// Package usergroup contains resource detail domain use-cases.
package usergroup

import (
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/usergroupclient"
)

type Service struct {
	log             *slog.Logger
	usergroupClient *usergroupclient.APIClient
}

func NewService(usergroupClient *usergroupclient.APIClient) *Service {
	return &Service{
		log:             logging.Get("usergroup-service"),
		usergroupClient: usergroupClient,
	}
}
