# EPOS Data Retriever Service

Service that executes structured outbound HTTP requests, stores retrieval metadata, caches non-binary payloads, and offers free-text search on cached payload text.

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

Database environment variables:

- `DATA_RETRIEVER_DB_HOST` (default `localhost`)
- `DATA_RETRIEVER_DB_PORT` (default `5432`)
- `DATA_RETRIEVER_DB_NAME` (default `data_retriever`)
- `DATA_RETRIEVER_DB_USER` (default `data_retriever`)
- `DATA_RETRIEVER_DB_PASSWORD` (default `data_retriever`)

## OpenAPI

- Contract: `swagger/openapi.yaml`
- Runtime spec: `GET /openapi.yaml`
- Swagger UI: `GET /swagger`

## Database migrations

- SQL migrations: `migrations/`
