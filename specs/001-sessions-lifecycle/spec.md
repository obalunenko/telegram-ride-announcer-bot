# Feature Specification: Sessions, Users, and Bot Lifecycle

**Feature Directory**: `specs/001-sessions-lifecycle`

**Feature Branch**: N/A (migrated from existing code on `master`; first built in #4 and #12, August 2023)

**Created**: 2026-11-15

**Status**: migrated

**Input**: Reverse-engineered by `/speckit-brownfield-migrate` from `internal/service/middleware.go`, `internal/service/service.go`, `internal/ops/{sessions,users}.go`, `internal/repository/{users,sessions,states}/`, and `cmd/rideannouncer/main.go`.

**Bugfix**: 2026-11-15 — BUG-001 One session and dialog state per user and chat instead of per user.

**Bugfix**: 2026-11-15 — BUG-002 Storage failures in the session middleware get an error reply.

**Bugfix**: 2026-11-15 — BUG-003 The stored user profile follows the sender's Telegram profile.

## Project Scope and Compatibility

- **Affected boundaries**: `cmd/rideannouncer/` (startup, signals, shutdown
  timeout), `internal/service/` (middleware chain, session context,
  `Start`/`Shutdown`), `internal/ops/` (user and session operations),
  `internal/repository/{users,sessions,states}/` (storage),
  `internal/models/` (`User`, `Session`, `UserState`).
- **Bot commands and menu**: N/A. Every command and text handler depends on
  the session this feature provides.
- **Dialog states**: Creates the user's state as `StateStart` on first
  contact and restores the saved state on every later message.
- **Messages**: The farewell message `I'm going to sleep. Bye, <first name>!`
  on shutdown.
- **Storage**: Users, sessions, and states in memory. Nothing survives a
  restart.
- **Chats**: Private and group chats. ~~The session keeps the chat of the
  user's first message.~~ Each chat of a user has its own session and dialog
  (BUG-001).
- **Configuration**: `RIDE_ANNOUNCER_TELEGRAM_TOKEN` (required).
- **Release impact**: N/A (existing behavior).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - First Contact Registers the User (Priority: P1)

A Telegram user writes to the bot for the first time. The bot remembers who
they are and opens a session for them, so every command can address them by
name and keep track of their dialog.

**Why this priority**: Every command and dialog needs a session; without it,
handlers do nothing.

**Independent Test**: Send any message from a new user through the middleware
chain and check that the next handler receives a session with the user's
profile, the chat ID, and `StateStart` (`TestSetSessionMiddleware`, case
"new user").

**Acceptance Scenarios**:

1. **Given** a user the bot has never seen, **When** they send any message,
   **Then** the bot stores their Telegram ID, username, first name, and last
   name, creates a dialog state `StateStart`, and creates a session bound to
   the chat of that message.
2. **Given** a new user, **When** the session is created, **Then** the
   handler for the message runs with that session in its context.
3. **Given** a user with a session in one chat, **When** they write in
   another chat, **Then** the bot creates a separate session in `StateStart`
   for that chat, and the dialog in the first chat is unchanged (BUG-001).

---

### User Story 2 - Returning User Keeps Their Dialog (Priority: P1)

A user who is in the middle of a dialog sends the next message. The bot
restores their session with the saved state and draft trip, so the dialog
continues where it stopped.

**Why this priority**: Multi-step dialogs such as trip creation depend on it.

**Independent Test**: Save a session in some state, send a message, and check
that the next handler receives that state (`TestSetSessionMiddleware`, case
"existing session").

**Acceptance Scenarios**:

1. **Given** a user with a saved session in state X, **When** they send a
   message, **Then** the handler receives the session in state X with the
   same chat ID.
2. **Given** a saved state that references a draft trip, **When** the session
   is loaded, **Then** the draft trip is loaded with it.
3. **Given** a known user who changed their username or name in Telegram,
   **When** they send a message, **Then** the stored profile is updated
   before the handler runs (BUG-003).

---

### User Story 3 - Graceful Shutdown With a Farewell (Priority: P2)

When the bot is stopped, users with an open session are told that the bot is
going to sleep, and updates that are already being handled are finished.

**Why this priority**: Users learn that the bot stopped answering, and no
update is cut off midway.

**Independent Test**: Start the service against the fake Bot API, open a
session, call `Shutdown`, and check the farewell message
(`TestService_PublishTrip`, `TestService_CancelTrip`, `TestService_Commands`).

**Acceptance Scenarios**:

1. **Given** the bot is running, **When** it receives SIGTERM, SIGINT,
   SIGQUIT, or SIGHUP, **Then** it starts shutting down and logs the signal
   as the reason.
2. **Given** users with open sessions, **When** the bot shuts down, **Then**
   each of them receives `I'm going to sleep. Bye, <first name>!` in their
   session's chat, and their session is deleted.
3. **Given** updates still being handled, **When** the bot shuts down,
   **Then** it stops long polling and waits for those handlers for up to
   10 seconds.

---

### User Story 4 - Robust Update Handling (Priority: P3)

The bot keeps working when an update cannot be handled: it ignores updates
that have no user to talk to, survives handler panics, and logs every update
with the user and chat it belongs to.

**Why this priority**: Operational safety; users do not see it directly.

**Independent Test**: `TestSetSessionMiddleware` (ignored updates and storage
failures), `TestPanicRecovery`, `TestLoggerMiddleware`,
`TestHandlers_NoSession`.

**Acceptance Scenarios**:

1. **Given** an update without a message or without a sender, **When** it
   arrives, **Then** no session is created and no handler runs.
2. **Given** a message sent by the bot itself, **When** it arrives, **Then**
   no session is created and no handler runs.
3. **Given** a handler that panics, **When** it handles an update, **Then**
   the panic is recovered and logged, and the bot keeps handling other
   updates.
4. **Given** a user with a session, **When** any handler logs, **Then** the
   log entry carries `user_id` and `chat_id`.
5. **Given** a handler that runs without a session in its context, **When**
   it is called, **Then** it logs an error and returns without replying.
6. **Given** loading or creating the user or session fails, **When** a
   message arrives, **Then** the bot replies "Sorry, something went wrong.
   Please try again." to that chat and drops the update (BUG-002).

---

### Edge Cases

- **Same user in two chats**: ~~the session keeps the chat of the user's first
  message. Replies and trip announcements for later messages from another
  chat go to that first chat. (Gap 1.)~~ each chat has its own session and
  dialog; replies go to the chat the user writes in (BUG-001).
- **Storage failure while loading or creating the user or session**: the
  error is logged and the update is dropped; ~~the user gets no reply.
  (Gap 2.)~~ the user gets the error reply (BUG-002).
- **Changed Telegram profile**: ~~the stored username and names are never
  refreshed after first contact. (Gap 3.)~~ the stored profile is updated on
  the next message (BUG-003).
- **Session list fails during shutdown**: `Shutdown` returns before it stops
  long polling and the handler, so no farewell is sent and in-flight updates
  are not awaited. (Gap 4.)
- **Bot restart**: all users, sessions, states, and draft trips are lost;
  every user starts again with `StateStart`.
- **Session deleted on shutdown**: the user's state stays in memory until the
  process exits.
- **Several updates from the same user at once**: each handler works on its
  own copy of the session; the last saved copy wins.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The bot MUST create a user record (Telegram ID, username, first
  name, last name) on the first message from an unknown user, and MUST
  update the record when a later message shows a different username, first
  name, or last name (BUG-003).
- **FR-002**: ~~The bot MUST keep exactly one session and one dialog state per
  user; the first state is `StateStart`.~~ The bot MUST keep exactly one
  session and one dialog state per user and chat; the first state is
  `StateStart` (BUG-001).
- **FR-003**: A new session MUST be bound to the chat of the message that
  created it.
- **FR-004**: On every message from a known user, the bot MUST load the saved
  session, dialog state, and draft trip of that user in that chat and pass
  them to the handler (BUG-001).
- **FR-005**: The bot MUST ignore updates without a message, messages without
  a sender, and messages sent by the bot itself.
- **FR-006**: When loading or creating the user or session fails, the bot
  MUST log the error, reply "Sorry, something went wrong. Please try again."
  to the chat of the message, and drop the update (BUG-002).
- **FR-007**: The bot MUST recover from handler panics, log them, and keep
  handling updates.
- **FR-008**: Log entries of a handled update MUST carry `user_id` and
  `chat_id` when the update has a session.
- **FR-009**: The bot MUST read the token from
  `RIDE_ANNOUNCER_TELEGRAM_TOKEN` and exit with an error when it is missing or
  the bot cannot be created.
- **FR-010**: On SIGTERM, SIGINT, SIGQUIT, or SIGHUP the bot MUST send each
  open session the farewell message and delete the session.
- **FR-011**: On shutdown the bot MUST stop long polling and wait up to
  10 seconds for updates in flight.
- **FR-012**: Shutdown MUST use a context that is not canceled but keeps the
  logger, so Bot API calls made during shutdown succeed.

### Key Entities *(include if feature involves data)*

- **User**: a Telegram user known to the bot: Telegram ID, username, first
  name, last name. ~~Created once, never updated.~~ Created on first contact,
  updated when the Telegram profile changes (BUG-003).
- **Session**: the user's conversation with the bot in one chat: user, chat
  ID, and the dialog state. ~~One per user.~~ One per user and chat (BUG-001).
- **Dialog state** (`UserState`): the user's current `models.State` step and
  the draft trip of the dialog, if any. ~~One per user;~~ One per user and
  chat (BUG-001); referenced by the session.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first message from a new user reaches its handler with a
  session in `StateStart` and the user's profile.
- **SC-002**: A returning user's next message reaches its handler in the
  state saved by the previous step.
- **SC-003**: Every user with an open session receives exactly one farewell
  message when the bot stops.
- **SC-004**: A panic in one handler does not stop the bot from handling the
  next update.

## Assumptions

- The bot runs as a single instance with long polling and keeps all data in
  memory.
- Log level `DEBUG` and text format are fixed in `cmd/rideannouncer/main.go`.
- The farewell message is sent to the session's chat, which may be a group.
- Depends on the Telegram Bot API.

## Gaps

Found during migration; none of them is fixed by this spec.

1. ~~The session's chat ID never changes after the first message, so replies
   and announcements can go to a different chat than the one the user is
   writing in.~~ Fixed by BUG-001.
2. ~~Storage failures in `setSessionMiddleware` drop the update without
   telling the user.~~ Fixed by BUG-002.
3. ~~User profiles are never refreshed, so trip cards can show an old
   username.~~ Fixed by BUG-003.
4. `Service.Shutdown` returns early when listing sessions fails and skips
   stopping long polling and the handler.
5. After a recovered panic the user gets no reply; the stack trace goes to
   stdout through `fmt.Println` instead of the logger.
6. `internal/ops/sessions.go`, `internal/ops/users.go`, and the
   `users`, `sessions`, and `states` repositories have no tests of their own
   (constitution Known Deviation).
7. Unused repository API: `states.ListStates`, `states.GetStateByID`,
   `users.List`, and `states.State.Command` (always empty). The log level
   cannot be configured.
