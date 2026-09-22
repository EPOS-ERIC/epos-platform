# EPOS Data Retriever Service

Service that executes structured outbound HTTP requests, stores retrieval metadata, and persists output schemas for resource versions.

The service uses shared Go packages from `packages/go/` for structured logging and PostgreSQL connection bootstrap.

## gRPC API

- Contract: `packages/protobuf/data_retriever/v1/data_retriever.proto`
- Service: `data_retriever.v1.DataRetrieverService`
- Health: standard gRPC health service
- `RetrieveStream` requires `resource_id` and `version_id`; callers do not provide output parameters. On a successful complete response, the service detects JSON, XML, and CSV payloads, extracts their fields or tags, and atomically replaces the stored output parameters for that resource version. Extraction buffers a complete payload of at most `DATA_RETRIEVER_SCHEMA_MAX_BYTES` (default `2 MiB`); unsupported and larger payloads retain the previously discovered parameters.
- `RetrieveStream` streams the upstream body in 64 KiB chunks with bounded memory and no size limit; payloads are never persisted, only metadata (status, content type, hash, exact byte size). A retrieval record is stored before streaming starts and finalized with the exact hash and size once the stream completes; interrupted streams keep an empty hash and `-1` size. Redirects are never followed: a 3xx upstream response yields the status code, the upstream `Location` in `redirect_url`, and the upstream `Content-Disposition` (original file name) in `content_disposition`. When the upstream omits `Content-Disposition`, one is synthesized as `inline; filename="<name>"` from the file name in the request URL. `DATA_RETRIEVER_HTTP_TIMEOUT` bounds upstream response headers only; bodies stream until the caller cancels.

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
- `DATA_RETRIEVER_HTTP_TIMEOUT` (default `30s`): outbound HTTP response-header timeout
- `DATA_RETRIEVER_SCHEMA_MAX_BYTES` (default `2097152`): maximum complete payload size used for schema extraction; set to `0` to disable extraction
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
