// Package datasets manages dataset versions and their lifecycle.
package datasets

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// TODO: maybe we should move these permissions constants to the domain? or maybe we should make them constants in the db so that we have the sqlc constants and use those? or maybe have a migration that sets them up in the db but not hardcoding them in the sql but instead in the domain so that the migration uses them and they are the source of truth?
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
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	List(context.Context, uuid.UUID) ([]domain.ResourceVersionSummary, error)
	GetPublished(context.Context, uuid.UUID) (domain.DatasetVersion, error)
	GetActive(context.Context, uuid.UUID) (domain.DatasetVersion, error)
	CreateDraft(context.Context, string, uuid.UUID, domain.DatasetInput) (domain.DatasetVersion, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, domain.DatasetDraftInput) (domain.DatasetVersion, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, domain.DatasetRejectionInput) (domain.DatasetVersion, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Discard(context.Context, uuid.UUID, uuid.UUID) error
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (domain.DatasetVersion, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (domain.DatasetVersion, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
}

type Authorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (domain.Result, error)
	AuthorizeResource(context.Context, string, domain.ResourceType, uuid.UUID, string) (domain.Result, error)
	CanReadResourceVersions(context.Context, string, domain.ResourceType, uuid.UUID) (domain.Result, error)
	CanReadDatasetVersion(context.Context, *string, uuid.UUID, uuid.UUID) (domain.Result, error)
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
		log:        log.With("component", "datasets-service"),
	}
}

func (s *Service) CreateDraft(ctx context.Context, actor domain.Actor, groupID uuid.UUID, value domain.DatasetInput) (domain.DatasetVersion, error) {
	if err := s.authorizeGroup(ctx, actor, groupID, permissionCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft creation", err)
	}

	dataset, err := s.store.CreateDraft(ctx, string(actor), groupID, value)
	if err != nil {
		return domain.DatasetVersion{}, operationError("create draft", err)
	}

	return dataset, nil
}

func (s *Service) ReplaceDraft(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID, value domain.DatasetDraftInput) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionEdit); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft replacement", err)
	}

	dataset, err := s.store.ReplaceDraft(ctx, datasetID, versionID, value)
	if err != nil {
		return domain.DatasetVersion{}, operationError("replace draft", err)
	}

	return dataset, nil
}

func (s *Service) Read(ctx context.Context, actor *domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	result, err := s.authorizer.CanReadDatasetVersion(ctx, actor.OptionalString(), datasetID, versionID)
	if err != nil {
		return domain.DatasetVersion{}, operationError("authorize dataset read", err)
	}
	if !result.Allowed {
		return domain.DatasetVersion{}, fmt.Errorf("%w: %s", domain.ErrDatasetForbidden, result.Reason)
	}

	dataset, err := s.store.Get(ctx, datasetID, versionID)
	if err != nil {
		return domain.DatasetVersion{}, operationError("get dataset version", err)
	}

	return dataset, nil
}

func (s *Service) ListVersions(ctx context.Context, actor domain.Actor, datasetID uuid.UUID) ([]domain.ResourceVersionSummary, error) {
	result, err := s.authorizer.CanReadResourceVersions(ctx, string(actor), domain.Dataset, datasetID)
	if err != nil {
		return nil, operationError("authorize dataset version history", err)
	}
	if !result.Allowed {
		if result.Reason == "resource not found" {
			return nil, domain.ErrDatasetNotFound
		}
		return nil, fmt.Errorf("%w: %s", domain.ErrDatasetForbidden, result.Reason)
	}

	versions, err := s.store.List(ctx, datasetID)
	return versions, operationError("list dataset versions", err)
}

func (s *Service) Submit(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionSubmit); err != nil {
		return domain.DatasetVersion{}, operationError("authorize submission", err)
	}
	dataset, err := s.store.Submit(ctx, string(actor), datasetID, versionID)
	return dataset, operationError("submit draft", err)
}

func (s *Service) Withdraw(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionWithdraw); err != nil {
		return domain.DatasetVersion{}, operationError("authorize withdrawal", err)
	}
	dataset, err := s.store.Withdraw(ctx, datasetID, versionID)
	return dataset, operationError("withdraw submission", err)
}

func (s *Service) Reject(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID, value domain.DatasetRejectionInput) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionReject); err != nil {
		return domain.DatasetVersion{}, operationError("authorize rejection", err)
	}
	dataset, err := s.store.Reject(ctx, string(actor), datasetID, versionID, value)
	return dataset, operationError("reject submission", err)
}

func (s *Service) Reopen(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionReopen); err != nil {
		return domain.DatasetVersion{}, operationError("authorize reopening", err)
	}
	dataset, err := s.store.Reopen(ctx, datasetID, versionID)
	return dataset, operationError("reopen rejected version", err)
}

func (s *Service) Publish(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionPublish); err != nil {
		return domain.DatasetVersion{}, operationError("authorize publication", err)
	}
	dataset, err := s.store.Publish(ctx, string(actor), datasetID, versionID)
	return dataset, operationError("publish version", err)
}

func (s *Service) Discard(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) error {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionDiscard); err != nil {
		return operationError("authorize discard", err)
	}
	return operationError("discard version", s.store.Discard(ctx, datasetID, versionID))
}

func (s *Service) Unpublish(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionUnpublish); err != nil {
		return domain.DatasetVersion{}, operationError("authorize unpublication", err)
	}
	dataset, err := s.store.Unpublish(ctx, string(actor), datasetID, versionID)
	return dataset, operationError("unpublish version", err)
}

func (s *Service) DraftFromPublication(ctx context.Context, actor domain.Actor, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft creation", err)
	}
	dataset, err := s.store.DraftFromPublication(ctx, string(actor), datasetID)
	return dataset, operationError("create draft from publication", err)
}

func (s *Service) DraftFromLatestArchive(ctx context.Context, actor domain.Actor, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, permissionCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft creation", err)
	}
	dataset, err := s.store.DraftFromLatestArchive(ctx, string(actor), datasetID)
	return dataset, operationError("create draft from archive", err)
}

func (s *Service) CopyPublished(ctx context.Context, actor domain.Actor, sourceDatasetID, targetGroupID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeGroup(ctx, actor, targetGroupID, permissionCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize dataset copy", err)
	}
	dataset, err := s.store.CopyPublished(ctx, string(actor), sourceDatasetID, targetGroupID)
	return dataset, operationError("copy published dataset", err)
}

func (s *Service) authorizeGroup(ctx context.Context, actor domain.Actor, groupID uuid.UUID, permission string) error {
	result, err := s.authorizer.AuthorizeGroup(ctx, string(actor), groupID, permission)
	if err != nil {
		return fmt.Errorf("authorize group: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", domain.ErrDatasetForbidden, result.Reason)
	}
	return nil
}

func (s *Service) authorizeResource(ctx context.Context, actor domain.Actor, datasetID uuid.UUID, permission string) error {
	result, err := s.authorizer.AuthorizeResource(ctx, string(actor), domain.Dataset, datasetID, permission)
	if err != nil {
		return fmt.Errorf("authorize dataset: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", domain.ErrDatasetForbidden, result.Reason)
	}
	return nil
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrDatasetForbidden) || errors.Is(err, domain.ErrDatasetConflict) || errors.Is(err, domain.ErrDatasetNotFound) || errors.Is(err, domain.ErrInvalidDataset) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, domain.ErrDatasetInternal, err)
}
