// Package db provides PostgreSQL connections, migrations, and persistence adapters.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
	authdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/authz"
	categorydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/categories"
	contactpointdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/contactpoints"
	datasetdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dataset"
	discoverydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/discovery"
	groupdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/groups"
	identitydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/identity"
	importdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/metadataimport"
	organizationdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/organizations"
	softwaredb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/software"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	pgxgeom "github.com/twpayne/pgx-geom"
)

var log = logging.Get("db")

//go:embed migrations/*.sql
var embedMigrations embed.FS

type PostgresDB struct {
	db                   *pgxpool.Pool
	Authorization        *authdb.Repo
	Identity             *identitydb.Repo
	Groups               *groupdb.Repo
	Categories           *categorydb.Repo
	Organizations        *organizationdb.Repo
	ContactPoints        *contactpointdb.Repo
	Discovery            *discoverydb.Repo
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

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres pool config: %w", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// initialize the pgxgeom library for using wkt in go
		if err := pgxgeom.Register(ctx, conn); err != nil {
			return fmt.Errorf("register PostGIS codec: %w", err)
		}
		return nil
	}
	db, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
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
		Categories:           categorydb.New(db),
		Organizations:        organizationdb.New(db),
		ContactPoints:        contactpointdb.New(db),
		Discovery:            discoverydb.New(db),
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
