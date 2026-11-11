package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewGroupInput validates and normalizes editable group fields.
func NewGroupInput(value GroupInput) (GroupInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Description = optionalTrimmedString(value.Description)
	value.Label = optionalTrimmedString(value.Label)
	value.Logo = optionalTrimmedString(value.Logo)
	value.Page = optionalTrimmedString(value.Page)
	value.Color = optionalTrimmedString(value.Color)
	if value.Name == "" {
		return GroupInput{}, invalidField("name", "Name must not be blank.", nil)
	}
	if value.ConceptID == uuid.Nil {
		return GroupInput{}, invalidField("concept_id", "Concept ID is required.", nil)
	}
	return value, nil
}

// NewGroup creates complete public group metadata from its identity and input.
func NewGroup(id uuid.UUID, value GroupInput) (Group, error) {
	if id == uuid.Nil {
		return Group{}, apperror.Default(apperror.CodeValidationFailed)
	}
	value, err := NewGroupInput(value)
	if err != nil {
		return Group{}, internalValidationError(err)
	}
	return Group{
		ID:          id,
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		ConceptID:   value.ConceptID,
	}, nil
}
