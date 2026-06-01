// Package server provides gateway HTTP transport wiring and server runtime.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
)

func Run(cfg config.Config, metadataClient *metadataclient.APIClient) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log := logging.Get("server")

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

	discoverySvc := discovery.NewService(metadataClient, cfg.Discovery.FanOutLimit)
	resourcesSvc := resources.NewService(metadataClient)
	router := NewRouter(discoverySvc, resourcesSvc)
	handler := requestTimeoutHandler(router, cfg.Server.RequestTimeout)

	srv := &http.Server{
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

func requestTimeoutHandler(next http.Handler, timeout time.Duration) http.Handler {
	return http.TimeoutHandler(next, timeout, "request timeout\n")
}
