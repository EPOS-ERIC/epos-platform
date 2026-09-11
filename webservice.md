We decided to change the parameter schema to follow the OpenAPI schema as
closely as possible. This is an EPOS model inspired by OpenAPI, not a stored
OpenAPI document, so we can add or remove features without coupling the
platform to the OpenAPI specification. It keeps OpenAPI's parameter structure
but uses EPOS-native schema types such as `float` and `datetime`.

The URL template contains the path structure and path placeholders. Query
parameters are declared separately because they use the normal query-parameter
format. Request headers are also declared as parameters. The parameter
location is one of `path`, `query`, or `header`.

Each web service has a required `type`, one of `WMS`, `GEOJSON`, `DYNAMIC`,
`WMTS`, `WFS`, or `COVJSON`. `visible_on` is optional. When present, it contains
one or more of `GRAPH`, `TABLE`, and `MAP`; order does not matter. `null` or
omission means visibility targets are not configured.

Each web service has a required HTTP method. The initial supported methods are
`GET`, `POST`, `PUT`, and `DELETE`, matching the current retriever. A request
body is optional. When present, it is described by a content type, a body
template, and a JSON array of parameters used by that template.

Each parameter has the usual metadata:

```yaml
- name: contactPointId
  in: path
  position: 0
  label: Contact point ID
  description: Identifier of the contact point.
  required: true
  multiple_values: false
  explode: false
  read_only: false
  schema:
    type: string
```

The JSON/API field `in` is stored as the SQL column `location`. Top-level
parameter names are unique within one web service, regardless of location.
This avoids ambiguity when clients submit values. A path parameter must also
correspond to a placeholder in `template_url`; the application validates this
relationship. `position` is an integer ordering priority and may be negative.
It does not need to be unique. Parameters with equal positions are ordered
alphabetically by name. It is used to preserve form and template-parameter
ordering.

Request headers use the same parameter shape:

```yaml
- name: X-Client-Version
  in: header
  position: 0
  label: Client version
  description: Version sent to the remote service.
  required: false
  multiple_values: false
  schema:
    type: string
```

For more particular parameters, such as a CQL filter, the schema can use a
custom `template` type:

```yaml
- name: cql_filter
  in: query
  position: 0
  label: CQL filter
  description: Spatial filter generated from the selected bounds.
  required: true
  multiple_values: false
  explode: false
  read_only: false
  schema:
    type: template
    template: "{(geom,{{.minLatitude}},{{.maxLatitude}})}"
    parameters:
      - name: minLatitude
        position: 0
        label: Minimum latitude
        description: Lower latitude bound.
        required: true
        multiple_values: false
        read_only: false
        schema:
          type: float
          minimum: -90
          maximum: 90
      - name: maxLatitude
        position: 1
        label: Maximum latitude
        description: Upper latitude bound.
        required: true
        multiple_values: false
        read_only: false
        schema:
          type: float
          minimum: -90
          maximum: 90
```

The template is persisted as data. Its syntax and references must be parsed
and validated by the application when metadata is written. Nested template
parameters do not have `in`, because they are only inputs used to render the
outer parameter. Template parameters use the same scalar schemas as regular
parameters and cannot themselves be templates in the initial model.

The complete web-service representation can look like this:

```json
{
  "id": "a2a419dd-8b75-4be7-ade1-576753fa57a6",
  "name": "My WebService",
  "description": "my webservice description",
  "template_url": "https://myurl.com/{minMagnitude}/",
  "method": "GET",
  "type": "GEOJSON",
  "visible_on": null,
  "body_content_type": null,
  "body_template": null,
  "body_parameters": null,
  "format": "http://publications.europa.eu/resource/authority/file-type/JSON",
  "provider": "957d658c-7995-5eff-8e6b-972684a3e73f",
  "documentation_title": "EMSC EventID usage specifications",
  "documentation_description": "Some description of the API Documentation",
  "documentation_uri": "https://www.emsc-csem.org/Files/epos/specifications/Specs_EventID-WS.pdf",
  "spatial_coverage": "POLYGON ((-180 90, 180 90, 180 -90, -180 -90, -180 90))",
  "contact_points": ["6e2a1913-0056-5925-b53e-b4073ad5e463"],
  "parameters": [
    {
      "name": "minMagnitude",
      "in": "path",
      "position": 0,
      "label": "Minimum magnitude",
      "description": "Lower magnitude bound.",
      "required": true,
      "multiple_values": false,
      "explode": false,
      "read_only": false,
      "schema": {
        "type": "float"
      }
    }
  ]
}
```

For a service with a body, the body fields can look like this:

