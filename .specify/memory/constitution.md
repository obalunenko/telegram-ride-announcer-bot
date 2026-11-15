<!--
Sync Impact Report
- Version change: 1.0.1 → 1.0.2 (PATCH: resolved known deviations)
- Modified: Known Deviations (removed the stale `internal/input/` and
  `internal/puzzles/` lint exclusions and the `scrum-report` OCI label, both
  fixed in the repository)
- Added sections: none; Removed sections: none
- Templates: spec-template.md, plan-template.md, tasks-template.md, AGENTS.md —
  no change needed
- Follow-up: none
- Previous: 1.0.0 → 1.0.1 factual corrections from
  /speckit-brownfield-validate; template → 1.0.0 initial ratification from the
  brownfield scan
-->

# Telegram Ride Announcer Bot Constitution

## Project Identity

Telegram Ride Announcer Bot schedules and announces planned bicycle trips in
Telegram chats. A user creates a trip in a step-by-step dialog (name, date,
description, confirmation); the bot publishes the trip card to the chat and
pins it. The project is one Go module, `rideannouncer`, with a single entry
point and layered packages under `internal/`. It talks to the Telegram Bot
API through telego long polling and keeps all data in memory. Shell scripts,
Docker, and goreleaser support development and release, not the runtime.

## Core Principles

### I. Keep the Layers: `cmd` → `service` → `ops` → `repository`

- `cmd/rideannouncer/` is the composition root: it reads the environment,
  builds the repositories, `service/backends`, `telegram.Bot`, and
  `service.Service`, defines the command menu, and handles signals and
  shutdown. It holds no bot behavior.
- `internal/service/` holds the Telegram handlers, middleware, and dialog
  flows. It reaches storage only through `internal/ops`; it MUST NOT call
  repository methods directly. It imports repository packages only for the
  types in its `backends` interface and for sentinel errors.
- `internal/ops/` holds the storage operations and converts repository
  records into `internal/models` types. It MUST NOT import telego,
  `internal/telegram`, or `internal/service`.
- `internal/repository/*` packages MUST NOT import `internal/models`,
  `internal/ops`, `internal/service`, or telego. They use their own record
  types and plain IDs (`int64`, `uuid.UUID`).
- `internal/models/` holds the domain types and the dialog `State`; it
  depends on no other package of this module.
- `internal/telegram/` wraps the telego client (bot identity, name,
  description, command menu) and MUST NOT import other `internal/` packages;
  its tests use the fake server in `internal/telegram/telegramtest`.

Evidence: imports of every package; `cmd/rideannouncer/main.go`;
`internal/service/backends.go` and `internal/ops/backends.go`.

### II. Keep Storage Behind `Repository` Interfaces

- Each package under `internal/repository/` (`trips`, `users`, `sessions`,
  `states`) exposes a `Repository` interface, its record types, sentinel
  errors (`ErrNotFound`, `ErrAlreadyExists`), and constructors. Today the
  only implementation is `NewInMemory()`.
- Implementations MUST be safe for concurrent use; the in-memory ones guard
  their maps with `sync.RWMutex`.
- A new storage backend implements the existing `Repository` interface and
  is selected in `cmd/rideannouncer`; `service` and `ops` keep depending on
  the interfaces through `backends`.
- All data is lost when the bot restarts. A feature spec MUST state whether
  its data has to survive a restart.

Evidence: `internal/repository/*/*.go`; `internal/service/backends/backends.go`.

### III. Model Dialogs as `models.State` Transitions

- Multi-step dialogs are transitions of `models.State`, stored in the user's
  session (`Session.UserState`) together with the draft trip.
- A new step adds a `State` constant before `stateSentinel`, regenerates
  `internal/models/state_string.go` with `make codegen`, and is routed in
  `textHandler`, `newTripHandler`, and `createTrip` in `internal/service/`.
  Every state that a step sets MUST have a branch in `createTrip`.
- A step MUST save the session before it sends the reply that asks for the
  next input (`replyInTrip`), so the user's answer finds the session in the
  new state.
- `/start` and `/help` reset the session to `StateStart`.
- One session exists per user, created by `setSessionMiddleware` on the
  user's first message, and keeps the chat ID of that message.

Evidence: `internal/models/models.go`; `internal/service/handlers.go`
(`createTrip`, `replyInTrip`, `textHandler`); `internal/service/middleware.go`.

### IV. Keep the Command and Message Contract Consistent

- A bot command is a `Cmd*` constant in `internal/service/service.go`, a
  handler registered in `initHandlers`, and an entry with a description and
  an `enabled` flag in `commands` in `cmd/rideannouncer/main.go`. All three
  change together.
- Only enabled commands appear in the Telegram menu. A command that is not
  implemented uses `notImplementedHandler` and stays disabled.
- Multi-line user messages (welcome, help, trip card) are rendered by
  `internal/service/renderer` from embedded `templates/*.gotmpl`. A new or
  changed template MUST have a golden file in `renderer/testdata/` and a
  test that compares against it.

Evidence: `internal/service/service.go`, `cmd/rideannouncer/main.go`,
`internal/telegram/bot.go` (`EnabledCommands`), `internal/service/renderer/`.

