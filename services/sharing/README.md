# EPOS Sharing

Go gRPC service for personal and official configuration persistence. PostgreSQL is the system of record; database migrations are embedded in the binary and applied at startup.

## API and documentation

- gRPC service: `configuration.v1.ConfigurationService`
- Default listen address: `:8200`
- Authorization model: [sharing.md](sharing.md)

External clients normally call the Gateway, which validates bearer tokens and forwards trusted actor and management metadata to Sharing.

## Configuration

| Variable      | Default     |
| ------------- | ----------- |
| `SERVER_PORT` | `8200`      |
| `DB_HOST`     | `localhost` |
| `DB_PORT`     | `5432`      |
| `DB_NAME`     | `sharing`   |
| `DB_USER`     | `sharing`   |
| `DB_PASSWORD` | `sharing`   |

`LOG_LEVEL` optionally sets shared structured logging to `DEBUG`, `INFO`, `WARN`, or `ERROR`. Standard OpenTelemetry SDK environment variables configure telemetry export.

## Development

The easiest way to run the service and its PostgreSQL database is the repository Tilt stack. From the repository root, run `make up`.

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
