---

description: "Migrated task list for Trip Creation and Publication"
---

# Tasks: Trip Creation and Publication

**Input**: Design documents from `/specs/003-trip-creation/`

**Prerequisites**: plan.md, spec.md; depends on `specs/001-sessions-lifecycle` and `specs/002-start-help-menu`

**Status**: migrated. All tasks describe code that already exists, so they are marked complete. Gaps are listed at the end as notes, not as open tasks.

**Bugfix**: 2026-11-15 — BUG-001 Updated from bugfix patch

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

## Phase 1: Domain and Storage

- [x] T001 `Trip` and the trip states `StateNewTrip` through `StateNewTripConfirm` in `internal/models/models.go`; `state_string.go` via `make codegen`
- [x] T002 [P] Trips repository with `CreateTrip`, `GetTripByID`, `UpdateTrip`, `DeleteTrip` (soft delete), `ListTrips`, `ListTripsByUser` in `internal/repository/trips/trips.go`
- [x] T003 ⚠️ Reopened `CreateTrip`, `GetTrip` (with the creator), `UpdateTrip`, `DeleteTrip` in `internal/ops/trips.go` (depends on T002) (reopened — BUG-001; closed by T018)
- [x] T004 [P] Trip card template in `internal/service/renderer/templates/trip.gotmpl`, `Trip` in `internal/service/renderer/renderer.go`, `renderTrip` in `internal/service/helper.go`

## Phase 2: User Story 1 - Create a Trip in a Dialog (P1)

- [x] T005 [US1] `/newtrip` handler that resets to `StateNewTrip` and calls `createTrip` in `internal/service/trips.go`
- [x] T006 [US1] `createTrip` state switch and `replyInTrip` (save, then send) in `internal/service/handlers.go`
- [x] T007 [US1] `startTrip` creates the draft and asks for the name in `internal/service/handlers.go`
- [x] T008 [US1] `setTripName` with the `today`/`tomorrow` keyboard, `setTripDate`, and `setTripDescription` with the card and the `yes`/`no` keyboard in `internal/service/handlers.go`
- [x] T009 [US1] Route trip states from `textHandler` to `newTripHandler` in `internal/service/handlers.go`
- [x] T010 [US1] Register `/newtrip` in `initHandlers` in `internal/service/service.go` and enable it in `commands` in `cmd/rideannouncer/main.go`
- [x] T011 [P] [US1] Step-by-step prompts and saved states in `TestService_PublishTrip` in `internal/service/service_internal_test.go`
- [x] T012 [P] [US1] Failure cases for each step in `TestService_CreateTrip`; new-trip cases in `TestHandlers_Failure` and `TestHandlers_NoSession` in `internal/service/handlers_internal_test.go`
- [x] T013 [P] [US1] Golden test `TestTemplates_Trip` in `internal/service/renderer/renderer_test.go`

## Phase 3: User Story 2 - Publish the Trip (P1)

- [x] T014 [US2] ⚠️ Reopened `confirmTrip` and `publishTrip` (creator check, announcement, pin, reset to `StateStart`) in `internal/service/handlers.go` (reopened — BUG-001; closed by T018)
- [x] T015 [P] [US2] Announcement, `pinChatMessage`, and final state in `TestService_PublishTrip`

## Phase 4: User Story 3 - Cancel the Trip (P2)

- [x] T016 [US3] `cancelTrip` soft-deletes the draft and resets to `StateNewTrip` in `internal/service/handlers.go`
- [x] T017 [P] [US3] `TestService_CancelTrip` in `internal/service/service_internal_test.go`

## Phase 5: Bugfixes

- [x] T018 [US2] Pass `Completed` in `ops.UpdateTrip`; add `Completed` to `models.Trip` and map it in `GetTrip` and `ListTripsByUser` in `internal/ops/trips.go`, `internal/models/models.go` (BUG-001)
- [x] T019 [P] [US2] Tests for the trips repository in `internal/repository/trips/trips_test.go` and for `internal/ops/trips_test.go`; completed check in `TestService_PublishTrip` (BUG-001)

## Gaps (not tasks)

Follow-ups found during migration; see [spec.md](spec.md#gaps). Track them with `/speckit-bugfix-report` or a new `/speckit-specify` feature.

- ~~Bug: `ops.UpdateTrip` drops `Completed`, so published trips are never marked completed; `models.Trip` has no completed field.~~ Fixed by BUG-001.
- Any confirmation answer except exactly `no` publishes.
- `/newtrip` during the dialog becomes the answer to the current step.
- Free-text date without validation or time.
- Failed steps give the user no reply; pin failure after the announcement.
- Free text after cancel starts a new dialog.
- Announcement goes to the session's chat (001 Gap 1).
- Abandoned drafts stay in storage.
- `@` without a username; no length limits.
- Planned steps never built (track, photo, subscribing the creator); `StateNewTripTime` and `StateNewTripPublish` unused.
