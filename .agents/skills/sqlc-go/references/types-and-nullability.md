# Types and nullability with pgx/v5

This repo uses `sql_package: "pgx/v5"` with `emit_pointers_for_null_types: true`. Nullable standard scalar fields use pointers, not `pgtype.*` or `sql.Null*`. Follow service's actual `sqlc.yaml` if config differs.

## Type mapping (Postgres → Go, pgx/v5)

| Postgres                    | Non-null Go            | Nullable Go             |
| --------------------------- | ---------------------- | ----------------------- |
| `text`, `varchar`           | `string`               | `*string`               |
| `int4`, `serial`            | `int32`                | `*int32`                |
| `int8`, `bigserial`         | `int64`                | `*int64`                |
| `int2`                      | `int16`                | `*int16`                |
| `bool`                      | `bool`                 | `*bool`                 |
| `numeric`                   | `pgtype.Numeric`       | `*pgtype.Numeric`       |
| `float4` / `float8`         | `float32` / `float64`  | `*float32` / `*float64` |
| `timestamp` / `timestamptz` | `time.Time`            | `*time.Time`            |
| `date`                      | `time.Time`            | `*time.Time`            |
| `uuid`                      | `uuid.UUID`            | `*uuid.UUID`            |
| `jsonb`/`json`              | `[]byte` (or override) | `[]byte`                |
| `bytea`                     | `[]byte`               | `[]byte`                |
| arrays                      | `[]string` etc.        | same — nil slice = NULL |

`numeric` may remain `pgtype.Numeric`; pointer wraps nullable value. This repo overrides `timestamptz` to `time.Time` and `uuid` to `uuid.UUID`.

Generated enum `NullX` wrappers may remain where sqlc cannot emit pointers. Follow generated types; never hand-edit output.

## Using pointer values

```go
// read
if user.MiddleName != nil { fmt.Println(*user.MiddleName) }

// write non-null
middleName := "Ann"
params.MiddleName = &middleName

// write explicit NULL
params.MiddleName = nil
```

Use `nil` for NULL. Check service for existing pointer helpers before adding new ones.

## sqlc.arg vs sqlc.narg

```sql
-- name: SearchUsers :many
SELECT * FROM users
WHERE (sqlc.narg(name_filter)::text IS NULL OR name ILIKE '%' || sqlc.narg(name_filter) || '%')
  AND status = sqlc.arg(status);
```

- `sqlc.arg(x)` — explicit non-null param.
- `sqlc.narg(x)` — explicit nullable pointer param. Use for optional filters with `col = sqlc.narg(x) OR sqlc.narg(x) IS NULL`, not dynamic SQL.

## overrides: in sqlc.yaml

```yaml
overrides:
  - db_type: "uuid"
    go_type: "github.com/google/uuid.UUID"
  - column: "users.metadata"
    go_type: "encoding/json.RawMessage"
```

Check this block before assuming default mapping. Pointer overrides control nullable custom types.
