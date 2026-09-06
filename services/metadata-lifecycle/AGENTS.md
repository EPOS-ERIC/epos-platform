# Metadata Lifecycle Service

## Architecture Rules

- Keep business logic in the owning domain package like `identity/`, `authz/`, `groups/`.
- Keep domain packages independent from `httpapi/`, `grpcapi/`, `server/`, protobuf-generated packages, and PostgreSQL implementation packages. Depend on interfaces where an adapter is required.
- Keep RDF/Turtle parsing and mapping in `turtle/`; do not put parser-specific behavior in domain or HTTP packages.
- Keep HTTP routing, transport types, request validation, and response mapping in `httpapi/`. Keep handlers thin and delegate use-case behavior to domain services.
- Keep gRPC registration, request validation, metadata extraction, status mapping, and protobuf/domain conversion in `grpcapi/`. gRPC handlers must be thin and delegate use-case behavior to domain services.
- Keep server startup, shutdown, runtime lifecycle, and dependency wiring in `server/` and `main.go`. Do not put business logic there.
- Keep PostgreSQL bootstrap, migrations, repositories, and persistence adapters in `db/`. Do not leak database or SQLC types into HTTP contracts or domain types.
- Define API transport types separately from domain types. Map explicitly instead of aliasing types across boundaries.
- Extend the package that owns a responsibility. Add a new package only for a distinct domain or adapter boundary.

## gRPC And Mapping

- Protobuf messages are transport types, not domain types. Never expose them from domain services, repositories, or database adapters.
- Map exactly once at the gRPC boundary: `protobuf request -> domain input` before invoking a service and `domain result -> protobuf response` after it returns. Do not route protobuf messages through HTTP transport types or persistence types.
- Keep mappings for one resource family together in a dedicated `*_mapping.go` file or focused mapper type. Do not accumulate unrelated resource, lifecycle, validation, and mapping logic in one RPC implementation file.
- Name conversions by direction: `toDomainX` for protobuf-to-domain and `toProtoX` for domain-to-protobuf. Do not use ambiguous `mapX` or `convertX` names.
- Share representation helpers for UUIDs, timestamps, optional scalars, enums, and actor metadata. Do not duplicate their validation or nullability behavior across RPC services.
- Mapping validates only transport representation: missing nested messages, malformed UUIDs or timestamps, unknown enums, and protobuf presence. Domain constructors own business validation and normalization.
- Parse required actor metadata once at the gRPC transport boundary. Pass the typed `domain.Actor` to services; do not reparse it in services or encode it into protobuf business messages.
- Convert domain errors to gRPC status errors at the gRPC boundary. Do not return raw domain errors or database errors from RPC methods.

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

