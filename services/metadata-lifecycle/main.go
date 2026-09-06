package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/server"
)

func main() {
	log := logging.New("metadata-lifecycle").With("service", "metadata-lifecycle")

	log.Info("metadata-lifecycle service started")

	cfg, err := config.Load()
	if err != nil {
		log.Error("error loading config", "err", err)
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(log, cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			log.Error("database close failed", "error", closeErr)
		}
	}()

	if err := server.Run(ctx, cfg, log, database); err != nil {
		log.Error("grpc server failed", "error", err)
		panic(err)
	}
}
