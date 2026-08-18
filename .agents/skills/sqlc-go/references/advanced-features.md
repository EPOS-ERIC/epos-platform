# Advanced sqlc features

## sqlc.embed(table)

Nest full joined-table struct, not flat columns:

```sql
-- name: GetPostWithAuthor :one
SELECT sqlc.embed(posts), sqlc.embed(users)
FROM posts JOIN users ON users.id = posts.author_id
WHERE posts.id = $1;
```

```go
type GetPostWithAuthorRow struct {
    Post Post
    User User
}
```

Prefer over manual listing/aliasing: avoids name collisions, reads better. Table or alias must appear in `FROM`/`JOIN`, not arbitrary subquery.

## sqlc.slice(name)

pgx/v5 only. Real Go slice param for `IN (...)`:

```sql
-- name: GetUsersByIDs :many
SELECT * FROM users WHERE id IN (sqlc.slice(ids));
```

```go
func (q *Queries) GetUsersByIDs(ctx context.Context, ids []int64) ([]User, error)
```

Older queries may use `WHERE id = ANY($1::bigint[])` with `[]int64` — valid, but `sqlc.slice` is more direct for new queries.

## sqlc.arg / sqlc.narg

See `types-and-nullability.md`: `arg` for explicit non-null params; `narg` for nullable params (optional filters).

## sqlc.yaml config knobs

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/query"
    schema: "db/schema.sql"
    gen:
      go:
        package: "db"
        out: "db"
        sql_package: "pgx/v5"
        emit_pointers_for_null_types: true
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
```

- **`sql_package`** — `pgx/v5` vs `database/sql`; drives driver types and generated query code.
- **`emit_pointers_for_null_types`** — on = nullable fields use pointers; preferred in this repo. Off may generate `pgtype.*` or `sql.Null*` depending on `sql_package`.
- **`emit_interface`** — generates `Querier` interface for mocking; check test dependency.
- **`emit_exact_table_names`** — off = singularized structs (`users` → `User`); on = literal (`Users`).
- **`emit_empty_slices`** — on = `:many` returns `[]T{}`, not `nil`, for zero rows; affects `== nil` checks.
- **`overrides`** — see `types-and-nullability.md`.
- **`rename`** — per-column struct field renames.

Surprising generated code: check config before assuming bug. Most surprises are intended config behavior.
