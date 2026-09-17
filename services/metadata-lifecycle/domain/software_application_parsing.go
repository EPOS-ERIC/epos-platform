//nolint:dupl // Software application and source-code contracts intentionally mirror each other.
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewSoftwareApplicationInput validates and normalizes caller-controlled fields.
func NewSoftwareApplicationInput(value SoftwareApplicationInput) (SoftwareApplicationInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareApplicationInput{}, fmt.Errorf("software application name is required: %w", ErrSoftwareApplicationInvalid)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return SoftwareApplicationInput{}, fmt.Errorf("software application description is required: %w", ErrSoftwareApplicationInvalid)
	}
	value.MainEntityOfPage = optionalTrimmedString(value.MainEntityOfPage)
	value.License = optionalTrimmedString(value.License)
	value.Version = optionalTrimmedString(value.Version)
	value.DownloadURL = optionalTrimmedString(value.DownloadURL)
	value.Requirements = optionalTrimmedString(value.Requirements)
	value.OperatingSystem = optionalTrimmedString(value.OperatingSystem)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return SoftwareApplicationInput{}, fmt.Errorf("software application keywords: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	value.Keywords = keywords
	if err := validateUUIDList(value.CreatorIDs, "creator"); err != nil {
		return SoftwareApplicationInput{}, fmt.Errorf("software application creators: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	if err := validateUUIDList(value.ContactPointIDs, "contact point"); err != nil {
		return SoftwareApplicationInput{}, fmt.Errorf("software application contact points: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	if err := validateUUIDList(value.CategoryIDs, "category"); err != nil {
		return SoftwareApplicationInput{}, fmt.Errorf("software application categories: %w: %w", err, ErrSoftwareApplicationInvalid)
	}

	identifiers := make([]IdentifierInput, len(value.Identifiers))
	seen := make(map[string]struct{}, len(value.Identifiers))
	for i, identifier := range value.Identifiers {
		identifier, err := NewIdentifierInput(identifier)
		if err != nil {
			return SoftwareApplicationInput{}, fmt.Errorf("software application identifier: %w: %w", err, ErrSoftwareApplicationInvalid)
		}
		key := string(identifier.Type) + "\x00" + identifier.Value
		if _, exists := seen[key]; exists {
			return SoftwareApplicationInput{}, fmt.Errorf("duplicate software application identifier %q: %w", key, ErrSoftwareApplicationInvalid)
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
		return SoftwareApplicationDraftInput{}, fmt.Errorf("revision must be positive: %w", ErrSoftwareApplicationInvalid)
	}
	metadata, err := NewSoftwareApplicationInput(value.Metadata)
	if err != nil {
		return SoftwareApplicationDraftInput{}, err
	}
	value.Metadata = metadata
	return value, nil
}

// NewSoftwareApplicationRejectionInput validates and normalizes a rejection note.
func NewSoftwareApplicationRejectionInput(value SoftwareApplicationRejectionInput) (SoftwareApplicationRejectionInput, error) {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return SoftwareApplicationRejectionInput{}, fmt.Errorf("rejection note is required: %w", ErrSoftwareApplicationInvalid)
	}
	return value, nil
}

// NewSoftwareApplicationMetadata validates persisted or imported metadata.
func NewSoftwareApplicationMetadata(value SoftwareApplicationMetadata) (SoftwareApplicationMetadata, error) {
	if value.ID == uuid.Nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application metadata ID is required: %w", ErrSoftwareApplicationInvalid)
	}
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application name is required: %w", ErrSoftwareApplicationInvalid)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application description is required: %w", ErrSoftwareApplicationInvalid)
	}
	value.MainEntityOfPage = optionalTrimmedString(value.MainEntityOfPage)
	value.License = optionalTrimmedString(value.License)
	value.Version = optionalTrimmedString(value.Version)
	value.DownloadURL = optionalTrimmedString(value.DownloadURL)
	value.Requirements = optionalTrimmedString(value.Requirements)
	value.OperatingSystem = optionalTrimmedString(value.OperatingSystem)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application keywords: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	value.Keywords = keywords

	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.Identifiers))
	for _, identifier := range value.Identifiers {
		if err := validateIdentifier(identifier, true); err != nil {
			return SoftwareApplicationMetadata{}, fmt.Errorf("software application identifier: %w: %w", err, ErrSoftwareApplicationInvalid)
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return SoftwareApplicationMetadata{}, fmt.Errorf("duplicate software application identifier %q: %w", identifier.ID, ErrSoftwareApplicationInvalid)
		}
		seenIdentifiers[identifier.ID] = struct{}{}
	}
	if err := validateContactPoints(value.ContactPoints, "software application contact point"); err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application contact points: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	if err := validateOrganizations(value.Creators, "creator"); err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application creators: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return SoftwareApplicationMetadata{}, fmt.Errorf("software application categories: %w: %w", err, ErrSoftwareApplicationInvalid)
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
	group Group,
	createdAt, updatedAt time.Time,
	rejectionNote *string,
	metadata SoftwareApplicationMetadata,
) (SoftwareApplicationVersion, error) {
	if softwareApplicationID == uuid.Nil || versionID == uuid.Nil {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application and version IDs are required: %w", ErrSoftwareApplicationInvalid)
	}
	if revision < 1 {
		return SoftwareApplicationVersion{}, fmt.Errorf("revision must be positive: %w", ErrSoftwareApplicationInvalid)
	}
	if !status.Valid() {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application status %q is invalid: %w", status, ErrSoftwareApplicationInvalid)
	}
	actor, err := NewActor(string(createdBy))
	if err != nil {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application creator: %w: %w", err, ErrSoftwareApplicationInvalid)
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
		return SoftwareApplicationVersion{}, fmt.Errorf("software application group: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	metadata, err = NewSoftwareApplicationMetadata(metadata)
	if err != nil {
		return SoftwareApplicationVersion{}, err
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application timestamps are required: %w", ErrSoftwareApplicationInvalid)
	}
	rejectionNote, err = NewRejectionNote(status, rejectionNote)
	if err != nil {
		return SoftwareApplicationVersion{}, fmt.Errorf("software application rejection note: %w: %w", err, ErrSoftwareApplicationInvalid)
	}
	return SoftwareApplicationVersion{
		SoftwareApplicationID: softwareApplicationID,
		VersionID:             versionID,
		Revision:              revision,
		Status:                status,
		CreatedBy:             actor,
		Group:                 normalizedGroup,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
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
		identifiers[i] = IdentifierInput{Type: identifier.Type, Value: identifier.Value}
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
