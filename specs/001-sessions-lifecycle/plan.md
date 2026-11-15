# Implementation Plan: Sessions, Users, and Bot Lifecycle

**Feature Directory**: `specs/001-sessions-lifecycle` | **Branch**: N/A (migrated)

**Date**: 2026-11-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-sessions-lifecycle/spec.md`

**Note**: Reverse-engineered from existing code by `/speckit-brownfield-migrate`; it describes what was built, not a proposal.

**Bugfix**: 2026-11-15 — BUG-001 Updated from bugfix patch

## Summary

Every update passes through a telego middleware chain that recovers panics,
attaches the root context, loads or creates the user and their session, and
adds `user_id`/`chat_id` to the logger. A session joins three in-memory
records (user, session, dialog state) into `models.Session` in `internal/ops`.
`cmd/rideannouncer` wires the repositories, and on a signal the service sends
a farewell to every open session, deletes it, stops long polling, and waits
for in-flight updates.

## Technical Context

**Language/Version**: Go 1.27, single module
`github.com/obalunenko/telegram-ride-announcer-bot` with committed `vendor/`.

**Primary Dependencies**: telego (`telegohandler` middleware, long polling,
`telegoutil`), gofrs/uuid, obalunenko/logger, getenv, version.

**Storage**: In memory: `users` (map by user ID), `sessions` (~~map by user
ID~~ map by user ID and chat ID, BUG-001), `states` (map by state ID,
~~searched by user ID~~ unique per user and chat and loaded through the
session's state ID, BUG-001), each guarded by `sync.RWMutex`. All data is
lost on restart.

**Testing**: `internal/service/middleware_internal_test.go` (table-driven
`TestSetSessionMiddleware` with failing repository stubs,
`TestLoggerMiddleware`, `TestSetContextMiddleware`, `TestPanicRecovery`);
`TestHandlers_NoSession` in `handlers_internal_test.go`; farewell checks in
`service_internal_test.go` against `internal/telegram/telegramtest`.

**Target Platform**: Long-running bot binary and Docker image. CGO disabled.

**Project Type**: Telegram bot, one binary with layered `internal/` packages.

**Performance Goals**: N/A; none established.

**Constraints**: Shutdown has 10 seconds (`shutdownTimeout`); handlers may
run concurrently, so repositories must be safe for concurrent use.

**Scale/Scope**: About 1,000 lines in 10 files; no commands of its own.

## Constitution Check

*Retrospective check of the existing implementation against constitution 1.0.1.*

- [x] I. Layers hold: middleware calls `ops` only; `ops` converts repository
      records to `models`; repositories import only stdlib and uuid;
      `cmd/rideannouncer` only wires dependencies.
- [x] II. Users, sessions, and states sit behind `Repository` interfaces with
      `NewInMemory` implementations guarded by `sync.RWMutex`; the spec
      states that nothing survives a restart.
- [x] III. The session stores the `models.State` and draft trip; new users
      start in `StateStart`.
- [x] IV. N/A: no commands or templates of its own.
- [ ] V. Middleware behavior is tested against `telegramtest`, but
      `internal/ops/{sessions,users}.go` and the three repositories have no
      tests of their own (Known Deviation; spec Gap 6).
- [x] VI. No generated code involved.
- [ ] `README.md` does not document `RIDE_ANNOUNCER_TELEGRAM_TOKEN` (Known
      Deviation).
- [x] Known deviations are not extended.

## Project Structure

### Documentation (this feature)

```text
specs/001-sessions-lifecycle/
├── spec.md
├── plan.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/rideannouncer/
└── main.go             # Token, logger, repositories, signals, shutdown context
internal/
├── service/
│   ├── service.go      # New, Start, Shutdown, initHandlers (middleware order)
│   ├── middleware.go   # panicRecovery, setContext, setSession, logger
│   ├── context.go      # contextWithSession, sessionFromContext
│   ├── helper.go       # sendMessage to the session's chat
│   └── backends/       # Set of repositories
├── ops/
│   ├── users.go        # GetUser, CreateUser
│   └── sessions.go     # GetSession, CreateSession, UpdateSession, ListSessions, DeleteSession
├── repository/
│   ├── users/          # User records by Telegram ID
│   ├── sessions/       # Session records by user ID (chat ID, state ID)
│   └── states/         # Dialog state records (step, trip ID)
└── models/models.go    # User, Session, UserState, State
```

**Structure Decision**: The session is split across three repositories and
joined in `internal/ops`: a `sessions.Session` holds the chat and the state
ID, a `states.State` holds the step and the draft trip ID, and the trip is
loaded through `ops.GetTrip`. `ops.toSession` loads the state by the
session's `StateID` for both `GetSession` and `ListSessions` (BUG-001).

## Design Decisions (as built)

1. **Middleware order** (`initHandlers`): `panicRecovery` →
   `setContextMiddleware(ctx)` → `setSessionMiddleware` → `loggerMiddleware`,
   then command handlers and the text handler. Panic recovery comes first so
   it also covers the session lookup.
2. **Root context in updates**: `setContextMiddleware` replaces each update's
   context with the service context, so handlers get the configured logger.
3. **Session in context**: the loaded `*models.Session` is stored under a
   private key (`context.go`); handlers read it with `sessionFromContext` and
   save changes through `ops.UpdateSession`.
4. ~~**One record per user**: `sessions` is keyed by user ID and rejects a
   second session (`ErrAlreadyExists`); `states.CreateState` returns the
   existing state with `ErrAlreadyExists`.~~ **One record per user and
   chat** (BUG-001): `sessions` is keyed by user ID and chat ID and rejects a
   second session in the same chat (`ErrAlreadyExists`);
   `states.CreateState` returns the existing state of the user in the chat
   with `ErrAlreadyExists`, and `ops.CreateSession` reuses it.
5. **Shutdown context**: `main` waits for a signal, then calls `Shutdown`
   with `context.WithoutCancel(ctx)` and a 10-second timeout, because the
   root context is already canceled.
6. **Shutdown order**: farewell and session deletion for each session first,
   then the stop function from `initHandlers` cancels polling and calls
   `handler.StopWithContext`.

## Complexity Tracking

No constitution violations to justify. Gaps are listed in [spec.md](spec.md#gaps).
