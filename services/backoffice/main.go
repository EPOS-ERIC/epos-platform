package main

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/config"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/server"
)

func main() {
	log := logging.Get("backoffice")
	defer func() {
		if err := logging.Close(); err != nil {
			log.Error("failed to close logging", "error", err)
		}
	}()

	log.Info("backoffice service started")

	cfg, err := config.Load()
	if err != nil {
		log.Error("error loading config", "err", err)
		panic(err)
	}

	log.Debug("config loaded", "listen", cfg.ServerPort)

	if err := server.Run(cfg); err != nil {
		log.Error("server failed", "error", err)
		panic("server failed")
	}
}
