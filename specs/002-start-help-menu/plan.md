# Implementation Plan: Start, Help, and Command Menu

**Feature Directory**: `specs/002-start-help-menu` | **Branch**: N/A (migrated)

**Date**: 2026-11-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-start-help-menu/spec.md`

**Note**: Reverse-engineered from existing code by `/speckit-brownfield-migrate`; it describes what was built, not a proposal.

**Bugfix**: 2026-11-15 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-11-15 — BUG-002 Updated from bugfix patch

**Bugfix**: 2026-11-15 — BUG-003 Updated from bugfix patch

## Summary

Commands are `Cmd*` constants in `internal/service/service.go`, registered
with telego handlers in `initHandlers` and declared with a description and an
enabled flag in `cmd/rideannouncer/main.go`. `telegram.NewBot` reads the
bot's profile and menu and updates the name, description, and enabled
commands only when they differ. `/start` and `/help` reset the dialog state
and render embedded templates; unknown commands, free text, and stub
commands get fallback replies.

## Technical Context

**Language/Version**: Go 1.27, single module
`github.com/obalunenko/telegram-ride-announcer-bot` with committed `vendor/`.

**Primary Dependencies**: telego (`telegohandler.CommandEqual`,
`AnyCommand`, `AnyMessageWithText`; Bot API profile and menu methods),
`text/template` with `embed`, obalunenko/logger.

**Storage**: N/A beyond saving the session state (001).

**Testing**: `TestService_Commands`, `TestService_PublishTrip`,
`TestHandlers_Failure`, `TestHandlers_NoSession` in `internal/service/`;
`TestCommand`, `TestCommands_EnabledCommands`, `TestNewBot`,
`TestNewBot_Error`, `TestNewBot_UpdateOnStart` in
`internal/telegram/bot_test.go`; golden tests in
`internal/service/renderer/renderer_test.go` and `TestParseTemplate`. All
run against `internal/telegram/telegramtest`.

**Target Platform**: Long-running bot binary and Docker image. CGO disabled.

**Project Type**: Telegram bot, one binary with layered `internal/` packages.

**Performance Goals**: N/A; none established.

**Constraints**: Only enabled commands are registered in the menu; profile
updates must not stop startup.

**Scale/Scope**: About 560 lines; 2 working commands, 4 stub commands, 2
fallbacks, startup sync of 3 profile fields.

## Constitution Check

*Retrospective check of the existing implementation against constitution 1.0.1.*

- [x] I. Handlers live in `internal/service`; the Telegram wrapper in
      `internal/telegram` imports no other `internal/` package; `main` only
      declares the menu.
- [x] II. N/A: no storage of its own.
- [x] III. `/start` and `/help` reset the session to `StateStart`.
- [x] IV. Each command has a `Cmd*` constant, a handler in `initHandlers`,
      and a menu entry with an enabled flag; stub commands use
      `notImplementedHandler` and are disabled; `welcome` and `help` have
      golden tests.
- [x] V. Handlers and the bot wrapper are tested against `telegramtest` with
      table-driven testify tests.
- [x] VI. No generated code involved.
- [x] ~~`README.md` does not document the bot commands (Known Deviation).~~
      `README.md` lists the commands (BUG-001).
- [x] Known deviations are not extended.

## Project Structure

### Documentation (this feature)

```text
specs/002-start-help-menu/
├── spec.md
├── plan.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/rideannouncer/
└── main.go                 # botName, botDescription, commands (with enabled flags)
internal/
├── service/
│   ├── service.go          # Cmd* constants, New (loads templates), initHandlers
│   ├── handlers.go         # startHandler, helpHandler, notFound/unsupported/notImplementedHandler, textHandler
│   └── renderer/
│       ├── renderer.go     # Renderer: Welcome, Help, Trip
│       ├── templates/      # welcome.gotmpl, help.gotmpl
│       └── testdata/       # welcome.golden, help.golden
└── telegram/
    ├── bot.go              # Command, Commands, NewBot, updateOnStart, maybeUpdate*
    └── telegramtest/       # Fake Bot API
```

**Structure Decision**: The menu definition lives in `main` with the rest of
the wiring; `internal/telegram` owns the sync with Telegram; handlers read
the synced menu through `bot.Commands()` for `/help`.

## Design Decisions (as built)

1. **Handler order** (`initHandlers`): exact command handlers first, then
   `notFoundHandler` for `AnyCommand`, then `textHandler` for any text, which
   answers "Command not found" when the user is not in a dialog.
2. **Stub commands**: registered with `notImplementedHandler` and disabled in
   the menu, so they are hidden but still answer.
3. **Sync only on difference**: `maybeUpdateBotName` and
   `maybeUpdateDescriptionBot` compare with the values read at startup;
   `maybeUpdateCommands` compares the enabled commands with the registered
   ones by name and description and sends `setMyCommands` with the enabled
   list only when they differ.
4. **Non-fatal updates**: update failures are logged and the cached values
   stay unchanged; read failures in `NewBot` are fatal.
5. **Dialog reset** (BUG-002): `resetDialog` in `handlers.go` discards the
   draft trip through `discardDraft`, saves the session in `StateStart`, and
   sends "Your unfinished trip is discarded." when there was a draft. 003
   reuses `discardDraft` for `/newtrip` and `/cancel`. A failed reset or
   render replies with `replyError` (BUG-003).
6. **Templates**: embedded with `//go:embed templates/*.gotmpl` and parsed in
   `renderer.New`; parse errors are joined and fail `service.New`.

## Complexity Tracking

No constitution violations to justify. Gaps are listed in [spec.md](spec.md#gaps).
