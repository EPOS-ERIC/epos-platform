package main

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/config"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/server"
)

func main() {
	log := logging.Get("user-group")
	defer func() {
		if err := logging.Close(); err != nil {
			log.Error("failed to close logging", "error", err)
		}
	}()

	log.Info("user-group service started")

	cfg, err := config.Load()
	if err != nil {
		log.Error("error loading config", "error", err)
		panic("error loading config")
	}

	log.Debug("config loaded", "listen", cfg.ServerPort)

	if err := server.Run(cfg); err != nil {
		log.Error("server failed", "error", err)
		panic("server failed")
	}
}
