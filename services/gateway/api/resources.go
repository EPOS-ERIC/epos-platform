package api

import "time"

// TODO: improve this package and the openapi spec by explicitly set required and not fields
type Dataset struct {
	ID                 string           `json:"id"`
	Status             VersioningStatus `json:"status,omitempty"`
	Created            time.Time        `json:"created"`
	Modified           time.Time        `json:"modified"`
	Title              string           `json:"title,omitempty"`
	VersionInfo        string           `json:"version_info,omitempty"`
	ContactPoints      []string         `json:"contact_points,omitempty"` // just the ids of the contact points, not the full objects (can't show the emails)
	Keywords           []string         `json:"keywords,omitempty"`
	Concepts           []string         `json:"concepts,omitempty"`
	Description        string           `json:"description,omitempty"`
	Publisher          []string         `json:"publisher,omitempty"`
	Issued             time.Time        `json:"issued"`
	Spatial            string           `json:"spatial,omitempty"`
	Identifiers        []Identifier     `json:"identifiers,omitempty"`
	Temporal           TemporalExtent   `json:"temporal"`
	Distributions      []SearchItem     `json:"distributions,omitempty"` // just the link to the Distributions not aggregated
	AccessRights       string           `json:"access_rights,omitempty"`
	AccrualPeriodicity string           `json:"accrual_periodicity,omitempty"`
	IsVersionOf        string           `json:"is_version_of,omitempty"`
	QualityAnnotation  []string         `json:"quality_annotation,omitempty"`
}

type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SoftwareApplication struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Concepts    []string `json:"concepts,omitempty"`
	DownloadURL string   `json:"download_url,omitempty"`
}

type SoftwareSourceCode struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Description         string   `json:"description,omitempty"`
	ProgrammingLanguage string   `json:"programming_language,omitempty"`
	Keywords            []string `json:"keywords,omitempty"`
	Concepts            []string `json:"concepts,omitempty"`
	Repository          string   `json:"repository,omitempty"`
}

type TemporalExtent struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TODO main entities:
// Dataset
// Distribution
// SoftwareSourceCode
// SoftwareApplication
// Webservice
// Organization
// Operation: aggiungere anche l'api documentation

// only aggregate "small" entities, like temporal or spatial, main entities have their own endpoint
