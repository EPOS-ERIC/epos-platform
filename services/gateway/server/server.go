// Package server provides gateway HTTP transport wiring and server runtime.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/canonical"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	groupservice "github.com/EPOS-ERIC/epos-platform/services/gateway/groups"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	metadataimports "github.com/EPOS-ERIC/epos-platform/services/gateway/imports"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
)

func Run(cfg config.Config, log *slog.Logger, metadataLifecycleClient *clients.MetadataLifecycle, authService *auth.Service) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	baseLog := log
	log = log.With("component", "server")

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

	lifecycleSvc := lifecycle.NewService(baseLog, metadataLifecycleClient)
	identitySvc := identity.NewService(metadataLifecycleClient)
	canonicalSvc := canonical.NewService(metadataLifecycleClient)
	discoverySvc := discovery.NewService(metadataLifecycleClient)
	groupsSvc := groupservice.NewService(metadataLifecycleClient)
	importsSvc := metadataimports.NewService(metadataLifecycleClient)
	router := NewRouter(baseLog, canonicalSvc, discoverySvc, groupsSvc, lifecycleSvc, identitySvc, importsSvc, metadataLifecycleClient, authService, cfg.Server.CORSAllowOrigin)
	handler := handlers.NewTimeoutHandler(baseLog, router, cfg.Server.RequestTimeout)

	srv := &http.Server{ //nolint:exhaustruct_v5
		BaseContext:  func(net.Listener) context.Context { return ctx },
		Addr:         cfg.Server.Listen,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Info("starting server", "listen", cfg.Server.Listen)
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
