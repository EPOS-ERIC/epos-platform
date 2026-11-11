//nolint:dupl // Software application and source-code contracts intentionally mirror each other.
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewSoftwareApplicationInput validates and normalizes caller-controlled fields.
func NewSoftwareApplicationInput(value SoftwareApplicationInput) (SoftwareApplicationInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareApplicationInput{}, invalidField("name", "Name must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return SoftwareApplicationInput{}, invalidField("description", "Description must not be blank.", nil)
	}
	value.MainEntityOfPage = optionalTrimmedString(value.MainEntityOfPage)
	value.License = optionalTrimmedString(value.License)
	value.Version = optionalTrimmedString(value.Version)
	value.DownloadURL = optionalTrimmedString(value.DownloadURL)
	value.Requirements = optionalTrimmedString(value.Requirements)
	value.OperatingSystem = optionalTrimmedString(value.OperatingSystem)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return SoftwareApplicationInput{}, invalidField("keywords", "Keywords must not be blank and may contain only letters, numbers, spaces, colon, hyphen, and underscore.", err)
	}
	value.Keywords = keywords
	if err := validateUUIDListField(value.CreatorIDs, "creators", "Creator"); err != nil {
		return SoftwareApplicationInput{}, err
	}
	if err := validateUUIDListField(value.ContactPointIDs, "contact_points", "Contact point"); err != nil {
		return SoftwareApplicationInput{}, err
	}
	if err := validateUUIDListField(value.CategoryIDs, "categories", "Category"); err != nil {
		return SoftwareApplicationInput{}, err
	}

	identifiers := make([]IdentifierInput, len(value.Identifiers))
	seen := make(map[string]struct{}, len(value.Identifiers))
	for i, identifier := range value.Identifiers {
		identifier, err := NewIdentifierInput(identifier)
		if err != nil {
			return SoftwareApplicationInput{}, fmt.Errorf("validate software application identifier: %w", apperror.PrefixTarget(err, fmt.Sprintf("identifiers[%d]", i)))
		}
		key := string(identifier.Type) + "\x00" + identifier.Value
		if _, exists := seen[key]; exists {
			return SoftwareApplicationInput{}, invalidField(fmt.Sprintf("identifiers[%d]", i), "Identifier must be unique.", nil)
		}
		seen[key] = struct{}{}
		identifiers[i] = identifier
	}
	value.Identifiers = identifiers
	return value, nil
}

// NewSoftwareApplicationDraftInput validates a draft replacement command.
func NewSoftwareApplicationDraftInput(value SoftwareApplicationDraftInput) (SoftwareApplicationDraftInput, error) {
	if value.Revision < 1 {
		return SoftwareApplicationDraftInput{}, invalidField("revision", "Revision must be positive.", nil)
	}
	metadata, err := NewSoftwareApplicationInput(value.Metadata)
	if err != nil {
		return SoftwareApplicationDraftInput{}, fmt.Errorf("validate software application draft metadata: %w", apperror.PrefixTarget(err, "metadata"))
	}
	value.Metadata = metadata
	return value, nil
}

// NewSoftwareApplicationRejectionInput validates and normalizes a rejection note.
func NewSoftwareApplicationRejectionInput(value SoftwareApplicationRejectionInput) (SoftwareApplicationRejectionInput, error) {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return SoftwareApplicationRejectionInput{}, invalidField("note", "Rejection note must not be blank.", nil)
	}
	return value, nil
}

// NewSoftwareApplicationMetadata validates persisted or imported metadata.
func NewSoftwareApplicationMetadata(value SoftwareApplicationMetadata) (result SoftwareApplicationMetadata, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application metadata ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application name is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application description is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.MainEntityOfPage = optionalTrimmedString(value.MainEntityOfPage)
	value.License = optionalTrimmedString(value.License)
	value.Version = optionalTrimmedString(value.Version)
	value.DownloadURL = optionalTrimmedString(value.DownloadURL)
	value.Requirements = optionalTrimmedString(value.Requirements)
	value.OperatingSystem = optionalTrimmedString(value.OperatingSystem)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application keywords: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.Keywords = keywords

	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.Identifiers))
	for _, identifier := range value.Identifiers {
		if err := validateIdentifier(identifier, true); err != nil {
			return SoftwareApplicationMetadata{}, fmt.Errorf("software application identifier: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return SoftwareApplicationMetadata{}, fmt.Errorf("duplicate software application identifier %q: %w", identifier.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seenIdentifiers[identifier.ID] = struct{}{}
	}
	if err := validateContactPoints(value.ContactPoints, "software application contact point"); err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application contact points: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateOrganizations(value.Creators, "creator"); err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application creators: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application categories: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return value, nil
}

// NewSoftwareApplicationVersion validates a complete persisted version.
//
//nolint:dupl // Resource version constructors intentionally share lifecycle validation.
func NewSoftwareApplicationVersion(
	softwareApplicationID, versionID uuid.UUID,
	revision int32,
	status Status,
	createdBy Actor,
	submittedBy *string,
	group Group,
	createdAt, updatedAt time.Time,
	rejectedBy *string,
	rejectedAt *time.Time,
	rejectionNote *string,
	metadata SoftwareApplicationMetadata,
) (result SoftwareApplicationVersion, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if softwareApplicationID == uuid.Nil || versionID == uuid.Nil {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application and version IDs are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if revision < 1 {
		return SoftwareApplicationVersion{}, fmt.Errorf("revision must be positive: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !status.Valid() {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application status %q is invalid: %w", status, apperror.Default(apperror.CodeValidationFailed))
	}
	actor, err := NewActor(string(createdBy))
	if err != nil {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application creator: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	var submittedByActor *Actor
	if submittedBy != nil {
		value, err := NewActor(*submittedBy)
		if err != nil {
			return SoftwareApplicationVersion{}, fmt.Errorf("software application submitter: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
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
		ConceptID:   group.ConceptID,
	})
	if err != nil {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application group: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	metadata, err = NewSoftwareApplicationMetadata(metadata)
	if err != nil {
		return SoftwareApplicationVersion{}, err
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application timestamps are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	rejectionNote, err = NewRejectionNote(status, rejectionNote)
	if err != nil {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application rejection note: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return SoftwareApplicationVersion{
		SoftwareApplicationID: softwareApplicationID,
		VersionID:             versionID,
		Revision:              revision,
		Status:                status,
		CreatedBy:             actor,
		SubmittedBy:           submittedByActor,
		Group:                 normalizedGroup,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
		RejectedBy:            rejectedBy,
		RejectedAt:            rejectedAt,
		RejectionNote:         rejectionNote,
		Metadata:              metadata,
	}, nil
}

// Validate validates persisted or imported software application metadata.
// TODO: delete this and make the callers instead just create their object using the correct parsing functions
func (value SoftwareApplicationMetadata) Validate() error {
	_, err := NewSoftwareApplicationMetadata(value)
	return err
}

// Input returns caller-shaped fields from persisted or imported metadata.
func (value SoftwareApplicationMetadata) Input() SoftwareApplicationInput {
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
	return SoftwareApplicationInput{
		Name:             value.Name,
		Description:      value.Description,
		MainEntityOfPage: value.MainEntityOfPage,
		License:          value.License,
		Version:          value.Version,
		Keywords:         value.Keywords,
		DownloadURL:      value.DownloadURL,
		Requirements:     value.Requirements,
		OperatingSystem:  value.OperatingSystem,
		CreatorIDs:       creatorIDs,
		ContactPointIDs:  contactPointIDs,
		CategoryIDs:      categoryIDs,
		Identifiers:      identifiers,
	}
}
