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
	MonitoringDistributions(context.Context) ([]domain.MonitoringDistribution, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	GetDistributionDatasetVersion(context.Context, uuid.UUID) (uuid.UUID, uuid.UUID, error)
	List(context.Context, uuid.UUID) ([]domain.ResourceVersionSummary, error)
	GetPublished(context.Context, uuid.UUID) (domain.DatasetVersion, error)
	GetActive(context.Context, uuid.UUID) (domain.DatasetVersion, error)
	CreateDraft(context.Context, string, uuid.UUID, domain.DatasetInput) (domain.DatasetVersion, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, domain.DatasetDraftInput) (domain.DatasetVersion, []uuid.UUID, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, domain.RejectionInput) (domain.DatasetVersion, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Discard(context.Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error)
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error)
}

type Authorizer interface {
	AuthorizeGlobal(context.Context, string, domain.GlobalPermission) (domain.Result, error)
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
	if log == nil {
		panic("datasets: nil logger")
	}
	if store == nil {
		panic("datasets: nil store")
	}
	if authorizer == nil {
		panic("datasets: nil authorizer")
	}
	if notifier == nil {
		panic("datasets: nil notifier")
	}
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "datasets-service"),
		notifier:   notifier,
	}
}

func (s *Service) CreateDraft(ctx context.Context, actor string, groupID uuid.UUID, value domain.DatasetInput) (domain.DatasetVersion, error) {
	if err := s.authorizeGroup(ctx, actor, groupID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, operationError("authorize draft creation", err)
	}

	dataset, err := s.store.CreateDraft(ctx, actor, groupID, value)
	if err != nil {
		return domain.DatasetVersion{}, operationError("create draft", err)
	}

	return dataset, nil
}

func (s *Service) ReplaceDraft(ctx context.Context, actor string, datasetID, versionID uuid.UUID, value domain.DatasetDraftInput) (domain.DatasetVersion, []uuid.UUID, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceEdit); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize draft replacement", err)
	}

	dataset, removed, err := s.store.ReplaceDraft(ctx, datasetID, versionID, value)
	if err != nil {
		return domain.DatasetVersion{}, nil, operationError("replace draft", err)
	}

	return dataset, removed, nil
}

func (s *Service) Read(ctx context.Context, actor *string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	result, err := s.authorizer.CanReadDatasetVersion(ctx, actor, datasetID, versionID)
	if apperror.IsCode(err, apperror.CodeNotFound) {
		return domain.DatasetVersion{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.DatasetVersion{}, operationError("authorize dataset read", err)
	}

	if !result.Allowed {
		return domain.DatasetVersion{}, apperror.Default(apperror.CodeNotFound)
	}

	dataset, err := s.store.Get(ctx, datasetID, versionID)
	if err != nil {
		return domain.DatasetVersion{}, operationError("get dataset version", err)
	}

	return dataset, nil
}

func (s *Service) PrepareDownload(ctx context.Context, actor *string, distributionID uuid.UUID) (domain.DistributionDownload, error) {
	datasetID, versionID, distribution, err := s.readDistribution(ctx, actor, distributionID)
	if err != nil {
		return domain.DistributionDownload{}, err
	}

	if distribution.Type != domain.DistributionTypeDownloadableFile || distribution.DownloadableFile == nil {
		return domain.DistributionDownload{}, apperror.Field("distributionId", "Distribution is not a downloadable file.", nil)
	}

	return domain.DistributionDownload{
		ResourceID: datasetID,
		VersionID:  versionID,
		URL:        distribution.DownloadableFile.URL,
	}, nil
}

func (s *Service) ListVersions(ctx context.Context, actor string, datasetID uuid.UUID) ([]domain.ResourceVersionSummary, error) {
	result, err := s.authorizer.CanReadResourceVersions(ctx, actor, domain.Dataset, datasetID)
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
func (s *Service) Submit(ctx context.Context, actor string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceSubmit); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize submission", err)
	}

	dataset, err := s.store.Submit(ctx, actor, datasetID, versionID)
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

	notification, buildErr := s.notifier.Build(ctx, domain.ResourceSubmitted{
		ActorID:  actor,
		GroupID:  dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourceSubmitted, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func (s *Service) Withdraw(ctx context.Context, actor string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceWithdraw); err != nil {
		return domain.DatasetVersion{}, operationError("authorize withdrawal", err)
	}
	dataset, err := s.store.Withdraw(ctx, datasetID, versionID)
	return dataset, operationError("withdraw submission", err)
}

func (s *Service) Reject(ctx context.Context, actor string, datasetID, versionID uuid.UUID, value domain.RejectionInput) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceReject); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize rejection", err)
	}

	dataset, err := s.store.Reject(ctx, actor, datasetID, versionID, value)
	if err != nil {
		return domain.DatasetVersion{}, nil, operationError("reject submission", err)
	}

	notification, buildErr := s.notifier.Build(ctx, domain.ResourceRejected{
		ActorID:  actor,
		GroupID:  dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourceRejected, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func (s *Service) Reopen(ctx context.Context, actor string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceReopen); err != nil {
		return domain.DatasetVersion{}, operationError("authorize reopening", err)
	}
	dataset, err := s.store.Reopen(ctx, datasetID, versionID)
	return dataset, operationError("reopen rejected version", err)
}

//nolint:dupl // Resource lifecycle notification paths intentionally mirror each other.
func (s *Service) Publish(ctx context.Context, actor string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourcePublish); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize publication", err)
	}

	dataset, err := s.store.Publish(ctx, actor, datasetID, versionID)
	if err != nil {
		return domain.DatasetVersion{}, nil, operationError("publish version", err)
	}

	notification, buildErr := s.notifier.Build(ctx, domain.ResourcePublished{
		ActorID:  actor,
		GroupID:  dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourcePublished, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func (s *Service) Discard(ctx context.Context, actor string, datasetID, versionID uuid.UUID) ([]uuid.UUID, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceDiscard); err != nil {
		return nil, operationError("authorize discard", err)
	}

	removed, err := s.store.Discard(ctx, datasetID, versionID)
	return removed, operationError("discard version", err)
}

//nolint:dupl // Resource lifecycle notification paths intentionally mirror each other.
func (s *Service) Unpublish(ctx context.Context, actor string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, []domain.Notification, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceUnpublish); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize unpublication", err)
	}

	dataset, err := s.store.Unpublish(ctx, actor, datasetID, versionID)
	if err != nil {
		return domain.DatasetVersion{}, nil, operationError("unpublish version", err)
	}

	notification, buildErr := s.notifier.Build(ctx, domain.ResourceUnpublished{
		ActorID:  actor,
		GroupID:  dataset.Group.ID,
		Resource: datasetNotificationResource(dataset),
	})
	if buildErr != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventResourceUnpublished, "error", buildErr)
		notification = nil
	}

	return dataset, notification, nil
}

