// Package db opens the converter PostgreSQL database and applies migrations.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/postgres"
	"github.com/EPOS-ERIC/epos-platform/services/converter/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate go tool sqlc generate -f ../sqlc.yaml

//go:embed migrations/*.sql
var migrations embed.FS

type Database struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, log *slog.Logger, cfg config.Config) (*Database, error) {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("open migrations: %w", err)
	}
	pool, err := postgres.Open(ctx, log, postgres.Config{
		Host: cfg.DBHost, Port: cfg.DBPort, Database: cfg.DBName,
		User: cfg.DBUser, Password: cfg.DBPassword,
	}, files)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &Database{Pool: pool}, nil
}

func (d *Database) Close() { d.Pool.Close() }
