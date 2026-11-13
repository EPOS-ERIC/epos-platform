# Gateway Service

## Workflow

- Run `make generate` after changing the Gateway OpenAPI specification; it regenerates `gen/gatewayapi`.

## Package Boundaries

- Keep routing, HTTP parsing, transport validation, actor extraction, response encoding, and Gateway API mapping in `server/handlers/`. Handlers must not implement Gateway-specific authorization, business rules, or multi-client orchestration.
- Keep Gateway-owned business concepts, orchestration state, and domain errors in `domain/`. Do not mirror an upstream service's domain model only to proxy it; orchestrated resources may define full domain representations when their response requires them.
- Keep Gateway-specific authorization, business rules, and multi-client orchestration in domain-oriented service packages. An endpoint requiring any of these must call the owning service; the service applies the policy and calls its clients. A service may also expose related operations when it already owns that resource's orchestration.
- Pure proxy endpoints do not need a Gateway service or Gateway domain DTO. Their handlers may call a concrete Gateway-owned client adapter directly. When a service already owns orchestration for the same resource, its related endpoints may use that service and its domain types instead of maintaining a separate proxy path.
- Keep outbound connection setup, gRPC metadata, deadlines, retries, transport mapping, and gRPC-status-to-Gateway-error mapping in `clients/`. Clients do not own Gateway authorization, business rules, or multi-client orchestration. Handlers must not call generated gRPC clients directly.
- Generated protobuf responses may pass from `clients/` to handlers for pure proxies or unchanged response data alongside separate orchestration enrichment. Never expose protobuf types in the external API; map them to explicit API types defined by OpenAPI.
- Orchestration paths map protobuf to a minimal Gateway domain type before applying Gateway behavior. Do not store protobuf messages in Gateway domain objects or request context.
- Do not create Gateway domain DTOs, services, or mappers that merely duplicate an upstream protobuf shape. Remove an obsolete proxy-only domain layer when the last caller migrates.
- Keep Gateway HTTP middleware in `server/`.
- Keep package-private helper code in `internal/`.
- Keep `main.go` limited to application wiring and startup.
- Extend an existing package when its responsibility matches. Create a new top-level package only for a genuine gateway domain or adapter boundary.

## API Boundaries

- Treat `docs/openapi.yaml` and embedded docs assets as the public Gateway HTTP contract. Keep routes, `api/` types, status codes, response bodies, and docs aligned.
- Treat `api/` as public Gateway HTTP request and response shapes. Handler owns mapping from an upstream protobuf response to `api/` for pure proxy endpoints.
- For pure proxies use `protobuf response -> api response`. For orchestration map fields used for behavior to minimal Gateway domain types; an otherwise unchanged protobuf response may pass through alongside separate enrichment.
- Clients convert upstream gRPC failures to Gateway domain errors. Handlers map those errors to documented HTTP responses and never expose upstream diagnostics.
- Keep handlers thin: parse input, call a service, map domain errors to HTTP status, and encode the response.
- Authentication and generic access checks may run in HTTP security middleware. Pass the authenticated principal to services that enforce Gateway-owned permissions; an actor ID alone is not proof of authorization.
- Upstream services remain authoritative for their own data and permissions. Do not duplicate upstream authorization in Gateway unless Gateway owns an additional policy.
- Separate explicit privileged operations from automatic orchestration. Enforce explicit permissions in the owning service; constrain automatic operations to the authorized workflow and its results, without exposing an authorization bypass to callers.
- A Gateway service boundary is not a network security boundary. When upstream services trust Gateway-provided identity or permissions, restrict or authenticate direct access to those services.
- Keep URL construction such as `href` in `server/handlers/`.
- Validate new configuration fields and behavior in `config/Validate`.

## Error Handling

- Use the shared `packages/go/apperror` contract.
- Gateway clients own gRPC-status decoding and classify upstream availability/deadline failures. Handlers map application errors to documented HTTP responses.
- Keep HTTP Problem fields safe: the type identifies the application outcome, title follows the HTTP reason phrase, detail is client-safe, and target is included only when known. Never expose upstream diagnostics or error causes.
- Preserve nested validation paths and array indices when mapping ogen validation errors. Use actionable safe details for known validation failures; use generic safe detail for unknown validator errors.
- Log/report errors at the HTTP boundary: expected client errors at debug level; 5xx errors at error level and on the active trace.

## Mapping Rules

- Map exactly once at each transport boundary. For a pure proxy that calls a client directly: `api request -> protobuf request` in the handler, then `protobuf response -> api response` in the handler. When using a shared service's domain types, map between API and domain in the handler and between domain and protobuf in the client.
- For Gateway orchestration: map fields used by the service to minimal Gateway domain types. Unchanged protobuf responses may pass through alongside separate enrichment for mapping by the handler.
- Keep request and response mappings for one resource family together in a dedicated `*_mapping.go` file. Do not place mapping functions in handler method files except small private helpers used by that one handler.
- Name conversions by source and destination: `toProtoX`, `toAPIX`, and `fromProtoX`. Do not use ambiguous names such as `mapX`, `convertX`, or names that mix destination concepts such as `toAPIProtoX`.
- Define shared protobuf-to-OpenAPI primitives (UUID, timestamp, nullable scalar, enum) once in `mapping_helpers.go`. Resource mappings must use those helpers instead of recreating nullable or timestamp semantics.
- Mapping functions validate only representation-level invariants: required nested messages, enum values, UUID parsing, timestamp validity, and OpenAPI nullability. Business validation belongs in domain services.
- A mapping function must not make network calls, read configuration, inspect authentication, or mutate state.

## gRPC

- Keep generated gRPC clients behind concrete Gateway-owned adapters in `clients/`. Handlers must not call generated gRPC clients directly.
- Pass actor identity through gRPC metadata from Gateway clients. Do not add actor identity to protobuf business requests or Gateway domain objects.
- Gateway clients may return protobuf responses unchanged for pure proxies or passthrough enrichment. For orchestration decisions, adapter methods map required fields to minimal Gateway domain types.
- When a Gateway service returns an unchanged protobuf response, add a comment explaining the passthrough, such as `// Protobuf passes through unchanged; no response orchestration.`

## Generated Assets

- Do not edit `gen/` manually. Change the owning API specification and regenerate its client.
