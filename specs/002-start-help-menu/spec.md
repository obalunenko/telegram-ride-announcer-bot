# Feature Specification: Start, Help, and Command Menu

**Feature Directory**: `specs/002-start-help-menu`

**Feature Branch**: N/A (migrated from existing code on `master`)

**Created**: 2026-11-15

**Status**: migrated

**Input**: Reverse-engineered by `/speckit-brownfield-migrate` from `internal/service/handlers.go` (`startHandler`, `helpHandler`, `notFoundHandler`, `unsupportedHandler`, `notImplementedHandler`), `internal/service/service.go`, `cmd/rideannouncer/main.go`, `internal/telegram/bot.go`, and `internal/service/renderer/`.

**Bugfix**: 2026-11-15 — BUG-001 The welcome and the README describe only built features.

**Bugfix**: 2026-11-15 — BUG-002 `/start` and `/help` discard an unfinished trip and say so.

**Bugfix**: 2026-11-15 — BUG-003 `/start` and `/help` reply with the error message when they fail.

## Project Scope and Compatibility

- **Affected boundaries**: `cmd/rideannouncer/` (bot name, description,
  command menu), `internal/service/` (command constants, handler
  registration, start/help/fallback handlers), `internal/service/renderer/`
  (`welcome` and `help` templates), `internal/telegram/` (bot identity and
  startup sync).
- **Bot commands and menu**: `/start` and `/help` (enabled); fallback for
  unknown commands and free text; `/trips`, `/subscribe`, `/unsubscribe`,
  `/subscribed` answer "Not implemented yet" and are disabled in the menu.
  The menu shows `/start`, `/help`, `/newtrip`, `/mytrips`.
- **Dialog states**: `/start` and `/help` reset the session to `StateStart`
  and discard the draft trip of an unfinished `/newtrip` (BUG-002).
- **Messages**: welcome (`welcome.gotmpl`), help (`help.gotmpl`), "Command
  not found. Use /help command to see all available commands.", "Not
  implemented yet. Use /help command to see all available commands."
- **Storage**: Saves the session state only (see 001).
- **Chats**: Replies go to the session's chat.
- **Configuration**: Bot name `Ride Announcer Bot` and description are
  constants in `cmd/rideannouncer/main.go`.
- **Release impact**: N/A (existing behavior).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Welcome on `/start` (Priority: P1)

A user sends `/start` and gets a welcome that greets them by first name,
describes what the bot is for, and points them to `/help`.

**Why this priority**: It is the first thing every user sees.

**Independent Test**: Send `/start` through the fake Bot API and check the
greeting (`TestService_PublishTrip`); render the template against
`welcome.golden` (`TestTemplates_Welcome`).

**Acceptance Scenarios**:

1. **Given** a user named John, **When** they send `/start`, **Then** the bot
   replies with a message that starts with `Hello, John! 👋 Welcome to
   <bot username>.` and mentions `/help`.
2. **Given** a user in any dialog state, **When** they send `/start`,
   **Then** their state is reset to `StateStart`.
3. **Given** any user, **When** they read the welcome, **Then** it lists only
   what the bot does today: creating and announcing a trip, and showing the
   trips they created (BUG-001).

---

### User Story 2 - Command List on `/help` (Priority: P1)

A user sends `/help` and gets the list of commands they can use, each with a
short description.

**Why this priority**: It is how users discover the bot's commands.

**Independent Test**: Send `/help` and check the help header
(`TestService_PublishTrip`); render the template against `help.golden`
(`TestTemplates_Help`).

**Acceptance Scenarios**:

1. **Given** the bot's registered menu, **When** a user sends `/help`,
   **Then** the bot replies with `Welcome to <bot username> Help!` and one
   line `/<command> - <description>` per menu command.
2. **Given** a user in any dialog state, **When** they send `/help`, **Then**
   their state is reset to `StateStart`.
3. **Given** a user in the middle of `/newtrip`, **When** they send `/start`
   or `/help`, **Then** the bot deletes the draft trip, replies "Your
   unfinished trip is discarded.", and then sends the welcome or the help
   (BUG-002).

---

### User Story 3 - Fallback Replies (Priority: P2)

A user sends something the bot does not handle, or a command that is not
built yet, and gets a reply that points them to `/help`.

**Why this priority**: The bot never stays silent on valid input.

**Independent Test**: `TestService_Commands` sends an unknown command, free
text, and each stub command, and checks the replies;
`TestTextHandler_UnknownState` covers an unknown dialog state.

**Acceptance Scenarios**:

1. **Given** any user, **When** they send an unknown command such as
   `/unknown`, **Then** the bot replies "Command not found. Use /help command
   to see all available commands."
2. **Given** a user who is not in a dialog, **When** they send free text,
   **Then** the bot replies with the same "Command not found" message.
3. **Given** any user, **When** they send `/trips`, `/subscribe`,
   `/unsubscribe`, or `/subscribed`, **Then** the bot replies "Not
   implemented yet. Use /help command to see all available commands."

---

### User Story 4 - Bot Profile and Menu Sync on Startup (Priority: P2)

When the bot starts, it makes its Telegram name, description, and command
menu match the code, and changes only what differs.

