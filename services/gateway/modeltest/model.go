package main

import "time"

type Dataset struct {
	UID string

	Title                string
	Description          string
	Distributions        []Distribution
	SpatialCoverage      *Spatial
	TemporalCoverage     *Temporal
	PersistentIdentifier *[]Identifier
	ContactPoints        []ContactPoint
	Providers            []Organization
	Categories           []string

	// could be a separate struct?
	Status          string // could be an enum
	Editor          string
	Group           string
	Created         time.Time
	Updated         time.Time
	OriginalDataset *string // if this is not created from scratch it will have the id of the dataset from which we drafted, otherwise the same as the uid
}

type Distribution struct {
	Title            string
	Description      string
	WebService       *WebService
	License          string
	DownloadableFile *DownloadableFile
}

type DownloadableFile struct {
	URL    string
	Format string
}

type WebService struct {
	Name             string
	Description      string
	TemplateURL      string
	Parameters       []Parameter
	Format           string
	Providers        []Organization
	Documentation    *Documentation
	SpatialCoverage  *Spatial
	TemporalCoverage *Temporal
	ContactPoints    []ContactPoint
}
type Documentation struct {
	Title       string
	Description string
	URI         string
}

type Organization struct {
	Name string
}

type Coordinate struct {
	Latitude  float64
	Longitude float64
}

type Spatial struct {
	Point   *Coordinate
	Polygon []Coordinate
}
type Temporal struct {
	Start *time.Time
	End   *time.Time
}

type Identifier struct {
	Type  string // could be an enum?
	Value string
}

type ContactPoint struct {
	Email string
	Role  string // could be an enum?
}
