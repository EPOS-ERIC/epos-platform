# EPOS Data Retriever Service

Service that executes structured outbound HTTP requests, stores retrieval metadata, and persists output schemas for resource versions.

The service uses shared Go packages from `packages/go/` for structured logging and PostgreSQL connection bootstrap.

## gRPC API

- Contract: `packages/protobuf/data_retriever/v1/data_retriever.proto`
- Service: `data_retriever.v1.DataRetrieverService`
- Health: standard gRPC health service
- `Retrieve` requires `resource_id`, `version_id`, and output parameters; it atomically replaces the schema for that resource version.
- Payloads are returned inline only at runtime for non-binary responses and are never persisted. Binary responses return a `redirect_url`, so large downloads bypass the service.

## Build and run

Run commands from `services/data-retriever/`:

```bash
make generate
make build
make test
make fmt
make lint
```

Run service binary:

```bash
DATA_RETRIEVER_GRPC_LISTEN_ADDR=:8100 ./data-retriever
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

- `DATA_RETRIEVER_GRPC_LISTEN_ADDR` (default `:8100`)
- `DATA_RETRIEVER_HTTP_TIMEOUT` (default `30s`): outbound HTTP request timeout
- `DATA_RETRIEVER_DB_CONNECT_MAX_ATTEMPTS` (default `20`)
- `DATA_RETRIEVER_DB_CONNECT_RETRY_DELAY` (default `2s`)

Standard OpenTelemetry SDK environment variables configure telemetry export. The repository development stack sends OTLP data to the local LGTM container.

## Health check

- Standard gRPC health service on the configured listen address.

## Database migrations

- Versioned SQL migrations are in `storage/migrations/`, embedded in the service binary, and applied with Goose.
- SQLC queries: `repository/queries/`; generated query code is under `repository/sqlc/` and must not be edited manually.

Migrations are applied automatically when the service starts.

## Docker

Build the production image from this directory with `make docker`. Set `IMAGE` to override the default image name and tag.
