# EPOS User Group Service

Service that stores users, groups, memberships, and role-based authorization data used by the orchestrator.

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

## Database migrations

- SQL migrations: `migrations/`
