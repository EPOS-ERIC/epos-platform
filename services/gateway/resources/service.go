// Package resources contains resource detail domain use-cases.
package resources

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

var log = logging.Get("resources-service")

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetConceptByID() error {
	return domain.ErrNotImplemented
}

func (s *Service) GetDistributionByID() error {
	return domain.ErrNotImplemented
}

func (s *Service) GetSoftwareApplicationByID() error {
	return domain.ErrNotImplemented
}

func (s *Service) GetSoftwareSourceCodeByID() error {
	return domain.ErrNotImplemented
}