```json
{
  "method": "POST",
  "body_content_type": "application/json",
  "body_template": "{\"minimum\": {{.minMagnitude}}, \"station\": \"{{.stationName}}\"}",
  "body_parameters": [
    {
      "name": "minMagnitude",
      "position": 0,
      "label": "Minimum magnitude",
      "description": "Lower magnitude bound.",
      "required": true,
      "multiple_values": false,
      "read_only": false,
      "schema": {
        "type": "float",
        "minimum": 0
      }
    },
    {
      "name": "stationName",
      "position": 1,
      "label": "Station name",
      "description": "Station identifier.",
      "required": true,
      "multiple_values": false,
      "read_only": false,
      "schema": {
        "type": "string",
        "pattern": "^[A-Za-z0-9]+$"
      }
    }
  ]
}
```

Body parameters do not have `in`, because they are only inputs used to render
the body template. `body_parameters` is an array, and an empty array is valid
when the body template has no parameters. The body template can contain any
media type; its rendered content is not validated against `body_content_type`
by the persistence model.

Persistence uses one generic `parameters` table. Its common metadata is shared
by web-service, body, and template-child parameters. Ownership is represented by
exactly one of `web_service_id`, `web_service_body_id`, or
`template_parameter_id`. Root web-service parameters also have `location` and
`explode`.

`web_service_bodies` stores the optional body definition separately from
`web_services`. `parameter_templates` stores template source for a template
parameter. Template children reference their parent parameter directly through
`parameters.template_parameter_id`.

The parameter table uses `schema_kind` plus typed columns: `text[]` for string
enums, `bigint[]` for integer enums, and `double precision[]` for float enums.
Only columns belonging to the selected schema kind may be populated. Database
constraints enforce ownership, basic ranges, enum membership, and scalar field
shape. Domain validation enforces regular expressions, date-time layout, and
template syntax and references.

`position` preserves the order used to generate forms. Parameter queries order
by owner, `position`, and `name`, not by the random parameter UUID. The same
ordering applies to template and body parameter scopes.

The database enforces relational and basic typed-column invariants. The application
must validate the schema discriminator, supported fields, value types,
defaults, enum membership, ranges, date-time values, template references,
path-placeholder relationships, body template syntax, and body template
references.

The metadata lifecycle domain can use the same `Parameter` and
`ParameterSchema` types for top-level parameters, template parameters, and
body parameters. Top-level parameters add an ID, a location, and an `explode`
flag; nested parameters do not. `explode` is valid only for query parameters
with `multiple_values: true`. When enabled, array values are sent as repeated
query keys. When disabled, they remain in one query parameter value;

The following is the proposed Go domain model. Imports are omitted.

```go
type HTTPMethod string

const (
	HTTPMethodGet    HTTPMethod = "GET"
	HTTPMethodPost   HTTPMethod = "POST"
	HTTPMethodPut    HTTPMethod = "PUT"
	HTTPMethodDelete HTTPMethod = "DELETE"
)

type WebServiceType string

const (
	WebServiceTypeWMS     WebServiceType = "WMS"
	WebServiceTypeGeoJSON WebServiceType = "GEOJSON"
	WebServiceTypeDynamic WebServiceType = "DYNAMIC"
	WebServiceTypeWMTS    WebServiceType = "WMTS"
	WebServiceTypeWFS     WebServiceType = "WFS"
	WebServiceTypeCOVJSON WebServiceType = "COVJSON"
)

type WebServiceVisibleOn string

const (
	WebServiceVisibleOnGraph WebServiceVisibleOn = "GRAPH"
	WebServiceVisibleOnTable WebServiceVisibleOn = "TABLE"
	WebServiceVisibleOnMap   WebServiceVisibleOn = "MAP"
)

type ParameterLocation string

const (
	ParameterLocationPath   ParameterLocation = "PATH"
	ParameterLocationQuery  ParameterLocation = "QUERY"
	ParameterLocationHeader ParameterLocation = "HEADER"
)

// Parameter is shared by top-level, template, and body parameters.
type Parameter struct {
	Position       int
	Name           string
	Label          string
	Description    *string
	Required       bool
	MultipleValues bool
	ReadOnly       bool
	Schema         ParameterSchema
}

type WebServiceParameterInput struct {
	Parameter
	Location ParameterLocation
	Explode  bool
}

type WebServiceParameter struct {
	ID uuid.UUID
	Parameter
	Location ParameterLocation
	Explode  bool
}

type RequestBody struct {
	ContentType string
	Template    string
	Parameters  []Parameter
}

type WebServiceInput struct {
	Name             string
	Description      string
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

type WebService struct {
	ID               uuid.UUID
	Name             string
	Description      string
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
```

