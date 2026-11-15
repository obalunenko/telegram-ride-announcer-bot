---

description: "Migrated task list for Sessions, Users, and Bot Lifecycle"
---

# Tasks: Sessions, Users, and Bot Lifecycle

**Input**: Design documents from `/specs/001-sessions-lifecycle/`

**Prerequisites**: plan.md, spec.md

**Status**: migrated. All tasks describe code that already exists, so they are marked complete. Gaps are listed at the end as notes, not as open tasks.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)

## Phase 1: Domain and Storage

- [x] T001 Define `User`, `Session`, `UserState`, and `State` with `StateStart` in `internal/models/models.go`
- [x] T002 [P] Users repository with `Create`, `GetBuID`, `List`, and `ErrAlreadyExists`/`ErrNotFound` in `internal/repository/users/users.go`
- [x] T003 [P] Sessions repository keyed by user ID with `CreateSession`, `GetSessionByUserID`, `UpdateSession`, `ListSessions`, `DeleteSession` in `internal/repository/sessions/sessions.go`
- [x] T004 [P] States repository with one state per user (`CreateState` returns the existing one with `ErrAlreadyExists`) in `internal/repository/states/states.go`
- [x] T005 Repository set with nil checks in `internal/service/backends/backends.go`

## Phase 2: Operations

- [x] T006 `GetUser` and `CreateUser` in `internal/ops/users.go` (depends on T002)
- [x] T007 `GetSession`, `CreateSession` (state `StateStart`), `UpdateSession`, `ListSessions`, `DeleteSession` joining sessions, states, and trips in `internal/ops/sessions.go` (depends on T003, T004)

## Phase 3: User Story 1 - First Contact Registers the User (P1)

- [x] T008 [US1] Create the user and the session on the first message in `setSessionMiddleware` in `internal/service/middleware.go`
- [x] T009 [US1] Store and read the session in the context in `internal/service/context.go`
- [x] T010 [P] [US1] Test case "new user" in `TestSetSessionMiddleware` in `internal/service/middleware_internal_test.go`

## Phase 4: User Story 2 - Returning User Keeps Their Dialog (P1)

- [x] T011 [US2] Load the existing session, state, and draft trip in `setSessionMiddleware` through `ops.GetSession`
- [x] T012 [P] [US2] Test case "existing session" in `TestSetSessionMiddleware`

## Phase 5: User Story 3 - Graceful Shutdown With a Farewell (P2)

- [x] T013 [US3] Signal handling, `context.WithCancelCause`, and a 10-second shutdown context with `context.WithoutCancel` in `cmd/rideannouncer/main.go`
- [x] T014 [US3] Farewell to every session, session deletion, then the stop function in `Service.Shutdown` in `internal/service/service.go`
- [x] T015 [US3] Stop long polling and wait for handlers with `StopWithContext` in `initHandlers` in `internal/service/service.go`
- [x] T016 [P] [US3] Farewell assertions after `Shutdown` in `TestService_PublishTrip`, `TestService_CancelTrip`, `TestService_Commands` in `internal/service/service_internal_test.go`

## Phase 6: User Story 4 - Robust Update Handling (P3)

- [x] T017 [US4] Skip updates without a message or sender and messages from the bot in `setSessionMiddleware`
- [x] T018 [P] [US4] `panicRecovery` in `internal/service/middleware.go`
- [x] T019 [P] [US4] `loggerMiddleware` and `setContextMiddleware` in `internal/service/middleware.go`
- [x] T020 [US4] Register the middleware in order in `initHandlers` in `internal/service/service.go`
- [x] T021 [P] [US4] Ignored-update and storage-failure cases in `TestSetSessionMiddleware`; `TestPanicRecovery`, `TestLoggerMiddleware`, `TestSetContextMiddleware` in `internal/service/middleware_internal_test.go`
- [x] T022 [P] [US4] `TestHandlers_NoSession` in `internal/service/handlers_internal_test.go`

## Phase 7: Startup

- [x] T023 Read `RIDE_ANNOUNCER_TELEGRAM_TOKEN`, initialize the logger, create the bot, repositories, backends, and service in `cmd/rideannouncer/main.go`

## Gaps (not tasks)

Follow-ups found during migration; see [spec.md](spec.md#gaps). Track them with `/speckit-bugfix-report` or a new `/speckit-specify` feature.

- Session chat ID is fixed to the first message's chat.
- Storage failures in `setSessionMiddleware` give the user no reply.
- User profiles are never refreshed.
- `Service.Shutdown` skips stopping polling when listing sessions fails.
- Panics give the user no reply and print the stack with `fmt.Println`.
- No tests for `internal/ops/{sessions,users}.go` and the `users`, `sessions`, `states` repositories.
- Unused API: `states.ListStates`, `states.GetStateByID`, `users.List`, `states.State.Command`.