### V. Test Behavior Changes Without the Real Telegram API

- Behavior changes in `internal/` MUST include or update tests. Untested
  packages are listed under Known Deviations; new code MUST NOT extend them.
  Tests for `cmd/rideannouncer` (`main`) and for `internal/telegram/telegramtest`
  (a test helper) are not required.
- Tests MUST NOT call the real Telegram Bot API. Handler, middleware, and
  bot tests use the fake server in `internal/telegram/telegramtest`.
- Tests live next to the code in `*_test.go` files, use
  `github.com/stretchr/testify` (`assert`, `require`, `suite`), and prefer
  table-driven cases with `t.Run`. Fixtures and golden files go in the
  package's `testdata/` directory.
- Tests MUST pass with the race detector (`make test` runs
  `go test -race ./...` through gotestsum).
- No numeric coverage threshold or test-first process is required.

Evidence: `internal/service/*_test.go`, `internal/telegram/bot_test.go`,
`internal/service/renderer/renderer_test.go`, `scripts/tests/run.sh`.

### VI. Do Not Hand-Edit Generated or Vendored Code

- `internal/models/state_string.go` is generated by stringer; regenerate it
  with `make codegen` after changing `State`.
- `vendor/` is committed and goreleaser builds with `mod: vendor`. After
  changing `go.mod`, run `make sync-vendor`.
- Spec Kit skills in `.claude/skills/` are generated; regenerate them with
  `make speckit-install`.

Evidence: `//go:generate` in `internal/models/models.go`,
`scripts/sync-vendor.sh`, `.goreleaser.yml`.

## Naming and Documentation

- Go packages use lowercase single-word names; Go files use lowercase
  snake_case names; shell scripts in `scripts/` use kebab-case. Both `_internal_test.go` (in `internal/service/`) and
  in-package `_test.go` files exist for white-box tests; no single rule is
  established.
- Every package has a package comment. New exported identifiers MUST have
  doc comments.
- Errors are sentinel values (`errors.New`) wrapped with
  `fmt.Errorf("...: %w", err)` and checked with `errors.Is`.
- Logging uses `github.com/obalunenko/logger` carried in `context.Context`;
  handlers add a `command_handler` field.
- Configuration comes from environment variables read with
  `github.com/obalunenko/getenv`. The bot token is
  `RIDE_ANNOUNCER_TELEGRAM_TOKEN`; new variables use the same
  `RIDE_ANNOUNCER_` prefix.
- Commits follow Conventional Commits (`feat`, `fix`, `chore`, `ci`, `docs`)
  with an optional scope; PRs are squash-merged into `master`.
- No feature-branch naming convention is established. Feature specs live in
  `specs/NNN-feature-name` through the feature-numbering hook.
- Update `README.md` when bot commands, environment variables, deployment,
  or user-visible behavior change.

## Quality Gates and Development Tools

- Tools are pinned in `tools/<name>/go.mod` with a `tool` directive and run
  through the Makefile with `go tool`; `make install-tools` prebuilds them.
- Before merge, these MUST pass: `make vet`, `make test`, `make lint-pipeline`,
  and `make build`. CI (`.github/workflows/go.yml`) runs the same checks; its
  golangci-lint and goreleaser come from their official actions, at the
  versions pinned in `tools/`.
- Code is formatted with gofumpt and goimports (`make format-project`). New
  code MUST respect the limits in `.golangci.yml` (funlen 100 lines /
  50 statements, gocyclo 15, lll 140, mnd, wsl, gosec, exhaustive).
- `.github/workflows/release.yml` builds `v*` tags with goreleaser
  (`.goreleaser.yml`); no release has been tagged yet. The bot also builds as
  a Docker image (`build/docker/rideannouncer/Dockerfile`,
  `make build-rideannouncer`) and runs locally with `make run-local`.
- Spec Kit is installed and checked locally through `make speckit-install`,
  `make speckit-update-check`, and `make speckit-update-test`. Keep Spec Kit
  checks out of CI.

## Known Deviations

These are current exceptions. They are not rules, and new features MUST NOT
extend them:

- `internal/ops`, `internal/repository/*`, and `internal/models` have no
  tests of their own; they are exercised only through `internal/service`
  tests.
- `StateNewTripTime` and `StateNewTripPublish` are declared and routed by
  `textHandler`, but no step sets them and `createTrip` has no branch for
  them.
- `README.md` documents neither the bot commands nor
  `RIDE_ANNOUNCER_TELEGRAM_TOKEN`.

## Governance

- This constitution supersedes conflicting templates and agent instructions.
  Plans MUST pass the Constitution Check in `plan-template.md`; any violation
  MUST be justified in its Complexity Tracking table.
- Amendments update this file and its Sync Impact Report, and propagate to the
  Spec Kit templates and `AGENTS.md`. Versioning follows semantic versioning:
  MAJOR for removed or redefined principles, MINOR for new principles or
  sections, PATCH for clarifications.
- Resolving a known deviation removes it from the list in a PATCH amendment.

**Version**: 1.0.2 | **Ratified**: 2026-11-15 | **Last Amended**: 2026-11-15
