//nolint:dupl // Software application and source-code contracts intentionally mirror each other.
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewSoftwareSourceCodeInput validates and normalizes caller-controlled fields.
func NewSoftwareSourceCodeInput(value SoftwareSourceCodeInput) (SoftwareSourceCodeInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareSourceCodeInput{}, invalidField("name", "Name must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return SoftwareSourceCodeInput{}, invalidField("description", "Description must not be blank.", nil)
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
		return SoftwareSourceCodeInput{}, invalidField("keywords", "Keywords must not be blank and may contain only letters, numbers, spaces, colon, hyphen, and underscore.", err)
	}
	value.Keywords = keywords
	if err := validateUUIDListField(value.CreatorIDs, "creators", "Creator"); err != nil {
		return SoftwareSourceCodeInput{}, err
	}
	if err := validateUUIDListField(value.ContactPointIDs, "contact_points", "Contact point"); err != nil {
		return SoftwareSourceCodeInput{}, err
	}
	if err := validateUUIDListField(value.CategoryIDs, "categories", "Category"); err != nil {
		return SoftwareSourceCodeInput{}, err
	}

	identifiers := make([]IdentifierInput, len(value.Identifiers))
	seen := make(map[string]struct{}, len(value.Identifiers))
	for i, identifier := range value.Identifiers {
		identifier, err := NewIdentifierInput(identifier)
		if err != nil {
			return SoftwareSourceCodeInput{}, fmt.Errorf("validate software source-code identifier: %w", apperror.PrefixTarget(err, fmt.Sprintf("identifiers[%d]", i)))
		}
		key := string(identifier.Type) + "\x00" + identifier.Value
		if _, exists := seen[key]; exists {
			return SoftwareSourceCodeInput{}, invalidField(fmt.Sprintf("identifiers[%d]", i), "Identifier must be unique.", nil)
		}
		seen[key] = struct{}{}
		identifiers[i] = identifier
	}
	value.Identifiers = identifiers
	return value, nil
}

// NewSoftwareSourceCodeDraftInput validates a draft replacement command.
func NewSoftwareSourceCodeDraftInput(value SoftwareSourceCodeDraftInput) (SoftwareSourceCodeDraftInput, error) {
	if value.Revision < 1 {
		return SoftwareSourceCodeDraftInput{}, invalidField("revision", "Revision must be positive.", nil)
	}
	metadata, err := NewSoftwareSourceCodeInput(value.Metadata)
	if err != nil {
		return SoftwareSourceCodeDraftInput{}, fmt.Errorf("validate software source-code draft metadata: %w", apperror.PrefixTarget(err, "metadata"))
	}
	value.Metadata = metadata
	return value, nil
}

// NewSoftwareSourceCodeRejectionInput validates and normalizes a rejection note.
func NewSoftwareSourceCodeRejectionInput(value SoftwareSourceCodeRejectionInput) (SoftwareSourceCodeRejectionInput, error) {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return SoftwareSourceCodeRejectionInput{}, invalidField("note", "Rejection note must not be blank.", nil)
	}
	return value, nil
}

