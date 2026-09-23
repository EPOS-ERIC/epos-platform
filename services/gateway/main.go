// Package main wires and starts the gateway service binary.
package main

import (
	"flag"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/notifications"
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

	log.Debug("config loaded", "listen", cfg.Server.Listen)

	var dispatcher clients.NotificationDispatcher
	if cfg.Notifications.Mailgun.Enable {
		mailer, mailerErr := notifications.NewMailgun(log, cfg.Notifications.Mailgun)
		if mailerErr != nil {
			log.Error("invalid Mailgun notification configuration", "error", mailerErr)
			panic("invalid Mailgun notification configuration")
		}
		dispatcher = mailer
	}

	metadataLifecycleClient, err := clients.NewMetadataLifecycle(
		log,
		cfg.Services.MetadataLifecycle.URL,
		dispatcher,
	)
	if err != nil {
		log.Error("invalid metadata-lifecycle service configuration", "error", err)
		panic("invalid metadata-lifecycle service configuration")
	}

	authService, err := auth.NewService(log, cfg.Services.Auth.URL)
	if err != nil {
		log.Error("invalid auth service configuration", "error", err)
		panic("invalid auth service configuration")
	}

	if err := server.Run(*cfg, log, metadataLifecycleClient, authService); err != nil {
		log.Error("server failed", "error", err)
		panic("server failed")
	}
}
