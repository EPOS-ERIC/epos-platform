package domain

import "time"

type DiscoveryFilter struct {
	Query              string
	Types              []string
	Concepts           []string
	GroupID            string
	SpatialFilter      string
	DateFrom           *time.Time
	DateTo             *time.Time
	Provider           string
	Keywords           string
	EditorID           string
	VersioningStatuses []string
}

type DiscoveryItem struct {
	Type       string
	ResourceID string
	VersionID  string
	Name       string
	Status     string
	UpdatedAt  time.Time
	Concept    string
	// TODO: creator should be string string but group and the other references should be uuid for the id and string for the name
	CreatedBy     *DiscoveryReference
	Group         DiscoveryReference
	Providers     []DiscoveryReference
	Creators      []DiscoveryReference
	Categories    []DiscoveryReference
	Distributions []DiscoveryReference
}

type DiscoveryReference struct {
	ID   string
	Name string
}
