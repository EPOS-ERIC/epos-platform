package main

import (
	"flag"
	"os"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
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
		os.Exit(1)
	}

	log.Debug("config loaded", "listen", cfg.Server.Listen, "base_path", cfg.Server.BasePath)

	server.RunServer(*cfg)
}
