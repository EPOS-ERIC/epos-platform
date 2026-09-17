# EPOS Metadata Lifecycle

Go service that owns EPOS metadata persistence and lifecycle operations. It provides users, groups, memberships, authorization, categories, contact points, organizations, metadata import and discovery, plus lifecycle management for datasets, software applications, and software source code.

PostgreSQL with PostGIS is the system of record. Database migrations are embedded in the binary and applied at startup.

## API and documentation

- API base path: `/api/v1`
- Health check: `GET /health`
- OpenAPI document: `GET /openapi.yaml`
- Scalar UI: `GET /scalar/`
- Swagger UI: `GET /swagger/`
- Source contract: `docs/openapi.yaml`

Authenticated endpoints use the `X-EPOS-User-ID` actor header internally. External clients normally call the gateway, which validates the bearer token and supplies identity to this service.

## Configuration

| Variable          | Default              |
| ----------------- | -------------------- |
| `PORT`            | `5000`               |
| `REQUEST_TIMEOUT` | `30s`                |
| `READ_TIMEOUT`    | `15s`                |
| `WRITE_TIMEOUT`   | `30s`                |
| `IDLE_TIMEOUT`    | `60s`                |
| `DB_HOST`         | `localhost`          |
| `DB_PORT`         | `5432`               |
| `DB_NAME`         | `metadata-lifecycle` |
| `DB_USER`         | `metadata-lifecycle` |
| `DB_PASSWORD`     | `metadata-lifecycle` |

`LOG_LEVEL` optionally sets shared structured logging to `DEBUG`, `INFO`, `WARN`, or `ERROR`. Standard OpenTelemetry SDK environment variables configure telemetry export.

## Development

The easiest way to run the service and its PostGIS database is the repository Tilt stack. From the repository root, run `make up`.

To run only the Go process from this directory, provide a reachable database and run `make dev`.

## Build, generation, and checks

```bash
make generate
make build
make test
make fmt
make lint
```

- `make generate` runs Go generation, including SQLC.
- `make build` regenerates code before compiling.
- `make lint` formats before running golangci-lint.
- `make docker` builds the production image; set `IMAGE` to override its tag.

SQL queries and migrations are the source of truth for generated persistence code under `db/sqlc/`; do not edit generated files manually.

Resource lifecycle transitions and permissions are described in [lifecycle.md](lifecycle.md). The OpenAPI contract remains authoritative for HTTP request and response shapes.
