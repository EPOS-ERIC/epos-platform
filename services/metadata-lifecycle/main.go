package main

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
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

	log.Debug("config loaded", "listen", cfg.ServerPort)

	if err := server.Run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		panic(err)
	}
}
