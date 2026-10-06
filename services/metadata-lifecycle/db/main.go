// Package db provides PostgreSQL connections, migrations, and persistence adapters.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
	accessdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/access"
	activitydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/activity"
	authdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/authz"
	categorydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/categories"
	contactpointdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/contactpoints"
	datasetdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dataset"
	discoverydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/discovery"
	groupdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/groups"
	importdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/metadataimport"
	notificationdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/notifications"
	organizationdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/organizations"
	softwareapplicationsdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/softwareapplications"
	softwaresourcecodesdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/softwaresourcecodes"
	usersdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/users"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	pgxgeom "github.com/twpayne/pgx-geom"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type PostgresDB struct {
	db                   *pgxpool.Pool
	Activity             *activitydb.Repo
	Authorization        *authdb.Repo
	Users                *usersdb.Repo
	Groups               *groupdb.Repo
	Access               *accessdb.Repo
	Categories           *categorydb.Repo
	Organizations        *organizationdb.Repo
	ContactPoints        *contactpointdb.Repo
	Discovery            *discoverydb.Repo
	Datasets             *datasetdb.Repo
	Imports              *importdb.Repo
	SoftwareApplications *softwareapplicationsdb.Repo
	SoftwareSourceCodes  *softwaresourcecodesdb.Repo
	Notifications        *notificationdb.Repo
}

// Open connects to PostgreSQL, applies migrations, and builds repositories.
func Open(log *slog.Logger, cfg config.Config) (*PostgresDB, error) {
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
	if err := runMigrations(log, migrationDB); err != nil {
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
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithSpanNameFunc(queryOperationName),
	)
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// initialize the pgxgeom library for using wkt in go
		if err := pgxgeom.Register(ctx, conn); err != nil {
			return fmt.Errorf("register PostGIS codec: %w", err)
		}
		customTypes, err := conn.LoadTypes(ctx, []string{
			"web_service_visible_on",
			"_web_service_visible_on",
			"string_parameter_enum_option",
			"_string_parameter_enum_option",
			"integer_parameter_enum_option",
			"_integer_parameter_enum_option",
			"float_parameter_enum_option",
			"_float_parameter_enum_option",
		})
		if err != nil {
			return fmt.Errorf("load PostgreSQL types: %w", err)
		}
		conn.TypeMap().RegisterTypes(customTypes)
		return nil
	}
	db, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := otelpgx.RecordStats(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("record postgres pool stats: %w", err)
	}
	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres pool: %w", err)
	}

	// TODO: align the initialization of the db packages
	return &PostgresDB{
		db:                   db,
		Activity:             activitydb.New(log, db),
		Authorization:        authdb.New(db),
		Users:                usersdb.New(db),
		Groups:               groupdb.New(db),
		Access:               accessdb.New(db),
		Categories:           categorydb.New(db),
		Organizations:        organizationdb.New(db),
		ContactPoints:        contactpointdb.New(db),
		Discovery:            discoverydb.New(log, db),
		Datasets:             datasetdb.New(db),
		Imports:              importdb.New(db),
		SoftwareApplications: softwareapplicationsdb.New(db),
		SoftwareSourceCodes:  softwaresourcecodesdb.New(db),
		Notifications:        notificationdb.New(db),
	}, nil
}

func runMigrations(log *slog.Logger, db *sql.DB) error {
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
