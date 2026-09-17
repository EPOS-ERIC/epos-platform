package domain

import (
	"strings"

	"github.com/google/uuid"
)

// NewGroupInput validates and normalizes editable group fields.
func NewGroupInput(value GroupInput) (GroupInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Concept = strings.TrimSpace(value.Concept)
	value.Description = optionalTrimmedString(value.Description)
	value.Label = optionalTrimmedString(value.Label)
	value.Logo = optionalTrimmedString(value.Logo)
	value.Page = optionalTrimmedString(value.Page)
	value.Color = optionalTrimmedString(value.Color)
	if value.Name == "" || value.Concept == "" {
		return GroupInput{}, ErrInvalidGroup
	}
	if value.Concept != "DATA" && value.Concept != "FACILITIES" && value.Concept != "SOFTWARE" {
		return GroupInput{}, ErrInvalidGroup
	}
	return value, nil
}

// NewGroup creates complete public group metadata from its identity and input.
func NewGroup(id uuid.UUID, value GroupInput) (Group, error) {
	if id == uuid.Nil {
		return Group{}, ErrInvalidGroup
	}
	value, err := NewGroupInput(value)
	if err != nil {
		return Group{}, err
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
