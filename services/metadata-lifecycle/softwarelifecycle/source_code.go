//nolint:dupl // Application and source-code lifecycles intentionally keep distinct domain types.
package softwarelifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/google/uuid"
)

// SourceCode is software source code metadata with lifecycle identity.
type SourceCode struct {
	SourceCodeID uuid.UUID
	VersionID    uuid.UUID
	Revision     int32
	Status       metadata.DatasetStatus
	CreatedBy    string
	Group        metadata.CategoryScheme
	Metadata     metadata.SoftwareSourceCode
}

type SourceCodeRepository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (SourceCode, error)
	CreateDraft(context.Context, string, uuid.UUID, metadata.SoftwareSourceCode) (SourceCode, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, int32, metadata.SoftwareSourceCode) (SourceCode, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (SourceCode, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID) (SourceCode, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, string) (SourceCode, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID) (SourceCode, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (SourceCode, error)
	Discard(context.Context, uuid.UUID, uuid.UUID) error
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (SourceCode, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (SourceCode, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (SourceCode, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID, string) (SourceCode, error)
}

type SourceCodeService struct {
	authorizer SourceCodeAuthorizer
	repository SourceCodeRepository
}

// NewSourceCodeService creates a software source code lifecycle service.
func NewSourceCodeService(authorizer SourceCodeAuthorizer, repository SourceCodeRepository) *SourceCodeService {
	return &SourceCodeService{authorizer: authorizer, repository: repository}
}

// CreateDraft creates a software source code draft in groupID.
func (s *SourceCodeService) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value metadata.SoftwareSourceCode) (SourceCode, error) {
	if err := normalizeSourceCode(&value); err != nil {
		return SourceCode{}, err
	}
	if err := authorizeGroup(ctx, s.authorizer, actorID, groupID, permissionCreate); err != nil {
		return SourceCode{}, operationError("authorize source code draft creation", err)
	}
	sourceCode, err := s.repository.CreateDraft(ctx, actorID, groupID, value)
	return sourceCode, operationError("create source code draft", err)
}

// Read returns an authorized source code version and contact visibility.
func (s *SourceCodeService) Read(ctx context.Context, actorID *string, sourceCodeID, versionID uuid.UUID) (SourceCode, bool, error) {
	result, err := s.authorizer.CanReadSoftwareSourceCodeVersion(ctx, actorID, sourceCodeID, versionID)
	if err != nil {
		return SourceCode{}, false, operationError("authorize source code read", err)
	}
	if !result.Allowed {
		return SourceCode{}, false, fmt.Errorf("%w: %s", ErrDenied, result.Reason)
	}
	includeContacts := false
	if actorID != nil {
		contactResult, err := s.authorizer.CanReadContactPoint(ctx, *actorID)
		if err != nil {
			return SourceCode{}, false, operationError("authorize contact point read", err)
		}
		includeContacts = contactResult.Allowed
	}
	sourceCode, err := s.repository.Get(ctx, sourceCodeID, versionID)
	return sourceCode, includeContacts, operationError("get source code version", err)
}

// ReplaceDraft replaces a source code draft at revision.
func (s *SourceCodeService) ReplaceDraft(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID, revision int32, value metadata.SoftwareSourceCode) (SourceCode, error) {
	if revision < 1 {
		return SourceCode{}, fmt.Errorf("revision must be positive: %w", ErrInvalid)
	}
	if err := normalizeSourceCode(&value); err != nil {
		return SourceCode{}, err
	}
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionEdit); err != nil {
		return SourceCode{}, operationError("authorize source code draft replacement", err)
	}
	sourceCode, err := s.repository.ReplaceDraft(ctx, sourceCodeID, versionID, revision, value)
	return sourceCode, operationError("replace source code draft", err)
}

// Submit submits a source code draft for review.
func (s *SourceCodeService) Submit(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (SourceCode, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionSubmit); err != nil {
		return SourceCode{}, operationError("authorize source code submission", err)
	}
	sourceCode, err := s.repository.Submit(ctx, actorID, sourceCodeID, versionID)
	return sourceCode, operationError("submit source code draft", err)
}

// Withdraw withdraws a submitted source code version.
func (s *SourceCodeService) Withdraw(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (SourceCode, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionWithdraw); err != nil {
		return SourceCode{}, operationError("authorize source code withdrawal", err)
	}
	sourceCode, err := s.repository.Withdraw(ctx, sourceCodeID, versionID)
	return sourceCode, operationError("withdraw source code submission", err)
}

// Reject rejects a submitted source code version with note.
func (s *SourceCodeService) Reject(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID, note string) (SourceCode, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return SourceCode{}, fmt.Errorf("rejection note is required: %w", ErrInvalid)
	}
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionReject); err != nil {
		return SourceCode{}, operationError("authorize source code rejection", err)
	}
	sourceCode, err := s.repository.Reject(ctx, actorID, sourceCodeID, versionID, note)
	return sourceCode, operationError("reject source code submission", err)
}

