---

description: "Migrated task list for My Trips"
---

# Tasks: My Trips

**Input**: Design documents from `/specs/004-my-trips/`

**Prerequisites**: plan.md, spec.md; depends on `specs/001-sessions-lifecycle` and `specs/003-trip-creation`

**Status**: migrated. All tasks describe code that already exists, so they are marked complete. Gaps are listed at the end as notes, not as open tasks.

**Bugfix**: 2026-11-15 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-11-15 — BUG-002 Updated from bugfix patch

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US2)

## Phase 1: Storage

- [x] T001 `ListTripsByUser` (creator match, not deleted) in `internal/repository/trips/trips.go`
- [x] T002 `ListTripsByUser` converting records to `models.Trip` with the requesting user as creator in `internal/ops/trips.go` (depends on T001)

## Phase 2: User Story 1 - List My Trips (P1)

- [x] T003 [US1] `myTripsHandler` renders each trip under `Your trips:` in `internal/service/handlers.go`
- [x] T004 [US1] Register `/mytrips` in `initHandlers` in `internal/service/service.go` and enable it in `commands` in `cmd/rideannouncer/main.go`
- [x] T005 [P] [US1] `/mytrips` after publishing in `TestService_PublishTrip`; list and render failures in `TestHandlers_Failure`; `TestHandlers_NoSession`

## Phase 3: User Story 2 - No Trips Yet (P2)

- [x] T006 [US2] Empty-list hint pointing to `/newtrip` in `myTripsHandler`
- [x] T007 [P] [US2] Empty list after cancel in `TestService_CancelTrip`

## Phase 4: Bugfixes

- [x] T008 [US1] `publishedTrips` in `internal/service/mytrips.go`; `myTripsHandler` lists only published trips (BUG-001)
- [x] T009 [P] [US1] `TestPublishedTrips` in `internal/service/mytrips_internal_test.go`; `TestService_MyTripsHidesDrafts`; `newPublishedTrip` for `TestHandlers_Failure` (BUG-001)

- [x] T010 [US1] `sortByStart` in `internal/service/mytrips.go`; sort in `myTripsHandler` (BUG-002)
- [x] T011 [P] [US1] `TestSortByStart` in `internal/service/mytrips_internal_test.go` (BUG-002)

## Gaps (not tasks)

Follow-ups found during migration; see [spec.md](spec.md#gaps). Track them with `/speckit-bugfix-report` or a new `/speckit-specify` feature.

- ~~Drafts are listed with published trips (003 Gap 1).~~ Fixed by BUG-001.
- ~~Random order from map iteration.~~ Fixed by BUG-002.
- Long lists can exceed the Telegram message limit.
- List and render failures give the user no reply.
- Past trips are never filtered out.
- No tests for several trips, drafts, or ordering.
