package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/config"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type PostgresDB struct {
	db *sql.DB
	*sqlc.Queries
}

func Open(cfg config.Config) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBUser,
		cfg.DBPassword,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres db: %w", err)
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &PostgresDB{db: db, Queries: sqlc.New(db)}, nil
}

func runMigrations(db *sql.DB) error {
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("create migrations filesystem: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	ctx := context.Background()
	hasPending, err := provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("check pending migrations: %w", err)
	}

	if !hasPending {
		return nil
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func (d *PostgresDB) Close() error {
	if d.db == nil {
		return nil
	}
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}

func (d *PostgresDB) Health() error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := d.db.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (d *PostgresDB) PingContext(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := d.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
