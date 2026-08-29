package domain

import "github.com/google/uuid"

type ResourceType string

const (
	Dataset             ResourceType = "dataset"
	SoftwareApplication ResourceType = "software-application"
	SoftwareSourceCode  ResourceType = "software-source-code"
)

type Status string

const (
	Published Status = "PUBLISHED"
	Draft     Status = "DRAFT"
	Submitted Status = "SUBMITTED"
	Archived  Status = "ARCHIVED"
	Rejected  Status = "REJECTED"
)

func (s Status) Valid() bool {
	switch s {
	case Published, Draft, Submitted, Archived, Rejected:
		return true
	default:
		return false
	}
}

type IdentifierType string

type IdentifierInput struct {
	Type  IdentifierType
	Value string
}

type Identifier struct {
	ID    uuid.UUID
	Type  IdentifierType
	Value string
}

// OrganizationMetadata is embedded organization metadata used by resources.
type OrganizationMetadata struct {
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
	ContactPoints []ContactPoint
	Identifiers   []Identifier
}

// CategoryMetadata is embedded category metadata used by resources.
type CategoryMetadata struct {
	ID       uuid.UUID
	Name     string
	Scheme   Group
	Children []CategoryMetadata
}
