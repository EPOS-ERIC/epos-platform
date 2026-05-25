# EPOS User Group Service

Service that stores users, groups, memberships, and role-based authorization data used by the orchestrator.

Membership and friendship APIs are available to manage user roles in groups and draft-sharing relationships.

The service uses shared Go packages from `packages/go/` for structured logging, PostgreSQL connection bootstrap, and SQL migrations.

## API version

- Base path: `/api/v2`

## Supported group roles

- `VIEWER`
- `EDITOR`
- `REVIEWER`
- `ADMIN`

One role is assigned per user per group.

## Build and run

Run commands from `services/user-group/`:

```bash
make build
make test
make fmt
make lint
```

Run service binary:

```bash
USER_GROUP_LISTEN_ADDR=:8090 ./user-group
```

Logging environment variables:

- `LOG_LEVEL` (optional: `DEBUG`, `INFO`, `WARN`, `ERROR`; default `INFO`)
- `LOG_FILE` (optional: write JSON logs to file instead of stdout)

Database environment variables:

- `USER_GROUP_DB_HOST` (default `localhost`)
- `USER_GROUP_DB_PORT` (default `5432`)
- `USER_GROUP_DB_NAME` (default `user_group`)
- `USER_GROUP_DB_USER` (default `user_group`)
- `USER_GROUP_DB_PASSWORD` (default `user_group`)

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
