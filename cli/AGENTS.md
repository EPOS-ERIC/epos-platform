# CLI

## Boundaries

- `cmd/`: Cobra commands; keep handlers thin. `tui/`: terminal UI; no direct stdout while active. Use `display/` for CLI output and TUI components for feedback.
- `pkg/docker/` and `pkg/k8s/`: environment lifecycle. `command/`: subprocess helpers; reuse where applicable.
- `config/`: user settings (`epos-cli.yaml`). `db/`: local SQLite state, embedded goose migrations, and sqlc queries.
- Preserve Linux, macOS, and Windows compatibility. Check CLI help and behavior when changing commands.

## Deployment Config

- Docker: `pkg/docker/config/default.yaml`, models, validation, and embedded `templates/` render Compose and `.env`.
- Kubernetes: `pkg/k8s/config/helm/` embeds chart and `values.yaml`; `pkg/k8s/config/` owns models, validation, and rendering. Root `helm/` is a separate chart copy.
- Change related defaults, models, validation, templates, and render tests together when affected. Check both deployment modes and root chart for relevant changes; do not force unrelated changes.
- Preserve existing environment config and migration compatibility. Never edit migrations already tagged in git; add a new migration instead.

## Workflow

Run from `cli/`:

- `make test`: Go tests. `vet` prerequisite has no recipe; this does not run sqlc vet.
- `make lint`: runs `go mod tidy`, `go fix`, formatter, Helm lint, and golangci-lint; modifies files.
- `make build`: generate then build. `make build-release`: build for `GOOS`/`GOARCH` without generation; may overwrite existing binary.
- `make generate`: run `go generate ./...`, including sqlc. Load `sqlc-go` skill and read `sqlc.yaml` before SQL work; never edit `db/sqlc/` manually.
- For chart-only lint: `helm lint ./pkg/k8s/config/helm`. CI runs lint, release build, and tests; see `.github/workflows/cli-ci.yml`.
