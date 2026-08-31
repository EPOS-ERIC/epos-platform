// Package lifecycle contains resource lifecycle use-cases.
package lifecycle

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	metadataLifecycle *clients.MetadataLifecycle

	log *slog.Logger
}

// NewService builds lifecycle service for Metadata Lifecycle lifecycle operations.
func NewService(log *slog.Logger, metadataLifecycle *clients.MetadataLifecycle) *Service {
	return &Service{
		metadataLifecycle: metadataLifecycle,
		log:               log.With("component", "lifecycle-service"),
	}
}

func (s *Service) Create(ctx context.Context, actorID, groupID, resourceType string, input domain.CreateResourceInput) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.CreateResource(ctx, actorID, groupID, resourceType, input)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("create resource: %w", err)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	s.log.Debug("get resource version", "resourceType", resourceType, "resourceID", resourceID, "versionID", versionID)

	result, err := s.metadataLifecycle.GetResourceVersion(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		s.log.Debug("get resource version", "error", err)
		return domain.ResourceVersion{}, fmt.Errorf("get resource version: %w", err)
	}

	s.log.Debug("get resource version", "response", result)

	return result, nil
}

func (s *Service) ReplaceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.ReplaceResourceDraftInput) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.ReplaceResourceDraft(ctx, actorID, resourceType, resourceID, versionID, input)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("replace resource draft: %w", err)
	}
	return result, nil
}

func (s *Service) Discard(ctx context.Context, actorID, resourceType, resourceID, versionID string) error {
	if err := s.metadataLifecycle.DiscardResourceDraft(ctx, actorID, resourceType, resourceID, versionID); err != nil {
		return fmt.Errorf("discard resource draft: %w", err)
	}
	return nil
}

func (s *Service) Submit(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.SubmitResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("submit resource: %w", err)
	}
	return result, nil
}

func (s *Service) Withdraw(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.WithdrawResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("withdraw resource: %w", err)
	}
	return result, nil
}

func (s *Service) Reject(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.RejectResourceInput) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.RejectResource(ctx, actorID, resourceType, resourceID, versionID, input)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("reject resource: %w", err)
	}
	return result, nil
}

func (s *Service) Reopen(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.ReopenResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("reopen resource: %w", err)
	}
	return result, nil
}

func (s *Service) Publish(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.PublishResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("publish resource: %w", err)
	}
	return result, nil
}

func (s *Service) Unpublish(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.UnpublishResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("unpublish resource: %w", err)
	}
	return result, nil
}

func (s *Service) DraftFromPublication(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.CreateResourceDraftFromPublication(ctx, actorID, resourceType, resourceID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("create draft from publication: %w", err)
	}
	return result, nil
}

func (s *Service) DraftFromArchive(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.CreateResourceDraftFromArchive(ctx, actorID, resourceType, resourceID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("create draft from archive: %w", err)
	}
	return result, nil
}

func (s *Service) Copy(ctx context.Context, actorID, groupID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	result, err := s.metadataLifecycle.CopyResource(ctx, actorID, groupID, resourceType, resourceID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("copy resource: %w", err)
	}
	return result, nil
}
