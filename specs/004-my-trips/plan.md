# Implementation Plan: My Trips

**Feature Directory**: `specs/004-my-trips` | **Branch**: N/A (migrated)

**Date**: 2026-11-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-my-trips/spec.md`

**Note**: Reverse-engineered from existing code by `/speckit-brownfield-migrate`; it describes what was built, not a proposal.

## Summary

`myTripsHandler` asks `ops.ListTripsByUser` for the requesting user's trips;
the repository returns every trip whose creator matches and that is not
soft-deleted. The handler renders each trip with the `trip` template from
003, joins the cards under `Your trips:`, and sends one message to the
session's chat, or a hint to use `/newtrip` when the list is empty.

## Technical Context

**Language/Version**: Go 1.27, single module
`github.com/obalunenko/telegram-ride-announcer-bot` with committed `vendor/`.

**Primary Dependencies**: telego (`SendMessage` through `sendMessage`),
`text/template` (`trip` template).

**Storage**: Reads `internal/repository/trips` (map by trip ID, filtered by
creator and `DeletedAt`).

**Testing**: `/mytrips` with one published trip in `TestService_PublishTrip`,
the empty list in `TestService_CancelTrip`, `my trips: list trips` and
`my trips: render trip` in `TestHandlers_Failure`, and `TestHandlers_NoSession`.

**Target Platform**: Long-running bot binary and Docker image. CGO disabled.

**Project Type**: Telegram bot, one binary with layered `internal/` packages.

**Performance Goals**: N/A; none established.

**Constraints**: One reply message; no paging.

**Scale/Scope**: About 110 lines; one command.

## Constitution Check

*Retrospective check of the existing implementation against constitution 1.0.1.*

- [x] I. The handler reaches trips only through `ops.ListTripsByUser`.
- [x] II. Reads go through `trips.Repository`.
- [x] III. The handler does not touch the dialog state.
- [x] IV. `/mytrips` has its `Cmd*` constant, handler, and enabled menu
      entry; it reuses the `trip` template, which has a golden test.
- [ ] V. The handler is tested against `telegramtest`, but
      `ops.ListTripsByUser` and the repository method have no tests of their
      own (Known Deviation).
- [x] VI. No generated code involved.
- [ ] `README.md` does not document `/mytrips` (Known Deviation).
- [x] Known deviations are not extended.

## Project Structure

### Documentation (this feature)

```text
specs/004-my-trips/
├── spec.md
├── plan.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/rideannouncer/main.go     # /mytrips menu entry
internal/
├── service/
│   ├── handlers.go           # myTripsHandler
│   └── helper.go             # renderTrip, sendMessage
├── ops/trips.go              # ListTripsByUser (sets the requesting user as creator)
└── repository/trips/trips.go # ListTripsByUser (creator match, not deleted)
```

**Structure Decision**: The list reuses the trip card from 003 instead of a
list-specific template.

## Design Decisions (as built)

1. **Creator filter in the repository**: `ListTripsByUser` scans all trips
   and keeps those with a matching `CreatedBy` and no `DeletedAt`.
2. **Creator from the session**: `ops.ListTripsByUser` sets the requesting
   user as the creator of every listed trip instead of loading it.
3. **One message**: all cards are concatenated into one reply; a render
   failure aborts the whole reply.

## Complexity Tracking

No new violations to justify. Gaps are listed in [spec.md](spec.md#gaps).
