# Feature Specification: Trip Creation and Publication

**Feature Directory**: `specs/003-trip-creation`

**Feature Branch**: N/A (migrated from existing code on `master`; first built in #4, August 2023)

**Created**: 2026-11-15

**Status**: migrated

**Input**: Reverse-engineered by `/speckit-brownfield-migrate` from `internal/service/trips.go` (`newTripHandler`), `internal/service/handlers.go` (`createTrip` through `publishTrip`, `textHandler`), `internal/service/helper.go` (`renderTrip`), `internal/ops/trips.go`, `internal/repository/trips/`, and `internal/service/renderer/templates/trip.gotmpl`.

**Bugfix**: 2026-11-15 — BUG-001 Publishing marks the trip completed.

**Bugfix**: 2026-11-15 — BUG-002 Only `yes` publishes and only `no` cancels; other answers are asked again.

## Project Scope and Compatibility

- **Affected boundaries**: `internal/service/` (dialog steps and routing),
  `internal/service/renderer/` (`trip` template), `internal/ops/` (trip
  operations), `internal/repository/trips/` (trip storage),
  `internal/models/` (`Trip`, `State`), `cmd/rideannouncer/` (menu entry).
- **Bot commands and menu**: `/newtrip` ("create new trip", enabled).
- **Dialog states**: `StateNewTrip` → `StateNewTripName` →
  `StateNewTripDate` → `StateNewTripDescription` → `StateNewTripConfirm` →
  `StateStart` on publish, or `StateNewTrip` on cancel.
- **Messages**: "Please enter trip name"; `Your trip name "<name>". Please
  select date and time` with a `today`/`tomorrow` keyboard; "Please enter
  trip description"; the trip card followed by "Please confirm" with a
  `yes`/`no` keyboard; "Trip is published!" followed by the card; "Your trip
  is canceled. Thank you!".
- **Storage**: A draft trip is stored on the first step and updated on each
  step; cancel soft-deletes it. Nothing survives a restart.
- **Chats**: All prompts, the announcement, and the pin go to the session's
  chat (see 001).
- **Configuration**: N/A.
- **Release impact**: N/A (existing behavior).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a Trip in a Dialog (Priority: P1)

A user sends `/newtrip`, and the bot asks for the trip name, the date, and the
description one at a time, then shows the trip card and asks to confirm.

**Why this priority**: Creating trips is the bot's main purpose.

**Independent Test**: Run the dialog against the fake Bot API and check each
prompt and the saved state after each answer (`TestService_PublishTrip`,
`TestService_CancelTrip`).

**Acceptance Scenarios**:

1. **Given** a user who is not in a dialog, **When** they send `/newtrip`,
   **Then** the bot creates a draft trip owned by them, replies "Please enter
   trip name", and saves the state `StateNewTripName`.
2. **Given** the state `StateNewTripName`, **When** the user sends
   `Morning ride`, **Then** the bot saves the name, replies `Your trip name
   "Morning ride". Please select date and time` with `today` and `tomorrow`
   buttons, and saves `StateNewTripDate`.
3. **Given** the state `StateNewTripDate`, **When** the user sends `today`,
   **Then** the bot saves the date as entered, replies "Please enter trip
   description", and saves `StateNewTripDescription`.
4. **Given** the state `StateNewTripDescription`, **When** the user sends
   `Easy pace`, **Then** the bot saves the description, replies with the trip
   card (`Title`, `Description`, `Date`, `Created By: @<username>`) followed
   by "Please confirm" with `yes` and `no` buttons, and saves
   `StateNewTripConfirm`.
5. **Given** any step, **When** the bot replies, **Then** the new state is
   already saved, so the user's answer is handled by the next step.

---

### User Story 2 - Publish the Trip (Priority: P1)

The user confirms the trip, and the bot announces it in the chat and pins the
announcement.

**Why this priority**: Announcing the trip is the point of creating it.

**Independent Test**: Answer `yes` at the confirmation step and check the
announcement, the `pinChatMessage` call, and the state
(`TestService_PublishTrip`).

**Acceptance Scenarios**:

1. **Given** the state `StateNewTripConfirm`, **When** the user answers
   `yes` (in any case, with or without surrounding spaces, BUG-002), **Then**
   the bot sends `Trip is published!` followed by the trip card to the
   session's chat and pins that message.
4. **Given** the state `StateNewTripConfirm`, **When** the user answers
   anything other than `yes` or `no`, **Then** the bot replies "Please answer
   "yes" to publish the trip or "no" to cancel it." with the same buttons and
   the state stays `StateNewTripConfirm` (BUG-002).
2. **Given** the trip was published, **When** the user sends another message,
   **Then** their state is `StateStart` and they can start a new trip.
3. **Given** the trip was published, **When** it is loaded from storage,
   **Then** it is completed; a trip that is not completed is a draft
   (BUG-001).

---

### User Story 3 - Cancel the Trip (Priority: P2)

The user answers `no` at the confirmation step, and the bot discards the
draft.

**Why this priority**: Users can back out of a trip they do not want to
announce.

**Independent Test**: Answer `no` at the confirmation step and check the
reply, the state, and that `/mytrips` shows no trips
(`TestService_CancelTrip`).

**Acceptance Scenarios**:

1. **Given** the state `StateNewTripConfirm`, **When** the user answers `no`,
   **Then** the bot deletes the draft trip, replies "Your trip is canceled.
   Thank you!", and saves the state `StateNewTrip` without a trip.
2. **Given** a canceled trip, **When** the user lists their trips, **Then**
   the canceled trip is not listed.

---

### Edge Cases

- ~~**Any answer other than exactly `no`** at the confirmation step publishes
  the trip, including `No`, `nope`, or `maybe`. (Gap 2.)~~ **Unclear answer
  at the confirmation step**: the bot asks again (BUG-002).
- **`/newtrip` sent during the dialog** is handled as the answer to the
  current step; for example, it becomes the trip name. (Gap 3.)
- **Free-text date**: any text is stored as the date; nothing is parsed or
  validated, and no time is asked. (Gap 4.)
- **A step fails** (storage, rendering, or a Bot API call): the error is
  logged and the user gets no reply. A pin failure happens after the
  announcement was sent and the state was saved. (Gap 5.)
- **Free text after cancel**: the state `StateNewTrip` makes the next text
  message start a new trip dialog. (Gap 6.)
- **Dialog in a private chat**: the announcement is sent and pinned in the
  private chat, not in a group. (Gap 7.)
- **Interrupted dialog** (`/start`, `/help`, a failure, or a restart): the
  draft trip stays in storage. (Gap 8.)
- **User without a username**: the card shows `Created By: @`. (Gap 9.)
- **Draft owned by another user** in the session: the description and publish
  steps fail with "is not the creator of the trip".
- **Unexpected state** in `createTrip` (for example `StateNewTripPublish`):
  logged; nothing happens.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `/newtrip` MUST start the trip dialog for a user who is not in
  it, creating a draft trip owned by that user.
- **FR-002**: The dialog MUST ask for the name, then the date, then the
  description, saving each answer to the draft trip.
- **FR-003**: The date step MUST offer `today` and `tomorrow` as one-time
  keyboard buttons and MUST accept any text.
- **FR-004**: After the description, the bot MUST show the trip card and ask
  to confirm with `yes` and `no` buttons.
- **FR-005**: Each step MUST save the new dialog state before sending the
  prompt for the next answer.
- **FR-006**: The answer `no` (ignoring case and surrounding spaces, BUG-002)
  MUST delete the draft trip, reply "Your trip is canceled. Thank you!", and
  set the state to `StateNewTrip`.
- **FR-007**: ~~Any other answer~~ The answer `yes` (ignoring case and
  surrounding spaces, BUG-002) MUST publish the trip: send `Trip is
  published!` with the trip card to the session's chat, pin that message, and
  set the state to `StateStart`. Any other answer MUST be asked again without
  changing the state (BUG-002).
- **FR-008**: The description and publish steps MUST refuse a draft trip that
  the user did not create.
- **FR-009**: The trip card MUST show the title, description, date, and the
  creator as `@<username>`, rendered from the `trip` template.
- **FR-010**: Free text from a user in a trip state MUST be routed to the trip
  dialog; free text from a user in `StateStart` MUST get the "Command not
  found" reply.
- **FR-011**: A deleted trip MUST NOT be returned by trip lookups or lists.
- **FR-012**: Publishing ~~SHOULD mark the trip as completed; the current code
  requests it but does not store it. (Gap 1.)~~ MUST mark the trip as
  completed in storage (BUG-001).
- **FR-013**: A failed step MUST log the error.

### Key Entities *(include if feature involves data)*

- **Trip**: a planned bicycle trip: name, date (free text), description,
  creator, creation and update time, completed flag, and deletion time
  (soft delete).
- **Draft trip**: the trip referenced by the user's dialog state until it is
  published or canceled.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can create and publish a trip in one dialog of five
  messages (`/newtrip`, name, date, description, `yes`).
- **SC-002**: Every published trip is announced and pinned in the chat of
  the dialog.
- **SC-003**: A canceled trip does not appear in the user's trip list.
- **SC-004**: After each answer, the saved state matches the next expected
  step.

## Assumptions

- Dates are stored as entered; `today` and `tomorrow` are not converted to
  calendar dates.
- The bot has the right to pin messages in the chat where the trip is
  published.
- Users do not run two trip dialogs at once; there is one dialog state per
  user.
- Depends on the Telegram Bot API (`sendMessage`, `pinChatMessage`).

## Gaps

Found during migration; none of them is fixed by this spec.

1. ~~Published trips are never marked completed: `ops.UpdateTrip` does not pass
   `Completed` to the repository, and `models.Trip` has no completed field,
   so drafts and published trips cannot be told apart.~~ Fixed by BUG-001.
2. ~~Any confirmation answer except exactly `no` publishes the trip.~~ Fixed by
   BUG-002.
3. `/newtrip` during the dialog is taken as the answer to the current step.
4. The date is free text: no parsing, no validation, and no time step,
   although the prompt asks for "date and time".
5. Failed steps give the user no reply; a pin failure is reported only after
   the announcement was sent and the state saved.
6. After cancel, the state `StateNewTrip` makes the next free text start a
   new dialog.
7. The announcement goes to the session's chat (001 Gap 1), so a dialog in a
   private chat is announced and pinned there.
8. Abandoned drafts stay in storage.
9. Users without a username appear as `@`; names and descriptions have no
   length limits.
10. Planned steps in the `createTrip` comments were never built: a track link
    or file, a photo, and subscribing the creator. `StateNewTripTime` and
    `StateNewTripPublish` are unused (constitution Known Deviation).
