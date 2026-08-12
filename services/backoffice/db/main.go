// Package db provides PostgreSQL connections, migrations, and persistence adapters.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/config"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type PostgresDB struct {
	db       *pgxpool.Pool
	Identity *IdentityRepository
	Datasets *DatasetRepository
}

// IdentityRepository provides identity persistence through generated queries.
type IdentityRepository struct {
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

	migrationDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open migration db: %w", err)
	}
	if err := migrationDB.Ping(); err != nil {
		_ = migrationDB.Close()
		return nil, fmt.Errorf("ping migration db: %w", err)
	}
	if err := runMigrations(migrationDB); err != nil {
		_ = migrationDB.Close()
		return nil, err
	}
	if err := migrationDB.Close(); err != nil {
		return nil, fmt.Errorf("close migration db: %w", err)
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres pool: %w", err)
	}

	return &PostgresDB{db: db, Identity: &IdentityRepository{Queries: sqlc.New(db)}, Datasets: NewDatasetRepository(db)}, nil
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
	d.db.Close()
	return nil
}

func (d *PostgresDB) Health() error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := d.db.Ping(context.Background()); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (d *PostgresDB) PingContext(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := d.db.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
