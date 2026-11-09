// Package resource orchestrates resource reads and conversion discovery.
package resource

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatav1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type metadataLifecycle interface {
	GetResourceVersion(context.Context, string, domain.ResourceType, uuid.UUID, uuid.UUID) (*metadatav1.GetResourceVersionResponse, domain.ResourceVersion, error)
	CopyResource(context.Context, string, string, metadatav1.ResourceType, string) (*metadatav1.CopyPublishedResourceResponse, map[uuid.UUID]uuid.UUID, error)
	CreateResourceDraftFromPublication(context.Context, string, metadatav1.ResourceType, string) (*metadatav1.DraftResourceFromPublicationResponse, map[uuid.UUID]uuid.UUID, error)
	CreateResourceDraftFromArchive(context.Context, string, metadatav1.ResourceType, string) (*metadatav1.DraftResourceFromLatestArchiveResponse, map[uuid.UUID]uuid.UUID, error)
	ReplaceResourceDraft(context.Context, string, string, string, *metadatav1.ReplaceResourceDraftRequest) (*metadatav1.ReplaceResourceDraftResponse, []uuid.UUID, error)
	DiscardResourceDraft(context.Context, string, metadatav1.ResourceType, string, string) ([]uuid.UUID, error)
}

type converter interface {
	AvailableConversions(context.Context) ([]domain.AvailableConversion, error)
	CopyRelations(context.Context, string, map[uuid.UUID]uuid.UUID) error
	RemoveRelations(context.Context, string, []uuid.UUID) error
}

type Service struct {
	metadata  metadataLifecycle
	converter converter
	log       *slog.Logger
}

func New(log *slog.Logger, metadata metadataLifecycle, converter converter) *Service {
	return &Service{
		metadata:  metadata,
		converter: converter,
		log:       log,
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

	resource.ConversionsByDistribution = make(map[uuid.UUID][]domain.AvailableConversion, len(conversions))
	for _, conversion := range conversions {
		resource.ConversionsByDistribution[conversion.DistributionID] = append(resource.ConversionsByDistribution[conversion.DistributionID], conversion)
	}

	return response, resource, nil
}

func (s *Service) Copy(ctx context.Context, actor, group string, kind metadatav1.ResourceType, id string) (*metadatav1.CopyPublishedResourceResponse, error) {
	response, pairs, err := s.metadata.CopyResource(ctx, actor, group, kind, id)
	if err != nil {
		return nil, err
	}

	if err := s.converter.CopyRelations(ctx, actor, pairs); err != nil {
		s.log.ErrorContext(ctx, "copy converter relations failed", "resource_id", id, "distribution_pairs", pairs, "error", err)
	}

	return response, nil
}

func (s *Service) DraftPublication(ctx context.Context, actor string, kind metadatav1.ResourceType, id string) (*metadatav1.DraftResourceFromPublicationResponse, error) {
	response, pairs, err := s.metadata.CreateResourceDraftFromPublication(ctx, actor, kind, id)
	if err != nil {
		return nil, err
	}

	if err := s.converter.CopyRelations(ctx, actor, pairs); err != nil {
		s.log.ErrorContext(ctx, "copy converter relations failed", "resource_id", id, "distribution_pairs", pairs, "error", err)
	}

	return response, nil
}

func (s *Service) DraftArchive(ctx context.Context, actor string, kind metadatav1.ResourceType, id string) (*metadatav1.DraftResourceFromLatestArchiveResponse, error) {
	response, pairs, err := s.metadata.CreateResourceDraftFromArchive(ctx, actor, kind, id)
	if err != nil {
		return nil, err
	}

	if err := s.converter.CopyRelations(ctx, actor, pairs); err != nil {
		s.log.ErrorContext(ctx, "copy converter relations failed", "resource_id", id, "distribution_pairs", pairs, "error", err)
	}

	return response, nil
}

func (s *Service) Replace(ctx context.Context, actor, id, version string, request *metadatav1.ReplaceResourceDraftRequest) (*metadatav1.ReplaceResourceDraftResponse, error) {
	response, removed, err := s.metadata.ReplaceResourceDraft(ctx, actor, id, version, request)
	if err != nil {
		return nil, err
	}

	if err := s.converter.RemoveRelations(ctx, actor, removed); err != nil {
		s.log.ErrorContext(ctx, "remove converter relations failed", "resource_id", id, "removed_distribution_ids", removed, "error", err)
	}

	return response, nil
}

func (s *Service) Discard(ctx context.Context, actor string, kind metadatav1.ResourceType, id, version string) error {
	removed, err := s.metadata.DiscardResourceDraft(ctx, actor, kind, id, version)
	if err != nil {
		return err
	}

	if err := s.converter.RemoveRelations(ctx, actor, removed); err != nil {
		s.log.ErrorContext(ctx, "remove converter relations failed", "resource_id", id, "removed_distribution_ids", removed, "error", err)
	}

	return nil
}
