// Package resources contains resource detail domain use-cases.
package resources

import (
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
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

func (s *Service) GetConceptByID(id string) (api.Concept, error) {
	return api.Concept{}, domain.ErrNotImplemented
}

func (s *Service) GetSoftwareApplicationByID(id string) (api.SoftwareApplication, error) {
	return api.SoftwareApplication{}, domain.ErrNotImplemented
}

func (s *Service) GetSoftwareSourceCodeByID(id string) (api.SoftwareSourceCode, error) {
	return api.SoftwareSourceCode{}, domain.ErrNotImplemented
}

func (s *Service) GetWebserviceByID(id string) (api.Webservice, error) {
	return api.Webservice{}, domain.ErrNotImplemented
}

func (s *Service) GetOrganizationByID(id string) (api.Organization, error) {
	return api.Organization{}, domain.ErrNotImplemented
}
