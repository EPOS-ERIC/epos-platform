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
	log := logging.Get("gateway")
	defer func() {
		if err := logging.Close(); err != nil {
			log.Error("failed to close logging", "error", err)
		}
	}()

	log.Info("gateway service started")

	cfgPath := flag.String("config", "/etc/gateway/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Error("error loading config", "err", err)
		panic("error loading config")
	}

	log.Debug("config loaded", "listen", cfg.Server.Listen)

	metadataLifecycleClient, err := clients.NewMetadataLifecycle(cfg.Services.MetadataLifecycle.URL, cfg.Upstream)
	if err != nil {
		log.Error("invalid metadata-lifecycle service configuration", "error", err)
		panic("invalid metadata-lifecycle service configuration")
	}

	authService, err := auth.NewService(cfg.Services.Auth.URL, cfg.Upstream)
	if err != nil {
		log.Error("invalid auth service configuration", "error", err)
		panic("invalid auth service configuration")
	}

	if err := server.Run(*cfg, metadataLifecycleClient, authService); err != nil {
		log.Error("server failed", "error", err)
		panic("server failed")
	}
}
