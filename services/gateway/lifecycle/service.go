// Package lifecycle contains resource lifecycle use-cases.
package lifecycle

import (
	"context"

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
	return s.backoffice.CreateResource(ctx, actorID, groupID, resourceType, input)
}

func (s *Service) Get(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	return s.backoffice.GetResourceVersion(ctx, actorID, resourceType, resourceID, versionID)
}

func (s *Service) ReplaceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.ReplaceResourceDraftInput) (domain.ResourceVersion, error) {
	return s.backoffice.ReplaceResourceDraft(ctx, actorID, resourceType, resourceID, versionID, input)
}

func (s *Service) Discard(ctx context.Context, actorID, resourceType, resourceID, versionID string) error {
	return s.backoffice.DiscardResourceDraft(ctx, actorID, resourceType, resourceID, versionID)
}

func (s *Service) Submit(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	return s.backoffice.SubmitResource(ctx, actorID, resourceType, resourceID, versionID)
}

func (s *Service) Withdraw(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	return s.backoffice.WithdrawResource(ctx, actorID, resourceType, resourceID, versionID)
}

func (s *Service) Reject(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.RejectResourceInput) (domain.ResourceVersion, error) {
	return s.backoffice.RejectResource(ctx, actorID, resourceType, resourceID, versionID, input)
}

func (s *Service) Reopen(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	return s.backoffice.ReopenResource(ctx, actorID, resourceType, resourceID, versionID)
}

func (s *Service) Publish(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	return s.backoffice.PublishResource(ctx, actorID, resourceType, resourceID, versionID)
}

func (s *Service) Unpublish(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	return s.backoffice.UnpublishResource(ctx, actorID, resourceType, resourceID, versionID)
}

func (s *Service) DraftFromPublication(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	return s.backoffice.CreateResourceDraftFromPublication(ctx, actorID, resourceType, resourceID)
}

func (s *Service) DraftFromArchive(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	return s.backoffice.CreateResourceDraftFromArchive(ctx, actorID, resourceType, resourceID)
}

func (s *Service) Copy(ctx context.Context, actorID, groupID, resourceType, resourceID string, input domain.CopyResourceInput) (domain.ResourceVersion, error) {
	return s.backoffice.CopyResource(ctx, actorID, groupID, resourceType, resourceID, input)
}
