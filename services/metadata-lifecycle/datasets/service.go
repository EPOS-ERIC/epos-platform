// Package datasets manages dataset versions and their lifecycle.
package datasets

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
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
	AuthorizeGroup(context.Context, string, uuid.UUID, domain.GroupPermission) (domain.Result, error)
	AuthorizeResource(context.Context, string, domain.ResourceType, uuid.UUID, domain.GroupPermission) (domain.Result, error)
	CanReadResourceVersions(context.Context, string, domain.ResourceType, uuid.UUID) (domain.Result, error)
	CanReadDatasetVersion(context.Context, *string, uuid.UUID, uuid.UUID) (domain.Result, error)
}

type Notifier interface {
	Build(context.Context, domain.NotificationEvent) ([]domain.Notification, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	log        *slog.Logger
	notifier   Notifier
}

func New(log *slog.Logger, store Store, authorizer Authorizer, notifier Notifier) *Service {
	// TODO: in all services we should have a validation check that all inputs are injected correctly and panic if not
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "datasets-service"),
		notifier:   notifier,
	}
}

func (s *Service) CreateDraft(ctx context.Context, actor domain.Actor, groupID uuid.UUID, value domain.DatasetInput) (domain.DatasetVersion, error) {
	if err := s.authorizeGroup(ctx, actor, groupID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft creation", err)
	}

	dataset, err := s.store.CreateDraft(ctx, string(actor), groupID, value)
	if err != nil {
		return domain.DatasetVersion{}, operationError("create draft", err)
	}

	return dataset, nil
}

func (s *Service) ReplaceDraft(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID, value domain.DatasetDraftInput) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceEdit); err != nil {
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
	if apperror.IsCode(err, apperror.CodeNotFound) {
		return domain.DatasetVersion{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.DatasetVersion{}, operationError("authorize dataset read", err)
	}
	if !result.Allowed {
		return domain.DatasetVersion{}, fmt.Errorf("%w: %s", apperror.Default(apperror.CodePermissionDenied), result.Reason)
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
			return nil, apperror.Default(apperror.CodeNotFound)
		}
		return nil, fmt.Errorf("%w: %s", apperror.Default(apperror.CodePermissionDenied), result.Reason)
	}

	versions, err := s.store.List(ctx, datasetID)
	return versions, operationError("list dataset versions", err)
}

