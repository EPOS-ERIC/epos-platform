// Package db provides PostgreSQL connections, migrations, and persistence adapters.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	sharedpostgres "github.com/EPOS-ERIC/epos-platform/packages/go/postgres"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/config"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/db/configurations"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate go tool sqlc generate -f ../sqlc.yaml

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

type PostgresDB struct {
	db             *pgxpool.Pool
	Configurations *configurations.Repo
}

// Open connects to PostgreSQL, applies migrations, and builds repositories.
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
		return nil, err
	}

	return &PostgresDB{
		db:             pool,
		Configurations: configurations.New(pool),
	}, nil
}

// Close closes the PostgreSQL connection pool.
func (d *PostgresDB) Close() {
	if d.db == nil {
		return
	}
	d.db.Close()
}
