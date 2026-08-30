package domain

import (
	"time"

	"github.com/google/uuid"
)

// DiscoveryFilter contains validated search criteria for resource discovery.
type DiscoveryFilter struct {
	Query              *string
	Types              []ResourceType
	Concepts           []TopLevelConcept
	GroupID            *uuid.UUID
	SpatialFilter      *SpatialFilter
	DateFrom           *time.Time
	DateTo             *time.Time
	Providers          []uuid.UUID
	Keywords           []string
	EditorID           *string
	VersioningStatuses []Status
}

// SpatialFilter contains a validated WGS84 geometry used for discovery.
type SpatialFilter struct {
	Point   *Coordinate
	Polygon []Coordinate
}

// DiscoveryCandidate is the resource summary returned by persistence.
type DiscoveryCandidate struct {
	Type       ResourceType
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	GroupID    uuid.UUID
	Status     Status
	Name       string
	UpdatedAt  time.Time
	Concept    TopLevelConcept
	CreatedBy  *DiscoveryCreatedBy
}

// DiscoveryItem is a resource summary returned by discovery.
type DiscoveryItem struct {
	Type          ResourceType
	ResourceID    uuid.UUID
	VersionID     uuid.UUID
	Name          string
	Status        Status
	UpdatedAt     time.Time
	Concept       TopLevelConcept
	CreatedBy     *DiscoveryCreatedBy
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

// DiscoveryCreatedBy identifies the user who created a resource version.
type DiscoveryCreatedBy struct {
	ID   string
	Name string
}
