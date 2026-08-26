// Package dbtx contains shared PostgreSQL transaction helpers.
package dbtx

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run executes fn in one PostgreSQL transaction.
func Run(ctx context.Context, db *pgxpool.Pool, queries *sqlc.Queries, fn func(*sqlc.Queries) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(queries.WithTx(tx)); err != nil {
		return fmt.Errorf("execute transaction: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
