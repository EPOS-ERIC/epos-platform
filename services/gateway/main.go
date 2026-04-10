package main

import (
	"os"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port    int           `env:"SERVER_PORT"`
	Timeout time.Duration `env:"RESPONSE_TIMEOUT"`
}

func main() {
	// TODO: improve logging
	log := logging.Get("orchestrator")
	defer func() {
		if err := logging.Close(); err != nil {
			log.Error("failed to close logging", "error", err)
		}
	}()

	log.Info("orchestrator service started")

	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		// TODO
		log.Error("error")
		os.Exit(1)
	}

	serve()
}
