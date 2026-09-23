---
name: sqlc-go
description: Use for sqlc in this Go monorepo — .sql query files, sqlc.yaml, sqlc generate, generated Go code, query annotations (:one/:many/:exec), pointer nulls, sqlc.embed/arg/narg/slice. Trigger on any query file, queries.go/models.go, or new DB query mention.
---

# sqlc in this Go monorepo

sqlc generates type-safe Go from plain `.sql` files — not ORM. `.sql` is source of truth (field names, types, nullability, signatures). Generated code is build artifact like protobuf output: never hand-edit.

## Repo convention

Each service has own `sqlc.yaml` at root. **Always read it first** — never assume schema/query paths, package name, or `sql_package` carry over. Most target `sql_package: "pgx/v5"`; verify per service.

## Workflow

1. Edit schema DDL if change touches table structure.
2. Edit/add `.sql` queries with right annotation (below).
3. Run `make generate` from directory containing service's `sqlc.yaml`.
4. Never hand-edit generated output — fix `.sql`, regenerate.
5. Run `make lint` to catch mismatches.

## Query annotations

Top agent mistake: wrong annotation fails compile or silently returns wrong shape.

| Annotation                            | Returns                      | Use when                                                               |
| ------------------------------------- | ---------------------------- | ---------------------------------------------------------------------- |
| `:one`                                | `(Row, error)`               | One row expected. None → `pgx.ErrNoRows`; caller handles it.           |
| `:many`                               | `([]Row, error)`             | Zero+ rows. Empty → empty slice, no error.                             |
| `:exec`                               | `error`                      | No rows; affected count unneeded.                                      |
| `:execrows`                           | `(int64, error)`             | `:exec` + affected-row count, e.g. conditional UPDATE/DELETE.          |
| `:execresult`                         | `(pgconn.CommandTag, error)` | Rare; full result metadata beyond count.                               |
| `:batchexec`/`:batchmany`/`:batchone` | Batch (pgx only)             | Many param sets, one `pgx.Batch` round trip.                           |
| `:copyfrom`                           | Bulk insert (pgx only)       | Postgres `COPY`; no `ON CONFLICT`/`RETURNING`.                         |
| `:execlastid`                         | n/a here                     | `database/sql` only; invalid with pgx/v5. Use `RETURNING id` + `:one`. |

Details/examples: `references/query-annotations.md`.

## Nullable & custom types (pgx/v5)

This repo sets `emit_pointers_for_null_types: true`. Nullable standard scalar columns use pointers (`*string`, `*int32`, `*time.Time`, etc.) — **not** `pgtype.*` or `sql.Null*`. Use `nil` for SQL NULL; nil-check before dereferencing.

Custom nullable overrides must set `pointer: true`, as with `uuid` and `timestamptz` in this repo's `sqlc.yaml`.

Generated enum `NullX` wrappers may remain where sqlc cannot emit pointers. Never hand-edit generated output.

Nullability is schema-driven: Go column nullable iff DDL nullable or query makes it nullable, e.g. `LEFT JOIN`, unprovable aggregate/expression.

Type table, `sqlc.arg`/`sqlc.narg`, `overrides:`: `references/types-and-nullability.md`.

## PostgreSQL derived and composite types (pgx/v5)

sqlc generates Go types and query signatures from the schema, but it does not register PostgreSQL runtime codecs in pgx. Custom derived types must be loaded and registered on every connection, normally in the pool's `AfterConnect` callback:

```go
types, err := conn.LoadTypes(ctx, []string{"my_enum", "_my_enum"})
if err != nil {
	return fmt.Errorf("load PostgreSQL types: %w", err)
}
conn.TypeMap().RegisterTypes(types)
```

For a custom enum or composite array, register both the element type and its PostgreSQL array type (usually the `_type_name` name). PostgreSQL OIDs are instance-specific; never hard-code them. Without registration, pgx cannot encode or decode custom arrays even when sqlc generated the correct Go slice type.

For custom PostgreSQL composite types, define matching Go structs manually, map them with `sqlc.yaml` `overrides` (by `db_type` or column), and load/register the composite type and any array type in pgx. Do not hand-edit generated `db/sqlc` code. See `db/types/types.go` and the composite-column overrides in `sqlc.yaml` for the repository pattern.

## Advanced features

- **`sqlc.embed(table)`** — return a table as a nested generated struct (`row.Post`, `row.User`). Use it when a query combines a full table with other selected tables or values, especially joins. For a single-table query that returns the full table, select its columns explicitly so sqlc returns the table struct directly instead of wrapping it in a nested field. Use explicit columns for narrower projections or when avoiding coupling to future table columns.
- **`sqlc.arg(name)` / `sqlc.narg(name)`** — named params; `narg` makes Go param nullable for optional filters.
- **`sqlc.slice(name)`** — Go slice param for `IN (...)` queries (pgx/v5 only), replacing manual `= ANY($1::type[])`.

Examples and config reference: `references/advanced-features.md`.

## Before handing back sqlc work

- [ ] Checked service's actual `sqlc.yaml`; no assumptions
- [ ] Annotation matches query result shape
- [ ] No hand-edits to generated output
- [ ] Ran `make generate` from right service directory
- [ ] Used pointers for nullable fields; nil-checked before dereferencing
- [ ] `make build` passes after regenerating
