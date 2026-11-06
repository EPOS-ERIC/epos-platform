package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

type AccessRight string

const (
	AccessRightNonPublic    AccessRight = "NON_PUBLIC"
	AccessRightNormal       AccessRight = "NORMAL"
	AccessRightOpDatpro     AccessRight = "OP_DATPRO"
	AccessRightPublic       AccessRight = "PUBLIC"
	AccessRightRestricted   AccessRight = "RESTRICTED"
	AccessRightConfidential AccessRight = "CONFIDENTIAL"
	AccessRightSensitive    AccessRight = "SENSITIVE"
)

func (value AccessRight) Valid() bool {
	switch value {
	case AccessRightNonPublic, AccessRightNormal, AccessRightOpDatpro, AccessRightPublic,
		AccessRightRestricted, AccessRightConfidential, AccessRightSensitive:
		return true
	default:
		return false
	}
}

type AccrualPeriodicity string

const (
	AccrualPeriodicityTriennial        AccrualPeriodicity = "TRIENNIAL"
	AccrualPeriodicityBiennial         AccrualPeriodicity = "BIENNIAL"
	AccrualPeriodicityAnnual           AccrualPeriodicity = "ANNUAL"
	AccrualPeriodicitySemiannual       AccrualPeriodicity = "SEMIANNUAL"
	AccrualPeriodicityThreeTimesAYear  AccrualPeriodicity = "THREE_TIMES_A_YEAR"
	AccrualPeriodicityQuarterly        AccrualPeriodicity = "QUARTERLY"
	AccrualPeriodicityBimonthly        AccrualPeriodicity = "BIMONTHLY"
	AccrualPeriodicityMonthly          AccrualPeriodicity = "MONTHLY"
	AccrualPeriodicitySemimonthly      AccrualPeriodicity = "SEMIMONTHLY"
	AccrualPeriodicityBiweekly         AccrualPeriodicity = "BIWEEKLY"
	AccrualPeriodicityThreeTimesAMonth AccrualPeriodicity = "THREE_TIMES_A_MONTH"
	AccrualPeriodicityWeekly           AccrualPeriodicity = "WEEKLY"
	AccrualPeriodicitySemiweekly       AccrualPeriodicity = "SEMIWEEKLY"
	AccrualPeriodicityThreeTimesAWeek  AccrualPeriodicity = "THREE_TIMES_A_WEEK"
	AccrualPeriodicityDaily            AccrualPeriodicity = "DAILY"
	AccrualPeriodicityContinuous       AccrualPeriodicity = "CONTINUOUS"
	AccrualPeriodicityIrregular        AccrualPeriodicity = "IRREGULAR"
)

func (value AccrualPeriodicity) Valid() bool {
	switch value {
	case AccrualPeriodicityTriennial, AccrualPeriodicityBiennial, AccrualPeriodicityAnnual,
		AccrualPeriodicitySemiannual, AccrualPeriodicityThreeTimesAYear, AccrualPeriodicityQuarterly,
		AccrualPeriodicityBimonthly, AccrualPeriodicityMonthly, AccrualPeriodicitySemimonthly,
		AccrualPeriodicityBiweekly, AccrualPeriodicityThreeTimesAMonth, AccrualPeriodicityWeekly,
		AccrualPeriodicitySemiweekly, AccrualPeriodicityThreeTimesAWeek, AccrualPeriodicityDaily,
		AccrualPeriodicityContinuous, AccrualPeriodicityIrregular:
		return true
	default:
		return false
	}
}

// DatasetInput contains caller-controlled dataset version fields.
type DatasetInput struct {
	Title                      string
	Description                string
	Keywords                   []string
	AccrualPeriodicity         *AccrualPeriodicity
	QualityAnnotation          *string
	AccessRight                *AccessRight
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
	AccrualPeriodicity         *AccrualPeriodicity
	QualityAnnotation          *string
	AccessRight                *AccessRight
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
	ResourceID       *uuid.UUID
	Title            string
	Description      string
	License          *string
	Type             DistributionType
	ByteSize         *int64
	WebService       *WebServiceInput
	DownloadableFile *DownloadableFile
}

// Distribution is persisted distribution metadata.
type Distribution struct {
	ID               uuid.UUID
	ResourceID       uuid.UUID
	Title            string
	Description      string
	License          *string
	Type             DistributionType
	ByteSize         *int64
	WebService       *WebService
	DownloadableFile *DownloadableFile
	NativeFormats    []NativeFormat
}

