package domain

import (
	"time"

	"github.com/google/uuid"
)

// SoftwareSourceCodeInput contains caller-controlled software source-code fields.
type SoftwareSourceCodeInput struct {
	Name                 string
	Description          string
	ProgrammingLanguage  *string
	Keywords             string
	Repository           *string
	License              *string
	SoftwareRequirements *string
	RuntimePlatform      *string
	MainEntityOfPage     *string
	Version              *string
	CreatorIDs           []uuid.UUID
	ContactPointIDs      []uuid.UUID
	CategoryIDs          []uuid.UUID
	Identifiers          []IdentifierInput
}

// SoftwareSourceCodeDraftInput contains a validated draft replacement command.
type SoftwareSourceCodeDraftInput struct {
	Revision int32
	Metadata SoftwareSourceCodeInput
}

// SoftwareSourceCodeRejectionInput contains a validated rejection command.
type SoftwareSourceCodeRejectionInput struct {
	Note string
}

// SoftwareSourceCodeMetadata is persisted or imported software source-code metadata.
type SoftwareSourceCodeMetadata struct {
	ID                   uuid.UUID
	Name                 string
	Description          string
	ProgrammingLanguage  *string
	Keywords             string
	Repository           *string
	License              *string
	SoftwareRequirements *string
	RuntimePlatform      *string
	MainEntityOfPage     *string
	Version              *string
	Creators             []OrganizationMetadata
	Identifiers          []Identifier
	ContactPoints        []ContactPoint
	Categories           []CategoryMetadata
}

// SoftwareSourceCodeVersion is software source-code metadata with lifecycle identity.
type SoftwareSourceCodeVersion struct {
	SoftwareSourceCodeID uuid.UUID
	VersionID            uuid.UUID
	Revision             int32
	Status               Status
	CreatedBy            Actor
	Group                Group
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Metadata             SoftwareSourceCodeMetadata
}
