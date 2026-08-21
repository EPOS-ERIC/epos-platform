# Gateway Service

## Workflow

- Run the relevant generation target after changing an upstream API specification. Currently, `make generate` reads `../backoffice/docs/openapi.yaml` and regenerates `gen/backoffice`.
- Run `make test` for tests.
- `make lint` formats before linting. Do not use gofmt on files individually, run make lint.

## Package Boundaries

- Keep routing, HTTP parsing, response encoding, and gateway API mapping in `server/handlers/`.
- Keep use-case contracts, domain types, and domain errors in `domain/`. Do not add JSON tags, HTTP URLs, generated models, or transport dependencies there.
- Keep business logic and orchestration in domain-oriented service packages such as `lifecycle/` and `auth/`.
- Keep outbound adapter wiring in `clients/`; keep reusable outbound HTTP and cache behavior in `upstream/`.
- Keep shared HTTP middleware in `packages/go/middleware` instead of duplicating generic middleware in this service.
- Keep package-private helper code in `internal/`.
- Keep `main.go` limited to application wiring and startup.
- Extend an existing package when its responsibility matches. Create a new top-level package only for a genuine gateway domain or adapter boundary.

## API Boundaries

- Treat `docs/openapi.yaml` and embedded docs assets as the public gateway contract. Keep routes, `api/` types, status codes, response bodies, and docs aligned.
- Treat `api/` as gateway request and response shapes. Map explicitly between `api/` and `domain/`; do not alias them.
- Keep generated upstream clients inside `clients/`. Wrap each generated client with a gateway-owned adapter there, map its models to gateway domain contracts, and do not expose generated APIs or models to services or handlers.
- Convert upstream failures into gateway domain errors. Return only documented gateway error responses, normally `application/problem+json`; keep upstream diagnostics internal.
- Keep handlers thin: parse input, call a service, map domain errors to HTTP status, and encode the response.
- Keep URL construction such as `href` in `server/handlers/`.
- Validate new configuration fields and behavior in `config/Validate`.

## Generated Assets

- Do not edit `gen/` manually. Change the owning API specification and regenerate its client.
