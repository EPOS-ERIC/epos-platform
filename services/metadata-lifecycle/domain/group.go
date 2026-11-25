package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

type ImportedConcept string // TODO: remove this when the real import is implemented

const (
	ImportedConceptData       ImportedConcept = "DATA"
	ImportedConceptFacilities ImportedConcept = "FACILITIES"
	ImportedConceptSoftware   ImportedConcept = "SOFTWARE"
)

func (c ImportedConcept) Valid() bool {
	switch c {
	case ImportedConceptData, ImportedConceptFacilities, ImportedConceptSoftware:
		return true
	default:
		return false
	}
}

// Group is complete public group metadata returned by the domain.
type Group struct {
	ID          uuid.UUID
	Name        string
	Description *string
	Label       *string
	Logo        *string
	Page        *string
	Color       *string
	Position    int32
	ConceptID   uuid.UUID
}

// GroupInput contains fields editable through group CRUD.
type GroupInput struct {
	Name        string
	Description *string
	Label       *string
	Logo        *string
	Page        *string
	Color       *string
	Position    int32
	ConceptID   uuid.UUID
}

// Normalize validates and normalizes editable group fields.
func (value *GroupInput) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	value.Description = optionalTrimmedString(value.Description)
	value.Label = optionalTrimmedString(value.Label)
	value.Logo = optionalTrimmedString(value.Logo)
	value.Page = optionalTrimmedString(value.Page)
	value.Color = optionalTrimmedString(value.Color)
	if value.Name == "" {
		return apperror.Field("name", "Name must not be blank.", nil)
	}
	if value.ConceptID == uuid.Nil {
		return apperror.Field("concept_id", "Concept ID is required.", nil)
	}
	return nil
}

// Normalize validates complete public group metadata.
func (value *Group) Normalize() error {
	if value.ID == uuid.Nil {
		return apperror.Default(apperror.CodeInternal)
	}
	input := GroupInput{
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		ConceptID:   value.ConceptID,
	}
	if err := input.Normalize(); err != nil {
		return internalValidationError(err)
	}
	value.Name = input.Name
	value.Description = input.Description
	value.Label = input.Label
	value.Logo = input.Logo
	value.Page = input.Page
	value.Color = input.Color
	return nil
}
