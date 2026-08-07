package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) GetWebserviceByID(ctx context.Context, id string) (api.Webservice, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.Webservice{}, err
	}
	webservice, err := s.resources.GetWebService(ctx, parsedID)
	if err != nil {
		return api.Webservice{}, fmt.Errorf("fetch resources webservice '%s': %w", id, err)
	}

	return mapWebservice(webservice), nil
}
