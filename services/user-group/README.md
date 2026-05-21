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

## OpenAPI

- Contract: `openapi.yaml`
- Runtime spec: `GET /openapi.yaml`
- Swagger UI: `GET /swagger`

## Database migrations

- SQL migrations: `migrations/`
