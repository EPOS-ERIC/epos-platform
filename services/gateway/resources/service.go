// Package resources contains resource detail domain use-cases.
package resources

import (
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

type Service struct {
	log            *slog.Logger
	metadataClient *metadataclient.APIClient
}

func NewService(metadataClient *metadataclient.APIClient) *Service {
	return &Service{
		log:            logging.Get("resources-service"),
		metadataClient: metadataClient,
	}
}
