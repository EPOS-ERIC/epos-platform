// Package softwareapplications manages software application versions and their lifecycle.
package softwareapplications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	List(context.Context, uuid.UUID) ([]domain.ResourceVersionSummary, error)
	GetPublished(context.Context, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	GetActive(context.Context, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	CreateDraft(context.Context, string, uuid.UUID, domain.SoftwareApplicationInput) (domain.SoftwareApplicationVersion, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, domain.SoftwareApplicationDraftInput) (domain.SoftwareApplicationVersion, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, domain.SoftwareApplicationRejectionInput) (domain.SoftwareApplicationVersion, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Discard(context.Context, uuid.UUID, uuid.UUID) error
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
}

type Authorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, domain.GroupPermission) (domain.Result, error)
	AuthorizeResource(context.Context, string, domain.ResourceType, uuid.UUID, domain.GroupPermission) (domain.Result, error)
	CanReadResourceVersions(context.Context, string, domain.ResourceType, uuid.UUID) (domain.Result, error)
	CanReadSoftwareApplicationVersion(context.Context, *string, uuid.UUID, uuid.UUID) (domain.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	log        *slog.Logger
}

func New(log *slog.Logger, store Store, authorizer Authorizer) *Service {
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "software-applications-service"),
	}
}

func (s *Service) CreateDraft(ctx context.Context, actor domain.Actor, groupID uuid.UUID, value domain.SoftwareApplicationInput) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeGroup(ctx, actor, groupID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize draft creation", err)
	}

	application, err := s.store.CreateDraft(ctx, string(actor), groupID, value)
	if err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("create draft", err)
	}
	return application, nil
}

func (s *Service) ReplaceDraft(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID, value domain.SoftwareApplicationDraftInput) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceEdit); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize draft replacement", err)
	}

	application, err := s.store.ReplaceDraft(ctx, applicationID, versionID, value)
	if err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("replace draft", err)
	}
	return application, nil
}

func (s *Service) Read(ctx context.Context, actor *domain.Actor, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	result, err := s.authorizer.CanReadSoftwareApplicationVersion(ctx, actor.OptionalString(), applicationID, versionID)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.SoftwareApplicationVersion{}, domain.ErrSoftwareApplicationNotFound
	}
	if err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize application read", err)
	}
	if !result.Allowed {
		return domain.SoftwareApplicationVersion{}, fmt.Errorf("%w: %s", domain.ErrSoftwareApplicationForbidden, result.Reason)
	}

	application, err := s.store.Get(ctx, applicationID, versionID)
	if err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("get application version", err)
	}
	return application, nil
}

func (s *Service) ListVersions(ctx context.Context, actor domain.Actor, applicationID uuid.UUID) ([]domain.ResourceVersionSummary, error) {
	result, err := s.authorizer.CanReadResourceVersions(ctx, string(actor), domain.SoftwareApplication, applicationID)
	if err != nil {
		return nil, operationError("authorize application version history", err)
	}
	if !result.Allowed {
		if result.Reason == "resource not found" {
			return nil, domain.ErrSoftwareApplicationNotFound
		}
		return nil, fmt.Errorf("%w: %s", domain.ErrSoftwareApplicationForbidden, result.Reason)
	}

	versions, err := s.store.List(ctx, applicationID)
	return versions, operationError("list application versions", err)
}

func (s *Service) Submit(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceSubmit); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize submission", err)
	}
	application, err := s.store.Submit(ctx, string(actor), applicationID, versionID)
	return application, operationError("submit draft", err)
}

func (s *Service) Withdraw(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceWithdraw); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize withdrawal", err)
	}
	application, err := s.store.Withdraw(ctx, applicationID, versionID)
	return application, operationError("withdraw submission", err)
}

func (s *Service) Reject(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID, value domain.SoftwareApplicationRejectionInput) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceReject); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize rejection", err)
	}
	application, err := s.store.Reject(ctx, string(actor), applicationID, versionID, value)
	return application, operationError("reject submission", err)
}

func (s *Service) Reopen(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceReopen); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize reopening", err)
	}
	application, err := s.store.Reopen(ctx, applicationID, versionID)
	return application, operationError("reopen rejected version", err)
}

func (s *Service) Publish(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourcePublish); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize publication", err)
	}
	application, err := s.store.Publish(ctx, string(actor), applicationID, versionID)
	return application, operationError("publish version", err)
}

func (s *Service) Discard(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID) error {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceDiscard); err != nil {
		return operationError("authorize discard", err)
	}
	return operationError("discard version", s.store.Discard(ctx, applicationID, versionID))
}

func (s *Service) Unpublish(ctx context.Context, actor domain.Actor, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceUnpublish); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize unpublication", err)
	}
	application, err := s.store.Unpublish(ctx, string(actor), applicationID, versionID)
	return application, operationError("unpublish version", err)
}

func (s *Service) DraftFromPublication(ctx context.Context, actor domain.Actor, applicationID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize draft creation", err)
	}
	application, err := s.store.DraftFromPublication(ctx, string(actor), applicationID)
	return application, operationError("create draft from publication", err)
}

func (s *Service) DraftFromLatestArchive(ctx context.Context, actor domain.Actor, applicationID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeResource(ctx, actor, applicationID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize draft creation", err)
	}
	application, err := s.store.DraftFromLatestArchive(ctx, string(actor), applicationID)
	return application, operationError("create draft from archive", err)
}

func (s *Service) CopyPublished(ctx context.Context, actor domain.Actor, sourceApplicationID, targetGroupID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	if err := s.authorizeGroup(ctx, actor, targetGroupID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.SoftwareApplicationVersion{}, operationError("authorize application copy", err)
	}
	application, err := s.store.CopyPublished(ctx, string(actor), sourceApplicationID, targetGroupID)
	return application, operationError("copy published application", err)
}

func (s *Service) authorizeGroup(ctx context.Context, actor domain.Actor, groupID uuid.UUID, permission domain.GroupPermission) error {
	result, err := s.authorizer.AuthorizeGroup(ctx, string(actor), groupID, permission)
	if err != nil {
		return fmt.Errorf("authorize group: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", domain.ErrSoftwareApplicationForbidden, result.Reason)
	}
	return nil
}

func (s *Service) authorizeResource(ctx context.Context, actor domain.Actor, applicationID uuid.UUID, permission domain.GroupPermission) error {
	result, err := s.authorizer.AuthorizeResource(ctx, string(actor), domain.SoftwareApplication, applicationID, permission)
	if err != nil {
		return fmt.Errorf("authorize application: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", domain.ErrSoftwareApplicationForbidden, result.Reason)
	}
	return nil
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrSoftwareApplicationForbidden) || errors.Is(err, domain.ErrSoftwareApplicationConflict) || errors.Is(err, domain.ErrSoftwareApplicationNotFound) || errors.Is(err, domain.ErrSoftwareApplicationInvalid) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, domain.ErrSoftwareApplicationInternal, err)
}
