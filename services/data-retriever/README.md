# EPOS Data Retriever Service

Service that executes structured outbound HTTP requests, stores retrieval metadata, caches non-binary payloads, and offers free-text search on cached payload text.

The service uses shared Go packages from `packages/go/` for structured logging, PostgreSQL connection bootstrap, and SQL migrations.

## API version

- Base path: `/api/v2`

## Build and run

Run commands from `services/data-retriever/`:

```bash
make build
make test
make fmt
make lint
```

Run service binary:

```bash
DATA_RETRIEVER_LISTEN_ADDR=:8100 ./data-retriever
```

Logging environment variables:

- `LOG_LEVEL` (optional: `DEBUG`, `INFO`, `WARN`, `ERROR`; default `INFO`)

Database environment variables:

- `DATA_RETRIEVER_DB_HOST` (default `localhost`)
- `DATA_RETRIEVER_DB_PORT` (default `5432`)
- `DATA_RETRIEVER_DB_NAME` (default `data_retriever`)
- `DATA_RETRIEVER_DB_USER` (default `data_retriever`)
- `DATA_RETRIEVER_DB_PASSWORD` (default `data_retriever`)

Runtime environment variables:

- `DATA_RETRIEVER_HTTP_TIMEOUT` (default `30s`): outbound HTTP request timeout
- `DATA_RETRIEVER_MIGRATIONS_DIR` (default `migrations`): SQL migration directory
- `DATA_RETRIEVER_DB_CONNECT_MAX_ATTEMPTS` (default `20`)
- `DATA_RETRIEVER_DB_CONNECT_RETRY_DELAY` (default `2s`)

Standard OpenTelemetry SDK environment variables configure telemetry export. The repository development stack sends OTLP data to the local LGTM container.

## OpenAPI

- Contract: `swagger/openapi.yaml`
- Runtime spec: `GET /openapi.yaml`
- Swagger UI: `GET /swagger`

## Health check

- Endpoint: `GET /health`
- Returns `200` with `{"status":"ok"}` when the service is healthy and database is reachable.
- Returns `503` with `{"status":"degraded","reason":"database unreachable"}` when database ping fails.

## Database migrations

- SQL migrations: `migrations/`
- Runtime migration runner is shared via `packages/go/migrations`.

Migrations are applied automatically when the service starts.

## Docker

Build the production image from this directory with `make docker`. Set `IMAGE` to override the default image name and tag.
