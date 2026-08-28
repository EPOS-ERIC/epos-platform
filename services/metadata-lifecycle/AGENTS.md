# Metadata Lifecycle Service

## Architecture Rules

- Keep business logic in the owning domain package like `identity/`, `authz/`, `groups/`.
- Keep domain packages independent from `httpapi/`, `server/`, and PostgreSQL implementation packages. Depend on interfaces where an adapter is required.
- Keep RDF/Turtle parsing and mapping in `turtle/`; do not put parser-specific behavior in domain or HTTP packages.
- Keep HTTP routing, transport types, request validation, and response mapping in `httpapi/`. Keep handlers thin and delegate use-case behavior to domain services.
- Keep server startup, shutdown, runtime lifecycle, and dependency wiring in `server/` and `main.go`. Do not put business logic there.
- Keep PostgreSQL bootstrap, migrations, repositories, and persistence adapters in `db/`. Do not leak database or SQLC types into HTTP contracts or domain types.
- Define API transport types separately from domain types. Map explicitly instead of aliasing types across boundaries.
- Extend the package that owns a responsibility. Add a new package only for a distinct domain or adapter boundary.

## SQLC

- Load the `sqlc-go` skill before working with SQLC, SQL queries, migrations, or generated database code.
- Treat SQL queries and migrations as the source of truth. Do not edit `db/sqlc/` manually.
- Read this service's `sqlc.yaml` before changing queries or generation settings.
- Run `make generate` after SQL or migration changes. Run `make build` to verify generated code compiles.

## Domain And Mapping

- PUT only, no PATCH. Required field is `T`, nullable scalar is `*T`, collection is `[]T`; absent and `null` both decode to `nil` and mean clear. Do not use `optionalString`, `Field[T]`, or `*[]T`.
- `XInput` holds caller-supplied fields, `X` is `XInput` plus system IDs. `NewXInput` owns all input validation; `NewX(id, XInput, children…)` delegates to `NewXInput` and child constructors. Pass `XInput` as a struct, not flattened params. Keep fields exported and re-validate in `NewX` intentionally.
- Split child `ChildInput`/`Child` only when a second caller needs it. Do not embed `XInput` in `X` unless names and types are an identical subset.
- Use one request type for create and replace PUT when possible; define required DTO fields as `T` and validate via `NewXInput`, no pre-check in mapping. Service and DB trust validated input; DB reads reassemble through `NewX`.
- Per package: `types.go` plus `mapping.go` plus logic/handler/repo. `domain/` is `*_types.go` plus `*_parsing.go`; handler `mapping.go` is DTO to domain, repo `mapping.go` is `sqlc` to domain including `toCreate*Params` and `toDomain*`. Prefer generic `mapSlice` over per-type loops.

## Workflow

- `make generate` runs `go generate ./...`.
- `make build` regenerates SQLC output before building.
- `make test` runs `go test ./...`.
- `make lint` formats before linting. Do not use gofmt on files individually, run make lint.
