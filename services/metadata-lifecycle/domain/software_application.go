//nolint:dupl // Software application and source-code contracts intentionally mirror each other.
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// SoftwareApplicationInput contains caller-controlled software application fields.
type SoftwareApplicationInput struct {
	Name             string
	Description      string
	MainEntityOfPage *string
	License          *string
	Version          *string
	Keywords         []string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	CreatorIDs       []uuid.UUID
	ContactPointIDs  []uuid.UUID
	CategoryIDs      []uuid.UUID
	Identifiers      []IdentifierInput
}

// SoftwareApplicationDraftInput contains a validated draft replacement command.
type SoftwareApplicationDraftInput struct {
	Revision int32
	Metadata SoftwareApplicationInput
}

// SoftwareApplicationRejectionInput contains a validated rejection command.
type SoftwareApplicationRejectionInput struct {
	Note string
}

// SoftwareApplicationMetadata is persisted or imported software application metadata.
type SoftwareApplicationMetadata struct {
	ID               uuid.UUID
	Name             string
	Description      string
	MainEntityOfPage *string
	License          *string
	Version          *string
	Keywords         []string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	Creators         []OrganizationMetadata
	Identifiers      []Identifier
	ContactPoints    []ContactPoint
	Categories       []CategoryMetadata
}

// SoftwareApplicationVersion is software application metadata with lifecycle identity.
type SoftwareApplicationVersion struct {
	SoftwareApplicationID uuid.UUID
	VersionID             uuid.UUID
	Revision              int32
	Status                Status
	CreatedBy             Actor
	SubmittedBy           *Actor
	Group                 Group
	CreatedAt             time.Time
	UpdatedAt             time.Time
	RejectedBy            *string
	RejectedAt            *time.Time
	RejectionNote         *string
	Metadata              SoftwareApplicationMetadata
}

// Normalize validates and normalizes caller-controlled fields.
func (value *SoftwareApplicationInput) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return apperror.Field("name", "Name must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return apperror.Field("description", "Description must not be blank.", nil)
	}
	if err := normalizeOptionalHTTPURL(value.MainEntityOfPage, "main_entity_of_page"); err != nil {
		return err
	}
	value.License = optionalTrimmedString(value.License)
	value.Version = optionalTrimmedString(value.Version)
	if err := normalizeOptionalHTTPURL(value.DownloadURL, "download_url"); err != nil {
		return err
	}
	value.Requirements = optionalTrimmedString(value.Requirements)
	value.OperatingSystem = optionalTrimmedString(value.OperatingSystem)
	keywords, err := normalizeKeywordInput(value.Keywords)
	if err != nil {
		return err
	}
	value.Keywords = keywords
	if err := validateUUIDListField(value.CreatorIDs, "creators", "Creator"); err != nil {
		return err
	}
	if err := validateUUIDListField(value.ContactPointIDs, "contact_points", "Contact point"); err != nil {
		return err
	}
	if err := validateUUIDListField(value.CategoryIDs, "categories", "Category"); err != nil {
		return err
	}

	identifiers, err := normalizeIdentifierInputs(value.Identifiers, "identifiers", "software application")
	if err != nil {
		return err
	}
	value.Identifiers = identifiers
	return nil
}

// Normalize validates a draft replacement command.
func (value *SoftwareApplicationDraftInput) Normalize() error {
	if value.Revision < 1 {
		return apperror.Field("revision", "Revision must be positive.", nil)
	}
	if err := value.Metadata.Normalize(); err != nil {
		return fmt.Errorf("validate software application draft metadata: %w", apperror.PrefixTarget(err, "metadata"))
	}
	return nil
}

// Normalize validates and normalizes a rejection note.
func (value *SoftwareApplicationRejectionInput) Normalize() error {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return apperror.Field("note", "Rejection note must not be blank.", nil)
	}
	return nil
}

// Normalize validates persisted or imported metadata.
func (value *SoftwareApplicationMetadata) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return fmt.Errorf("software application metadata ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return fmt.Errorf("software application name is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return fmt.Errorf("software application description is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if err := normalizeOptionalHTTPURL(value.MainEntityOfPage, "main_entity_of_page"); err != nil {
		return err
	}
	value.License = optionalTrimmedString(value.License)
	value.Version = optionalTrimmedString(value.Version)
	if err := normalizeOptionalHTTPURL(value.DownloadURL, "download_url"); err != nil {
		return err
	}
	value.Requirements = optionalTrimmedString(value.Requirements)
	value.OperatingSystem = optionalTrimmedString(value.OperatingSystem)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return fmt.Errorf("software application keywords: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.Keywords = keywords

	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.Identifiers))
	for _, identifier := range value.Identifiers {
		if err := validateIdentifier(identifier); err != nil {
			return fmt.Errorf("software application identifier: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return fmt.Errorf("duplicate software application identifier %q: %w", identifier.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seenIdentifiers[identifier.ID] = struct{}{}
	}
	if err := validateContactPoints(value.ContactPoints, "software application contact point"); err != nil {
		return fmt.Errorf("software application contact points: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateOrganizations(value.Creators, "creator"); err != nil {
		return fmt.Errorf("software application creators: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return fmt.Errorf("software application categories: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

// Normalize validates a complete persisted version.
//
//nolint:dupl // Resource version constructors intentionally share lifecycle validation.
func (value *SoftwareApplicationVersion) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.SoftwareApplicationID == uuid.Nil || value.VersionID == uuid.Nil {
		return fmt.Errorf("software application and version IDs are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Revision < 1 {
		return fmt.Errorf("revision must be positive: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !value.Status.Valid() {
		return fmt.Errorf("software application status %q is invalid: %w", value.Status, apperror.Default(apperror.CodeValidationFailed))
	}
	actor, err := NewActor(string(value.CreatedBy))
	if err != nil {
		return fmt.Errorf("software application creator: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.CreatedBy = actor
	if value.SubmittedBy != nil {
		submittedBy, err := NewActor(string(*value.SubmittedBy))
		if err != nil {
			return fmt.Errorf("software application submitter: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		value.SubmittedBy = &submittedBy
	}
	if err := value.Group.Normalize(); err != nil {
		return fmt.Errorf("software application group: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := value.Metadata.Normalize(); err != nil {
		return err
	}
	if value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		return fmt.Errorf("software application timestamps are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.RejectionNote, err = normalizeRejectionNote(value.Status, value.RejectionNote)
	if err != nil {
		return fmt.Errorf("software application rejection note: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}
