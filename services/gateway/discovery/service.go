// Package discovery contains discovery domain use-cases.
package discovery

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/integrations/metadatalifecycle"
)

var log = logging.Get("discovery-service")

type Service struct {
	metadataClient *metadatalifecycle.Client
}

func NewService(metadataClient *metadatalifecycle.Client) *Service {
	return &Service{metadataClient: metadataClient}
}

func (s *Service) GetConceptTree() error {
	return domain.ErrNotImplemented
}
