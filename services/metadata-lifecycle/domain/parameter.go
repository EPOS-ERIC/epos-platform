package domain

import (
	"time"

	"github.com/google/uuid"
)

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

// AllowsMultipleValues reports false because a template renders a single value.
func (TemplateSchemaInput) AllowsMultipleValues() bool { return false }

// DefaultValue returns nil because templates resolve through their child parameters.
func (TemplateSchemaInput) DefaultValue() AccessValue { return nil }
