package domain

import (
	"time"

	"github.com/google/uuid"
)

// SoftwareApplicationInput contains caller-controlled software application fields.
type SoftwareApplicationInput struct {
	Name             string
	Description      string
	MainEntityOfPage *string
	License          *string
	Version          *string
	Keywords         []string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	CreatorIDs       []uuid.UUID
	ContactPointIDs  []uuid.UUID
	CategoryIDs      []uuid.UUID
	Identifiers      []IdentifierInput
}

// SoftwareApplicationDraftInput contains a validated draft replacement command.
type SoftwareApplicationDraftInput struct {
	Revision int32
	Metadata SoftwareApplicationInput
}

// SoftwareApplicationRejectionInput contains a validated rejection command.
type SoftwareApplicationRejectionInput struct {
	Note string
}

// SoftwareApplicationMetadata is persisted or imported software application metadata.
type SoftwareApplicationMetadata struct {
	ID               uuid.UUID
	Name             string
	Description      string
	MainEntityOfPage *string
	License          *string
	Version          *string
	Keywords         []string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	Creators         []OrganizationMetadata
	Identifiers      []Identifier
	ContactPoints    []ContactPoint
	Categories       []CategoryMetadata
}

// SoftwareApplicationVersion is software application metadata with lifecycle identity.
type SoftwareApplicationVersion struct {
	SoftwareApplicationID uuid.UUID
	VersionID             uuid.UUID
	Revision              int32
	Status                Status
	CreatedBy             Actor
	Group                 Group
	CreatedAt             time.Time
	UpdatedAt             time.Time
	RejectionNote         *string
	Metadata              SoftwareApplicationMetadata
}
