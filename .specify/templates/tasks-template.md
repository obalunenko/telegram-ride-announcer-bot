---

description: "Task list template for Telegram Ride Announcer Bot feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[NNN-feature-name]/`

**Prerequisites**: plan.md (required), spec.md (required for user stories); research.md, data-model.md, quickstart.md, and contracts/ when relevant

**Tests**: Behavior changes in `internal/` MUST include or update tests (constitution V). Use Go `testing` with testify `assert`/`require`/`suite`, table-driven `t.Run` cases, and fixtures or golden files in the package's `testdata/`. Handler and bot tests use the fake Bot API in `internal/telegram/telegramtest`; tests MUST NOT call the real Telegram API.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Within a story, follow the layers: `internal/models` → `internal/repository/*` → `internal/ops` → `internal/service` (and `renderer`) → `cmd/rideannouncer`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies); it does not itself request spawning additional agents
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Entry point: `cmd/rideannouncer/main.go` (command menu, wiring, shutdown)
- Bot: `internal/service/` (`service.go` commands and handler registration, `handlers.go` and `trips.go` flows, `middleware.go`), `internal/service/renderer/` (templates and golden files), `internal/telegram/`
- Domain and storage: `internal/models/`, `internal/ops/`, `internal/repository/{trips,users,sessions,states}/`
- Tests: `*_test.go` next to the code; fixtures and golden files in `<package>/testdata/`; fake Bot API in `internal/telegram/telegramtest/`
- Generated: `internal/models/state_string.go` (`make codegen`), `vendor/` (`make sync-vendor`)
- Docs: `README.md`

<!--
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.

  The /speckit-tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements and affected packages from plan.md
  - Entities from data-model.md
  - Interface contracts from contracts/ (bot commands, dialogs, messages), if any

  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment

  Remove phases and examples unrelated to the feature. Never add databases,
  web frontends, or authentication without a feature requirement.

  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Feature Preparation

**Purpose**: Changes every story depends on; skip what the feature does not need

- [ ] T001 Add or update dependencies in `go.mod` and run `make sync-vendor`
- [ ] T002 [P] Add golden files or fixtures in `[package]/testdata/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Domain and storage changes that MUST be complete before any handler work

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on the plan):

- [ ] T003 Extend domain types in `internal/models/models.go`
- [ ] T004 Add dialog states before `stateSentinel` in `internal/models/models.go`, then run `make codegen` to update `internal/models/state_string.go`
- [ ] T005 [P] Extend the `Repository` interface and `NewInMemory` implementation in `internal/repository/[name]/[name].go` with tests in `internal/repository/[name]/[name]_test.go`
- [ ] T006 Add the storage operation and record → model conversion in `internal/ops/[name].go` with tests in `internal/ops/[name]_test.go` (depends on T005)

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own, e.g., a `go test` case against `telegramtest` plus a manual run with `make run-local`]

### Tests for User Story 1

- [ ] T007 [P] [US1] Table-driven handler tests for [command or step] against `telegramtest` in `internal/service/handlers_internal_test.go`
- [ ] T008 [P] [US1] Golden test for the [name] template in `internal/service/renderer/renderer_test.go` with `internal/service/renderer/testdata/[name].golden`

### Implementation for User Story 1

- [ ] T009 [US1] Add or change the template in `internal/service/renderer/templates/[name].gotmpl` and its method in `internal/service/renderer/renderer.go`
- [ ] T010 [US1] Implement the handler or dialog step in `internal/service/handlers.go`; save the session before asking for the next input (depends on T006)
- [ ] T011 [US1] Add new states to the trip states routed by `textHandler` to `tripAnswerHandler` (`internal/service/trips.go`) and give them a branch in `createTrip`
- [ ] T012 [US1] Add the `Cmd*` constant and register the handler in `initHandlers` in `internal/service/service.go`
- [ ] T013 [US1] Add or enable the command in `commands` in `cmd/rideannouncer/main.go` (depends on T012)
- [ ] T014 [US1] Add validation, user-facing error replies, and error wrapping (`fmt.Errorf("...: %w", err)`) for [inputs]

**Checkpoint**: At this point, User Story 1 should work in Telegram and be testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 2

- [ ] T015 [P] [US2] Table-driven tests for [behavior] in `[package]/[file]_test.go`

### Implementation for User Story 2

- [ ] T016 [US2] Implement [behavior] in `internal/[package]/[file].go`
- [ ] T017 [US2] Update the handler in `internal/service/[file].go`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and quality gates for the whole feature

- [ ] TXXX [P] Update `README.md` (bot commands, environment variables, deployment, behavior)
- [ ] TXXX Run `make codegen` if `State` changed
- [ ] TXXX Run `make format-project`
- [ ] TXXX Run `make vet` and `make lint-pipeline`
- [ ] TXXX Run `make test`
- [ ] TXXX Run `make build`
- [ ] TXXX Try the flow in Telegram with `make run-local` (needs `RIDE_ANNOUNCER_TELEGRAM_TOKEN`), if the dialog or messages changed
- [ ] TXXX Run quickstart.md validation, if present

---

## Dependencies & Execution Order

### Phase Dependencies

- **Feature Preparation (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - Proceed sequentially in priority order (P1 → P2 → P3), or in parallel when stories touch different files
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- `internal/models` before `internal/repository/*` before `internal/ops`
- `internal/ops` and `internal/service/renderer` before `internal/service` handlers
- `internal/service` before `cmd/rideannouncer`
- Tests alongside or before the code they cover; they MUST pass with `-race`
- Story complete before moving to next priority

### Parallel Opportunities

- Test and golden-file tasks marked [P] in different packages can run in parallel
- Repository changes in different `internal/repository/*` packages can run in parallel
- Template work in `internal/service/renderer/` can run in parallel with `internal/ops` changes
- Different user stories can be worked on in parallel when they touch different files; most handler work shares `internal/service/handlers.go`, so plan it sequentially

---

## Parallel Example: User Story 1

```bash
# Before the handler change (T010), prepare the tests and the template together:
Task: "Table-driven handler tests in internal/service/handlers_internal_test.go"
Task: "Golden test in internal/service/renderer/renderer_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Feature Preparation
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run the story's tests and try it in Telegram
5. Open a PR if ready

### Incremental Delivery

1. Complete Preparation + Foundational → Foundation ready
2. Add User Story 1 → Test independently → PR (MVP!)
3. Add User Story 2 → Test independently → PR
4. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Do not hand-edit generated files or `vendor/`
- Commit with Conventional Commits (`feat:`, `fix:`, `chore:`, ...)
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
