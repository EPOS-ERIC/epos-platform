package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) GetSoftwareApplicationByID(ctx context.Context, id string) (api.SoftwareApplication, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.SoftwareApplication{}, err
	}
	software, err := s.resources.GetSoftwareApplication(ctx, parsedID)
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("fetch resources software application '%s': %w", id, err)
	}
	return mapSoftwareApplication(software), nil
}

func (s *Service) GetSoftwareSourceCodeByID(ctx context.Context, id string) (api.SoftwareSourceCode, error) {
	parsedID, err := resourceID(id)
	if err != nil {
		return api.SoftwareSourceCode{}, err
	}
	software, err := s.resources.GetSoftwareSourceCode(ctx, parsedID)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("fetch resources software source code '%s': %w", id, err)
	}
	return mapSoftwareSourceCode(software), nil
}
