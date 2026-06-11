# AGENTS

## Build and Test Workflow

- `make generate` runs `go generate ./...` and refreshes `db/sqlc`
- `make build` runs `generate` first, then builds binary
- `make lint` runs `gofmt` via `golangci-lint fmt`, then `golangci-lint run ./...`
- `make test` runs `go test ./...`

## Service Conventions

- Keep HTTP routing in `server/router.go`; keep handler logic in `server/handlers/`.
- Keep DB bootstrap in `db/`; use `db.Open(cfg)` plus embedded goose migrations and generated `sqlc` queries.
- Keep request parsing and JSON encoding in handlers; keep route wiring thin and explicit.
- Use chi route groups to mirror v1 shape under `/api/v2`.
- Keep `db/sqlc/` generated. Do not edit it manually; regenerate through `go generate`.
- Keep config fields in `config/Config`; validate new config behavior in `Load` or a dedicated validator if added later.
- Use `context.Context` for DB calls and health checks. Prefer request context for handler work.
- Keep everything strongly typed. Avoid `any` and `interface{}` unless truly needed.
- Use shared `packages/go/logging` and shared middleware. Do not duplicate recovery/logger plumbing.
- Treat `docs/openapi.yaml` and embedded docs assets as source for served API docs. Keep route changes aligned with docs updates.
- Do not use `%q` in error messages or logs. Prefer `'%s'` if quoting is needed.

## Testing

- Prefer **table-driven tests** where practical. Follow existing patterns in repository.
- Add unit tests for new logic or bug fixes when feasible without excessive mocking.
- Do not add tests for pure refactors unless behavior changes.
- Keep tests **deterministic**:
  - No `time.Sleep`.
  - No randomness.

## Logging

Use shared logging library consistently:

```go
var log = logging.Get("component-name")

// then use:
log.Debug(...)
log.Info(...)
```

- Use [slog](https://pkg.go.dev/log/slog)-style structured logging
- Log on error-prone or failure paths
- Do **not** over-log. Avoid noise in happy paths
- Do not use `%q` in wrapped/logged strings. Always use `'%s'` instead if necessary

## Architecture and Safety

### Service structure

- `main.go` wires config, db, and server.
- `server/` holds router and handlers only.
- `db/` holds postgres bootstrap and query surface.
- `db/sqlc/` is generated code only.
- `config/` owns env loading.
- `docs/` serves OpenAPI and UI assets.

- Follow existing package structure and naming conventions.
- Extend existing package areas when responsibility matches.
- Prefer simple, idiomatic Go over clever or complex abstractions.
- Use loop variables directly in closures. Do not add `idx := i` or `v := value` shadowing unless non-range variable actually needs rebinding.
- Maintain strong type safety.
- Avoid `interface{}` or `any` unless absolutely necessary; if used, document why.
