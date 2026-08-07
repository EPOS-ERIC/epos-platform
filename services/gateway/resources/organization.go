package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) GetOrganizationByID(ctx context.Context, id string) (api.Organization, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.Organization{}, err
	}
	organization, err := s.resources.GetOrganization(ctx, parsedID)
	if err != nil {
		return api.Organization{}, fmt.Errorf("fetch resources organization '%s': %w", id, err)
	}

	return mapOrganization(organization), nil
}
