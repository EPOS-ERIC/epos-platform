// Package domain contains metadata-lifecycle domain values shared by adapters.
package domain

import "github.com/google/uuid"

// Organization is a complete canonical organization returned by the domain.
type Organization struct {
	ID            uuid.UUID
	Name          string
	Street        *string
	Locality      *string
	PostalCode    *string
	Country       *string
	LogoURL       *string
	PageURL       *string
	Email         *string
	LEICode       *string
	Identifiers   []OrganizationIdentifier
	Parents       []uuid.UUID
	ContactPoints []uuid.UUID
}

// OrganizationIdentifier is a persisted organization identifier.
type OrganizationIdentifier struct {
	ID    uuid.UUID
	Type  string
	Value string
}

// OrganizationInput contains fields editable through organization CRUD.
type OrganizationInput struct {
	Name          string
	Street        *string
	Locality      *string
	PostalCode    *string
	Country       *string
	LogoURL       *string
	PageURL       *string
	Email         *string
	LEICode       *string
	Identifiers   []OrganizationIdentifierInput
	Parents       []uuid.UUID
	ContactPoints []uuid.UUID
}

// OrganizationIdentifierInput is an identifier supplied by a caller.
type OrganizationIdentifierInput struct {
	Type  string
	Value string
}
