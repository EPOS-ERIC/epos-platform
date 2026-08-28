package domain

import "github.com/google/uuid"

// ContactPoint is a complete canonical contact point returned by the domain.
type ContactPoint struct {
	ID    uuid.UUID
	Email string
	Role  string
}

// ContactPointInput contains fields editable through contact-point CRUD.
type ContactPointInput struct {
	Email string
	Role  string
}
