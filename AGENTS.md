# Project Instructions

Read `.specify/memory/constitution.md` for the project's architecture and
behavioral contracts. This repository is one Go module with a single
Telegram bot, `rideannouncer`, built from layered packages under `internal/`.

## Work Boundaries

- Entry point: `cmd/rideannouncer/` wires the environment, repositories,
  `service/backends`, `telegram.Bot`, and `service.Service`, and defines the
  command menu. Keep bot behavior out of it.
- Bot work: `internal/service/` (handlers, middleware, dialog flows),
  `internal/service/renderer/` (message templates and golden files), and
  `internal/telegram/` (telego wrapper and the `telegramtest` fake Bot API).
  `internal/service` reaches storage only through `internal/ops`. A command
  changes its `Cmd*` constant, handler registration, and menu entry together.
- Domain and storage work: `internal/models/` (domain types and dialog
  `State`), `internal/ops/` (storage operations and record → model
  conversion), and `internal/repository/{trips,users,sessions,states}/`
  (`Repository` interfaces with in-memory implementations). Repositories do
  not import `models`, `ops`, `service`, or telego.
- Tooling work: `Makefile`, `scripts/` (except `scripts/speckit/`), `tools/`,
  `build/docker/`, `deployments/docker-compose/`, `.goreleaser.yml`,
  `.golangci*.yml`, `sonar-project.properties`, and `.github/`.
- Dependencies: `go.mod` and `go.sum`; run `make sync-vendor` after changing
  them.
- Spec Kit work: `.specify/`, `.speckit/feature-numbering/`,
  `scripts/speckit/`, `CLAUDE.md`, Spec Kit hooks in `.claude/settings.json`,
  and generated Claude Code skills in `.claude/skills/`.
  The local feature-numbering extension source is `.speckit/feature-numbering/`.
  Regenerate skills through `make speckit-install`; do not edit generated
  copies. Preserve installed hooks and project customizations when refreshing
  tooling.
- Generated code: `internal/models/state_string.go` comes from `make codegen`
  (stringer). `vendor/` comes from `make sync-vendor`. Do not edit them by hand.
- User documentation: update `README.md` when bot commands, environment
  variables, deployment, or user-visible behavior change.

## Validation

- Run `make vet`, `make test`, `make lint-pipeline`, `make build`; CI runs the
  same checks. Tools are pinned in `tools/<name>/go.mod` and built with
  `go tool` on first use (`make install-tools` prebuilds them); bump a tool
  there, not in CI.
- Format with `make format-project` (gofumpt and goimports).
- Tests never call the real Telegram API; use
  `internal/telegram/telegramtest`. `make test` runs with `-race`.
- Try dialog or message changes in Telegram with `make run-local`
  (needs `RIDE_ANNOUNCER_TELEGRAM_TOKEN`).
- Spec Kit checks (`make speckit-update-check`, `make speckit-update-test`) are
  local; keep them out of CI.

## Observed Conventions

Lowercase Go package names, snake_case Go filenames, colocated `*_test.go`
files with table-driven testify tests, doc comments on exported identifiers,
sentinel errors wrapped with `%w`, and logging through
`github.com/obalunenko/logger` in context. Commits follow Conventional Commits
and PRs are squash-merged into `master`. No feature-branch naming convention or
numeric coverage threshold is established. Feature directories use
`specs/NNN-feature-name` through the installed numbering hook.
