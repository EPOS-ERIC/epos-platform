//nolint:dupl // Software application and source-code contracts intentionally mirror each other.
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewSoftwareSourceCodeInput validates and normalizes caller-controlled fields.
func NewSoftwareSourceCodeInput(value SoftwareSourceCodeInput) (SoftwareSourceCodeInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareSourceCodeInput{}, fmt.Errorf("software source code name is required: %w", ErrSoftwareSourceCodeInvalid)
	}
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return SoftwareSourceCodeInput{}, fmt.Errorf("software source code keywords: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	value.Keywords = keywords
	if err := validateUUIDList(value.CreatorIDs, "creator"); err != nil {
		return SoftwareSourceCodeInput{}, fmt.Errorf("software source code creators: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	if err := validateUUIDList(value.ContactPointIDs, "contact point"); err != nil {
		return SoftwareSourceCodeInput{}, fmt.Errorf("software source code contact points: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	if err := validateUUIDList(value.CategoryIDs, "category"); err != nil {
		return SoftwareSourceCodeInput{}, fmt.Errorf("software source code categories: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}

	identifiers := make([]IdentifierInput, len(value.Identifiers))
	seen := make(map[string]struct{}, len(value.Identifiers))
	for i, identifier := range value.Identifiers {
		identifier, err := NewIdentifierInput(identifier)
		if err != nil {
			return SoftwareSourceCodeInput{}, fmt.Errorf("software source code identifier: %w: %w", err, ErrSoftwareSourceCodeInvalid)
		}
		key := string(identifier.Type) + "\x00" + identifier.Value
		if _, exists := seen[key]; exists {
			return SoftwareSourceCodeInput{}, fmt.Errorf("duplicate software source code identifier %q: %w", key, ErrSoftwareSourceCodeInvalid)
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
		return SoftwareSourceCodeDraftInput{}, fmt.Errorf("revision must be positive: %w", ErrSoftwareSourceCodeInvalid)
	}
	metadata, err := NewSoftwareSourceCodeInput(value.Metadata)
	if err != nil {
		return SoftwareSourceCodeDraftInput{}, err
	}
	value.Metadata = metadata
	return value, nil
}

// NewSoftwareSourceCodeRejectionInput validates and normalizes a rejection note.
func NewSoftwareSourceCodeRejectionInput(value SoftwareSourceCodeRejectionInput) (SoftwareSourceCodeRejectionInput, error) {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return SoftwareSourceCodeRejectionInput{}, fmt.Errorf("rejection note is required: %w", ErrSoftwareSourceCodeInvalid)
	}
	return value, nil
}

// NewSoftwareSourceCodeMetadata validates persisted or imported metadata.
func NewSoftwareSourceCodeMetadata(value SoftwareSourceCodeMetadata) (SoftwareSourceCodeMetadata, error) {
	if value.ID == uuid.Nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code metadata ID is required: %w", ErrSoftwareSourceCodeInvalid)
	}
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code name is required: %w", ErrSoftwareSourceCodeInvalid)
	}
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code keywords: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	value.Keywords = keywords

	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.Identifiers))
	for _, identifier := range value.Identifiers {
		if err := validateIdentifier(identifier, true); err != nil {
			return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code identifier: %w: %w", err, ErrSoftwareSourceCodeInvalid)
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return SoftwareSourceCodeMetadata{}, fmt.Errorf("duplicate software source code identifier %q: %w", identifier.ID, ErrSoftwareSourceCodeInvalid)
		}
		seenIdentifiers[identifier.ID] = struct{}{}
	}
	if err := validateContactPoints(value.ContactPoints, "software source code contact point"); err != nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code contact points: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	if err := validateOrganizations(value.Creators, "creator"); err != nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code creators: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return SoftwareSourceCodeMetadata{}, fmt.Errorf("software source code categories: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	return value, nil
}

// NewSoftwareSourceCodeVersion validates a complete persisted version.
func NewSoftwareSourceCodeVersion(
	softwareSourceCodeID, versionID uuid.UUID,
	revision int32,
	status Status,
	createdBy Actor,
	group Group,
	createdAt, updatedAt time.Time,
	metadata SoftwareSourceCodeMetadata,
) (SoftwareSourceCodeVersion, error) {
	if softwareSourceCodeID == uuid.Nil || versionID == uuid.Nil {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code and version IDs are required: %w", ErrSoftwareSourceCodeInvalid)
	}
	if revision < 1 {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("revision must be positive: %w", ErrSoftwareSourceCodeInvalid)
	}
	if !status.Valid() {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code status %q is invalid: %w", status, ErrSoftwareSourceCodeInvalid)
	}
	actor, err := NewActor(string(createdBy))
	if err != nil {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code creator: %w: %w", err, ErrSoftwareSourceCodeInvalid)
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
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code group: %w: %w", err, ErrSoftwareSourceCodeInvalid)
	}
	metadata, err = NewSoftwareSourceCodeMetadata(metadata)
	if err != nil {
		return SoftwareSourceCodeVersion{}, err
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		return SoftwareSourceCodeVersion{}, fmt.Errorf("software source code timestamps are required: %w", ErrSoftwareSourceCodeInvalid)
	}
	return SoftwareSourceCodeVersion{
		SoftwareSourceCodeID: softwareSourceCodeID,
		VersionID:            versionID,
		Revision:             revision,
		Status:               status,
		CreatedBy:            actor,
		Group:                normalizedGroup,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
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
