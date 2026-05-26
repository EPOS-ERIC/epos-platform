// Package resources contains resource detail domain use-cases.
package resources

import (
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	log *slog.Logger
}

func NewService() *Service {
	return &Service{
		log: logging.Get("resources-service"),
	}
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

func (s *Service) GetWebserviceByID() error {
	return domain.ErrNotImplemented
}

func (s *Service) GetOrganizationByID() error {
	return domain.ErrNotImplemented
}
