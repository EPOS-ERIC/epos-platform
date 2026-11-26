// Package db manages Data Retriever PostgreSQL persistence.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	sharedpostgres "github.com/EPOS-ERIC/epos-platform/packages/go/postgres"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/config"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/db/retrievals"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate go tool sqlc generate -f ../sqlc.yaml

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

type PostgresDB struct {
	db         *pgxpool.Pool
	Retrievals *retrievals.Repo
}

func (d *PostgresDB) Ping(ctx context.Context) error {
	return d.db.Ping(ctx)
}

// Open connects to PostgreSQL, applies embedded migrations, and constructs repositories.
func Open(ctx context.Context, log *slog.Logger, cfg config.Config) (*PostgresDB, error) {
	migrations, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create migrations filesystem: %w", err)
	}

	pool, err := sharedpostgres.Open(ctx, log, sharedpostgres.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		Database: cfg.DBName,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
	}, migrations)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	return &PostgresDB{
		db:         pool,
		Retrievals: retrievals.New(pool),
	}, nil
}

// Close closes the PostgreSQL connection pool.
func (d *PostgresDB) Close() {
	if d.db != nil {
		d.db.Close()
	}
}