// Reopen reopens a rejected source code version.
func (s *SourceCodeService) Reopen(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (SourceCode, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionReopen); err != nil {
		return SourceCode{}, operationError("authorize source code reopening", err)
	}
	sourceCode, err := s.repository.Reopen(ctx, sourceCodeID, versionID)
	return sourceCode, operationError("reopen rejected source code version", err)
}

// Publish publishes a submitted source code version.
func (s *SourceCodeService) Publish(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (SourceCode, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionPublish); err != nil {
		return SourceCode{}, operationError("authorize source code publication", err)
	}
	sourceCode, err := s.repository.Publish(ctx, actorID, sourceCodeID, versionID)
	return sourceCode, operationError("publish source code version", err)
}

// Discard removes a source code draft or rejected version.
func (s *SourceCodeService) Discard(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) error {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionDiscard); err != nil {
		return operationError("authorize source code discard", err)
	}
	return operationError("discard source code version", s.repository.Discard(ctx, sourceCodeID, versionID))
}

// Unpublish archives a published source code version.
func (s *SourceCodeService) Unpublish(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (SourceCode, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionUnpublish); err != nil {
		return SourceCode{}, operationError("authorize source code unpublication", err)
	}
	sourceCode, err := s.repository.Unpublish(ctx, actorID, sourceCodeID, versionID)
	return sourceCode, operationError("unpublish source code version", err)
}

// DraftFromPublication creates a draft from published source code metadata.
func (s *SourceCodeService) DraftFromPublication(ctx context.Context, actorID string, sourceCodeID uuid.UUID) (SourceCode, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionCreate); err != nil {
		return SourceCode{}, operationError("authorize source code draft creation", err)
	}
	sourceCode, err := s.repository.DraftFromPublication(ctx, actorID, sourceCodeID)
	return sourceCode, operationError("create source code draft from publication", err)
}

// DraftFromLatestArchive creates a draft from the latest archived version.
func (s *SourceCodeService) DraftFromLatestArchive(ctx context.Context, actorID string, sourceCodeID uuid.UUID) (SourceCode, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, authz.ResourceTypeSoftwareSourceCode, sourceCodeID, permissionCreate); err != nil {
		return SourceCode{}, operationError("authorize source code draft creation", err)
	}
	sourceCode, err := s.repository.DraftFromLatestArchive(ctx, actorID, sourceCodeID)
	return sourceCode, operationError("create source code draft from archive", err)
}

// CopyPublished copies published source code metadata into targetGroupID.
func (s *SourceCodeService) CopyPublished(ctx context.Context, actorID string, sourceSourceCodeID, targetGroupID uuid.UUID, sourceIRI string) (SourceCode, error) {
	sourceIRI = strings.TrimSpace(sourceIRI)
	if sourceIRI == "" {
		return SourceCode{}, fmt.Errorf("source IRI is required: %w", ErrInvalid)
	}
	if err := authorizeGroup(ctx, s.authorizer, actorID, targetGroupID, permissionCreate); err != nil {
		return SourceCode{}, operationError("authorize source code copy", err)
	}
	sourceCode, err := s.repository.CopyPublished(ctx, actorID, sourceSourceCodeID, targetGroupID, sourceIRI)
	return sourceCode, operationError("copy published source code", err)
}

func normalizeSourceCode(value *metadata.SoftwareSourceCode) error {
	value.SourceIRI = strings.TrimSpace(value.SourceIRI)
	value.Name = strings.TrimSpace(value.Name)
	if value.SourceIRI == "" {
		return fmt.Errorf("source IRI is required: %w", ErrInvalid)
	}
	if value.Name == "" {
		return fmt.Errorf("name is required: %w", ErrInvalid)
	}
	if err := validateIdentifiers(value.Identifiers); err != nil {
		return err
	}
	if err := validateReferences("creator", organizationIDs(value.Creators)); err != nil {
		return err
	}
	if err := validateReferences("contact point", contactPointIDs(value.ContactPoints)); err != nil {
		return err
	}
	if err := validateReferences("category", categoryIDs(value.Categories)); err != nil {
		return err
	}
	return nil
}
