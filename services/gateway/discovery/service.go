// Package discovery contains discovery domain use-cases.
package discovery

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

var log = logging.Get("discovery-service")

type Service struct {
	metadataClient *metadataclient.APIClient
}

func NewService(metadataClient *metadataclient.APIClient) *Service {
	return &Service{metadataClient: metadataClient}
}

func (s *Service) GetConceptTree() error {
	return domain.ErrNotImplemented
}
