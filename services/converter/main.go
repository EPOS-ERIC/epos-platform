package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/packages/go/grpcserver"
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
	"github.com/EPOS-ERIC/epos-platform/services/converter/catalogue"
	"github.com/EPOS-ERIC/epos-platform/services/converter/config"
	"github.com/EPOS-ERIC/epos-platform/services/converter/db"
	dbcatalogue "github.com/EPOS-ERIC/epos-platform/services/converter/db/catalogue"
	"github.com/EPOS-ERIC/epos-platform/services/converter/grpcapi"
	"github.com/EPOS-ERIC/epos-platform/services/converter/plugin"
	"google.golang.org/grpc"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("converter version %s\n", version)
		return
	}
	if err := run(); err != nil {
		logging.New("converter").Error("converter stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	log := logging.New("converter").With("service", "converter")
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdown, err := observability.Start(ctx, observability.Config{})
	if err != nil {
		return fmt.Errorf("start observability: %w", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			log.Error("observability shutdown failed", "error", err)
		}
	}()
	database, err := db.Open(ctx, log, cfg)
	if err != nil {
		return err
	}
	defer database.Close()
	repository := dbcatalogue.New(database.Pool)
	manager, err := plugin.New(cfg.PluginsPath, cfg.MaxBytes, cfg.Timeout, repository)
	if err != nil {
		return err
	}
	service := catalogue.New(repository, manager, log)
	syncCtx, cancelSync := context.WithCancel(ctx)
	syncDone := make(chan struct{})
	go func() {
		defer close(syncDone)
		service.RunSync(syncCtx, cfg.SyncInterval)
	}()
	defer func() {
		cancelSync()
		<-syncDone
	}()
	return grpcserver.Run(ctx, grpcserver.Config{
		Address: ":" + cfg.ServerPort,
		Logger:  log,
		ValidationError: func(_ *grpc.UnaryServerInfo, err error) error {
			return apperror.Field("request", "Invalid request.", err)
		},
		Register: func(server *grpc.Server) {
			converterv1.RegisterConverterServiceServer(server, grpcapi.New(service, manager, cfg.MaxBytes, log))
		},
	})
}
