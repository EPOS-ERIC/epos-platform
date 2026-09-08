package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

// DatasetInput contains caller-controlled dataset version fields.
type DatasetInput struct {
	Title                 string
	Description           string
	Keywords              []string
	AccrualPeriodicity    *string
	QualityAnnotation     *string
	AccessRight           *string
	Distributions         []DistributionInput
	SpatialCoverage       geom.T
	TemporalCoverage      *Temporal
	PersistentIdentifiers []IdentifierInput
	ContactPointIDs       []uuid.UUID
	ProviderIDs           []uuid.UUID
	CategoryIDs           []uuid.UUID
	SourceCreated         *time.Time
	SourceUpdated         *time.Time
}

// DatasetDraftInput contains a validated draft replacement command.
type DatasetDraftInput struct {
	Revision int32
	Metadata DatasetInput
}

// DatasetRejectionInput contains a validated rejection command.
type DatasetRejectionInput struct {
	Note string
}

// DatasetMetadata is persisted dataset version metadata.
type DatasetMetadata struct {
	ID                    uuid.UUID
	Title                 string
	Description           string
	Keywords              []string
	AccrualPeriodicity    *string
	QualityAnnotation     *string
	AccessRight           *string
	Distributions         []Distribution
	SpatialCoverage       geom.T
	TemporalCoverage      *Temporal
	PersistentIdentifiers []Identifier
	ContactPoints         []ContactPoint
	Providers             []OrganizationMetadata
	Categories            []CategoryMetadata
	SourceCreated         *time.Time
	SourceUpdated         *time.Time
}

// DistributionInput contains caller-controlled distribution fields.
type DistributionInput struct {
	Title            string
	Description      string
	License          *string
	Type             DistributionType
	Issued           *time.Time
	Modified         *time.Time
	ByteSize         *int64
	WebService       *WebServiceInput
	DownloadableFile *DownloadableFile
}

// Distribution is persisted distribution metadata.
type Distribution struct {
	ID               uuid.UUID
	Title            string
	Description      string
	License          *string
	Type             DistributionType
	Issued           *time.Time
	Modified         *time.Time
	ByteSize         *int64
	WebService       *WebService
	DownloadableFile *DownloadableFile
}

// WebServiceInput contains caller-controlled web-service fields.
type WebServiceInput struct {
	Name             string
	Description      string
	TemplateURL      string
	Parameters       []Parameter
	Format           string
	ProviderID       *uuid.UUID
	Documentation    *Documentation
	SpatialCoverage  geom.T
	TemporalCoverage *Temporal
	ContactPointIDs  []uuid.UUID
}

// WebService is persisted web-service metadata.
type WebService struct {
	ID               uuid.UUID
	Name             string
	Description      string
	TemplateURL      string
	Parameters       []Parameter
	Format           string
	Provider         *OrganizationMetadata
	Documentation    *Documentation
	SpatialCoverage  geom.T
	TemporalCoverage *Temporal
	ContactPoints    []ContactPoint
}

type DownloadableFile struct {
	URL    string
	Format string
}

type Documentation struct {
	Title       *string
	Description *string
	URI         string
}

type Temporal struct {
	Start *time.Time
	End   *time.Time
}

type Parameter interface {
	GetMetadata() ParameterMetadata
	Kind() ParameterKind
}

type ParameterMetadata struct {
	ID             uuid.UUID
	Kind           ParameterKind
	Variable       string
	Property       string
	Label          string
	Required       bool
	MultipleValues bool
	ReadOnly       bool
}

type StringParameter struct {
	Metadata      ParameterMetadata
	Default       *string
	AllowedValues []string
	Pattern       string
}

func (p StringParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p StringParameter) Kind() ParameterKind            { return ParameterKindString }

type BooleanParameter struct {
	Metadata ParameterMetadata
	Default  *bool
}

func (p BooleanParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p BooleanParameter) Kind() ParameterKind            { return ParameterKindBoolean }

type IntegerParameter struct {
	Metadata      ParameterMetadata
	Minimum       *int64
	Maximum       *int64
	Default       *int64
	AllowedValues []int64
}

func (p IntegerParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p IntegerParameter) Kind() ParameterKind            { return ParameterKindInteger }

type FloatParameter struct {
	Metadata      ParameterMetadata
	Minimum       *float64
	Maximum       *float64
	Default       *float64
	AllowedValues []float64
}

func (p FloatParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p FloatParameter) Kind() ParameterKind            { return ParameterKindFloat }

type DateTimeParameter struct {
	Metadata ParameterMetadata
	Minimum  *time.Time
	Maximum  *time.Time
	Default  *time.Time
	Pattern  string
}

func (p DateTimeParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p DateTimeParameter) Kind() ParameterKind            { return ParameterKindDateTime }

type DistributionType string

// TODO: use better constants
const (
	DistributionTypeWebService       DistributionType = "http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"
	DistributionTypeDownloadableFile DistributionType = "http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE"
)

func (t DistributionType) Valid() bool {
	return t == DistributionTypeDownloadableFile || t == DistributionTypeWebService
}

type ParameterKind string

const (
	ParameterKindString   ParameterKind = "STRING"
	ParameterKindBoolean  ParameterKind = "BOOLEAN"
	ParameterKindInteger  ParameterKind = "INTEGER"
	ParameterKindFloat    ParameterKind = "FLOAT"
	ParameterKindDateTime ParameterKind = "DATETIME"
)

func (k ParameterKind) Valid() bool {
	switch k {
	case ParameterKindString, ParameterKindBoolean, ParameterKindInteger, ParameterKindFloat, ParameterKindDateTime:
		return true
	default:
		return false
	}
}

// DatasetVersion is dataset metadata with lifecycle identity.
type DatasetVersion struct {
	DatasetID     uuid.UUID
	VersionID     uuid.UUID
	Revision      int32
	Status        Status
	CreatedBy     Actor
	Group         Group
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RejectionNote *string
	Metadata      DatasetMetadata
}
