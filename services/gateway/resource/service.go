// Package resource orchestrates resource reads and conversion discovery.
package resource

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatav1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type metadataLifecycle interface {
	GetResourceVersion(context.Context, string, domain.ResourceType, uuid.UUID, uuid.UUID) (*metadatav1.GetResourceVersionResponse, domain.ResourceVersion, error)
}

type converter interface {
	AvailableConversions(context.Context) ([]domain.AvailableConversion, error)
}

type Service struct {
	metadata  metadataLifecycle
	converter converter
}

func New(metadata metadataLifecycle, converter converter) *Service {
	return &Service{
		metadata:  metadata,
		converter: converter,
	}
}

func (s *Service) GetVersion(ctx context.Context, actor string, resourceType domain.ResourceType, resourceID, versionID uuid.UUID) (*metadatav1.GetResourceVersionResponse, domain.ResourceVersion, error) {
	response, resource, err := s.metadata.GetResourceVersion(ctx, actor, resourceType, resourceID, versionID)
	if err != nil {
		return nil, domain.ResourceVersion{}, fmt.Errorf("get resource version: %w", err)
	}

	if !resource.IsDataset {
		return response, resource, nil
	}

	conversions, err := s.converter.AvailableConversions(ctx)
	if apperror.IsCode(err, apperror.CodeUpstreamUnavailable) || apperror.IsCode(err, apperror.CodeUpstreamTimeout) {
		return response, resource, nil
	}
	if err != nil {
		return nil, domain.ResourceVersion{}, fmt.Errorf("list available conversions: %w", err)
	}

	byDistribution := make(map[uuid.UUID][]domain.AvailableConversion)
	for _, conversion := range conversions {
		byDistribution[conversion.DistributionID] = append(byDistribution[conversion.DistributionID], conversion)
	}

	for i := range resource.Distributions {
		resource.Distributions[i].Conversions = byDistribution[resource.Distributions[i].ID]
	}

	return response, resource, nil
}
