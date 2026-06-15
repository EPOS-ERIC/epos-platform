# AGENTS

## Build and Test Workflow

- `make generate` regenerates `gen/metadataclient` from metadata-lifecycle OpenAPI spec
- `make lint test` formats + checks if changes ok

## Service Conventions

- Keep request routing, HTTP parsing, and response encoding in `server/`; keep business logic in domain-oriented packages outside transport layer.
- Keep service-specific outbound client wiring in `clients/`; keep reusable HTTP/cache behavior in `upstream/`.
- Keep shared HTTP middleware in `packages/go/middleware`; do not duplicate generic chi/logging recovery plumbing in services.
- Keep shared domain errors and contracts in `domain/`.
- Keep helper code that should not become public package API in `internal/`.
- Extend existing package areas when responsibility matches; create new top-level packages only when they represent genuine new gateway domain or adapter boundary.
- Treat `api/` as gateway response shapes and mapping targets. Do not leak generated client models directly from handlers.
- Validate new config in `config/Validate` when adding fields or behavior controlled by config.
- Use `context.Context` for outbound or cancellable work. Do not start background work without clear shutdown behavior.
- Keep handlers thin: extract params, call service, map domain errors to HTTP status, write response.
- Do not use `%q` in error messages or logs. If quoted formatting is necessary, prefer `'%s'`.
- Keep everything strongly typed. Do not introduce `any` or `interface{}` unless absolutely necessary and only after explicit user confirmation.
- Prefer typed structs and explicit fields over dynamic maps. If upstream payload forces dynamic data, confine it to mapping layer.
- Use shared `packages/go/logging` logger with structured fields. Log useful failure or boundary events, not routine happy-path noise.
- Do not edit `gen/` manually. Regenerate it.
- Treat `docs/openapi.yaml` and embedded docs assets as source for served API docs. Keep route changes aligned with docs updates.

## Testing

- Prefer **table-driven tests** where practical. Follow existing patterns in repository.
- Add unit tests for new logic or bug fixes when feasible without excessive mocking.
- Do not add tests for pure refactors unless behavior changes.
- Metadata-lifecycle-backed gateway endpoint tests prefer committed JSON fixtures in `testdata/metadata-lifecycle/` over inline payload.
- When adding fixture-backed tests, record source URLs in `testdata/metadata-lifecycle-fixtures.json` so fixtures can be regenerated consistently.
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

### Gateway service structure

- Keep package layout domain-first:
  - `discovery/`, `resources/` = domain use-cases.
  - `integrations/` = outbound adapters to external services (for example `integrations/metadatalifecycle`).
  - `http/` = inbound transport adapter (router, handlers, middleware).
  - `main.go` = single binary entrypoint and dependency wiring.
  - `config/` = configuration load/validate.
  - `gen/` = generated OpenAPI code (do not edit manually).
- Dependency direction is mandatory:
  - Domain packages must not import `http/` or `integrations/`.
  - Adapter packages (`http/`, `integrations/`) can import domain packages.
  - Keep business logic in domain packages, not transport or client adapter packages.

- Follow existing package structure and naming conventions.
- If similar functionality already exists, **extend it** instead of creating parallel implementation.
- Use `context.Context` for cancellable or long-running operations.
- Prefer simple, idiomatic Go over clever or complex abstractions.
- Use loop variables directly in closures. Do not add `idx := i` or `v := value` shadowing unless non-range variable actually needs rebinding.
- Maintain strong type safety:
  - Use typed enums (custom types with `const` values).
  - Avoid `interface{}` or `any` unless absolutely necessary; if used, document why.
- Don't edit `gen/` files manually.
