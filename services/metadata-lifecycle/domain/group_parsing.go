package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewTopLevelConcept validates and normalizes a top-level concept.
func NewTopLevelConcept(value string) (TopLevelConcept, error) {
	concept := TopLevelConcept(strings.ToUpper(strings.TrimSpace(value)))
	if !concept.Valid() {
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
	return concept, nil
}

// NewGroupInput validates and normalizes editable group fields.
func NewGroupInput(value GroupInput) (GroupInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Concept = strings.TrimSpace(value.Concept)
	value.Description = optionalTrimmedString(value.Description)
	value.Label = optionalTrimmedString(value.Label)
	value.Logo = optionalTrimmedString(value.Logo)
	value.Page = optionalTrimmedString(value.Page)
	value.Color = optionalTrimmedString(value.Color)
	if value.Name == "" {
		return GroupInput{}, invalidField("name", "Name must not be blank.", nil)
	}
	if value.Concept == "" {
		return GroupInput{}, invalidField("concept", "Concept is required.", nil)
	}
	concept, err := NewTopLevelConcept(value.Concept)
	if err != nil {
		return GroupInput{}, invalidField("concept", "Concept must be DATA, FACILITIES, or SOFTWARE.", nil)
	}
	value.Concept = string(concept)
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
		Concept:     value.Concept,
	}, nil
}
