# EPOS Data Retriever Service

Data Retriever executes prepared outbound HTTP requests, streams response bodies, records retrieval metadata, and discovers output parameters from supported text formats.

## gRPC API

- Listen address: `GRPC_LISTEN_ADDR` (default `:8100`)
- Contract: `packages/protobuf/data_retriever/v1/data_retriever.proto`
- `RetrieveStream` executes a fully prepared HTTP method, URL, headers and optional raw body. It preserves URL encoding, repeated query keys and body bytes. No template rendering, DRY mode, curl generation or dedicated bearer-token field is supported.
- GET and DELETE bodies are rejected, including explicitly empty bodies.
- Web-service access persists retrieval records and discovers output parameters. Downloads set `skip_persistence` to skip both.
- `GetRetrieval` returns persisted retrieval metadata and discovered output parameters.
- Standard gRPC health service is enabled for readiness checks.


## Build, test, and run

Run commands from `services/data-retriever/`:

```bash
make generate
make build
make test
make lint
```

`make dev` generates SQLC code and runs the service. Database migrations are embedded from `db/migrations/` and applied on startup through the shared PostgreSQL package.

## Configuration

- `GRPC_LISTEN_ADDR` (default `:8100`)
- `DB_HOST` (default `localhost`)
- `DB_PORT` (default `5432`)
- `DB_NAME` (default `data_retriever`)
- `DB_USER` (default `data_retriever`)
- `DB_PASSWORD` (default `data_retriever`)
- `HTTP_TIMEOUT` (default `30s`): upstream response-header timeout
- `SCHEMA_MAX_BYTES` (default `2097152`): maximum response bytes sampled for schema discovery; `0` disables discovery.
- `MAX_RESPONSE_BYTES` (default `1073741824`): maximum response body bytes streamed from upstream.

## Sensitive data

Request body text is persisted verbatim, including credentials. Request body text is never returned or logged. `Authorization`, `Cookie`, and `Set-Cookie` request headers are redacted before persistence; credential values are never logged..

## Development stack

Run the repository stack with `make up`. Data Retriever listens on gRPC port `8100`; its PostgreSQL database is exposed locally on port `5433`.
