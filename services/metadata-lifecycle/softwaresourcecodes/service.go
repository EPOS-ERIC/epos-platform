// Package softwaresourcecodes manages software source-code versions and their lifecycle.
package softwaresourcecodes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

const (
	permissionCreate    = "resource.create"
	permissionEdit      = "resource.edit"
	permissionSubmit    = "resource.submit"
	permissionWithdraw  = "resource.withdraw"
	permissionReject    = "resource.reject"
	permissionReopen    = "resource.reopen"
	permissionPublish   = "resource.publish"
	permissionDiscard   = "resource.discard"
	permissionUnpublish = "resource.unpublish"
)

type Store interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	GetPublished(context.Context, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	GetActive(context.Context, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	CreateDraft(context.Context, string, uuid.UUID, domain.SoftwareSourceCodeInput) (domain.SoftwareSourceCodeVersion, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, domain.SoftwareSourceCodeDraftInput) (domain.SoftwareSourceCodeVersion, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, domain.SoftwareSourceCodeRejectionInput) (domain.SoftwareSourceCodeVersion, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Discard(context.Context, uuid.UUID, uuid.UUID) error
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
}

type Authorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (domain.Result, error)
	AuthorizeResource(context.Context, string, domain.ResourceType, uuid.UUID, string) (domain.Result, error)
	CanReadSoftwareSourceCodeVersion(context.Context, *string, uuid.UUID, uuid.UUID) (domain.Result, error)
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
		log:        log.With("component", "software-source-codes-service"),
	}
}

func (s *Service) CreateDraft(ctx context.Context, actor domain.Actor, groupID uuid.UUID, value domain.SoftwareSourceCodeInput) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeGroup(ctx, actor, groupID, permissionCreate); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize draft creation", err)
	}
	sourceCode, err := s.store.CreateDraft(ctx, string(actor), groupID, value)
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("create draft", err)
	}
	return sourceCode, nil
}

func (s *Service) ReplaceDraft(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID, value domain.SoftwareSourceCodeDraftInput) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionEdit); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize draft replacement", err)
	}
	sourceCode, err := s.store.ReplaceDraft(ctx, sourceCodeID, versionID, value)
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("replace draft", err)
	}
	return sourceCode, nil
}

func (s *Service) Read(ctx context.Context, actor *domain.Actor, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	result, err := s.authorizer.CanReadSoftwareSourceCodeVersion(ctx, actor.OptionalString(), sourceCodeID, versionID)
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize source code read", err)
	}
	if !result.Allowed {
		return domain.SoftwareSourceCodeVersion{}, fmt.Errorf("%w: %s", domain.ErrSoftwareSourceCodeForbidden, result.Reason)
	}

	sourceCode, err := s.store.Get(ctx, sourceCodeID, versionID)
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("get source code version", err)
	}
	return sourceCode, nil
}

func (s *Service) Submit(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionSubmit); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize submission", err)
	}
	sourceCode, err := s.store.Submit(ctx, string(actor), sourceCodeID, versionID)
	return sourceCode, operationError("submit draft", err)
}

func (s *Service) Withdraw(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionWithdraw); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize withdrawal", err)
	}
	sourceCode, err := s.store.Withdraw(ctx, sourceCodeID, versionID)
	return sourceCode, operationError("withdraw submission", err)
}

func (s *Service) Reject(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID, value domain.SoftwareSourceCodeRejectionInput) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionReject); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize rejection", err)
	}
	sourceCode, err := s.store.Reject(ctx, string(actor), sourceCodeID, versionID, value)
	return sourceCode, operationError("reject submission", err)
}

func (s *Service) Reopen(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionReopen); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize reopening", err)
	}
	sourceCode, err := s.store.Reopen(ctx, sourceCodeID, versionID)
	return sourceCode, operationError("reopen rejected version", err)
}

func (s *Service) Publish(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionPublish); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize publication", err)
	}
	sourceCode, err := s.store.Publish(ctx, string(actor), sourceCodeID, versionID)
	return sourceCode, operationError("publish version", err)
}

func (s *Service) Discard(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID) error {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionDiscard); err != nil {
		return operationError("authorize discard", err)
	}
	return operationError("discard version", s.store.Discard(ctx, sourceCodeID, versionID))
}

func (s *Service) Unpublish(ctx context.Context, actor domain.Actor, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionUnpublish); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize unpublication", err)
	}
	sourceCode, err := s.store.Unpublish(ctx, string(actor), sourceCodeID, versionID)
	return sourceCode, operationError("unpublish version", err)
}

func (s *Service) DraftFromPublication(ctx context.Context, actor domain.Actor, sourceCodeID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionCreate); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize draft creation", err)
	}
	sourceCode, err := s.store.DraftFromPublication(ctx, string(actor), sourceCodeID)
	return sourceCode, operationError("create draft from publication", err)
}

func (s *Service) DraftFromLatestArchive(ctx context.Context, actor domain.Actor, sourceCodeID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeResource(ctx, actor, sourceCodeID, permissionCreate); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize draft creation", err)
	}
	sourceCode, err := s.store.DraftFromLatestArchive(ctx, string(actor), sourceCodeID)
	return sourceCode, operationError("create draft from archive", err)
}

func (s *Service) CopyPublished(ctx context.Context, actor domain.Actor, sourceSourceCodeID, targetGroupID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	if err := s.authorizeGroup(ctx, actor, targetGroupID, permissionCreate); err != nil {
		return domain.SoftwareSourceCodeVersion{}, operationError("authorize source code copy", err)
	}
	sourceCode, err := s.store.CopyPublished(ctx, string(actor), sourceSourceCodeID, targetGroupID)
	return sourceCode, operationError("copy published source code", err)
}

func (s *Service) authorizeGroup(ctx context.Context, actor domain.Actor, groupID uuid.UUID, permission string) error {
	result, err := s.authorizer.AuthorizeGroup(ctx, string(actor), groupID, permission)
	if err != nil {
		return fmt.Errorf("authorize group: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", domain.ErrSoftwareSourceCodeForbidden, result.Reason)
	}
	return nil
}

func (s *Service) authorizeResource(ctx context.Context, actor domain.Actor, sourceCodeID uuid.UUID, permission string) error {
	result, err := s.authorizer.AuthorizeResource(ctx, string(actor), domain.SoftwareSourceCode, sourceCodeID, permission)
	if err != nil {
		return fmt.Errorf("authorize source code: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", domain.ErrSoftwareSourceCodeForbidden, result.Reason)
	}
	return nil
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrSoftwareSourceCodeForbidden) || errors.Is(err, domain.ErrSoftwareSourceCodeConflict) || errors.Is(err, domain.ErrSoftwareSourceCodeNotFound) || errors.Is(err, domain.ErrSoftwareSourceCodeInvalid) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, domain.ErrSoftwareSourceCodeInternal, err)
}
