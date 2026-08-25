// Package lifecycle contains resource lifecycle use-cases.
package lifecycle

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	backoffice *clients.Backoffice
}

// NewService builds lifecycle service for Backoffice lifecycle operations.
func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{backoffice: backoffice}
}

func (s *Service) Create(ctx context.Context, actorID, groupID, resourceType string, input domain.CreateResourceInput) (domain.ResourceVersion, error) {
	result, err := s.backoffice.CreateResource(ctx, actorID, groupID, resourceType, input)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("create resource: %w", err)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.GetResourceVersion(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("get resource version: %w", err)
	}
	return result, nil
}

func (s *Service) ReplaceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.ReplaceResourceDraftInput) (domain.ResourceVersion, error) {
	result, err := s.backoffice.ReplaceResourceDraft(ctx, actorID, resourceType, resourceID, versionID, input)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("replace resource draft: %w", err)
	}
	return result, nil
}

func (s *Service) Discard(ctx context.Context, actorID, resourceType, resourceID, versionID string) error {
	if err := s.backoffice.DiscardResourceDraft(ctx, actorID, resourceType, resourceID, versionID); err != nil {
		return fmt.Errorf("discard resource draft: %w", err)
	}
	return nil
}

func (s *Service) Submit(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.SubmitResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("submit resource: %w", err)
	}
	return result, nil
}

func (s *Service) Withdraw(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.WithdrawResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("withdraw resource: %w", err)
	}
	return result, nil
}

func (s *Service) Reject(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.RejectResourceInput) (domain.ResourceVersion, error) {
	result, err := s.backoffice.RejectResource(ctx, actorID, resourceType, resourceID, versionID, input)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("reject resource: %w", err)
	}
	return result, nil
}

func (s *Service) Reopen(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.ReopenResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("reopen resource: %w", err)
	}
	return result, nil
}

func (s *Service) Publish(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.PublishResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("publish resource: %w", err)
	}
	return result, nil
}

func (s *Service) Unpublish(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.UnpublishResource(ctx, actorID, resourceType, resourceID, versionID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("unpublish resource: %w", err)
	}
	return result, nil
}

func (s *Service) DraftFromPublication(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.CreateResourceDraftFromPublication(ctx, actorID, resourceType, resourceID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("create draft from publication: %w", err)
	}
	return result, nil
}

func (s *Service) DraftFromArchive(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	result, err := s.backoffice.CreateResourceDraftFromArchive(ctx, actorID, resourceType, resourceID)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("create draft from archive: %w", err)
	}
	return result, nil
}

func (s *Service) Copy(ctx context.Context, actorID, groupID, resourceType, resourceID string, input domain.CopyResourceInput) (domain.ResourceVersion, error) {
	result, err := s.backoffice.CopyResource(ctx, actorID, groupID, resourceType, resourceID, input)
	if err != nil {
		return domain.ResourceVersion{}, fmt.Errorf("copy resource: %w", err)
	}
	return result, nil
}
