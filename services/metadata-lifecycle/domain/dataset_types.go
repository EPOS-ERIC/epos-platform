package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

// DatasetInput contains caller-controlled dataset version fields.
type DatasetInput struct {
	Title                      string
	Description                string
	Keywords                   []string
	AccrualPeriodicity         *string
	QualityAnnotation          *string
	AccessRight                *string
	Distributions              []DistributionInput
	SpatialCoverage            geom.T
	TemporalCoverage           *Temporal
	GeologicalTemporalCoverage *GeologicalTemporal
	PersistentIdentifiers      []IdentifierInput
	ContactPointIDs            []uuid.UUID
	ProviderIDs                []uuid.UUID
	CategoryIDs                []uuid.UUID
	SourceCreated              *time.Time
	SourceUpdated              *time.Time
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
	ID                         uuid.UUID
	Title                      string
	Description                string
	Keywords                   []string
	AccrualPeriodicity         *string
	QualityAnnotation          *string
	AccessRight                *string
	Distributions              []Distribution
	SpatialCoverage            geom.T
	TemporalCoverage           *Temporal
	GeologicalTemporalCoverage *GeologicalTemporal
	PersistentIdentifiers      []Identifier
	ContactPoints              []ContactPoint
	Providers                  []OrganizationMetadata
	Categories                 []CategoryMetadata
	SourceCreated              *time.Time
	SourceUpdated              *time.Time
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
	Description      *string
	TemplateURL      string
	Method           HTTPMethod
	Type             WebServiceType
	VisibleOn        []WebServiceVisibleOn
	Body             *RequestBody
	Parameters       []WebServiceParameterInput
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
	Description      *string
	TemplateURL      string
	Method           HTTPMethod
	Type             WebServiceType
	VisibleOn        []WebServiceVisibleOn
	Body             *RequestBody
	Parameters       []WebServiceParameter
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

type GeologicalTemporal struct {
	StartMYA *float64
	EndMYA   *float64
}

type HTTPMethod string

const (
	HTTPMethodGet    HTTPMethod = "GET"
	HTTPMethodPost   HTTPMethod = "POST"
	HTTPMethodPut    HTTPMethod = "PUT"
	HTTPMethodDelete HTTPMethod = "DELETE"
)

func (m HTTPMethod) Valid() bool {
	switch m {
	case HTTPMethodGet, HTTPMethodPost, HTTPMethodPut, HTTPMethodDelete:
		return true
	default:
		return false
	}
}

type WebServiceType string

const (
	WebServiceTypeWMS     WebServiceType = "WMS"
	WebServiceTypeGeoJSON WebServiceType = "GEOJSON"
	WebServiceTypeDynamic WebServiceType = "DYNAMIC"
	WebServiceTypeWMTS    WebServiceType = "WMTS"
	WebServiceTypeWFS     WebServiceType = "WFS"
	WebServiceTypeCOVJSON WebServiceType = "COVJSON"
)

func (t WebServiceType) Valid() bool {
	switch t {
	case WebServiceTypeWMS, WebServiceTypeGeoJSON, WebServiceTypeDynamic, WebServiceTypeWMTS, WebServiceTypeWFS, WebServiceTypeCOVJSON:
		return true
	default:
		return false
	}
}

type WebServiceVisibleOn string

const (
	WebServiceVisibleOnGraph WebServiceVisibleOn = "GRAPH"
	WebServiceVisibleOnTable WebServiceVisibleOn = "TABLE"
	WebServiceVisibleOnMap   WebServiceVisibleOn = "MAP"
)

func (v WebServiceVisibleOn) Valid() bool {
	switch v {
	case WebServiceVisibleOnGraph, WebServiceVisibleOnTable, WebServiceVisibleOnMap:
		return true
	default:
		return false
	}
}

type ParameterLocation string

const (
	ParameterLocationPath   ParameterLocation = "PATH"
	ParameterLocationQuery  ParameterLocation = "QUERY"
	ParameterLocationHeader ParameterLocation = "HEADER"
)

func (l ParameterLocation) Valid() bool {
	switch l {
	case ParameterLocationPath, ParameterLocationQuery, ParameterLocationHeader:
		return true
	default:
		return false
	}
}

// Parameter is a scalar body or template parameter.
type Parameter struct {
	Position    int
	Name        string
	Label       string
	Description *string
	Required    bool
	ReadOnly    bool
	Schema      ParameterSchema
}

type WebServiceParameterInput struct {
	Parameter
	Property *string
	Location ParameterLocation
	Explode  bool
}

type WebServiceParameter struct {
	ID uuid.UUID
	Parameter
	Property *string
	Location ParameterLocation
	Explode  bool
}

type RequestBody struct {
	ContentType string
	Template    string
	Parameters  []Parameter
}

type ParameterSchemaKind string

const (
	ParameterSchemaKindString   ParameterSchemaKind = "STRING"
	ParameterSchemaKindBoolean  ParameterSchemaKind = "BOOLEAN"
	ParameterSchemaKindInteger  ParameterSchemaKind = "INTEGER"
	ParameterSchemaKindFloat    ParameterSchemaKind = "FLOAT"
	ParameterSchemaKindDateTime ParameterSchemaKind = "DATETIME"
	ParameterSchemaKindTemplate ParameterSchemaKind = "TEMPLATE"
)

type ParameterSchema interface {
	Kind() ParameterSchemaKind
	parameterSchema()
}

type StringEnumOption struct {
	Value string
	Label *string
}

type StringSchema struct {
	Pattern        *string
	Enum           []StringEnumOption
	Default        *string
	MultipleValues bool
}

func (StringSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindString }
func (StringSchema) parameterSchema()          {}

type BooleanSchema struct {
	Default *bool
}

func (BooleanSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindBoolean }
func (BooleanSchema) parameterSchema()          {}

type IntegerEnumOption struct {
	Value int64
	Label *string
}

type IntegerSchema struct {
	Minimum        *int64
	Maximum        *int64
	Enum           []IntegerEnumOption
	Default        *int64
	MultipleValues bool
}

func (IntegerSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindInteger }
func (IntegerSchema) parameterSchema()          {}

type FloatEnumOption struct {
	Value float64
	Label *string
}

type FloatSchema struct {
	Minimum        *float64
	Maximum        *float64
	Enum           []FloatEnumOption
	Default        *float64
	MultipleValues bool
}

func (FloatSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindFloat }
func (FloatSchema) parameterSchema()          {}

type DateTimeSchema struct {
	Minimum *time.Time
	Maximum *time.Time
	Default *time.Time
	Layout  *string
}

func (DateTimeSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindDateTime }
func (DateTimeSchema) parameterSchema()          {}

type TemplateSchema struct {
	Template   string
	Parameters []Parameter
}

func (TemplateSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindTemplate }
func (TemplateSchema) parameterSchema()          {}

type DistributionType string

const (
	DistributionTypeWebService       DistributionType = "WEB_SERVICE"
	DistributionTypeDownloadableFile DistributionType = "DOWNLOADABLE_FILE"
)

func (t DistributionType) Valid() bool {
	return t == DistributionTypeDownloadableFile || t == DistributionTypeWebService
}

// DatasetVersion is dataset metadata with lifecycle identity.
type DatasetVersion struct {
	DatasetID     uuid.UUID
	VersionID     uuid.UUID
	Revision      int32
	Status        Status
	CreatedBy     Actor
	SubmittedBy   *Actor
	Group         Group
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RejectedBy    *string
	RejectedAt    *time.Time
	RejectionNote *string
	Metadata      DatasetMetadata
}
