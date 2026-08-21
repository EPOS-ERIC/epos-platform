// Package db provides PostgreSQL connections, migrations, and persistence adapters.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/config"
	authdb "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/authz"
	datasetdb "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/dataset"
	groupdb "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/groups"
	identitydb "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/identity"
	importdb "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/metadataimport"
	softwaredb "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/software"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var log = logging.Get("db")

//go:embed migrations/*.sql
var embedMigrations embed.FS

type PostgresDB struct {
	db                   *pgxpool.Pool
	Authorization        *authdb.Repo
	Identity             *identitydb.Repo
	Groups               *groupdb.Repo
	Datasets             *datasetdb.Repo
	Imports              *importdb.Repo
	SoftwareApplications *softwaredb.ApplicationRepo
	SoftwareSourceCodes  *softwaredb.SourceCodeRepo
}

// Open connects to PostgreSQL, applies migrations, and builds repositories.
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

	// TODO: align the initialization of the db packages
	return &PostgresDB{
		db:                   db,
		Authorization:        &authdb.Repo{Queries: sqlc.New(db)},
		Identity:             &identitydb.Repo{Queries: sqlc.New(db), DB: db},
		Groups:               groupdb.New(db),
		Datasets:             datasetdb.New(db),
		Imports:              importdb.New(db),
		SoftwareApplications: softwaredb.NewApplicationRepo(db),
		SoftwareSourceCodes:  softwaredb.NewSourceCodeRepo(db),
	}, nil
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
		log.Debug("database migrations are current")
		return nil
	}

	log.Info("applying database migrations")
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Info("database migrations applied")

	return nil
}

// Close closes the PostgreSQL connection pool.
func (d *PostgresDB) Close() error {
	if d.db == nil {
		return nil
	}
	d.db.Close()
	return nil
}

// PingContext checks PostgreSQL using ctx.
func (d *PostgresDB) PingContext(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := d.db.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
