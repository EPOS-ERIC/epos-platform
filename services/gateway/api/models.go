package api

import "time"

type SearchResponse struct {
	Total int32        `json:"total"`
	Items []SearchItem `json:"items"`
}

type SearchItem struct {
	Type             ResourceType `json:"type"`
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Href             string       `json:"href"`
	ServiceProviders []string     `json:"serviceProviders,omitempty"`
	Concepts         []string     `json:"concepts,omitempty"`
}

type Dataset struct {
	ID          string           `json:"id"`
	Status      VersioningStatus `json:"status,omitempty"`
	Created     time.Time        `json:"created"`
	Modified    time.Time        `json:"modified"`
	Title       string           `json:"title,omitempty"`
	VersionInfo string           `json:"version_info,omitempty"`
	// just the ids of the contact points, not the full objects (can't show the emails)
	ContactPoints      []string
	Keywords           []string       `json:"keywords,omitempty"`
	Concepts           []string       `json:"concepts,omitempty"`
	Description        string         `json:"description,omitempty"`
	Publisher          []string       `json:"publisher,omitempty"`
	Issued             time.Time      `json:"issued"`
	Spatial            string         `json:"spatial,omitempty"`
	Identifiers        []Identifier   `json:"identifiers,omitempty"`
	Temporal           TemporalExtent `json:"temporaly"`
	Distributions      []Distribution `json:"distributions,omitempty"`
	AccessRights       string         `json:"access_rights"`
	AccrualPeriodicity string         `json:"accrual_periodicity"`
	IsVersionOf        string         `json:"is_version_of"`
	QualityAnnotation  []string       `json:"quality_annotation"`
	// TODO: what is the difference between version and version info?
	Version string `json:"version"`
}

type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Distribution struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AccessURL   string `json:"accessUrl,omitempty"`
	Operation   string `json:"operation,omitempty"`
}

type SoftwareApplication struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Concepts    []string `json:"concepts,omitempty"`
	DownloadURL string   `json:"downloadUrl,omitempty"`
}

type SoftwareSourceCode struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Description         string   `json:"description,omitempty"`
	ProgrammingLanguage string   `json:"programmingLanguage,omitempty"`
	Keywords            []string `json:"keywords,omitempty"`
	Concepts            []string `json:"concepts,omitempty"`
	Repository          string   `json:"repository,omitempty"`
}

type Concept struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Children    []Concept `json:"children,omitempty"`
}

type ConceptScheme struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Logo        string    `json:"logo,omitempty"`
	Homepage    string    `json:"homepage,omitempty"`
	Color       string    `json:"color,omitempty"`
	OrderNumber int32     `json:"orderNumber,omitempty"`
	Type        string    `json:"type"`
	Children    []Concept `json:"children,omitempty"`
}

type ConceptTreeResponse struct {
	Schemes []ConceptScheme `json:"schemes"`
}

type TemporalExtent struct {
	Start time.Time `json:"starty"`
	End   time.Time `json:"endy"`
}

type ErrorResponse struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type ContactPoint struct {
	ID                string
	Email             string
	AvailableLanguage string
	ContactType       string
}
