# Gateway Service

## Workflow

- Run the relevant generation target after changing an upstream API specification. Currently, `make generate` reads `../metadata-lifecycle/docs/openapi.yaml` and regenerates `gen/metadata-lifecycle`.

## Package Boundaries

- Keep routing, HTTP parsing, response encoding, and Gateway API mapping in `server/handlers/`.
- Keep Gateway-owned business concepts, orchestration state, and domain errors in `domain/`. Do not mirror an upstream service's domain model only to proxy it.
- Keep Gateway orchestration in domain-oriented service packages. Create a service method only when it resolves Gateway-specific semantics, combines upstream calls, applies Gateway policy, or owns workflow state.
- Pure proxy endpoints do not need a Gateway service or Gateway domain DTO. Their handlers may call a concrete Gateway-owned client adapter directly.
- Keep outbound connection setup, gRPC metadata, deadlines, retries, and gRPC-status-to-Gateway-error mapping in `clients/`. Handlers must not call generated gRPC clients directly.
- Generated protobuf response types may cross from `clients/` into `server/handlers/` only for a pure proxy endpoint. Handler maps protobuf directly to `api/` response types. Never expose direct protobuf types to the external api, always define explicit api types following the openapi specification and map protobuf to those.
- Orchestration paths map protobuf to a minimal Gateway domain type before applying Gateway behavior. Do not store protobuf messages in Gateway domain objects or request context.
- Do not create Gateway domain DTOs, services, or mappers that merely duplicate an upstream protobuf shape. Remove an obsolete proxy-only domain layer when the last caller migrates.
- Keep shared HTTP middleware in `packages/go/middleware` instead of duplicating generic middleware in this service.
- Keep package-private helper code in `internal/`.
- Keep `main.go` limited to application wiring and startup.
- Extend an existing package when its responsibility matches. Create a new top-level package only for a genuine gateway domain or adapter boundary.

## API Boundaries

- Treat `docs/openapi.yaml` and embedded docs assets as the public Gateway HTTP contract. Keep routes, `api/` types, status codes, response bodies, and docs aligned.
- Treat `api/` as public Gateway HTTP request and response shapes. Handler owns mapping from an upstream protobuf response to `api/` for pure proxy endpoints.
- For pure proxies use `protobuf response -> api response`. For orchestration use `protobuf response -> minimal Gateway domain type -> api response or next action`.
- Clients convert upstream gRPC failures to Gateway domain errors. Handlers map those errors to documented HTTP responses and never expose upstream diagnostics.
- Keep handlers thin: parse input, call a service, map domain errors to HTTP status, and encode the response.
- Keep URL construction such as `href` in `server/handlers/`.
- Validate new configuration fields and behavior in `config/Validate`.

## Mapping Rules

- Map exactly once at each transport boundary. For a pure proxy: `api request -> protobuf request` in the handler, then `protobuf response -> api response` in the handler. Do not introduce a Gateway domain type between those mappings.
- For Gateway orchestration: client returns protobuf, the orchestration service maps only the fields it needs to a Gateway domain type, and the handler maps the final Gateway result to `api/`.
- Keep request and response mappings for one resource family together in a dedicated `*_mapping.go` file. Do not place mapping functions in handler method files except small private helpers used by that one handler.
- Name conversions by source and destination: `toProtoX`, `toAPIX`, and `fromProtoX`. Do not use ambiguous names such as `mapX`, `convertX`, or names that mix destination concepts such as `toAPIProtoX`.
- Define shared protobuf-to-OpenAPI primitives (UUID, timestamp, nullable scalar, enum) once in `mapping_helpers.go`. Resource mappings must use those helpers instead of recreating nullable or timestamp semantics.
- Mapping functions validate only representation-level invariants: required nested messages, enum values, UUID parsing, timestamp validity, and OpenAPI nullability. Business validation belongs in domain services.
- A mapping function must not make network calls, read configuration, inspect authentication, or mutate state.

## gRPC

- Keep generated gRPC clients behind concrete Gateway-owned adapters in `clients/`. Handlers must not call generated gRPC clients directly.
- Pass actor identity through gRPC metadata from Gateway clients. Do not add actor identity to protobuf business requests or Gateway domain objects.
- For pure proxy calls, Gateway client methods return generated protobuf responses unchanged. For orchestration calls, add a separate adapter method that maps the protobuf response to the minimal required Gateway domain type.

## Generated Assets

- Do not edit `gen/` manually. Change the owning API specification and regenerate its client.
