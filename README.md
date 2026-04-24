# EPOS Platform

Monorepo for EPOS platform services and shared tooling.

## Current scope

- `services/metadata-lifecycle`: Kotlin service that manages metadata lifecycle in GraphDB.
- `services/gateway`: Go service scaffold (build target available; test/lint/fmt targets still pending).
- `docker/`: local stack definitions (GraphDB + metadata-lifecycle service).
- `packages/scripts/`: shared scripts used by repository commands.

## Repository layout

```text
.
|- apps/
|- services/
|  |- gateway/
|  `- metadata-lifecycle/
|- packages/
|  |- go/
|  `- scripts/
|- docker/
|- helm/
`- cli/
```

## Quick start (Docker)

Run from repository root:

```bash
make up
```

This starts:

- GraphDB on `http://localhost:7200`
- Metadata Lifecycle API on `http://localhost:8080`

Useful endpoints:

- API docs (Swagger): `http://localhost:8080/swagger`
- OpenAPI spec: `http://localhost:8080/openapi.yaml`
- Health: `http://localhost:8080/health`

Stop the stack:

```bash
make down
```

## Development commands

From repository root:

```bash
make build
make test
make fmt
make lint
make populate
```

Notes:

- Root `test`, `fmt`, and `lint` currently run for `metadata-lifecycle` only.
- Gateway currently provides `make build` and `make generate` in `services/gateway/Makefile`.

## Service documentation

- Metadata Lifecycle: `services/metadata-lifecycle/README.md`
- Gateway: `services/gateway/README.md`

## Current priorities

- Complete metadata lifecycle hardening and validation.
- Improve GraphDB setup and operational defaults.
- Build out backoffice-facing integration surfaces.
