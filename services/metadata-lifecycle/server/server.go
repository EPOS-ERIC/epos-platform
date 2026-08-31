// Package server provides HTTP server runtime and dependency wiring.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi"
	accessapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/access"
	categoriesapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/categories"
	contactpointsapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/contactpoints"
	datasetapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/dataset"
	discoveryapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/discovery"
	groupsapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/groups"
	healthapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/health"
	importsapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/import"
	organizationsapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/organizations"
	resourceapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/resource"
	softwareapplicationapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/softwareapplication"
	softwaresourcecodeapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/softwaresourcecode"
	usersapi "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/users"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadataimport"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/turtle"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/users"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/access"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/categories"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/datasets"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwareapplications"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwaresourcecodes"
)

// Run starts the metadata-lifecycle HTTP server until shutdown.
func Run(cfg config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := observability.Start(ctx, observability.Config{
		MetricsExportInterval: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("setup observability: %w", err)
	}
	defer func() {
		if shutdownErr := otelShutdown(context.Background()); shutdownErr != nil {
			log.Error("otel shutdown failed", "error", shutdownErr)
		}
	}()

	database, err := db.Open(log, cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			log.Error("database close failed", "error", closeErr)
		}
	}()

	turtler := turtle.New(log)
	authorizer := authz.New(log, database.Authorization)
	usersSvc := users.New(log, database.Users, authorizer)
	importSvc := metadataimport.New(log, database.Imports, authorizer)
	groupsSvc := groups.New(log, database.Groups, authorizer)
	accessSvc := access.New(log, database.Access, authorizer)
	categoriesSvc := categories.New(log, database.Categories, authorizer)
	contactPointSvc := contactpoints.New(log, database.ContactPoints, authorizer)
	organizationsSvc := organizations.New(log, database.Organizations, authorizer)
	datasetSvc := datasets.New(log, database.Datasets, authorizer)
	applicationSvc := softwareapplications.New(log, database.SoftwareApplications, authorizer)
	sourceCodeSvc := softwaresourcecodes.New(log, database.SoftwareSourceCodes, authorizer)
	discoverySvc := discovery.New(log, database.Discovery, authorizer, datasetSvc, applicationSvc, sourceCodeSvc, groupsSvc)

	datasetsHandler := datasetapi.New(log, datasetSvc)
	applicationsHandler := softwareapplicationapi.New(log, applicationSvc)
	sourceCodesHandler := softwaresourcecodeapi.New(log, sourceCodeSvc)

	router := httpapi.NewRouter(httpapi.Router{
		Health:        healthapi.New(log, database),
		Imports:       importsapi.New(log, importSvc, turtler),
		Users:         usersapi.New(log, usersSvc),
		Groups:        groupsapi.New(log, groupsSvc),
		Access:        accessapi.New(log, accessSvc),
		Categories:    categoriesapi.New(log, categoriesSvc),
		ContactPoints: contactpointsapi.New(log, contactPointSvc),
		Organizations: organizationsapi.New(log, organizationsSvc),
		Resources:     resourceapi.New(log, datasetsHandler, applicationsHandler, sourceCodesHandler),
		Discovery:     discoveryapi.New(log, discoverySvc),

		Log: log,
	})

	srv := &http.Server{ //nolint:exhaustruct_v5
		BaseContext:  func(net.Listener) context.Context { return ctx },
		Addr:         ":" + cfg.ServerPort,
		Handler:      http.TimeoutHandler(router, cfg.RequestTimeout, "request timeout\n"),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Info("starting server", "listen", cfg.ServerPort)
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-srvErr:
		log.Error("server failed", "error", err)
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		log.Info("server terminated: stopping")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		return fmt.Errorf("shutdown server: %w", err)
	}

	log.Info("server shutdown complete")

	return nil
}
