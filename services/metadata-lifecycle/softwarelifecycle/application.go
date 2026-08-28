//nolint:dupl // Application and source-code lifecycles intentionally keep distinct domain types.
package softwarelifecycle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/google/uuid"
)

// Application is software application metadata with lifecycle identity.
type Application struct {
	ApplicationID uuid.UUID
	VersionID     uuid.UUID
	Revision      int32
	Status        metadata.DatasetStatus
	CreatedBy     string
	Group         metadata.CategoryScheme
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Metadata      metadata.SoftwareApplication
}

type ApplicationRepository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (Application, error)
	CreateDraft(context.Context, string, uuid.UUID, metadata.SoftwareApplication) (Application, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, int32, metadata.SoftwareApplication) (Application, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (Application, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID) (Application, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, string) (Application, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID) (Application, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (Application, error)
	Discard(context.Context, uuid.UUID, uuid.UUID) error
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (Application, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (Application, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (Application, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID) (Application, error)
}

type ApplicationService struct {
	authorizer ApplicationAuthorizer
	repository ApplicationRepository
}

// NewApplicationService creates a software application lifecycle service.
func NewApplicationService(authorizer ApplicationAuthorizer, repository ApplicationRepository) *ApplicationService {
	return &ApplicationService{authorizer: authorizer, repository: repository}
}

// CreateDraft creates a software application draft in groupID.
func (s *ApplicationService) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value metadata.SoftwareApplication) (Application, error) {
	if err := normalizeApplication(&value); err != nil {
		return Application{}, err
	}
	if err := authorizeGroup(ctx, s.authorizer, actorID, groupID, permissionCreate); err != nil {
		return Application{}, operationError("authorize application draft creation", err)
	}
	application, err := s.repository.CreateDraft(ctx, actorID, groupID, value)
	return application, operationError("create application draft", err)
}

// Read returns an authorized software application version and contact visibility.
func (s *ApplicationService) Read(ctx context.Context, actorID *string, applicationID, versionID uuid.UUID) (Application, bool, error) {
	result, err := s.authorizer.CanReadSoftwareApplicationVersion(ctx, actorID, applicationID, versionID)
	if err != nil {
		return Application{}, false, operationError("authorize application read", err)
	}
	if !result.Allowed {
		return Application{}, false, fmt.Errorf("%w: %s", ErrDenied, result.Reason)
	}
	includeContacts := false
	if actorID != nil {
		contactResult, err := s.authorizer.CanReadContactPoint(ctx, *actorID)
		if err != nil {
			return Application{}, false, operationError("authorize contact point read", err)
		}
		includeContacts = contactResult.Allowed
	}
	application, err := s.repository.Get(ctx, applicationID, versionID)
	return application, includeContacts, operationError("get application version", err)
}

// ReplaceDraft replaces a software application draft at revision.
func (s *ApplicationService) ReplaceDraft(ctx context.Context, actorID string, applicationID, versionID uuid.UUID, revision int32, value metadata.SoftwareApplication) (Application, error) {
	if revision < 1 {
		return Application{}, fmt.Errorf("revision must be positive: %w", ErrInvalid)
	}
	if err := normalizeApplication(&value); err != nil {
		return Application{}, err
	}
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionEdit); err != nil {
		return Application{}, operationError("authorize application draft replacement", err)
	}
	application, err := s.repository.ReplaceDraft(ctx, applicationID, versionID, revision, value)
	return application, operationError("replace application draft", err)
}

// Submit submits a software application draft for review.
func (s *ApplicationService) Submit(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (Application, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionSubmit); err != nil {
		return Application{}, operationError("authorize application submission", err)
	}
	application, err := s.repository.Submit(ctx, actorID, applicationID, versionID)
	return application, operationError("submit application draft", err)
}

// Withdraw withdraws a submitted software application version.
func (s *ApplicationService) Withdraw(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (Application, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionWithdraw); err != nil {
		return Application{}, operationError("authorize application withdrawal", err)
	}
	application, err := s.repository.Withdraw(ctx, applicationID, versionID)
	return application, operationError("withdraw application submission", err)
}

// Reject rejects a submitted software application version with note.
func (s *ApplicationService) Reject(ctx context.Context, actorID string, applicationID, versionID uuid.UUID, note string) (Application, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return Application{}, fmt.Errorf("rejection note is required: %w", ErrInvalid)
	}
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionReject); err != nil {
		return Application{}, operationError("authorize application rejection", err)
	}
	application, err := s.repository.Reject(ctx, actorID, applicationID, versionID, note)
	return application, operationError("reject application submission", err)
}

