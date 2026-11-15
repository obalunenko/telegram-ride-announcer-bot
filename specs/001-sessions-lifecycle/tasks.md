---

description: "Migrated task list for Sessions, Users, and Bot Lifecycle"
---

# Tasks: Sessions, Users, and Bot Lifecycle

**Input**: Design documents from `/specs/001-sessions-lifecycle/`

**Prerequisites**: plan.md, spec.md

**Status**: migrated. All tasks describe code that already exists, so they are marked complete. Gaps are listed at the end as notes, not as open tasks.

**Bugfix**: 2026-11-15 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-11-15 — BUG-002 Updated from bugfix patch

**Bugfix**: 2026-11-15 — BUG-003 Updated from bugfix patch

**Bugfix**: 2026-11-15 — BUG-004 Updated from bugfix patch

**Bugfix**: 2026-11-15 — BUG-005 Updated from bugfix patch

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
- [x] T014 [US3] Farewell to every session, session deletion, then the stop function in `Service.Shutdown` in `internal/service/service.go` (reopened — BUG-004; closed by T036)
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

## Phase 8: Bugfixes

- [x] T024 [US2] Key sessions by user and chat: `GetSession` and `DeleteSession` take the chat ID in `internal/repository/sessions/sessions.go` (BUG-001)
- [x] T025 [P] [US2] One state per user and chat with a `ChatID` field; drop `GetStateByUserID` in `internal/repository/states/states.go` (BUG-001)
- [x] T026 [US2] Pass the chat ID to `ops.GetSession`, `ops.CreateSession`, `ops.DeleteSession`; load the state by the session's state ID in `internal/ops/sessions.go` (depends on T024, T025; BUG-001)
- [x] T027 [US1] Use the message's chat in `setSessionMiddleware` in `internal/service/middleware.go` (depends on T026; BUG-001)
- [x] T028 [P] [US1] Case "same user in another chat" in `TestSetSessionMiddleware`; repository tests in `internal/repository/sessions/sessions_test.go` and `internal/repository/states/states_test.go` (BUG-001)

- [x] T029 [US4] `msgSomethingWentWrong` and `replyError` in `internal/service/helper.go` (BUG-002)
- [x] T030 [US4] Reply with the error message in every failure branch of `setSessionMiddleware` in `internal/service/middleware.go` (depends on T029; BUG-002)
- [x] T031 [P] [US4] Assert the error reply for failures and no reply for ignored updates in `TestSetSessionMiddleware` (BUG-002)

- [x] T032 [P] [US2] `Update` in `internal/repository/users/users.go` with tests in `internal/repository/users/users_test.go` (BUG-003)
- [x] T033 [US2] `UpdateUser` in `internal/ops/users.go` (depends on T032; BUG-003)
- [x] T034 [US2] `loadUser` gets, creates, or refreshes the user for `setSessionMiddleware` in `internal/service/middleware.go` (depends on T033; BUG-003)
- [x] T035 [P] [US2] Cases "changed profile" and "update user fails" in `TestSetSessionMiddleware` (BUG-003)

- [x] T036 [US3] Move the farewells into `sayGoodbye` and always run the stop functions in `Service.Shutdown` in `internal/service/service.go` (BUG-004)
- [x] T037 [P] [US3] `TestService_Shutdown_ListSessionsFails` in `internal/service/service_internal_test.go` (BUG-004)

- [x] T038 [US4] Log the panic stack as a field and reply with `replyError` in `panicRecovery` in `internal/service/middleware.go` (depends on T029; BUG-005)
- [x] T039 [P] [US4] Cases "message" and "update without a message" in `TestPanicRecovery` (BUG-005)

## Gaps (not tasks)

Follow-ups found during migration; see [spec.md](spec.md#gaps). Track them with `/speckit-bugfix-report` or a new `/speckit-specify` feature.

- ~~Session chat ID is fixed to the first message's chat.~~ Fixed by BUG-001.
- ~~Storage failures in `setSessionMiddleware` give the user no reply.~~ Fixed by BUG-002.
- ~~User profiles are never refreshed.~~ Fixed by BUG-003.
- ~~`Service.Shutdown` skips stopping polling when listing sessions fails.~~ Fixed by BUG-004.
- ~~Panics give the user no reply and print the stack with `fmt.Println`.~~ Fixed by BUG-005.
- No tests for `internal/ops/{sessions,users}.go` and the `users`, `sessions`, `states` repositories.
- Unused API: `states.ListStates`, `states.GetStateByID`, `users.List`, `states.State.Command`.