// WebServiceInput contains caller-controlled web-service fields.
type WebServiceInput struct {
	ResourceID       *uuid.UUID
	Type             WebServiceType
	Configuration    WebServiceConfigurationInput
	VisibleOn        []WebServiceVisibleOn
	ProviderID       *uuid.UUID
	DocumentationURI *string
	ContactPointIDs  []uuid.UUID
}

// WebService is persisted web-service metadata.
type WebService struct {
	ID               uuid.UUID
	ResourceID       uuid.UUID
	Type             WebServiceType
	Configuration    WebServiceConfiguration
	VisibleOn        []WebServiceVisibleOn
	Provider         *OrganizationMetadata
	DocumentationURI *string
	ContactPoints    []ContactPoint
}

// WebServiceConfigurationInput is one caller-controlled typed service configuration.
type WebServiceConfigurationInput interface {
	webServiceConfigurationInput()
}

// WebServiceConfiguration is one persisted typed service configuration.
type WebServiceConfiguration interface {
	Input() WebServiceConfigurationInput
	NativeFormats() []NativeFormat
	webServiceConfiguration()
}

type GenericWebServiceConfigurationInput struct {
	TemplateURL string
	Method      HTTPMethod
	Body        *RequestBodyInput
	Parameters  []WebServiceParameterInput
	Format      string
}

func (GenericWebServiceConfigurationInput) webServiceConfigurationInput() {}

type GenericWebServiceConfiguration struct {
	TemplateURL string
	Method      HTTPMethod
	Body        *RequestBody
	Parameters  []WebServiceParameter
	Format      string
}

func (GenericWebServiceConfiguration) webServiceConfiguration() {}

type WFSConfigurationInput struct {
	GetFeatureTemplateURL string
	Version               string
	FeatureTypes          []string
	DefaultFeatureType    string
	OutputFormats         []string
	DefaultOutputFormat   string
	ResultLimit           *int64
	ResultLimitEditable   bool
	Parameters            []WebServiceParameterInput
}

func (WFSConfigurationInput) webServiceConfigurationInput() {}

type WFSConfiguration struct {
	GetFeatureTemplateURL string
	Version               string
	FeatureTypes          []string
	DefaultFeatureType    string
	OutputFormats         []string
	DefaultOutputFormat   string
	ResultLimit           *int64
	ResultLimitEditable   bool
	Parameters            []WebServiceParameter
}

func (WFSConfiguration) webServiceConfiguration() {}

type WMSConfigurationInput struct {
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMSLayerInput
	TransparencyEditable     bool
	DefaultTransparency      bool
	GetMapParameters         []WebServiceParameterInput
	GetFeatureInfoParameters []WebServiceParameterInput
}

func (WMSConfigurationInput) webServiceConfigurationInput() {}

type WMSConfiguration struct {
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMSLayer
	TransparencyEditable     bool
	DefaultTransparency      bool
	GetMapParameters         []WebServiceParameter
	GetFeatureInfoParameters []WebServiceParameter
}

func (WMSConfiguration) webServiceConfiguration() {}

type WMSLayerInput struct {
	ResourceID   *uuid.UUID
	Name         string
	DefaultStyle string
	Styles       []string
}

type WMSLayer struct {
	ID           uuid.UUID
	ResourceID   uuid.UUID
	Name         string
	DefaultStyle string
	Styles       []string
}

type WMTSConfigurationInput struct {
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMTSLayerInput
	GetTileParameters        []WebServiceParameterInput
	GetFeatureInfoParameters []WebServiceParameterInput
}

func (WMTSConfigurationInput) webServiceConfigurationInput() {}

type WMTSConfiguration struct {
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMTSLayer
	GetTileParameters        []WebServiceParameter
	GetFeatureInfoParameters []WebServiceParameter
}

func (WMTSConfiguration) webServiceConfiguration() {}

type WMTSLayerInput struct {
	ResourceID   *uuid.UUID
	Name         string
	DefaultStyle *string
	Styles       []string
}

type WMTSLayer struct {
	ID           uuid.UUID
	ResourceID   uuid.UUID
	Name         string
	DefaultStyle *string
	Styles       []string
}

type DownloadableFile struct {
	URL    string
	Format string
}

