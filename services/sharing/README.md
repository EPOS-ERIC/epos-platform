# Sharing Service

Go port of the legacy `configuration-sharing-service` (Java/Spring). Stores
opaque workspace configuration documents plus descriptive metadata,
visibility flags and ownership.

Differences from the legacy service:

- No AES/salt encode/decode: the payload is stored as JSONB.
- No normalize/denormalize step: the payload is stored as canonical JSON
  (semantically identical to what the client sent).
- `configuration` is schemaless JSON so future payloads keep working.
- Metadata (`title`, `description`, `listOfServices`) lives in dedicated
  columns and stays queryable.
- New: official/unofficial flag, public/private visibility and ownership.

## API

gRPC (`ConfigurationService`, see
`packages/protobuf/configuration/v1/configuration.proto`):

- `CreateConfiguration` — persist a configuration; server generates the UUID
  when `id` is absent. Requires an authenticated caller; `is_official`
  requires an administrator.
- `GetConfiguration` — fetch one configuration by ID, enforcing visibility.
- `ListConfigurations` — paginated list (`page_size` 1–100, `offset`,
  optional `official` filter), newest first, scoped to what the caller may
  see.
- `UpdateConfiguration` — full replace (PUT semantics) by ID. Owner or
  administrator only; only administrators may change `is_official`.
- `DeleteConfiguration` — delete by ID. Owner or administrator only.

The caller identity (`x-epos-user-id`, `x-epos-admin`) arrives via gRPC
metadata attached by the gateway after bearer validation. Missing identity
means anonymous (public configurations only); mutating RPCs reject anonymous
callers with `Unauthenticated`.

HTTP is exposed through the gateway at `/v1/configurations`.

## Authorization matrix

| Operation | Anonymous | Owner | Other user | Admin |
| --- | --- | --- | --- | --- |
| Create unofficial | 401 | 201 | 201 | 201 |
| Create official | 401 | 403 | 403 | 201 |
| Read public | 200 | 200 | 200 | 200 |
| Read private | 404 | 200 | 404 | 200 |
| Replace/delete | 401 | 200/204 | 403/404 | 200/204 |
| Change official flag | 401 | 403 | 403/404 | 200 |

Official configurations must be public (enforced in the domain layer and by
a database `CHECK` constraint).

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `SHARING_GRPC_LISTEN_ADDR` | `:8200` | gRPC listen address |
| `SHARING_DB_HOST` | `localhost` | PostgreSQL host |
| `SHARING_DB_PORT` | `5432` | PostgreSQL port |
| `SHARING_DB_NAME` | `sharing` | Database name |
| `SHARING_DB_USER` | `sharing` | Database user |
| `SHARING_DB_PASSWORD` | `sharing` | Database password |
| `SHARING_DB_CONNECT_MAX_ATTEMPTS` | `20` | Connection retry attempts |
| `SHARING_DB_CONNECT_RETRY_DELAY` | `2s` | Delay between attempts |

## Development

```sh
make generate  # regenerate sqlc output
make build     # build the binary (runs generate first)
make test      # run unit tests
```
