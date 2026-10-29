package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/config"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/db"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/server"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("sharing version %s\n", version)
		return
	}
	log := logging.New("sharing").With("service", "sharing")

	cfg, err := config.Load()
	if err != nil {
		log.Error("error loading config", "err", err)
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otelShutdown, err := observability.Start(ctx, observability.Config{})
	if err != nil {
		log.Error("failed to setup observability", "error", err)
		panic(err)
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			log.Error("otel shutdown failed", "error", err)
		}
	}()

	log.Info("sharing service started")

	database, err := db.Open(ctx, log, cfg)
	if err != nil {
		panic(err)
	}
	defer database.Close()

	if err := server.Run(ctx, cfg, log, database); err != nil {
		log.Error("grpc server failed", "error", err)
		panic(err)
	}
}
