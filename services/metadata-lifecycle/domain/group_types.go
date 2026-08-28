package domain

import "github.com/google/uuid"

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
	Concept     string
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
	Concept     string
}
