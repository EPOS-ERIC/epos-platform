package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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

type ResourceVersionSummary struct {
	ResourceType  ResourceType
	ResourceID    uuid.UUID
	VersionID     uuid.UUID
	Revision      int32
	Status        Status
	Name          string
	CreatedBy     Actor
	CreatedAt     time.Time
	UpdatedAt     time.Time
	SubmittedBy   *string
	SubmittedAt   *time.Time
	PublishedBy   *string
	PublishedAt   *time.Time
	ArchivedBy    *string
	ArchivedAt    *time.Time
	RejectedBy    *string
	RejectedAt    *time.Time
	RejectionNote *string
}

// NewRejectionNote normalizes and validates a rejection note for a version status.
func NewRejectionNote(status Status, note *string) (*string, error) {
	note = normalizedOptionalString(note)
	switch status {
	case Published, Archived:
		if note != nil {
			return nil, errors.New("rejection note is not allowed for published or archived resource")
		}
	case Rejected:
		if note == nil {
			return nil, errors.New("rejection note is required for rejected resource")
		}
	case Draft, Submitted:
		// No rejection-note constraint, it's optional here
	default:
		return nil, fmt.Errorf("invalid status when checking rejection note: %q", status)
	}
	return note, nil
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
