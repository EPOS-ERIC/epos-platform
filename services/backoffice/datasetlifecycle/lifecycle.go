// Package datasetlifecycle manages dataset versions and their lifecycle.
package datasetlifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
)

var (
	ErrDenied   = errors.New("permission denied")
	ErrConflict = errors.New("lifecycle conflict")
	ErrNotFound = errors.New("dataset not found")
	ErrInvalid  = errors.New("invalid dataset request")
	ErrInternal = errors.New("dataset operation failed")
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

type Authorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (authz.Result, error)
	AuthorizeResource(context.Context, string, authz.ResourceType, uuid.UUID, string) (authz.Result, error)
	CanReadResource(context.Context, *string, authz.ResourceType, uuid.UUID) (authz.Result, error)
	CanReadDatasetVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error)
	CanReadContactPoint(context.Context, string) (authz.Result, error)
}

// Dataset is version metadata with lifecycle identity. Metadata.SourceIRI is
// logical dataset source IRI and is immutable after creation.
type Dataset struct {
	DatasetID uuid.UUID
	VersionID uuid.UUID
	Revision  int32
	Status    metadata.DatasetStatus
	CreatedBy string
	Group     metadata.CategoryScheme
	Metadata  metadata.Dataset
}

type Repository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (Dataset, error)
	GetPublished(context.Context, uuid.UUID) (Dataset, error)
	GetActive(context.Context, uuid.UUID) (Dataset, error)
	CreateDraft(context.Context, string, uuid.UUID, metadata.Dataset) (Dataset, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, int32, metadata.Dataset) (Dataset, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (Dataset, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID) (Dataset, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, string) (Dataset, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID) (Dataset, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (Dataset, error)
	Discard(context.Context, uuid.UUID, uuid.UUID) error
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (Dataset, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (Dataset, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (Dataset, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID, string) (Dataset, error)
}

type Service struct {
	authorizer Authorizer
	repository Repository
}

// New creates a dataset lifecycle service.
func New(authorizer Authorizer, repository Repository) *Service {
	return &Service{authorizer: authorizer, repository: repository}
}

// CreateDraft creates a dataset draft in groupID.
func (s *Service) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value metadata.Dataset) (Dataset, error) {
	if strings.TrimSpace(value.SourceIRI) == "" {
		return Dataset{}, fmt.Errorf("source IRI is required: %w", ErrInvalid)
	}
	if err := s.authorizeGroup(ctx, actorID, groupID, permissionCreate); err != nil {
		return Dataset{}, operationError("authorize draft creation", err)
	}
	value.SourceIRI = strings.TrimSpace(value.SourceIRI)
	dataset, err := s.repository.CreateDraft(ctx, actorID, groupID, value)
	if err != nil {
		return Dataset{}, operationError("create draft", err)
	}
	return dataset, nil
}

// ReplaceDraft replaces a dataset draft at revision.
func (s *Service) ReplaceDraft(ctx context.Context, actorID string, datasetID, versionID uuid.UUID, revision int32, value metadata.Dataset) (Dataset, error) {
	if revision < 1 {
		return Dataset{}, fmt.Errorf("revision must be positive: %w", ErrInvalid)
	}
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionEdit); err != nil {
		return Dataset{}, operationError("authorize draft replacement", err)
	}
	dataset, err := s.repository.ReplaceDraft(ctx, datasetID, versionID, revision, value)
	if err != nil {
		return Dataset{}, operationError("replace draft", err)
	}
	return dataset, nil
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrDenied) || errors.Is(err, ErrConflict) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalid) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, ErrInternal, err)
}

// Read returns an authorized dataset version and contact visibility.
func (s *Service) Read(ctx context.Context, actorID *string, datasetID, versionID uuid.UUID) (Dataset, bool, error) {
	result, err := s.authorizer.CanReadDatasetVersion(ctx, actorID, datasetID, versionID)
	if err != nil {
		return Dataset{}, false, operationError("authorize dataset read", err)
	}
	if !result.Allowed {
		return Dataset{}, false, fmt.Errorf("%w: %s", ErrDenied, result.Reason)
	}

	includeContacts := false
	if actorID != nil {
		contactResult, err := s.authorizer.CanReadContactPoint(ctx, *actorID)
		if err != nil {
			return Dataset{}, false, operationError("authorize contact point read", err)
		}
		includeContacts = contactResult.Allowed
	}

	dataset, err := s.repository.Get(ctx, datasetID, versionID)
	if err != nil {
		return Dataset{}, false, operationError("get dataset version", err)
	}
	return dataset, includeContacts, nil
}

// Submit submits a dataset draft for review.
func (s *Service) Submit(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (Dataset, error) {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionSubmit); err != nil {
		return Dataset{}, operationError("authorize submission", err)
	}
	dataset, err := s.repository.Submit(ctx, actorID, datasetID, versionID)
	return dataset, operationError("submit draft", err)
}

