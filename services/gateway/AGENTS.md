# Agents

## General Practices

Use [Effective Go](https://go.dev/doc/effective_go) as baseline for all code.

## Error Handling

- Handle only genuine **runtime** errors.
- If error can occur only from **build-time or configuration issue** (e.g. embedded `fs` assets missing), path must have test that exercises it.
- Check all returned errors (`errcheck` enabled). Prefer `err != nil` checks with early returns.
- Prefer early returns to reduce nesting and improve readability.

## Code Style and Conventions

### Linting and Formatting

- Follow all rules enforced by `golangci-lint` (see `.golangci.yaml`).
- Formatting via `golangci-lint fmt` (gofumpt + goimports).

### Naming

- `camelCase` for local variables and unexported functions.
- `PascalCase` for exported identifiers.
- Package names and file names use **single word**.

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
- Maintain strong type safety:
  - Use typed enums (custom types with `const` values).
  - Avoid `interface{}` or `any` unless absolutely necessary; if used, document why.
- Don't edit `gen/` files manually.
