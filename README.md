# EPOS Platform

Monorepo for EPOS platform services and shared tooling.

## Current scope

- `services/metadata-lifecycle`: Kotlin service that manages metadata lifecycle in GraphDB.
- `services/gateway`: Go service scaffold (build target available; test/lint/fmt targets still pending).
- `services/data-retriever`: Go service that executes templated outbound HTTP requests, caches non-binary payloads, and provides free-text search over cached payload text.
- `docker/`: local stack definitions (GraphDB + metadata-lifecycle service).
- `packages/scripts/`: shared scripts used by repository commands.

## Repository layout

```text
.
|- apps/
|- services/
|  |- gateway/
|  |- data-retriever/
|  `- metadata-lifecycle/
|- packages/
|  |- go/
|  `- scripts/
|- docker/
|- helm/
`- cli/
```

## Quick start (Development)

Run from repository root:

```bash
make up
```

This starts local development stack with Tilt:

- GraphDB on `http://localhost:7200`
- Metadata Lifecycle API on `http://localhost:8080`
- Gateway API on `http://localhost:5000`

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
make up
make down
make test
make fmt
make lint
make populate
```

Notes:

- Root `up` and `down` are development-only commands backed by Tilt.
- Root `test`, `fmt`, and `lint` fan out to current services.
- Service image and artifact build targets stay in each service `Makefile`.

## Service documentation

- Metadata Lifecycle: `services/metadata-lifecycle/README.md`
- Gateway: `services/gateway/README.md`
- Data Retriever: `services/data-retriever/README.md`

## Current priorities

- Complete metadata lifecycle hardening and validation.
- Improve GraphDB setup and operational defaults.
- Build out backoffice-facing integration surfaces.
