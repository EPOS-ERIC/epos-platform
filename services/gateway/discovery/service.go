// Package discovery contains discovery domain use-cases.
package discovery

import (
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/integrations/metadatalifecycle"
)

type Service struct {
	metadataClient *metadatalifecycle.Client
}

func NewService(metadataClient *metadatalifecycle.Client) *Service {
	return &Service{metadataClient: metadataClient}
}

func (s *Service) Search() error {
	return domain.ErrNotImplemented
}

func (s *Service) GetCategoryTree() error {
	return domain.ErrNotImplemented
}
