---

description: "Migrated task list for Start, Help, and Command Menu"
---

# Tasks: Start, Help, and Command Menu

**Input**: Design documents from `/specs/002-start-help-menu/`

**Prerequisites**: plan.md, spec.md; depends on `specs/001-sessions-lifecycle`

**Status**: migrated. All tasks describe code that already exists, so they are marked complete. Gaps are listed at the end as notes, not as open tasks.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)

## Phase 1: Foundation

- [x] T001 `Command`, `Commands`, `NewCommand`, `EnabledCommands` in `internal/telegram/bot.go`
- [x] T002 [P] Template renderer with embedded templates and parse-error joining in `internal/service/renderer/renderer.go`
- [x] T003 Declare `Cmd*` constants in `internal/service/service.go`
- [x] T004 Declare `botName`, `botDescription`, and `commands` with enabled flags in `cmd/rideannouncer/main.go` (depends on T001, T003)

## Phase 2: User Story 1 - Welcome on `/start` (P1)

- [x] T005 [P] [US1] Welcome template in `internal/service/renderer/templates/welcome.gotmpl` with `internal/service/renderer/testdata/welcome.golden`
- [x] T006 [US1] `startHandler` resets the state and renders the welcome in `internal/service/handlers.go`
- [x] T007 [P] [US1] Golden test `TestTemplates_Welcome` in `internal/service/renderer/renderer_test.go`

## Phase 3: User Story 2 - Command List on `/help` (P1)

- [x] T008 [P] [US2] Help template in `internal/service/renderer/templates/help.gotmpl` with `internal/service/renderer/testdata/help.golden`
- [x] T009 [US2] `helpHandler` resets the state and lists `bot.Commands()` in `internal/service/handlers.go`
- [x] T010 [P] [US2] Golden test `TestTemplates_Help`; `/start` and `/help` replies in `TestService_PublishTrip`; save and render failures in `TestHandlers_Failure`

## Phase 4: User Story 3 - Fallback Replies (P2)

- [x] T011 [US3] `notFoundHandler`, `unsupportedHandler`, and `notImplementedHandler` in `internal/service/handlers.go`
- [x] T012 [US3] Register exact commands, `AnyCommand`, and `AnyMessageWithText` in order in `initHandlers` in `internal/service/service.go`
- [x] T013 [P] [US3] Unknown command, free text, and stub command cases in `TestService_Commands`; `TestTextHandler_UnknownState`

## Phase 5: User Story 4 - Bot Profile and Menu Sync on Startup (P2)

- [x] T014 [US4] `NewBot` reads identity, description, and commands; `updateOnStart` with `maybeUpdateBotName`, `maybeUpdateDescriptionBot`, `maybeUpdateCommands` in `internal/telegram/bot.go`
- [x] T015 [P] [US4] Fake Bot API server in `internal/telegram/telegramtest/server.go`
- [x] T016 [P] [US4] `TestCommand`, `TestCommands_EnabledCommands`, `TestNewBot`, `TestNewBot_Error`, `TestNewBot_UpdateOnStart` in `internal/telegram/bot_test.go`

## Gaps (not tasks)

Follow-ups found during migration; see [spec.md](spec.md#gaps). Track them with `/speckit-bugfix-report` or a new `/speckit-specify` feature.

- The welcome message promises trip lists, subscriptions, and weather-based recommendations that do not exist.
- `/start` and `/help` silently abandon `/newtrip` and leave the draft trip.
- Save and render failures give the user no reply.
- `/help` can list a stale menu when the startup update failed.
- No service-level test for the `/help` command list.
- Bot username without `@`; default language only; no short description.
