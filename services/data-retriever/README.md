# EPOS Data Retriever Service

Data Retriever executes structured outbound HTTP requests, streams response bodies, records retrieval metadata, and discovers output parameters from supported text formats.

## gRPC API

- Listen address: `GRPC_LISTEN_ADDR` (default `:8100`)
- Contract: `packages/protobuf/data_retriever/v1/data_retriever.proto`
- `RetrieveStream` supports `DRY` curl generation and `EXECUTE` streaming.
- `GetRetrieval` returns persisted retrieval metadata and discovered output parameters.
- Standard gRPC health service is enabled for readiness checks.
- Public HTTP access goes through Gateway `POST /v1/access`.

In execute mode, Data Retriever sends response headers first, then bounded chunks. It does not follow redirects. Gateway relays upstream status and response content. Output parameters update only after a complete supported JSON, XML, or CSV body parses within `SCHEMA_MAX_BYTES`.

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
- `HTTP_TIMEOUT` (default `30s`): upstream response-header timeout; response bodies have no overall timeout.
- `SCHEMA_MAX_BYTES` (default `2097152`): maximum response bytes sampled for schema discovery; `0` disables discovery.

## Sensitive data

Request body text is persisted verbatim, including credentials. Request body text is never returned or logged. Dedicated bearer tokens remain in memory only. `Authorization`, `Cookie`, and `Set-Cookie` request headers are redacted before persistence; credential values are never logged.

## Development stack

Run the repository stack with `make up`. Data Retriever listens on gRPC port `8100`; its PostgreSQL database is exposed locally on port `5433`.