**Why this priority**: The menu users see in Telegram stays in line with the
commands the bot handles, without manual setup in BotFather.

**Independent Test**: `TestNewBot_UpdateOnStart` runs against the fake Bot
API with up-to-date, changed, and failing cases for the name, the
description, and the commands; `TestNewBot` and `TestNewBot_Error` cover
reading the current profile.

**Acceptance Scenarios**:

1. **Given** the bot starts, **When** it connects, **Then** it reads its
   identity, description, and registered commands from Telegram.
2. **Given** the registered name or description differs from the code,
   **When** the bot starts, **Then** it updates it; when it is equal, it
   makes no call.
3. **Given** the registered commands differ from the enabled commands in the
   code (added, removed, or changed description), **When** the bot starts,
   **Then** it registers exactly the enabled commands; when they are equal,
   it makes no call.
4. **Given** an update of the name, description, or commands fails, **When**
   the bot starts, **Then** it logs the error and keeps running with the
   previous value.
5. **Given** reading the identity, description, or commands fails, **When**
   the bot starts, **Then** it exits with an error.

---

### Edge Cases

- **`/start` or `/help` during `/newtrip`**: ~~the state is reset, but the
  draft trip stays in the session and in storage, and the user is not told
  that the dialog was abandoned. (Gap 2.)~~ the draft trip is deleted and
  the user is told (BUG-002).
- **Saving the session or rendering a template fails**: the error is logged
  and ~~the user gets no reply. (Gap 3.)~~ the user gets "Sorry, something
  went wrong. Please try again." (BUG-003).
- **Menu update failed on startup**: `/help` lists the commands that were
  registered in Telegram before, not the ones in the code. (Gap 4.)
- **Disabled commands typed by hand**: they still get the "Not implemented
  yet" reply.
- **Unknown dialog state**: free text is answered with "Command not found".

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `/start` MUST reply with the welcome message, greeting the user
  by first name and naming the bot by username. The welcome MUST describe
  only implemented features (BUG-001).
- **FR-002**: `/help` MUST reply with the help message listing each menu
  command as `/<command> - <description>`.
- **FR-003**: `/start` and `/help` MUST reset the user's dialog state to
  `StateStart`; when a trip dialog is unfinished, they MUST delete its draft
  trip and reply "Your unfinished trip is discarded." first (BUG-002).
- **FR-004**: Unknown commands, and free text outside a dialog, MUST get the
  "Command not found" reply pointing to `/help`.
- **FR-005**: Commands that are not implemented MUST get the "Not
  implemented yet" reply pointing to `/help`.
- **FR-006**: Each command MUST be declared in the code with a description
  and an enabled flag; only enabled commands are registered in the Telegram
  menu.
- **FR-007**: On startup the bot MUST read its identity, description, and
  registered commands, and exit with an error if any read fails.
- **FR-008**: On startup the bot MUST update its name, description, and menu
  only when they differ from the code.
- **FR-009**: A failed name, description, or menu update MUST be logged and
  MUST NOT stop the bot.
- **FR-010**: The welcome and help messages MUST be rendered from embedded
  templates; a template that cannot be parsed MUST make service creation
  fail.
- **FR-011**: When saving the session, discarding the draft, or rendering
  fails, the handler MUST log the error and ~~send no reply~~ reply "Sorry,
  something went wrong. Please try again." (BUG-003).
- **FR-012**: `README.md` MUST list the enabled commands and name the
  unimplemented ones (BUG-001).

### Key Entities *(include if feature involves data)*

- **Command**: a bot command with a name, a menu description, and an
  enabled flag.
- **Bot profile**: the bot's Telegram ID, username, name, description, and
  registered menu.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user who sends `/start` receives the welcome with their first
  name.
- **SC-002**: The Telegram menu shows exactly the enabled commands after the
  bot starts.
- **SC-003**: The bot makes no profile or menu update call when Telegram
  already matches the code.
- **SC-004**: Every unknown command, free-text message outside a dialog, and
  stub command gets a reply pointing to `/help`.

## Assumptions

- The menu and description are set for the default language only.
- Disabled commands stay registered as handlers so that typing them still
  gets a reply.
- Depends on the Telegram Bot API (`getMe`, `getMyDescription`,
  `getMyCommands`, `setMyName`, `setMyDescription`, `setMyCommands`).

## Gaps

Found during migration; none of them is fixed by this spec.

1. ~~The welcome message promises features that do not exist: a list of
   upcoming trips, subscriptions, a list of subscribed trips (stubs), and
   outfit and SPF recommendations based on the weather (no code at all).~~
   Fixed by BUG-001.
2. ~~`/start` and `/help` interrupt `/newtrip` silently and leave the draft
   trip in the session and in storage.~~ Fixed by BUG-002.
3. ~~Failures to save the session or render a template give the user no reply.~~
   Fixed by BUG-003.
4. `/help` lists `bot.Commands()`, which can be stale when the menu update
   failed on startup.
5. No service-level test checks that `/help` lists exactly the enabled
   commands; tests check only the header.
6. Minor: the bot username appears without `@`; only the default language
   is synced; the short description is not managed.
