package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/routes"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/services/metadata"
)

func RunServer(cfg config.Config) error {
	log := logging.Get("server")

	requestTimeout := cfg.Server.RequestTimeout
	if requestTimeout <= 0 {
		requestTimeout = 30 * time.Second
	}

	metadataService, err := metadata.NewService(cfg.Services.MetadataLifecycle.URL, requestTimeout)
	if err != nil {
		log.Error("invalid metadata service configuration", "error", err)
		return fmt.Errorf("build metadata service: %w", err)
	}

	httpServer := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      routes.NewHandler(metadataService),
		ReadTimeout:  requestTimeout,
		WriteTimeout: requestTimeout,
		IdleTimeout:  requestTimeout,
	}

	listenErr := make(chan error, 1)
	go func() {
		log.Info("starting server", "listen", cfg.Server.Listen)
		listenErr <- httpServer.ListenAndServe()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case err := <-listenErr:
		if err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "error", err)
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	case sig := <-sigChan:
		log.Info("received signal", "signal", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := <-listenErr; err != nil && err != http.ErrServerClosed {
		log.Error("server stopped with error", "error", err)
		return fmt.Errorf("server stopped with error: %w", err)
	}

	log.Info("server shutdown complete")

	return nil
}
