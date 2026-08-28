package domain

import (
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

// DiscoveryFilter contains validated search criteria for resource discovery.
type DiscoveryFilter struct {
	Query              *string
	Types              []ResourceType
	SpatialFilter      *string
	DateFrom           *time.Time
	DateTo             *time.Time
	Providers          []uuid.UUID
	Keywords           []string
	VersioningStatuses []Status
}

// DiscoveryCandidate is the resource summary returned by persistence.
type DiscoveryCandidate struct {
	Type       ResourceType
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	GroupID    uuid.UUID
	Status     Status
	Name       string
}

// DiscoveryItem is a resource summary returned by discovery.
type DiscoveryItem struct {
	Type          ResourceType
	ResourceID    uuid.UUID
	VersionID     uuid.UUID
	Name          string
	Status        Status
	Group         DiscoveryReference
	Providers     []DiscoveryReference
	Creators      []DiscoveryReference
	Categories    []DiscoveryReference
	Distributions []DiscoveryReference
}

// DiscoveryReference identifies related metadata attached to a resource.
type DiscoveryReference struct {
	ID   uuid.UUID
	Name string
}
