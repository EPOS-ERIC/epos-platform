package domain

import "github.com/google/uuid"

// TODO: Add software input, draft, version, and complete parsing types when software migration enters scope.

// SoftwareApplicationMetadata is software application metadata used by imports.
type SoftwareApplicationMetadata struct {
	ID               uuid.UUID
	Name             string
	Description      string
	MainEntityOfPage *string
	License          *string
	Version          *string
	Keywords         string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	Creators         []OrganizationMetadata
	Identifiers      []Identifier
	ContactPoints    []ContactPoint
	Categories       []CategoryMetadata
}

// SoftwareSourceCodeMetadata is software source-code metadata used by imports.
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
