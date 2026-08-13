// Package metadata defines metadata models and validation rules.
package metadata

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Dataset struct {
	ID                    uuid.UUID
	SourceIRI             string
	Title                 string
	Description           string
	Keywords              string
	AccrualPeriodicity    *string
	QualityAnnotation     *string
	AccessRight           *string
	Distributions         []Distribution
	SpatialCoverage       *Spatial
	TemporalCoverage      *Temporal
	PersistentIdentifiers []Identifier
	ContactPoints         []ContactPoint
	Providers             []Organization
	Categories            []Category
	SourceCreated         *time.Time
	SourceUpdated         *time.Time
	Status                DatasetStatus
	Created               time.Time
	Updated               time.Time
}

type Distribution struct {
	ID          uuid.UUID
	Title       string
	Description string
	License     *string
	Type        DistributionType
	Issued      *time.Time
	Modified    *time.Time
	ByteSize    *int64

	// mutually exclusive
	WebService       *WebService
	DownloadableFile *DownloadableFile
}

func (d Distribution) DeliveryType() (DistributionType, error) {
	switch {
	case d.WebService != nil && d.DownloadableFile == nil:
		return DistributionTypeWebService, nil
	case d.WebService == nil && d.DownloadableFile != nil:
		return DistributionTypeDownloadableFile, nil
	default:
		return "", fmt.Errorf("exactly one delivery method is required")
	}
}

type DownloadableFile struct {
	URL    string
	Format string
}

type WebService struct {
	ID               uuid.UUID
	Name             string
	Description      string
	TemplateURL      string
	Parameters       []Parameter
	Format           string
	Provider         *Organization
	Documentation    *Documentation
	SpatialCoverage  *Spatial
	TemporalCoverage *Temporal
	ContactPoints    []ContactPoint
}

type SoftwareApplication struct {
	ID               uuid.UUID
	SourceIRI        string
	Name             string
	Description      string
	MainEntityOfPage *string
	License          *string
	Version          *string
	Keywords         string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	Creators         []Organization
	Identifiers      []Identifier
	Categories       []Category
}

type SoftwareSourceCode struct {
	ID                   uuid.UUID
	SourceIRI            string
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
	Creators             []Organization
	Identifiers          []Identifier
	Categories           []Category
}
type Documentation struct {
	Title       *string
	Description *string
	URI         string
}

type Organization struct {
	ID          uuid.UUID
	SourceIRI   string
	Name        string
	Street      *string
	Locality    *string
	PostalCode  *string
	Country     *string
	LogoURL     *string
	PageURL     *string
	Email       *string
	LEICode     *string
	Identifiers []Identifier
}

type Category struct {
	ID        uuid.UUID
	SourceIRI string
	Name      string
	Scheme    CategoryScheme
	Children  []Category
}

type CategoryScheme struct {
	ID        uuid.UUID
	SourceIRI string
	Name      string
	Label     *string
	Logo      *string
	Page      *string
	Color     *string
	Position  int32
	Concept   TopLevelConcept
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
	ID        uuid.UUID
	SourceIRI string
	Type      IdentifierType
	Value     string
}

type ContactPoint struct {
	ID        uuid.UUID
	SourceIRI string
	Email     string
	Role      ContactPointRole
}
