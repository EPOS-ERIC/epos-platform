// Package db provides PostgreSQL connections, migrations, and persistence adapters.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	sharedpostgres "github.com/EPOS-ERIC/epos-platform/packages/go/postgres"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
	accessdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/access"
	authdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/authz"
	categorydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/categories"
	conceptdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/concepts"
	contactpointdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/contactpoints"
	contactrequestdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/contactrequests"
	datasetdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dataset"
	discoverydb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/discovery"
	groupdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/groups"
	importdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/metadataimport"
	notificationdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/notifications"
	organizationdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/organizations"
	softwareapplicationsdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/softwareapplications"
	softwaresourcecodesdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/softwaresourcecodes"
	usersdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxgeom "github.com/twpayne/pgx-geom"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type PostgresDB struct {
	db                   *pgxpool.Pool
	Authorization        *authdb.Repo
	Users                *usersdb.Repo
	Groups               *groupdb.Repo
	Concepts             *conceptdb.Repo
	Access               *accessdb.Repo
	Categories           *categorydb.Repo
	Organizations        *organizationdb.Repo
	ContactPoints        *contactpointdb.Repo
	ContactRequests      *contactrequestdb.Repo
	Discovery            *discoverydb.Repo
	Datasets             *datasetdb.Repo
	Imports              *importdb.Repo
	SoftwareApplications *softwareapplicationsdb.Repo
	SoftwareSourceCodes  *softwaresourcecodesdb.Repo
	Notifications        *notificationdb.Repo
}

// Open connects to PostgreSQL, applies migrations, and builds repositories.
func Open(ctx context.Context, log *slog.Logger, cfg config.Config) (*PostgresDB, error) {
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create migrations filesystem: %w", err)
	}

	db, err := sharedpostgres.Open(ctx, log, sharedpostgres.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		Database: cfg.DBName,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		AfterConnect: func(ctx context.Context, conn *pgx.Conn) error {
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
		},
	}, migrationsFS)
	if err != nil {
		return nil, err
	}

	// TODO: align the initialization of the db packages
	return &PostgresDB{
		db:                   db,
		Authorization:        authdb.New(db),
		Users:                usersdb.New(db),
		Groups:               groupdb.New(db),
		Concepts:             conceptdb.New(db),
		Access:               accessdb.New(db),
		Categories:           categorydb.New(db),
		Organizations:        organizationdb.New(db),
		ContactPoints:        contactpointdb.New(db),
		ContactRequests:      contactrequestdb.New(db),
		Discovery:            discoverydb.New(log, db),
		Datasets:             datasetdb.New(db),
		Imports:              importdb.New(db),
		SoftwareApplications: softwareapplicationsdb.New(db),
		SoftwareSourceCodes:  softwaresourcecodesdb.New(db),
		Notifications:        notificationdb.New(db),
	}, nil
}

// Close closes the PostgreSQL connection pool.
func (d *PostgresDB) Close() error {
	if d.db == nil {
		return nil
	}
	d.db.Close()
	return nil
}
