# Implementation Plan: [FEATURE]

**Feature Directory**: `specs/[NNN-feature-name]` | **Branch**: [Current branch, if applicable]

**Date**: [DATE] | **Spec**: [link]

**Input**: Feature specification from `/specs/[NNN-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  Defaults below describe the current project. Keep what applies, and add
  feature-specific details in the brackets.
-->

**Language/Version**: Go 1.27, single module
`github.com/obalunenko/telegram-ride-announcer-bot` with committed `vendor/`.

**Primary Dependencies**: telego (Telegram Bot API, long polling,
`telegohandler`), gofrs/uuid, obalunenko/logger, getenv, version.
[List feature-specific dependency changes; they require `make sync-vendor`.]

**Storage**: In memory only: `internal/repository/{trips,users,sessions,states}`
behind `Repository` interfaces; all data is lost on restart.
[Affected repositories and records, or N/A.]

**Testing**: Go `testing` with testify `assert`/`require`/`suite`,
table-driven `t.Run` cases, golden files in `testdata/`, and the fake Bot API
in `internal/telegram/telegramtest`; race detector on. [Relevant test files.]

**Target Platform**: Long-running bot on linux/darwin/windows binaries and a
Docker image. CGO disabled. [Feature-specific platform scope.]

**Project Type**: Telegram bot, one binary with layered `internal/` packages.

**Performance Goals**: [Feature-specific measurable goals or N/A; no general
latency or throughput goal is established.]

**Constraints**: Layers `cmd` → `service` → `ops` → `repository`; storage
through `Repository` interfaces; dialogs as `models.State` transitions; no
real Telegram API in tests. [Explain applicable constraints.]

**Scale/Scope**: [Feature scope: affected commands, dialog states, messages,
repositories.]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [ ] I. Layers hold: `service` reaches storage only through `ops`;
      repositories do not import `models`, `ops`, `service`, or telego;
      `ops` does not import telego; `cmd/rideannouncer` only wires
      dependencies.
- [ ] II. Storage changes go through `Repository` interfaces, stay safe for
      concurrent use, and the spec states whether data must survive a
      restart.
- [ ] III. New dialog steps add a `State` before `stateSentinel`, regenerate
      `state_string.go`, are routed in `textHandler`, `newTripHandler`, and
      `createTrip`, and save the session before asking for the next input.
- [ ] IV. Commands change their `Cmd*` constant, handler, and menu entry
      together; unimplemented commands stay disabled; changed templates have
      golden tests.
- [ ] V. Behavior changes in `internal/` include table-driven testify tests
      that pass with `-race` and use `telegramtest` instead of the real API.
- [ ] VI. Generated code is regenerated, not hand-edited (`make codegen` for
      `State`); `vendor/` is synced after `go.mod` changes.
- [ ] `README.md` is updated when commands, environment variables, or
      user-visible behavior change.
- [ ] Known deviations from the constitution are not extended.

## Project Structure

### Documentation (this feature)

```text
specs/[NNN-feature-name]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  Keep only the paths this feature touches and name the files that change.
  Tests live next to the code in *_test.go files.
-->

```text
cmd/
└── rideannouncer/      # Composition root: env, repositories, command menu, shutdown
internal/
├── service/            # Handlers, middleware, dialog flows (handlers.go, trips.go, middleware.go)
│   ├── renderer/       # Message templates (templates/*.gotmpl, testdata/*.golden)
│   └── backends/       # Set of repositories passed to the service
├── ops/                # Storage operations; repository records → models
├── repository/
│   ├── trips/          # Trip storage (Repository, NewInMemory)
│   ├── users/          # User storage
│   ├── sessions/       # Session storage
│   └── states/         # Dialog state storage
├── models/             # Domain types and State (stringer output)
└── telegram/           # telego wrapper: identity, menu, name, description
    └── telegramtest/   # Fake Bot API server for tests
build/docker/           # Bot image and bake config
deployments/docker-compose/
scripts/                # Build, test, lint, codegen, release scripts
tools/                  # Pinned tools, one go.mod per tool (go tool)
```

**Structure Decision**: [Name the affected packages and files, and explain any
new package or cross-package dependency.]

**Validation**: `make vet`, `make test`, `make lint-pipeline`, `make build`.
Add `make codegen` when `State` changes and `make sync-vendor` when
dependencies change. Try the flow in Telegram with `make run-local` when the
dialog or messages change.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., `service` calls a repository directly] | [specific problem] | [why an `ops` function is insufficient] |
| [e.g., storage outside `Repository` interfaces] | [current need] | [why implementing the interface is not feasible now] |
