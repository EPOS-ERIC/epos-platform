// Package discovery contains discovery domain use-cases.
package discovery

import (
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

type Service struct {
	metadataClient *metadataclient.APIClient
	fanOutLimit    int
	log            *slog.Logger
}

func NewService(metadataClient *metadataclient.APIClient, fanOutLimit int) *Service {
	return &Service{
		metadataClient: metadataClient,
		fanOutLimit:    fanOutLimit,
		log:            logging.Get("discovery-service"),
	}
}
