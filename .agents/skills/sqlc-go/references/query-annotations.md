# Query annotations in depth

Name comment directly above query:

```sql
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
```

Name becomes Go function name verbatim.

## :one

```go
func (q *Queries) GetUserByID(ctx context.Context, id int64) (User, error)
```

No match returns `pgx.ErrNoRows` (`"github.com/jackc/pgx/v5"`) — check with `errors.Is(err, pgx.ErrNoRows)`, not generic error handling. Also correct for `LEFT JOIN`/aggregate queries always returning one row, e.g. `COUNT(*)`.

## :many

```go
func (q *Queries) ListUsersByStatus(ctx context.Context, status string) ([]User, error)
```

Empty result = empty slice + nil error; never `ErrNoRows`.

## :exec / :execrows

```go
func (q *Queries) DeleteUser(ctx context.Context, id int64) error                    // :exec
func (q *Queries) DeactivateStaleUsers(ctx context.Context, cutoff time.Time) (int64, error) // :execrows
```

Use `:execrows` whenever the caller needs to know if anything actually happened (e.g. return 404 vs 200) — `:exec` silently ignores a no-op UPDATE/DELETE.

## :execresult

pgx-specific; returns raw `pgconn.CommandTag`. Rare: `:execrows` covers nearly every case.

## :batchexec / :batchmany / :batchone

pgx/v5 only. Queue many param sets; send in one round trip:

```sql
-- name: UpdateUserStatus :batchexec
UPDATE users SET status = $2 WHERE id = $1;
```

```go
batch := q.UpdateUserStatus(ctx, []UpdateUserStatusParams{...})
batch.Exec(func(i int, err error) { ... })
batch.Close()
```

Use when same query runs many times per request and latency matters; not one-off calls.

## :copyfrom

pgx/v5 only, backed by Postgres `COPY`. Bulk inserts; faster than N `INSERT`s:

```sql
-- name: BulkInsertEvents :copyfrom
INSERT INTO events (user_id, kind, payload) VALUES ($1, $2, $3);
```

Takes `[]BulkInsertEventsParams`. No `ON CONFLICT`/`RETURNING`: Postgres `COPY` supports neither. Use batched `INSERT ... ON CONFLICT` instead.

## :execlastid

`database/sql` only; invalid with pgx/v5. Postgres lacks MySQL-style autoincrement `LastInsertId`. Replace with:

```sql
-- name: CreateUser :one
INSERT INTO users (email, name) VALUES ($1, $2) RETURNING *;
```
