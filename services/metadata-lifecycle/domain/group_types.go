package domain

import "github.com/google/uuid"

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