//nolint:dupl // Resource lifecycle paths intentionally mirror each other.
func (s *Service) Submit(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceSubmit); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize submission", err)
	}

	dataset, err := s.store.Submit(ctx, string(actor), datasetID, versionID)
	if err != nil {
		if apperror.IsCode(err, apperror.CodeConflict) {
			return domain.DatasetVersion{}, nil, apperror.Wrap(
				apperror.CodeInvalidTransition,
				"The resource cannot be submitted from its current state.",
				err,
			)
		}
		return domain.DatasetVersion{}, nil, operationError("submit draft", err)
	}

	notification, buildErr := s.notifier.Build(ctx, domain.NotificationEvent{
		Kind:     domain.NotificationEventResourceSubmitted,
		ActorID:  actor,
		GroupID:  &dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourceSubmitted, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func (s *Service) Withdraw(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceWithdraw); err != nil {
		return domain.DatasetVersion{}, operationError("authorize withdrawal", err)
	}
	dataset, err := s.store.Withdraw(ctx, datasetID, versionID)
	return dataset, operationError("withdraw submission", err)
}

func (s *Service) Reject(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID, value domain.DatasetRejectionInput) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceReject); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize rejection", err)
	}

	dataset, err := s.store.Reject(ctx, string(actor), datasetID, versionID, value)
	if err != nil {
		return domain.DatasetVersion{}, nil, operationError("reject submission", err)
	}

	notification, buildErr := s.notifier.Build(ctx, domain.NotificationEvent{
		Kind:     domain.NotificationEventResourceRejected,
		ActorID:  actor,
		GroupID:  &dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourceRejected, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func (s *Service) Reopen(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceReopen); err != nil {
		return domain.DatasetVersion{}, operationError("authorize reopening", err)
	}
	dataset, err := s.store.Reopen(ctx, datasetID, versionID)
	return dataset, operationError("reopen rejected version", err)
}

//nolint:dupl // Resource lifecycle notification paths intentionally mirror each other.
func (s *Service) Publish(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourcePublish); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize publication", err)
	}

	dataset, err := s.store.Publish(ctx, string(actor), datasetID, versionID)
	if err != nil {
		return domain.DatasetVersion{}, nil, operationError("publish version", err)
	}

	notification, buildErr := s.notifier.Build(ctx, domain.NotificationEvent{
		Kind:     domain.NotificationEventResourcePublished,
		ActorID:  actor,
		GroupID:  &dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourcePublished, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func (s *Service) Discard(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) error {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceDiscard); err != nil {
		return operationError("authorize discard", err)
	}
	return operationError("discard version", s.store.Discard(ctx, datasetID, versionID))
}

//nolint:dupl // Resource lifecycle notification paths intentionally mirror each other.
func (s *Service) Unpublish(ctx context.Context, actor domain.Actor, datasetID, versionID uuid.UUID) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceUnpublish); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize unpublication", err)
	}

	dataset, err := s.store.Unpublish(ctx, string(actor), datasetID, versionID)
	if err != nil {
		return domain.DatasetVersion{}, nil, operationError("unpublish version", err)
	}

	notification, buildErr := s.notifier.Build(ctx, domain.NotificationEvent{
		Kind:     domain.NotificationEventResourceUnpublished,
		ActorID:  actor,
		GroupID:  &dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourceUnpublished, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func datasetNotificationResource(value domain.DatasetVersion) *domain.NotificationResource {
	return &domain.NotificationResource{
		Type:          domain.Dataset,
		ResourceID:    value.DatasetID,
		VersionID:     value.VersionID,
		Revision:      value.Revision,
		Status:        value.Status,
		Name:          value.Metadata.Title,
		SubmittedBy:   value.SubmittedBy.OptionalString(),
		RejectionNote: value.RejectionNote,
	}
}

func (s *Service) DraftFromPublication(ctx context.Context, actor domain.Actor, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft creation", err)
	}
	dataset, err := s.store.DraftFromPublication(ctx, string(actor), datasetID)
	return dataset, operationError("create draft from publication", err)
}

func (s *Service) DraftFromLatestArchive(ctx context.Context, actor domain.Actor, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft creation", err)
	}
	dataset, err := s.store.DraftFromLatestArchive(ctx, string(actor), datasetID)
	return dataset, operationError("create draft from archive", err)
}

func (s *Service) CopyPublished(ctx context.Context, actor domain.Actor, sourceDatasetID, targetGroupID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeGroup(ctx, actor, targetGroupID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize dataset copy", err)
	}
	dataset, err := s.store.CopyPublished(ctx, string(actor), sourceDatasetID, targetGroupID)
	return dataset, operationError("copy published dataset", err)
}

func (s *Service) authorizeGroup(ctx context.Context, actor domain.Actor, groupID uuid.UUID, permission domain.GroupPermission) error {
	result, err := s.authorizer.AuthorizeGroup(ctx, string(actor), groupID, permission)
	if err != nil {
		return fmt.Errorf("authorize group: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", apperror.Default(apperror.CodePermissionDenied), result.Reason)
	}
	return nil
}

func (s *Service) authorizeResource(ctx context.Context, actor domain.Actor, datasetID uuid.UUID, permission domain.GroupPermission) error {
	result, err := s.authorizer.AuthorizeResource(ctx, string(actor), domain.Dataset, datasetID, permission)
	if err != nil {
		return fmt.Errorf("authorize dataset: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", apperror.Default(apperror.CodePermissionDenied), result.Reason)
	}
	return nil
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if apperror.IsCode(err, apperror.CodePermissionDenied) || apperror.IsCode(err, apperror.CodeConflict) || apperror.IsCode(err, apperror.CodeNotFound) || apperror.IsCode(err, apperror.CodeValidationFailed) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, apperror.Default(apperror.CodeInternal), err)
}
