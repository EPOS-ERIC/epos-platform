// Package main wires and starts the gateway service binary.
package main

import (
	"flag"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/http"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/integrations/metadatalifecycle"
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

	requestTimeout := cfg.Server.RequestTimeout
	metadataClient, err := metadatalifecycle.NewClient(cfg.Services.MetadataLifecycle.URL, requestTimeout)
	if err != nil {
		log.Error("invalid metadata service configuration", "error", err)
		panic("invalid metadata service configuration")
	}

	if err := http.Run(*cfg, metadataClient); err != nil {
		log.Error("server failed", "error", err)
		panic("server failed")
	}
}