// Reopen reopens a rejected software application version.
func (s *ApplicationService) Reopen(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (Application, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionReopen); err != nil {
		return Application{}, operationError("authorize application reopening", err)
	}
	application, err := s.repository.Reopen(ctx, applicationID, versionID)
	return application, operationError("reopen rejected application version", err)
}

// Publish publishes a submitted software application version.
func (s *ApplicationService) Publish(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (Application, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionPublish); err != nil {
		return Application{}, operationError("authorize application publication", err)
	}
	application, err := s.repository.Publish(ctx, actorID, applicationID, versionID)
	return application, operationError("publish application version", err)
}

// Discard removes a software application draft or rejected version.
func (s *ApplicationService) Discard(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) error {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionDiscard); err != nil {
		return operationError("authorize application discard", err)
	}
	return operationError("discard application version", s.repository.Discard(ctx, applicationID, versionID))
}

// Unpublish archives a published software application version.
func (s *ApplicationService) Unpublish(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (Application, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionUnpublish); err != nil {
		return Application{}, operationError("authorize application unpublication", err)
	}
	application, err := s.repository.Unpublish(ctx, actorID, applicationID, versionID)
	return application, operationError("unpublish application version", err)
}

// DraftFromPublication creates a draft from published application metadata.
func (s *ApplicationService) DraftFromPublication(ctx context.Context, actorID string, applicationID uuid.UUID) (Application, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionCreate); err != nil {
		return Application{}, operationError("authorize application draft creation", err)
	}
	application, err := s.repository.DraftFromPublication(ctx, actorID, applicationID)
	return application, operationError("create application draft from publication", err)
}

// DraftFromLatestArchive creates a draft from the latest archived version.
func (s *ApplicationService) DraftFromLatestArchive(ctx context.Context, actorID string, applicationID uuid.UUID) (Application, error) {
	if err := authorizeResource(ctx, s.authorizer, actorID, domain.SoftwareApplication, applicationID, permissionCreate); err != nil {
		return Application{}, operationError("authorize application draft creation", err)
	}
	application, err := s.repository.DraftFromLatestArchive(ctx, actorID, applicationID)
	return application, operationError("create application draft from archive", err)
}

// CopyPublished copies published application metadata into targetGroupID.
func (s *ApplicationService) CopyPublished(ctx context.Context, actorID string, sourceApplicationID, targetGroupID uuid.UUID) (Application, error) {
	if err := authorizeGroup(ctx, s.authorizer, actorID, targetGroupID, permissionCreate); err != nil {
		return Application{}, operationError("authorize application copy", err)
	}
	application, err := s.repository.CopyPublished(ctx, actorID, sourceApplicationID, targetGroupID)
	return application, operationError("copy published application", err)
}

func normalizeApplication(value *metadata.SoftwareApplication) error {
	value.Name = strings.TrimSpace(value.Name)
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

func organizationIDs(values []metadata.Organization) []uuid.UUID {
	ids := make([]uuid.UUID, len(values))
	for i, value := range values {
		ids[i] = value.ID
	}
	return ids
}

func contactPointIDs(values []metadata.ContactPoint) []uuid.UUID {
	ids := make([]uuid.UUID, len(values))
	for i, value := range values {
		ids[i] = value.ID
	}
	return ids
}

func categoryIDs(values []metadata.Category) []uuid.UUID {
	ids := make([]uuid.UUID, len(values))
	for i, value := range values {
		ids[i] = value.ID
	}
	return ids
}
