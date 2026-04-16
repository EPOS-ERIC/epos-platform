# Agents

## General Practices

Use [Effective Go](https://go.dev/doc/effective_go) guide as baseline for all code.

## Error Handling

- Handle only genuine **runtime** errors.
- If error can occur only due to **build-time or configuration issue** (e.g. embedded `fs` assets missing), **panic** instead — do not silently swallow or return it.
- Every panic path must have corresponding test that exercises it.
- Check all returned errors (`errcheck` enabled). Prefer `err != nil` checks with early returns.
- Prefer early returns to reduce nesting and improve readability.

## Code Style and Conventions

### Linting and Formatting

- Follow all rules enforced by `golangci-lint` (see `.golangci.yaml`).
- Formatting enforced via `golangci-lint fmt` (gofumpt + goimports).

### Naming

- `camelCase` for local variables and unexported functions.
- `PascalCase` for exported identifiers.
- Package names and file names use **single word**.

## Testing

- Prefer **table-driven tests** where practical — follow existing patterns in repository.
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

- Use [slog](https://pkg.go.dev/log/slog)-style structured logging.
- Log on error-prone or failure paths.
- Do **not** over-log — avoid noise in happy paths.

## Architecture and Safety

- Follow existing package structure and naming conventions.
- If similar functionality already exists, **extend it** instead of creating parallel implementation.
- Use `context.Context` for cancellable or long-running operations.
- Keep code **portable** — must build and run correctly on Linux, macOS, and Windows. Avoid OS-specific behavior.
- Prefer simple, idiomatic Go over clever or complex abstractions.
- Maintain strong type safety:
  - Use typed enums (custom types with `const` values).
  - Avoid `interface{}` or `any` unless absolutely necessary; if used, document why.
