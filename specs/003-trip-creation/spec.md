# Feature Specification: Trip Creation and Publication

**Feature Directory**: `specs/003-trip-creation`

**Feature Branch**: N/A (migrated from existing code on `master`; first built in #4, August 2023)

**Created**: 2026-11-15

**Status**: migrated

**Input**: Reverse-engineered by `/speckit-brownfield-migrate` from `internal/service/trips.go` (`newTripHandler`), `internal/service/handlers.go` (`createTrip` through `publishTrip`, `textHandler`), `internal/service/helper.go` (`renderTrip`), `internal/ops/trips.go`, `internal/repository/trips/`, and `internal/service/renderer/templates/trip.gotmpl`.

**Bugfix**: 2026-11-15 — BUG-001 Publishing marks the trip completed.

**Bugfix**: 2026-11-15 — BUG-002 Only `yes` publishes and only `no` cancels; other answers are asked again.

**Bugfix**: 2026-11-15 — BUG-003 `/newtrip` during the dialog starts over; `/cancel` ends the dialog on any step.

**Bugfix**: 2026-11-15 — BUG-004 The dialog asks for a real date and a start time in the bot's time zone.

**Bugfix**: 2026-11-15 — BUG-005 Failed steps reply with the error message; a failed pin is explained.

**Bugfix**: 2026-11-15 — BUG-006 Canceling ends the dialog in `StateStart`.

**Bugfix**: 2026-11-15 — BUG-007 The announcement and the pin go to the chat of the dialog (through 001 BUG-001).

**Bugfix**: 2026-11-15 — BUG-008 Every command that ends the dialog deletes its draft.

## Project Scope and Compatibility

- **Affected boundaries**: `internal/service/` (dialog steps and routing),
  `internal/service/renderer/` (`trip` template), `internal/ops/` (trip
  operations), `internal/repository/trips/` (trip storage),
  `internal/models/` (`Trip`, `State`), `cmd/rideannouncer/` (menu entry).
- **Bot commands and menu**: `/newtrip` ("create new trip", enabled);
  `/cancel` ("cancel the trip you are creating", enabled, BUG-003).
- **Dialog states**: `StateNewTrip` → `StateNewTripName` →
  `StateNewTripDate` → `StateNewTripTime` (BUG-004) →
  `StateNewTripDescription` → `StateNewTripConfirm` → `StateStart` on
  publish, or ~~`StateNewTrip`~~ `StateStart` on cancel (BUG-006).
- **Messages**: ~~"Please enter trip name"~~ "Please enter trip name. Send
  /cancel to stop." (BUG-003); "Your unfinished trip is discarded." and
  "There is nothing to cancel." (BUG-003); ~~`Your trip name "<name>". Please
  select date and time`~~ `Your trip name "<name>". Please select the date or
  enter it as DD.MM.YYYY.` with a `today`/`tomorrow` keyboard; "Please enter
  the start time as HH:MM, for example 09:30."; the re-ask messages for an
  unreadable or past date or time (BUG-004); "Please enter trip
  description"; the trip card followed by "Please confirm" with a
  `yes`/`no` keyboard; "Trip is published!" followed by the card; "Your trip
  is canceled. Thank you!".
- **Storage**: A draft trip is stored on the first step and updated on each
  step; cancel soft-deletes it. Nothing survives a restart.
- **Chats**: All prompts, the announcement, and the pin go to the session's
  chat, which is the chat of the dialog since 001 BUG-001 (BUG-007).
- **Configuration**: `RIDE_ANNOUNCER_TIMEZONE`, the IANA time zone of trip
  dates and times (default `UTC`) (BUG-004).
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
   `tomorrow`, or a date as `DD.MM.YYYY` that is not in the past, **Then**
   the bot saves that day, asks for the start time as `HH:MM`, and saves
   `StateNewTripTime` (BUG-004).
3a. **Given** the state `StateNewTripTime`, **When** the user sends a time
   such as `09:30` and that start is in the future, **Then** the bot saves
   the trip start, replies "Please enter trip description", and saves
   `StateNewTripDescription` (BUG-004).
3b. **Given** the date or time step, **When** the user sends text that is
   not a date or time, or a start that has passed, **Then** the bot says so
   and asks again without changing the state (BUG-004).
4. **Given** the state `StateNewTripDescription`, **When** the user sends
   `Easy pace`, **Then** the bot saves the description, replies with the trip
   card (`Title`, `Description`, `Date` as `Sun, 15 Nov 2026 09:30 UTC` in
   the bot's time zone (BUG-004), `Created By: @<username>`) followed
   by "Please confirm" with `yes` and `no` buttons, and saves
   `StateNewTripConfirm`.
5. **Given** any step, **When** the bot replies, **Then** the new state is
   already saved, so the user's answer is handled by the next step.
6. **Given** a user in the middle of the dialog, **When** they send
   `/newtrip`, **Then** the bot deletes the draft, replies "Your unfinished
   trip is discarded.", and starts the dialog again (BUG-003).

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
5. **Given** the bot cannot pin messages in the chat, **When** the trip is
   published, **Then** the trip stays published and the bot replies "The
   trip is published, but I couldn't pin it. To pin announcements, make me
   an admin who can pin messages." (BUG-005).
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
   Thank you!", and saves the state ~~`StateNewTrip`~~ `StateStart` without
   a trip (BUG-006); free text afterwards gets "Command not found".
2. **Given** a canceled trip, **When** the user lists their trips, **Then**
   the canceled trip is not listed.
3. **Given** any step of the dialog, **When** the user sends `/cancel`,
   **Then** the bot deletes the draft, replies "Your unfinished trip is
   discarded.", and saves the state `StateStart` (BUG-003).
4. **Given** a user who is not in the dialog, **When** they send `/cancel`,
   **Then** the bot replies "There is nothing to cancel." (BUG-003).

---

### Edge Cases

- ~~**Any answer other than exactly `no`** at the confirmation step publishes
  the trip, including `No`, `nope`, or `maybe`. (Gap 2.)~~ **Unclear answer
  at the confirmation step**: the bot asks again (BUG-002).
- ~~**`/newtrip` sent during the dialog** is handled as the answer to the
  current step; for example, it becomes the trip name. (Gap 3.)~~
  **`/newtrip` sent during the dialog** starts over (BUG-003).
- ~~**Free-text date**: any text is stored as the date; nothing is parsed or
  validated, and no time is asked. (Gap 4.)~~ **Unreadable or past date or
  time**: asked again (BUG-004).
- **Midnight in the bot's time zone**: `today` is the current day in
  `RIDE_ANNOUNCER_TIMEZONE`, not in UTC (BUG-004).
- **A step fails** (storage, rendering, or a Bot API call): the error is
  logged and ~~the user gets no reply. A pin failure happens after the
  announcement was sent and the state was saved. (Gap 5.)~~ the user gets
  "Sorry, something went wrong. Please try again."; the state is kept, so
  the user can answer again. A failed pin keeps the trip published and is
  explained (BUG-005).
- ~~**Free text after cancel**: the state `StateNewTrip` makes the next text
  message start a new trip dialog. (Gap 6.)~~ **Free text after cancel**:
  "Command not found" (BUG-006).
- **Dialog in a private chat**: the announcement is sent and pinned in the
  private chat~~, not in a group. (Gap 7.)~~; a dialog in a group is
  announced and pinned in that group, even when the user talked to the bot
  privately before (BUG-007).
- **Interrupted dialog** (`/start`, `/help`, a failure, or a restart): ~~the
  draft trip stays in storage. (Gap 8.)~~ `/start`, `/help`, `/newtrip`, and
  `/cancel` delete the draft; a failure keeps the dialog; a restart loses all
  data (BUG-008).
- **User without a username**: the card shows `Created By: @`. (Gap 9.)
- **Draft owned by another user** in the session: the description and publish
  steps fail with "is not the creator of the trip".
- **Unexpected state** in `createTrip` (for example `StateNewTripPublish`):
  logged; nothing happens.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `/newtrip` MUST start the trip dialog ~~for a user who is not in
  it~~, creating a draft trip owned by that user; when a dialog is unfinished,
  it MUST delete that draft and reply "Your unfinished trip is discarded."
  first (BUG-003).
- **FR-002**: The dialog MUST ask for the name, then the date, then the start
  time (BUG-004), then the description, saving each answer to the draft
  trip.
- **FR-003**: The date step MUST offer `today` and `tomorrow` as one-time
  keyboard buttons and MUST accept ~~any text~~ `today`, `tomorrow`, or
  `DD.MM.YYYY` that is not in the past; the time step MUST accept `HH:MM`
  for a start in the future; anything else MUST be asked again (BUG-004).
- **FR-015**: Trip dates and times MUST be read and shown in the time zone
  from `RIDE_ANNOUNCER_TIMEZONE` (default `UTC`); the trip stores its start
  as a point in time (BUG-004).
- **FR-004**: After the description, the bot MUST show the trip card and ask
  to confirm with `yes` and `no` buttons.
- **FR-005**: Each step MUST save the new dialog state before sending the
  prompt for the next answer.
- **FR-006**: The answer `no` (ignoring case and surrounding spaces, BUG-002)
  MUST delete the draft trip, reply "Your trip is canceled. Thank you!", and
  set the state to ~~`StateNewTrip`~~ `StateStart` (BUG-006).
- **FR-007**: ~~Any other answer~~ The answer `yes` (ignoring case and
  surrounding spaces, BUG-002) MUST publish the trip: send `Trip is
  published!` with the trip card to the chat of the dialog (BUG-007), pin
  that message, and
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
  A command that ends an unfinished dialog MUST delete its draft (BUG-008).
- **FR-012**: Publishing ~~SHOULD mark the trip as completed; the current code
  requests it but does not store it. (Gap 1.)~~ MUST mark the trip as
  completed in storage (BUG-001).
- **FR-013**: A failed step, `/newtrip`, or `/cancel` MUST log the error and
  reply "Sorry, something went wrong. Please try again." without advancing
  the state; a failed pin MUST NOT fail publishing and MUST be explained to
  the user (BUG-005).
- **FR-014**: `/cancel` MUST end the trip dialog on any step: delete the
  draft, reply "Your unfinished trip is discarded.", and set the state to
  `StateStart`; outside of the dialog it MUST reply "There is nothing to
  cancel." (BUG-003).

### Key Entities *(include if feature involves data)*

- **Trip**: a planned bicycle trip: name, ~~date (free text)~~ start date and
  time (BUG-004), description,
  creator, creation and update time, completed flag, and deletion time
  (soft delete).
- **Draft trip**: the trip referenced by the user's dialog state until it is
  published or canceled.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can create and publish a trip in one dialog of ~~five~~
  six messages (`/newtrip`, name, date, time (BUG-004), description,
  `yes`).
- **SC-002**: Every published trip is announced and pinned in the chat of
  the dialog.
- **SC-003**: A canceled trip does not appear in the user's trip list.
- **SC-004**: After each answer, the saved state matches the next expected
  step.

## Assumptions

- ~~Dates are stored as entered; `today` and `tomorrow` are not converted to
  calendar dates.~~ `today` and `tomorrow` are resolved when the user
  answers, in the bot's time zone (BUG-004).
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
3. ~~`/newtrip` during the dialog is taken as the answer to the current step.~~
   Fixed by BUG-003.
4. ~~The date is free text: no parsing, no validation, and no time step,
   although the prompt asks for "date and time".~~ Fixed by BUG-004.
5. ~~Failed steps give the user no reply; a pin failure is reported only after
   the announcement was sent and the state saved.~~ Fixed by BUG-005.
6. ~~After cancel, the state `StateNewTrip` makes the next free text start a
   new dialog.~~ Fixed by BUG-006.
7. ~~The announcement goes to the session's chat (001 Gap 1), so a dialog in a
   private chat is announced and pinned there.~~ Fixed by 001 BUG-001;
   tested by BUG-007.
8. ~~Abandoned drafts stay in storage.~~ Fixed by 002 BUG-002 and BUG-003;
   recorded by BUG-008.
9. Users without a username appear as `@`; names and descriptions have no
   length limits.
10. Planned steps in the `createTrip` comments were never built: a track link
    or file, a photo, and subscribing the creator. `StateNewTripTime` and
    `StateNewTripPublish` are unused (constitution Known Deviation).
