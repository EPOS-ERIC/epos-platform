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
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/contactrequest"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/membership"
	sharingservice "github.com/EPOS-ERIC/epos-platform/services/gateway/sharing"
)

func Run(cfg config.Config, log *slog.Logger, metadataLifecycleClient *clients.MetadataLifecycle, semanticAggregatorClient *clients.SemanticAggregator, dataRetrieverClient *clients.DataRetriever, converterClient *clients.Converter, sharingClient *clients.Sharing, authService *auth.Service, contactRequestSender contactrequest.EmailSender) error {
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

	identitySvc := identity.NewService(baseLog, metadataLifecycleClient)
	membershipSvc := membership.NewService(baseLog, metadataLifecycleClient)
	sharingSvc := sharingservice.NewService(sharingClient)
	router := NewRouter(baseLog, identitySvc, membershipSvc, metadataLifecycleClient, semanticAggregatorClient, contactRequestSender, dataRetrieverClient, converterClient, sharingSvc, authService, cfg.Server.CORSAllowOrigin, cfg.Server.RequestTimeout)

	// WriteTimeout stays disabled because distribution downloads stream large responses.
	srv := &http.Server{
		BaseContext:  func(net.Listener) context.Context { return ctx },
		Addr:         cfg.Server.Listen,
		Handler:      router,
		ReadTimeout:  cfg.Server.RequestTimeout,
		WriteTimeout: 0,
		IdleTimeout:  cfg.Server.RequestTimeout,
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
