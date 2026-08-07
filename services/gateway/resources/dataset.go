package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) GetDatasetByID(ctx context.Context, id string) (api.Dataset, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.Dataset{}, err
	}
	dataset, err := s.resources.GetDataset(ctx, parsedID)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("fetch resources dataset '%s': %w", id, err)
	}

	mapped, err := mapDataset(dataset)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("map resources dataset '%s': %w", id, err)
	}
	return mapped, nil
}
