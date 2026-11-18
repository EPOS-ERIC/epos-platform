//nolint:dupl // Software application and source-code contracts intentionally mirror each other.
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// SoftwareSourceCodeInput contains caller-controlled software source-code fields.
type SoftwareSourceCodeInput struct {
	Name                 string
	Description          string
	ProgrammingLanguage  *string
	Keywords             []string
	Repository           *string
	License              *string
	SoftwareRequirements *string
	RuntimePlatform      *string
	MainEntityOfPage     *string
	Version              *string
	CreatorIDs           []uuid.UUID
	ContactPointIDs      []uuid.UUID
	CategoryIDs          []uuid.UUID
	Identifiers          []IdentifierInput
}

// SoftwareSourceCodeDraftInput contains a validated draft replacement command.
type SoftwareSourceCodeDraftInput struct {
	Revision int32
	Metadata SoftwareSourceCodeInput
}

// SoftwareSourceCodeRejectionInput contains a validated rejection command.
type SoftwareSourceCodeRejectionInput struct {
	Note string
}

// SoftwareSourceCodeMetadata is persisted or imported software source-code metadata.
type SoftwareSourceCodeMetadata struct {
	ID                   uuid.UUID
	Name                 string
	Description          string
	ProgrammingLanguage  *string
	Keywords             []string
	Repository           *string
	License              *string
	SoftwareRequirements *string
	RuntimePlatform      *string
	MainEntityOfPage     *string
	Version              *string
	Creators             []OrganizationMetadata
	Identifiers          []Identifier
	ContactPoints        []ContactPoint
	Categories           []CategoryMetadata
}

// SoftwareSourceCodeVersion is software source-code metadata with lifecycle identity.
type SoftwareSourceCodeVersion struct {
	SoftwareSourceCodeID uuid.UUID
	VersionID            uuid.UUID
	Revision             int32
	Status               Status
	CreatedBy            Actor
	SubmittedBy          *Actor
	Group                Group
	CreatedAt            time.Time
	UpdatedAt            time.Time
	RejectedBy           *string
	RejectedAt           *time.Time
	RejectionNote        *string
	Metadata             SoftwareSourceCodeMetadata
}

// Normalize validates and normalizes caller-controlled fields.
func (value *SoftwareSourceCodeInput) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return invalidField("name", "Name must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return invalidField("description", "Description must not be blank.", nil)
	}
	value.ProgrammingLanguage = optionalTrimmedString(value.ProgrammingLanguage)
	value.Repository = optionalTrimmedString(value.Repository)
	value.License = optionalTrimmedString(value.License)
	value.SoftwareRequirements = optionalTrimmedString(value.SoftwareRequirements)
	value.RuntimePlatform = optionalTrimmedString(value.RuntimePlatform)
	value.MainEntityOfPage = optionalTrimmedString(value.MainEntityOfPage)
	value.Version = optionalTrimmedString(value.Version)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return invalidField("keywords", "Keywords must not be blank and may contain only letters, numbers, spaces, colon, hyphen, and underscore.", err)
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

	identifiers, err := normalizeIdentifierInputs(value.Identifiers, "identifiers", "software source-code")
	if err != nil {
		return err
	}
	value.Identifiers = identifiers
	return nil
}

// Normalize validates a draft replacement command.
func (value *SoftwareSourceCodeDraftInput) Normalize() error {
	if value.Revision < 1 {
		return invalidField("revision", "Revision must be positive.", nil)
	}
	if err := value.Metadata.Normalize(); err != nil {
		return fmt.Errorf("validate software source-code draft metadata: %w", apperror.PrefixTarget(err, "metadata"))
	}
	return nil
}

// Normalize validates and normalizes a rejection note.
func (value *SoftwareSourceCodeRejectionInput) Normalize() error {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return invalidField("note", "Rejection note must not be blank.", nil)
	}
	return nil
}

// Normalize validates persisted or imported metadata.
func (value *SoftwareSourceCodeMetadata) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return fmt.Errorf("software source code metadata ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return fmt.Errorf("software source code name is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return fmt.Errorf("software source code description is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.ProgrammingLanguage = optionalTrimmedString(value.ProgrammingLanguage)
	value.Repository = optionalTrimmedString(value.Repository)
	value.License = optionalTrimmedString(value.License)
	value.SoftwareRequirements = optionalTrimmedString(value.SoftwareRequirements)
	value.RuntimePlatform = optionalTrimmedString(value.RuntimePlatform)
	value.MainEntityOfPage = optionalTrimmedString(value.MainEntityOfPage)
	value.Version = optionalTrimmedString(value.Version)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return fmt.Errorf("software source code keywords: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.Keywords = keywords

	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.Identifiers))
	for _, identifier := range value.Identifiers {
		if err := validateIdentifier(identifier); err != nil {
			return fmt.Errorf("software source code identifier: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return fmt.Errorf("duplicate software source code identifier %q: %w", identifier.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seenIdentifiers[identifier.ID] = struct{}{}
	}
	if err := validateContactPoints(value.ContactPoints, "software source code contact point"); err != nil {
		return fmt.Errorf("software source code contact points: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateOrganizations(value.Creators, "creator"); err != nil {
		return fmt.Errorf("software source code creators: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return fmt.Errorf("software source code categories: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

// Normalize validates a complete persisted version.
func (value *SoftwareSourceCodeVersion) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.SoftwareSourceCodeID == uuid.Nil || value.VersionID == uuid.Nil {
		return fmt.Errorf("software source code and version IDs are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Revision < 1 {
		return fmt.Errorf("revision must be positive: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !value.Status.Valid() {
		return fmt.Errorf("software source code status %q is invalid: %w", value.Status, apperror.Default(apperror.CodeValidationFailed))
	}
	actor, err := NewActor(string(value.CreatedBy))
	if err != nil {
		return fmt.Errorf("software source code creator: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.CreatedBy = actor
	if value.SubmittedBy != nil {
		submittedBy, err := NewActor(string(*value.SubmittedBy))
		if err != nil {
			return fmt.Errorf("software source code submitter: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		value.SubmittedBy = &submittedBy
	}
	if err := value.Group.Normalize(); err != nil {
		return fmt.Errorf("software source code group: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := value.Metadata.Normalize(); err != nil {
		return err
	}
	if value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		return fmt.Errorf("software source code timestamps are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.RejectionNote, err = normalizeRejectionNote(value.Status, value.RejectionNote)
	if err != nil {
		return fmt.Errorf("software source code rejection note: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}
