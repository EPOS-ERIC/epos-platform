// Package main wires and starts the gateway service binary.
package main

import (
	"flag"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server"
)

func main() {
	log := logging.New("gateway").With("service", "gateway")

	log.Info("gateway service started")

	cfgPath := flag.String("config", "/etc/gateway/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Error("error loading config", "err", err)
		panic("error loading config")
	}
	dataRetrieverClient, err := clients.NewDataRetriever(log, cfg.Services.DataRetriever.URL)
	if err != nil {
		log.Error("invalid data-retriever service configuration", "error", err)
		panic("invalid data-retriever service configuration")
	}

	log.Debug("config loaded", "listen", cfg.Server.Listen)

	metadataLifecycleClient, err := clients.NewMetadataLifecycle(
		log,
		cfg.Services.MetadataLifecycle.URL,
	)
	if err != nil {
		log.Error("invalid metadata-lifecycle service configuration", "error", err)
		panic("invalid metadata-lifecycle service configuration")
	}

	sharingClient, err := clients.NewSharing(log, cfg.Services.Sharing.URL)
	if err != nil {
		log.Error("invalid sharing service configuration", "error", err)
		panic("invalid sharing service configuration")
	}

	authService, err := auth.NewService(log, cfg.Services.Auth.URL, cfg.Upstream)
	if err != nil {
		log.Error("invalid auth service configuration", "error", err)
		panic("invalid auth service configuration")
	}

	if err := server.Run(*cfg, log, metadataLifecycleClient, dataRetrieverClient, sharingClient, authService); err != nil {
		log.Error("server failed", "error", err)
		panic("server failed")
	}
}
