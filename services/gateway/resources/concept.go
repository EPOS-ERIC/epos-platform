package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) GetConceptByID(ctx context.Context, id string) (api.Concept, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.Concept{}, err
	}
	concept, err := s.resources.GetConcept(ctx, parsedID)
	if err != nil {
		return api.Concept{}, fmt.Errorf("fetch resources concept '%s': %w", id, err)
	}

	return mapConcept(concept), nil
}