`WebServiceInput` and `WebService` intentionally declare their fields
explicitly. The persisted type does not embed the input type and does not
redeclare or shadow its `Parameters` field. The input contains parameters
without IDs; the persisted type contains parameters with database-generated
IDs.

`ParameterSchema` uses a closed interface. The `type` value in JSON and the
`Kind` value in Go are the discriminators for the possible schema variants:

```go
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

type StringSchema struct {
	Pattern *string
	Enum    []string
	Default *string
}

func (StringSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindString }
func (StringSchema) parameterSchema()          {}

type BooleanSchema struct {
	Default bool
}

func (BooleanSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindBoolean }
func (BooleanSchema) parameterSchema()          {}

type IntegerSchema struct {
	Minimum *int64
	Maximum *int64
	Enum    []int64
	Default *int64
}

func (IntegerSchema) Kind() ParameterSchemaKind { return ParameterSchemaKindInteger }
func (IntegerSchema) parameterSchema()           {}

type FloatSchema struct {
	Minimum *float64
	Maximum *float64
	Enum    []float64
	Default *float64
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
```

`FloatSchema` is both the domain and external representation name for floating
point values. Its schema type is `float`, matching the existing
`FloatParameter` terminology.

The variants are handled with a type switch at validation and persistence
boundaries:

```go
func validateParameterSchema(value ParameterSchema, allowTemplate bool) error {
	switch schema := value.(type) {
	case StringSchema:
		return validateStringSchema(schema)
	case BooleanSchema:
		return nil
	case IntegerSchema:
		return validateIntegerSchema(schema)
	case FloatSchema:
		return validateFloatSchema(schema)
	case DateTimeSchema:
		return validateDateTimeSchema(schema)
	case TemplateSchema:
		if !allowTemplate {
			return ErrTemplateNotAllowed
		}
		return validateTemplateSchema(schema)
	default:
		return ErrInvalidParameterSchema
	}
}
```

Domain constructors parse caller-supplied values into canonical domain values.
They return an error instead of allowing invalid input to reach persistence or
execution code:

```go
func NewWebServiceInput(value WebServiceInput) (WebServiceInput, error)
func NewWebService(value WebService) (WebService, error)

func NewWebServiceParameterInput(value WebServiceParameterInput) (WebServiceParameterInput, error)
func NewWebServiceParameter(value WebServiceParameter) (WebServiceParameter, error)

func NewRequestBody(value RequestBody) (RequestBody, error)

func parseParameter(value Parameter, allowTemplate bool) (Parameter, error)
func parseParameters(values []Parameter, allowTemplate bool) ([]Parameter, error)
func parseParameterSchema(value ParameterSchema, allowTemplate bool) (ParameterSchema, error)
```

`NewWebServiceInput` parses the existing web-service fields, top-level
parameters, and optional body. It validates path placeholders against the
top-level path parameters. `NewWebService` revalidates persisted records after
database reads, including the web-service ID, top-level parameter IDs, and
resolved provider and contact-point references.

`NewWebServiceParameterInput` parses the embedded `Parameter` with templates
allowed and requires a valid `Location`. `NewWebServiceParameter` additionally
requires a non-nil ID, then reuses the input parsing rules.

`NewRequestBody` trims and requires `ContentType` and `Template`, parses the
Go-template syntax without a custom function map, and parses body parameters
with templates disallowed.

`parseParameter` trims and requires `Name` and `Label`, normalizes an optional
description, and parses its schema. `parseParameters` parses every item,
rejects duplicate names, then returns parameters ordered by `Position` and
`Name`.

`parseParameterSchema` handles the schema-specific rules:

- `StringSchema`: compile `Pattern` when present; validate enum and default.
- `BooleanSchema`: default is always available because it is a required `bool`.
- `IntegerSchema`: validate bounds, enum values, and default.
- `FloatSchema`: validate finite bounds, enum values, and default.
- `DateTimeSchema`: validate bounds, default, and layout tokens.
- `TemplateSchema`: require a non-blank template, parse it, and parse nested parameters with templates disallowed.

Top-level web-service parameters call `parseParameterSchema` with
`allowTemplate=true`. Template and body parameter arrays call it with
`allowTemplate=false`.

The constructors define the valid domain boundary: values returned by a
`New...` or `parse...` function are valid and canonical. Exported Go structs
can still be made invalid with direct literals, so transport and persistence
mappers must always call these constructors before treating a value as a
domain object.

Each scope validates unique names and orders parameters by `Position`, then
`Name`. A nil enum means that the enum field was omitted; an empty enum is
invalid because it would allow no values. Boolean schemas always have a usable
`Default` value, including `false`.

