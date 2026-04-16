package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/routes"
)

func RunServer(cfg config.Config) {
	log := logging.Get("server")

	requestTimeout := cfg.Server.RequestTimeout
	if requestTimeout <= 0 {
		requestTimeout = 30 * time.Second
	}

	httpServer := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      routes.NewRouter(),
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
			panic("server failed")
		}
		return
	case sig := <-sigChan:
		log.Info("received signal", "signal", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		panic("server shutdown failed")
	}

	if err := <-listenErr; err != nil && err != http.ErrServerClosed {
		log.Error("server stopped with error", "error", err)
		panic("server stopped with error")
	}

	log.Info("server shutdown complete")
}
