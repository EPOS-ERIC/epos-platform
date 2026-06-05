# AGENTS

## Build and Test Workflow

- Run commands from `apps/platform` with `make build`, `make test`, `make fmt`, and `make lint`.
- The Angular project lives directly in this directory; keep app tooling aligned with repo tooling.
- Regenerate API types with `make codegen` after gateway OpenAPI changes.
- Local development proxies `/v1` to the gateway through `proxy.conf.cjs`.

## App Conventions

- Keep API base URL wiring in Angular providers, not hard-coded in components.
- Keep generated OpenAPI TypeScript under `src/app/api/generated/`; do not edit generated files manually.
- Prefer standalone Angular components and strict TypeScript types.
