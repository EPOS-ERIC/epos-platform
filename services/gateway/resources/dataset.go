package resources

import (
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func (s *Service) GetDatasetByID(id string) (api.Dataset, error) {
	return api.Dataset{}, domain.ErrNotImplemented
}
