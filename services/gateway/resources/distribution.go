package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) GetDistributionByID(ctx context.Context, id string) (api.Distribution, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.Distribution{}, err
	}
	distribution, err := s.resources.GetDistribution(ctx, parsedID)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("fetch resources distribution '%s': %w", id, err)
	}

	mapped, err := mapDistribution(distribution)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("map resources distribution '%s': %w", id, err)
	}
	return mapped, nil
}