// DistributionDownload is an authorized downloadable-file request target.
type DistributionDownload struct {
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	URL        string
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
	WebServiceTypeHTTP    WebServiceType = "HTTP"
	WebServiceTypeWMTS    WebServiceType = "WMTS"
	WebServiceTypeWFS     WebServiceType = "WFS"
	WebServiceTypeDynamic WebServiceType = "DYNAMIC"
)

func (t WebServiceType) Valid() bool {
	switch t {
	case WebServiceTypeWMS, WebServiceTypeHTTP, WebServiceTypeWMTS, WebServiceTypeWFS, WebServiceTypeDynamic:
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

type WebServiceParameterRole string

const (
	WebServiceParameterRoleFormat    WebServiceParameterRole = "FORMAT"
	WebServiceParameterRoleStartDate WebServiceParameterRole = "START_DATE"
	WebServiceParameterRoleEndDate   WebServiceParameterRole = "END_DATE"
	WebServiceParameterRoleMinLon    WebServiceParameterRole = "MIN_LON"
	WebServiceParameterRoleMaxLon    WebServiceParameterRole = "MAX_LON"
	WebServiceParameterRoleMinLat    WebServiceParameterRole = "MIN_LAT"
	WebServiceParameterRoleMaxLat    WebServiceParameterRole = "MAX_LAT"
	WebServiceParameterRoleLongitude WebServiceParameterRole = "LONGITUDE"
	WebServiceParameterRoleLatitude  WebServiceParameterRole = "LATITUDE"
)

func (r WebServiceParameterRole) Valid() bool {
	switch r {
	case WebServiceParameterRoleFormat,
		WebServiceParameterRoleStartDate,
		WebServiceParameterRoleEndDate,
		WebServiceParameterRoleMinLon,
		WebServiceParameterRoleMaxLon,
		WebServiceParameterRoleMinLat,
		WebServiceParameterRoleMaxLat,
		WebServiceParameterRoleLongitude,
		WebServiceParameterRoleLatitude:
		return true
	default:
		return false
	}
}

// ParameterInput contains caller-controlled scalar or template parameter fields.
type ParameterInput struct {
	ResourceID  *uuid.UUID
	Position    int
	Name        string
	Label       string
	Description *string
	Property    *string
	Role        *WebServiceParameterRole
	Required    bool
	ReadOnly    bool
	Schema      ParameterSchema
}

// Parameter is persisted scalar or template parameter metadata.
type Parameter struct {
	ParameterInput
	ID         uuid.UUID
	ResourceID uuid.UUID
}

type WebServiceParameterInput struct {
	ParameterInput
	Location ParameterLocation
	Explode  bool
}

type WebServiceParameter struct {
	Parameter
	Location ParameterLocation
	Explode  bool
}

type RequestBodyInput struct {
	Required    *bool
	ContentType string
	Template    string
	Parameters  []ParameterInput
}

type RequestBody struct {
	Required    bool
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
	HasEnum() bool
	AllowsMultipleValues() bool
	// DefaultValue returns the typed default with schema multiplicity, or nil if absent.
	// The result still needs value validation and datetime layout formatting.
	DefaultValue() AccessValue
	parameterSchema()
}

type StringEnumOption struct {
	Value string
	Label *string
}

type StringSchema struct {
	Pattern        *string
	Enum           []StringEnumOption
	Default        []string
	MultipleValues bool
}

func (StringSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindString }
func (StringSchema) parameterSchema()          {}

// HasEnum reports whether the schema declares allowed values.
func (value StringSchema) HasEnum() bool { return len(value.Enum) > 0 }

// AllowsMultipleValues reports whether the schema expects an array.
func (value StringSchema) AllowsMultipleValues() bool { return value.MultipleValues }

func defaultAccessValue[T any](defaults []T, multipleValues bool, scalar func(T) AccessScalar) AccessValue {
	if len(defaults) == 0 {
		return nil
	}
	if !multipleValues {
		return scalar(defaults[0])
	}
	result := make(AccessArray, len(defaults))
	for i, item := range defaults {
		result[i] = scalar(item)
	}
	return result
}

// DefaultValue returns the configured string defaults using the schema's value shape.
func (value StringSchema) DefaultValue() AccessValue {
	return defaultAccessValue(value.Default, value.MultipleValues, func(item string) AccessScalar {
		return AccessString(item)
	})
}

type BooleanSchema struct {
	Default *bool
}

func (BooleanSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindBoolean }
func (BooleanSchema) parameterSchema()          {}

// HasEnum reports false because boolean schemas do not declare enums.
func (BooleanSchema) HasEnum() bool { return false }

// AllowsMultipleValues reports false because boolean parameters are scalar.
func (BooleanSchema) AllowsMultipleValues() bool { return false }

// DefaultValue returns the boolean default, or nil when none is configured.
func (value BooleanSchema) DefaultValue() AccessValue {
	if value.Default == nil {
		return nil
	}
	return AccessBoolean(*value.Default)
}

type IntegerEnumOption struct {
	Value int64
	Label *string
}

type IntegerSchema struct {
	Minimum        *int64
	Maximum        *int64
	Enum           []IntegerEnumOption
	Default        []int64
	MultipleValues bool
}

func (IntegerSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindInteger }
func (IntegerSchema) parameterSchema()          {}

// HasEnum reports whether the schema declares allowed values.
func (value IntegerSchema) HasEnum() bool { return len(value.Enum) > 0 }

// AllowsMultipleValues reports whether the schema expects an array.
func (value IntegerSchema) AllowsMultipleValues() bool { return value.MultipleValues }

// DefaultValue returns the configured integer defaults using the schema's value shape.
func (value IntegerSchema) DefaultValue() AccessValue {
	return defaultAccessValue(value.Default, value.MultipleValues, func(item int64) AccessScalar {
		return AccessInteger(item)
	})
}

type FloatEnumOption struct {
	Value float64
	Label *string
}

type FloatSchema struct {
	Minimum        *float64
	Maximum        *float64
	Enum           []FloatEnumOption
	Default        []float64
	MultipleValues bool
}

func (FloatSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindFloat }
func (FloatSchema) parameterSchema()          {}

// HasEnum reports whether the schema declares allowed values.
func (value FloatSchema) HasEnum() bool { return len(value.Enum) > 0 }

// AllowsMultipleValues reports whether the schema expects an array.
func (value FloatSchema) AllowsMultipleValues() bool { return value.MultipleValues }

// DefaultValue returns the configured float defaults using the schema's value shape.
func (value FloatSchema) DefaultValue() AccessValue {
	return defaultAccessValue(value.Default, value.MultipleValues, func(item float64) AccessScalar {
		return AccessFloat(item)
	})
}

type DateTimeSchema struct {
	Minimum *time.Time
	Maximum *time.Time
	Default *time.Time
	Layout  *string
}

func (DateTimeSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindDateTime }
func (DateTimeSchema) parameterSchema()          {}

// HasEnum reports false because datetime schemas do not declare enums.
func (DateTimeSchema) HasEnum() bool { return false }

// AllowsMultipleValues reports false because datetime parameters are scalar.
func (DateTimeSchema) AllowsMultipleValues() bool { return false }

// DefaultValue returns the datetime default as RFC 3339 text, or nil if absent.
// Value resolution applies UTC normalization and the configured layout afterward.
func (value DateTimeSchema) DefaultValue() AccessValue {
	if value.Default == nil {
		return nil
	}
	return AccessString(value.Default.Format(time.RFC3339Nano))
}

type TemplateSchema struct {
	Template   string
	Parameters []Parameter
}

func (TemplateSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindTemplate }
func (TemplateSchema) parameterSchema()          {}

// HasEnum reports false because template schemas do not declare enums.
func (TemplateSchema) HasEnum() bool { return false }

// AllowsMultipleValues reports false because a template renders a single value.
func (TemplateSchema) AllowsMultipleValues() bool { return false }

// DefaultValue returns nil because templates resolve through their child parameters.
func (TemplateSchema) DefaultValue() AccessValue { return nil }

type TemplateSchemaInput struct {
	Template   string
	Parameters []ParameterInput
}

func (TemplateSchemaInput) Kind() ParameterSchemaKind { return ParameterSchemaKindTemplate }
func (TemplateSchemaInput) parameterSchema()          {}

// HasEnum reports false because template schemas do not declare enums.
func (TemplateSchemaInput) HasEnum() bool { return false }

// AllowsMultipleValues reports false because a template renders a single value.
func (TemplateSchemaInput) AllowsMultipleValues() bool { return false }

// DefaultValue returns nil because templates resolve through their child parameters.
func (TemplateSchemaInput) DefaultValue() AccessValue { return nil }

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
