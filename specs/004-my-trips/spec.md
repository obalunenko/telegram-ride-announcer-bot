# Feature Specification: My Trips

**Feature Directory**: `specs/004-my-trips`

**Feature Branch**: N/A (migrated from existing code on `master`)

**Created**: 2026-11-15

**Status**: migrated

**Input**: Reverse-engineered by `/speckit-brownfield-migrate` from `internal/service/handlers.go` (`myTripsHandler`), `internal/ops/trips.go` (`ListTripsByUser`), `internal/repository/trips/` (`ListTripsByUser`), and `internal/service/helper.go` (`renderTrip`).

**Bugfix**: 2026-11-15 — BUG-001 Only published trips are listed.

**Bugfix**: 2026-11-15 — BUG-002 Trips are listed by start, the nearest first.

## Project Scope and Compatibility

- **Affected boundaries**: `internal/service/` (handler), `internal/ops/`
  (`ListTripsByUser`), `internal/repository/trips/` (`ListTripsByUser`),
  `cmd/rideannouncer/` (menu entry).
- **Bot commands and menu**: `/mytrips` ("show trips you've created",
  enabled).
- **Dialog states**: N/A. `/mytrips` does not change the dialog state.
- **Messages**: `Your trips:` followed by one trip card per trip (the `trip`
  template from 003); "You have no trips yet. Use /newtrip command to create
  a new trip."
- **Storage**: Reads trips; nothing is written.
- **Chats**: The reply goes to the session's chat (see 001).
- **Configuration**: N/A.
- **Release impact**: N/A (existing behavior).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - List My Trips (Priority: P1)

A user sends `/mytrips` and gets the trips they created, each shown as a trip
card.

**Why this priority**: It is the only way for a creator to see their trips.

**Independent Test**: Publish a trip, send `/mytrips`, and check the reply
(`TestService_PublishTrip`).

**Acceptance Scenarios**:

1. **Given** a user who created a trip "Morning ride", **When** they send
   `/mytrips`, **Then** the bot replies with a message that starts with
   `Your trips:` and contains `Title: Morning ride`.
2. **Given** a user with several trips, **When** they send `/mytrips`,
   **Then** each trip appears as a card, separated by a blank line, ordered
   by start with the nearest first (BUG-002).
3. **Given** a user in the middle of `/newtrip`, **When** they send
   `/mytrips`, **Then** the dialog state is unchanged and the next answer
   continues the dialog; the draft of that dialog is not listed (BUG-001).

---

### User Story 2 - No Trips Yet (Priority: P2)

A user who has no trips sends `/mytrips` and is told how to create one.

**Why this priority**: The empty case points new users to `/newtrip`.

**Independent Test**: Cancel a trip, send `/mytrips`, and check the reply
(`TestService_CancelTrip`).

**Acceptance Scenarios**:

1. **Given** a user without trips, or whose only trip was canceled, **When**
   they send `/mytrips`, **Then** the bot replies "You have no trips yet. Use
   /newtrip command to create a new trip."

---

### Edge Cases

- **Drafts**: ~~the draft of the current dialog and abandoned drafts are
  listed like published trips, possibly with an empty name. (Gap 1.)~~ not
  listed (BUG-001).
- **Order**: ~~trips come in no fixed order and can change between calls.
  (Gap 2.)~~ by start, the nearest first; trips with the same start keep the
  order of creation (BUG-002).
- **Many trips**: the reply can exceed Telegram's message length limit; the
  send fails and the user gets nothing. (Gap 3.)
- **Listing or rendering fails**: logged; the user gets no reply. (Gap 4.)
- **Past trips**: never removed from the list. (Gap 5.)
- **Trips of other users**: never listed.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `/mytrips` MUST list the trips created by the requesting user
  that are not deleted and are published (BUG-001).
- **FR-002**: Each trip MUST be shown with the `trip` card (title,
  description, date, creator), after the header `Your trips:`, ordered by
  start with the nearest first (BUG-002).
- **FR-003**: When the user has no trips, the bot MUST reply with the hint to
  use `/newtrip`.
- **FR-004**: `/mytrips` MUST NOT change the user's dialog state.
- **FR-005**: Trips created by other users MUST NOT be listed.
- **FR-006**: When listing or rendering fails, the handler MUST log the
  error and send no reply.
- **FR-007**: `/mytrips` MUST be enabled in the Telegram menu as "show trips
  you've created".

### Key Entities *(include if feature involves data)*

- **Trip**: as defined in 003; listed by creator, excluding soft-deleted
  trips.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After publishing a trip, the creator sees it in `/mytrips`.
- **SC-002**: After canceling their only trip, the user sees the "no trips
  yet" hint.
- **SC-003**: `/mytrips` never shows another user's trips.

## Assumptions

- A user has few enough trips for one Telegram message.
- Listing includes every non-deleted trip, whatever its date.
- Depends on the Telegram Bot API (`sendMessage`).

## Gaps

Found during migration; none of them is fixed by this spec.

1. ~~Drafts, including the current dialog's draft and abandoned ones, are
   listed with published trips and cannot be told apart (003 Gap 1).~~ Fixed
   by BUG-001.
2. ~~The order comes from map iteration and is random.~~ Fixed by BUG-002.
3. Long lists can exceed Telegram's 4096-character message limit, so the
   reply fails.
4. Listing or rendering failures give the user no reply.
5. Past trips are never filtered out; dates are free text.
6. No tests for several trips, drafts in the list, or ordering.