The external schema type values are:

- `string`
- `boolean`
- `integer`
- `float`
- `datetime`
- `template`

Date-time values use the EPOS schema type `datetime`.
The `default`, `minimum`, and `maximum` values are stored as RFC3339 strings.
The optional `layout` field specifies how a validated date-time is serialized
for a remote service. If `layout` is omitted, RFC3339 serialization is used.

`layout` uses a platform-level token syntax, not a Go layout. The application
converts the tokens to a Go layout before formatting the date-time:

| Token  | Go layout |
| ------ | --------- |
| `YYYY` | `2006`    |
| `MM`   | `01`      |
| `DD`   | `02`      |
| `hh`   | `15`      |
| `mm`   | `04`      |
| `ss`   | `05`      |

The initial syntax treats characters not listed as tokens as literals. For
example, `YYYY-MM-DDThh:mm:ssZ` becomes
`2006-01-02T15:04:05Z`. The application must reject empty layouts and layouts
containing unsupported tokens.

Use `enum` for allowed values. Omit `enum` when values are unrestricted; do
not store an empty enum array. If `default` is present, it must have the
schema's type and must be one of the enum values.

**STRING**

```json
{
  "type": "string",
  "pattern": "^[A-Za-z0-9]+$",
  "enum": ["mystation", "mystation2"],
  "default": "mystation"
}
```

`pattern` is optional. Its regular-expression dialect must be fixed by the API
contract; the current server implementation should use patterns compatible
with Go's RE2-based regexp package.

**BOOLEAN**

```json
{
  "type": "boolean",
  "default": false
}
```

The `default` field is required for boolean schemas. It pre-fills the form,
and the value is sent unchanged if the user does not edit it. The parameter's
`required` field remains separate: it describes whether the remote service
requires the parameter, not whether the form needs a default.

**INTEGER**

```json
{
  "type": "integer",
  "minimum": 0,
  "maximum": 20,
  "enum": [5, 10],
  "default": 5
}
```

`minimum` and `maximum` are inclusive. Every enum value must satisfy both
bounds when bounds are present. `IntegerSchema` maps to Go `int64`.

**FLOAT**

```json
{
  "type": "float",
  "minimum": 5,
  "maximum": 200,
  "default": 105.5
}
```

`FloatSchema` maps to Go `float64`. Numeric values must be finite.

**DATETIME**

```json
{
  "type": "datetime",
  "layout": "YYYY-MM-DDThh:mm:ssZ",
  "minimum": "1990-01-01T00:00:00Z",
  "maximum": "2026-01-01T00:00:00Z",
  "default": "1998-01-01T00:00:00Z"
}
```

For date-time schemas, `minimum` and `maximum` are EPOS semantic constraints
on parsed timestamps. They are not standard JSON Schema numeric bounds. All
date-time values must be valid RFC3339 strings, and the minimum must not be
after the maximum. `layout` affects only remote-service serialization; it does
not change how these metadata values are parsed or validated.

**TEMPLATE**

```json
{
  "type": "template",
  "template": "{(geom,{{.minLatitude}},{{.maxLatitude}},{{.minLongitude}},{{.maxLongitude}})}",
  "parameters": [
    {
      "name": "minLatitude",
      "position": 0,
      "label": "Minimum latitude",
      "description": "Lower latitude bound.",
      "required": true,
      "multiple_values": false,
      "read_only": false,
      "schema": {
        "type": "float",
        "minimum": -90,
        "maximum": 90
      }
    },
    {
      "name": "maxLatitude",
      "position": 1,
      "label": "Maximum latitude",
      "description": "Upper latitude bound.",
      "required": true,
      "multiple_values": false,
      "read_only": false,
      "schema": {
        "type": "float",
        "minimum": -90,
        "maximum": 90
      }
    }
  ]
}
```

Template parameter names are unique within the template. Nested parameters do
not have IDs or locations. Recursive templates are not supported.

Allowed values are stored in typed array columns:

```json
{
  "type": "string",
  "enum": ["value-1", "value-2", "value-3"]
}
```

This avoids one database row per allowed value while preserving PostgreSQL and
Go element types.

This is efficient for the current access pattern, where the complete
parameter definition is returned and parameter values are not queried in SQL.
Large enums still consume database I/O, network bandwidth, and application
memory when fetched. No index should be added on enum arrays unless a real SQL
membership query appears.

The following are deferred until the execution design:

- URL construction and HTTP request execution.
- Authentication and credential storage.
- Body-template rendering and rendered media-type validation.
- Response-header persistence.
- Array and multi-value parameter serialization.
- Additional custom schema types.