// Withdraw withdraws a submitted dataset version.
func (s *Service) Withdraw(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (Dataset, error) {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionWithdraw); err != nil {
		return Dataset{}, operationError("authorize withdrawal", err)
	}
	dataset, err := s.repository.Withdraw(ctx, datasetID, versionID)
	return dataset, operationError("withdraw submission", err)
}

// Reject rejects a submitted dataset version with note.
func (s *Service) Reject(ctx context.Context, actorID string, datasetID, versionID uuid.UUID, note string) (Dataset, error) {
	if strings.TrimSpace(note) == "" {
		return Dataset{}, fmt.Errorf("rejection note is required: %w", ErrInvalid)
	}
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionReject); err != nil {
		return Dataset{}, operationError("authorize rejection", err)
	}
	dataset, err := s.repository.Reject(ctx, actorID, datasetID, versionID, strings.TrimSpace(note))
	return dataset, operationError("reject submission", err)
}

// Reopen reopens a rejected dataset version.
func (s *Service) Reopen(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (Dataset, error) {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionReopen); err != nil {
		return Dataset{}, operationError("authorize reopening", err)
	}
	dataset, err := s.repository.Reopen(ctx, datasetID, versionID)
	return dataset, operationError("reopen rejected version", err)
}

// Publish publishes a submitted dataset version.
func (s *Service) Publish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (Dataset, error) {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionPublish); err != nil {
		return Dataset{}, operationError("authorize publication", err)
	}
	dataset, err := s.repository.Publish(ctx, actorID, datasetID, versionID)
	return dataset, operationError("publish version", err)
}

// Discard removes a dataset draft or rejected version.
func (s *Service) Discard(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) error {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionDiscard); err != nil {
		return operationError("authorize discard", err)
	}
	return operationError("discard version", s.repository.Discard(ctx, datasetID, versionID))
}

// Unpublish archives a published dataset version.
func (s *Service) Unpublish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (Dataset, error) {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionUnpublish); err != nil {
		return Dataset{}, operationError("authorize unpublication", err)
	}
	dataset, err := s.repository.Unpublish(ctx, actorID, datasetID, versionID)
	return dataset, operationError("unpublish version", err)
}

// DraftFromPublication creates a draft from published dataset metadata.
func (s *Service) DraftFromPublication(ctx context.Context, actorID string, datasetID uuid.UUID) (Dataset, error) {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionCreate); err != nil {
		return Dataset{}, operationError("authorize draft creation", err)
	}
	dataset, err := s.repository.DraftFromPublication(ctx, actorID, datasetID)
	return dataset, operationError("create draft from publication", err)
}

// DraftFromLatestArchive creates a draft from the latest archived version.
func (s *Service) DraftFromLatestArchive(ctx context.Context, actorID string, datasetID uuid.UUID) (Dataset, error) {
	if err := s.authorizeResource(ctx, actorID, datasetID, permissionCreate); err != nil {
		return Dataset{}, operationError("authorize draft creation", err)
	}
	dataset, err := s.repository.DraftFromLatestArchive(ctx, actorID, datasetID)
	return dataset, operationError("create draft from archive", err)
}

// CopyPublished copies published dataset metadata into targetGroupID.
func (s *Service) CopyPublished(ctx context.Context, actorID string, sourceDatasetID, targetGroupID uuid.UUID, sourceIRI string) (Dataset, error) {
	if strings.TrimSpace(sourceIRI) == "" {
		return Dataset{}, fmt.Errorf("source IRI is required: %w", ErrInvalid)
	}
	if err := s.authorizeGroup(ctx, actorID, targetGroupID, permissionCreate); err != nil {
		return Dataset{}, operationError("authorize dataset copy", err)
	}
	dataset, err := s.repository.CopyPublished(ctx, actorID, sourceDatasetID, targetGroupID, strings.TrimSpace(sourceIRI))
	return dataset, operationError("copy published dataset", err)
}

func (s *Service) authorizeGroup(ctx context.Context, actorID string, groupID uuid.UUID, permission string) error {
	result, err := s.authorizer.AuthorizeGroup(ctx, actorID, groupID, permission)
	if err != nil {
		return fmt.Errorf("authorize group: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", ErrDenied, result.Reason)
	}
	return nil
}

func (s *Service) authorizeResource(ctx context.Context, actorID string, datasetID uuid.UUID, permission string) error {
	result, err := s.authorizer.AuthorizeResource(ctx, actorID, authz.ResourceTypeDataset, datasetID, permission)
	if err != nil {
		return fmt.Errorf("authorize dataset: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf("%w: %s", ErrDenied, result.Reason)
	}
	return nil
}
