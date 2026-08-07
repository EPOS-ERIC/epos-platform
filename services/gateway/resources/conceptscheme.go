package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) GetConceptSchemeByID(ctx context.Context, id string) (api.ConceptScheme, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.ConceptScheme{}, err
	}
	conceptScheme, err := s.resources.GetConceptScheme(ctx, parsedID)
	if err != nil {
		return api.ConceptScheme{}, fmt.Errorf("fetch resources concept scheme '%s': %w", id, err)
	}

	return mapConceptScheme(conceptScheme), nil
}
