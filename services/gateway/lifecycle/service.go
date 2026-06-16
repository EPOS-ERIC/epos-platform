// Package lifecycle contains resource detail domain use-cases.
package lifecycle

import (
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

type Service struct {
	log            *slog.Logger
	metadataClient *metadataclient.APIClient
}

// NewService builds lifecycle service for draft operations.
func NewService(metadataClient *metadataclient.APIClient) *Service {
	return &Service{
		log:            logging.Get("lifecycle-service"),
		metadataClient: metadataClient,
	}
}
