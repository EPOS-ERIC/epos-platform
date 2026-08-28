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
- `XInput` holds caller-supplied fields, `X` is `XInput` plus system IDs. `NewXInput` owns input validation and normalization; `NewX` revalidates and canonicalizes persisted values, including reads. Pass input as a struct. Services and DB trust validated input.
- Keep transport, domain, and persistence types separate and map explicitly at boundaries. Use one request type for create and replace when shapes match. Generate system IDs in storage.
- Split child input and persisted child types only when another caller needs the distinction.

## Actors And Errors

- Use typed `domain.Actor`. Parse at transport boundary with `NewActor`; services trust parsed actors and skip trim/empty checks.
- Optional actor: absent is allowed, malformed header writes `400`; required actor writes `401` when absent. Convert `domain.Actor` to `string` at adapter boundaries when string required.
- Handler package use shared service-error mapping function. Match wrapped sentinels with `errors.Is`, keep status/code pairs consistent.

## Workflow

- `make generate` runs `go generate ./...`.
- `make build` regenerates SQLC output before building.
- `make test` runs `go test ./...`.
- `make lint` formats before linting. Do not use gofmt on files individually, run make lint.
