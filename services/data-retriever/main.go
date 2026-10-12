package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/config"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/db"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/server"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/upstream"
)

func main() {
	log := logging.New("data-retriever").With("service", "data-retriever")
	if err := run(log); err != nil {
		log.Error("data-retriever stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otelShutdown, err := observability.Start(ctx, observability.Config{})
	if err != nil {
		return fmt.Errorf("setup observability: %w", err)
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			log.Error("otel shutdown failed", "error", err)
		}
	}()

	database, err := db.Open(ctx, log, cfg)
	if err != nil {
		return fmt.Errorf("open data-retriever database: %w", err)
	}
	defer database.Close()

	log.Info("starting data-retriever gRPC server", "addr", cfg.GRPCListenAddr, "header_timeout", cfg.HTTPTimeout.String())
	if err := server.Run(ctx, cfg, log, database.Retrievals, upstream.NewClient(cfg.HTTPTimeout), uint64(cfg.SchemaMaxBytes)); err != nil {
		return fmt.Errorf("run data-retriever gRPC server: %w", err)
	}
	return nil
}
