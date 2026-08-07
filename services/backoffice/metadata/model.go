// Package metadata defines metadata models and validation rules.
package metadata

import (
	"time"

	"github.com/google/uuid"
)

type Dataset struct {
	ID                    uuid.UUID      `json:"id"`
	SourceIRI             string         `json:"source_iri"`
	Title                 string         `json:"title"`
	Description           string         `json:"description"`
	Distributions         []Distribution `json:"distributions"`
	SpatialCoverage       *Spatial       `json:"spatial_coverage"`
	TemporalCoverage      *Temporal      `json:"temporal_coverage"`
	PersistentIdentifiers []Identifier   `json:"persistent_identifiers"`
	ContactPoints         []ContactPoint `json:"contact_points"`
	Providers             []Organization `json:"providers"`
	Categories            []Category     `json:"categories"`
	SourceCreated         *time.Time     `json:"source_created"`
	SourceUpdated         *time.Time     `json:"source_updated"`

	// versioning stuff
	Status            DatasetStatus `json:"status"`
	Editor            *User         `json:"editor"`
	Group             *Group        `json:"group"`
	Created           time.Time     `json:"created"`
	Updated           time.Time     `json:"updated"`
	OriginalDatasetID uuid.UUID     `json:"original_dataset_id"`
}

type Distribution struct {
	ID          uuid.UUID `json:"id"`
	SourceIRI   string    `json:"source_iri"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	License     *string   `json:"license"`

	// mutually exclusive
	WebService       *WebService       `json:"web_service"`
	DownloadableFile *DownloadableFile `json:"downloadable_file"`
}

type DownloadableFile struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

type WebService struct {
	ID               uuid.UUID      `json:"id"`
	SourceIRI        string         `json:"source_iri"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	TemplateURL      string         `json:"template_url"`
	Parameters       []Parameter    `json:"parameters"`
	Format           string         `json:"format"`
	Providers        []Organization `json:"providers"`
	Documentation    *Documentation `json:"documentation"`
	SpatialCoverage  *Spatial       `json:"spatial_coverage"`
	TemporalCoverage *Temporal      `json:"temporal_coverage"`
	ContactPoints    []ContactPoint `json:"contact_points"`
}
type Documentation struct {
	SourceIRI   string  `json:"source_iri"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	URI         string  `json:"uri"`
}

type Organization struct {
	ID          uuid.UUID    `json:"id"`
	SourceIRI   string       `json:"source_iri"`
	Name        string       `json:"name"`
	Street      *string      `json:"street"`
	Locality    *string      `json:"locality"`
	PostalCode  *string      `json:"postal_code"`
	Country     *string      `json:"country"`
	LogoURL     *string      `json:"logo_url"`
	PageURL     *string      `json:"page_url"`
	Email       *string      `json:"email"`
	Identifiers []Identifier `json:"identifiers"`
}

type Category struct {
	ID        uuid.UUID      `json:"id"`
	SourceIRI string         `json:"source_iri"`
	Name      string         `json:"name"`
	Scheme    CategoryScheme `json:"scheme"`
}

type CategoryScheme struct {
	ID        uuid.UUID       `json:"id"`
	SourceIRI string          `json:"source_iri"`
	Name      string          `json:"name"`
	Concept   TopLevelConcept `json:"concept"`
}

type User struct {
	ID             uuid.UUID  `json:"id"`
	AuthIdentifier string     `json:"auth_identifier"`
	FamilyName     *string    `json:"family_name"`
	GivenName      *string    `json:"given_name"`
	Email          *string    `json:"email"`
	IsSuperAdmin   bool       `json:"is_super_admin"`
	Status         UserStatus `json:"status"`
	Created        time.Time  `json:"created"`
	Updated        time.Time  `json:"updated"`
}

type Group struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsSystem    bool      `json:"is_system"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Spatial struct {
	Point   *Coordinate  `json:"point"`
	Polygon []Coordinate `json:"polygon"`
}
type Temporal struct {
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
}

type Identifier struct {
	ID        uuid.UUID      `json:"id"`
	SourceIRI string         `json:"source_iri"`
	Type      IdentifierType `json:"type"`
	Value     string         `json:"value"`
}

type ContactPoint struct {
	ID        uuid.UUID        `json:"id"`
	SourceIRI string           `json:"source_iri"`
	Email     string           `json:"email"`
	Role      ContactPointRole `json:"role"`
}