// NewSoftwareSourceCodeMetadata validates persisted or imported metadata.
func NewSoftwareSourceCodeMetadata(value SoftwareSourceCodeMetadata) (result SoftwareSourceCodeMetadata, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code metadata ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code name is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code description is required: %w", apperror.Default(apperror.CodeValidationFailed))
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
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code keywords: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.Keywords = keywords

	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.Identifiers))
	for _, identifier := range value.Identifiers {
		if err := validateIdentifier(identifier, true); err != nil {
			return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code identifier: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return SoftwareSourceCodeMetadata{}, fmt.Errorf("duplicate software source code identifier %q: %w", identifier.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seenIdentifiers[identifier.ID] = struct{}{}
	}
	if err := validateContactPoints(value.ContactPoints, "software source code contact point"); err != nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code contact points: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateOrganizations(value.Creators, "creator"); err != nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code creators: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code categories: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return value, nil
}

// NewSoftwareSourceCodeVersion validates a complete persisted version.
func NewSoftwareSourceCodeVersion(
	softwareSourceCodeID, versionID uuid.UUID,
	revision int32,
	status Status,
	createdBy Actor,
	submittedBy *string,
	group Group,
	createdAt, updatedAt time.Time,
	rejectedBy *string,
	rejectedAt *time.Time,
	rejectionNote *string,
	metadata SoftwareSourceCodeMetadata,
) (result SoftwareSourceCodeVersion, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if softwareSourceCodeID == uuid.Nil || versionID == uuid.Nil {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code and version IDs are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if revision < 1 {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("revision must be positive: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !status.Valid() {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code status %q is invalid: %w", status, apperror.Default(apperror.CodeValidationFailed))
	}
	actor, err := NewActor(string(createdBy))
	if err != nil {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code creator: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	var submittedByActor *Actor
	if submittedBy != nil {
		value, err := NewActor(*submittedBy)
		if err != nil {
			return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code submitter: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		submittedByActor = &value
	}
	normalizedGroup, err := NewGroup(group.ID, GroupInput{
		Name:        group.Name,
		Description: group.Description,
		Label:       group.Label,
		Logo:        group.Logo,
		Page:        group.Page,
		Color:       group.Color,
		Position:    group.Position,
		Concept:     group.Concept,
	})
	if err != nil {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code group: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	metadata, err = NewSoftwareSourceCodeMetadata(metadata)
	if err != nil {
		return SoftwareSourceCodeVersion{}, err
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code timestamps are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	rejectionNote, err = NewRejectionNote(status, rejectionNote)
	if err != nil {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code rejection note: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return SoftwareSourceCodeVersion{
		SoftwareSourceCodeID: softwareSourceCodeID,
		VersionID:            versionID,
		Revision:             revision,
		Status:               status,
		CreatedBy:            actor,
		SubmittedBy:          submittedByActor,
		Group:                normalizedGroup,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
		RejectedBy:           rejectedBy,
		RejectedAt:           rejectedAt,
		RejectionNote:        rejectionNote,
		Metadata:             metadata,
	}, nil
}

// Validate validates persisted or imported software source-code metadata.
// TODO: delete this and make the callers instead just create their object using the correct parsing functions
func (value SoftwareSourceCodeMetadata) Validate() error {
	_, err := NewSoftwareSourceCodeMetadata(value)
	return err
}

// Input returns caller-shaped fields from persisted or imported metadata.
func (value SoftwareSourceCodeMetadata) Input() SoftwareSourceCodeInput {
	identifiers := make([]IdentifierInput, len(value.Identifiers))
	for i, identifier := range value.Identifiers {
		identifiers[i] = IdentifierInput{
			ResourceID: &identifier.ResourceID,
			Type:       identifier.Type,
			Value:      identifier.Value,
		}
	}
	creatorIDs := make([]uuid.UUID, len(value.Creators))
	for i, creator := range value.Creators {
		creatorIDs[i] = creator.ID
	}
	contactPointIDs := make([]uuid.UUID, len(value.ContactPoints))
	for i, contact := range value.ContactPoints {
		contactPointIDs[i] = contact.ID
	}
	categoryIDs := make([]uuid.UUID, len(value.Categories))
	for i, category := range value.Categories {
		categoryIDs[i] = category.ID
	}
	return SoftwareSourceCodeInput{
		Name:                 value.Name,
		Description:          value.Description,
		ProgrammingLanguage:  value.ProgrammingLanguage,
		Keywords:             value.Keywords,
		Repository:           value.Repository,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
		CreatorIDs:           creatorIDs,
		ContactPointIDs:      contactPointIDs,
		CategoryIDs:          categoryIDs,
		Identifiers:          identifiers,
	}
}