func datasetNotificationResource(value domain.DatasetVersion) domain.NotificationResource {
	return domain.NotificationResource{
		Type:          domain.Dataset,
		ResourceID:    value.DatasetID,
		VersionID:     value.VersionID,
		Revision:      value.Revision,
		Status:        value.Status,
		Name:          value.Metadata.Title,
		SubmittedBy:   value.SubmittedBy,
		RejectionNote: value.RejectionNote,
	}
}

func (s *Service) DraftFromPublication(ctx context.Context, actor string, datasetID uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize draft creation", err)
	}
	dataset, pairs, err := s.store.DraftFromPublication(ctx, actor, datasetID)
	return dataset, pairs, operationError("create draft from publication", err)
}

func (s *Service) DraftFromLatestArchive(ctx context.Context, actor string, datasetID uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error) {
	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize draft creation", err)
	}
	dataset, pairs, err := s.store.DraftFromLatestArchive(ctx, actor, datasetID)
	return dataset, pairs, operationError("create draft from archive", err)
}

func (s *Service) CopyPublished(ctx context.Context, actor string, sourceDatasetID, targetGroupID uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error) {
	if err := s.authorizeGroup(ctx, actor, targetGroupID, domain.GroupPermissionResourceCreate); err != nil {
		return domain.DatasetVersion{}, nil, operationError("authorize dataset copy", err)
	}
	dataset, pairs, err := s.store.CopyPublished(ctx, actor, sourceDatasetID, targetGroupID)
	return dataset, pairs, operationError("copy published dataset", err)
}

func (s *Service) authorizeGroup(ctx context.Context, actor string, groupID uuid.UUID, permission domain.GroupPermission) error {
	result, err := s.authorizer.AuthorizeGroup(ctx, actor, groupID, permission)
	if err != nil {
		return fmt.Errorf("authorize group: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", apperror.Default(apperror.CodePermissionDenied), result.Reason)
	}
	return nil
}

func (s *Service) authorizeResource(ctx context.Context, actor string, datasetID uuid.UUID, permission domain.GroupPermission) error {
	result, err := s.authorizer.AuthorizeResource(ctx, actor, domain.Dataset, datasetID, permission)
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
	return apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("%s: %w", operation, err))
}
