package domain

import "time"

type DiscoveryFilter struct {
	Query              string
	Types              []string
	SpatialFilter      string
	DateFrom           *time.Time
	DateTo             *time.Time
	Provider           string
	Keywords           string
	VersioningStatuses []string
}

type DiscoveryItem struct {
	Type          string
	ResourceID    string
	VersionID     string
	Name          string
	Status        string
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
