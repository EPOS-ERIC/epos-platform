package dataset

import (
	"reflect"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func TestParameterMappingRoundTrip(t *testing.T) {
	minimum := int64(1)
	maximum := int64(10)
	floatDefault := 1.5
	booleanDefault := false
	date := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	description := "parameter description"
	stringDefault := "json"
	pattern := "^[a-z]+$"
	layout := "YYYY-MM-DD"
	tests := []struct {
		name  string
		value domain.ParameterInput
	}{
		{
			name: "string",
			value: domain.ParameterInput{
				Position:    -1,
				Name:        "string-value",
				Label:       "String value",
				Description: &description,
				Required:    true,
				Schema: domain.StringSchema{
					Default:        &stringDefault,
					Enum:           []domain.StringEnumOption{{Value: "json", Label: new("JSON")}, {Value: "xml"}},
					Pattern:        &pattern,
					MultipleValues: true,
				},
			},
		},
		{
			name: "boolean",
			value: domain.ParameterInput{
				Position: 2,
				Name:     "enabled",
				Label:    "Enabled",
				Schema:   domain.BooleanSchema{Default: &booleanDefault},
			},
		},
		{
			name: "integer",
			value: domain.ParameterInput{
				Position: 3,
				Name:     "count",
				Label:    "Count",
				Schema:   domain.IntegerSchema{Minimum: &minimum, Maximum: &maximum, Default: &minimum, Enum: []domain.IntegerEnumOption{{Value: 1}, {Value: 5}, {Value: 10}}},
			},
		},
		{
			name: "float",
			value: domain.ParameterInput{
				Position: 4,
				Name:     "ratio",
				Label:    "Ratio",
				Schema:   domain.FloatSchema{Default: &floatDefault, Enum: []domain.FloatEnumOption{{Value: 1.5}, {Value: 2.5}}},
			},
		},
		{
			name: "datetime",
			value: domain.ParameterInput{
				Position: 5,
				Name:     "when",
				Label:    "When",
				Schema:   domain.DateTimeSchema{Default: &date, Layout: &layout},
			},
		},
		{
			name: "template",
			value: domain.ParameterInput{
				Position: 6,
				Name:     "filter",
				Label:    "Filter",
				Schema: domain.TemplateSchemaInput{
					Template: "{{.value}}",
					Parameters: []domain.ParameterInput{{
						Position: 0,
						Name:     "value",
						Label:    "Value",
						Schema:   domain.StringSchema{},
					}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params, err := parameterParams(test.value, parameterOwner{}, nil, nil, nil, false)
			if err != nil {
				t.Fatalf("parameterParams() error = %v", err)
			}
			row := parameterRow(params)
			var template *sqlc.ParameterTemplate
			var children []sqlc.Parameter
			if schema, ok := test.value.Schema.(domain.TemplateSchemaInput); ok {
				template = &sqlc.ParameterTemplate{ParameterID: row.ID, Template: schema.Template}
				children = []sqlc.Parameter{parameterRowForSchema(schema.Parameters[0], row.ID)}
			}
			schemas := map[uuid.UUID]parameterSchemas{row.ID: parameterSchemaRow(test.value, row.ID)}
			if len(children) > 0 {
				schemas[children[0].ID] = parameterSchemaRow(test.value.Schema.(domain.TemplateSchemaInput).Parameters[0], children[0].ID)
			}
			got, err := domainParameter(row, template, children, schemas)
			if err != nil {
				t.Fatalf("domainParameter() error = %v", err)
			}
			if !reflect.DeepEqual(got.Input(), test.value) {
				t.Fatalf("parameter = %#v, want %#v", got.Input(), test.value)
			}
		})
	}
}

func parameterRow(value sqlc.AddLifecycleParameterParams) sqlc.Parameter {
	return sqlc.Parameter{
		ID:                         uuid.New(),
		Position:                   value.Position,
		Name:                       value.Name,
		Label:                      value.Label,
		Description:                value.Description,
		Required:                   value.Required,
		ReadOnly:                   value.ReadOnly,
		WebServiceID:               value.WebServiceID,
		WmsGetMapServiceID:         value.WmsGetMapServiceID,
		WmsGetFeatureInfoServiceID: value.WmsGetFeatureInfoServiceID,
		WebServiceBodyID:           value.WebServiceBodyID,
		TemplateParameterID:        value.TemplateParameterID,
		Role:                       value.Role,
		Location:                   value.Location,
		Explode:                    value.Explode,
		SchemaKind:                 value.SchemaKind,
	}
}

func parameterRowForSchema(value domain.ParameterInput, parentID uuid.UUID) sqlc.Parameter {
	params, err := parameterParams(value, parameterOwner{templateParameterID: &parentID}, nil, nil, nil, false)
	if err != nil {
		panic(err)
	}
	return parameterRow(params)
}

func parameterSchemaRow(value domain.ParameterInput, id uuid.UUID) parameterSchemas {
	result := parameterSchemas{}
	switch schema := value.Schema.(type) {
	case domain.StringSchema:
		result.string = sqlc.StringParameterSchema{ParameterID: id, Pattern: schema.Pattern, DefaultValue: schema.Default, Enum: toStringEnumOptions(schema.Enum), MultipleValues: schema.MultipleValues}
		result.stringOK = true
	case domain.BooleanSchema:
		result.boolean = sqlc.BooleanParameterSchema{ParameterID: id, DefaultValue: schema.Default}
		result.booleanOK = true
	case domain.IntegerSchema:
		result.integer = sqlc.IntegerParameterSchema{ParameterID: id, Minimum: schema.Minimum, Maximum: schema.Maximum, DefaultValue: schema.Default, Enum: toIntegerEnumOptions(schema.Enum), MultipleValues: schema.MultipleValues}
		result.integerOK = true
	case domain.FloatSchema:
		result.float = sqlc.FloatParameterSchema{ParameterID: id, Minimum: schema.Minimum, Maximum: schema.Maximum, DefaultValue: schema.Default, Enum: toFloatEnumOptions(schema.Enum), MultipleValues: schema.MultipleValues}
		result.floatOK = true
	case domain.DateTimeSchema:
		result.datetime = sqlc.DatetimeParameterSchema{ParameterID: id, Minimum: schema.Minimum, Maximum: schema.Maximum, DefaultValue: schema.Default, Layout: schema.Layout}
		result.datetimeOK = true
	}
	return result
}

func TestParameterParamsPreservesWebServiceOwner(t *testing.T) {
	serviceID := uuid.New()
	location := domain.ParameterLocationQuery
	params, err := parameterParams(domain.ParameterInput{
		Name:   "value",
		Label:  "Value",
		Schema: domain.StringSchema{},
	}, parameterOwner{webServiceID: &serviceID}, nil, nil, &location, true)
	if err != nil {
		t.Fatalf("parameterParams() error = %v", err)
	}
	if params.WebServiceID == nil || *params.WebServiceID != serviceID || params.Location == nil || *params.Location != sqlc.WebServiceParameterLocationQUERY || !params.Explode {
		t.Fatalf("parameter owner fields = %+v", params)
	}
}

func TestWebServicePersistenceMappingPreservesTypeAndVisibleOn(t *testing.T) {
	id := uuid.New()
	service := toDomainWebService(sqlc.WebService{
		ID:        id,
		Name:      "service",
		Type:      sqlc.WebServiceTypeHTTP,
		VisibleOn: []sqlc.WebServiceVisibleOn{sqlc.WebServiceVisibleOnMAP, sqlc.WebServiceVisibleOnGRAPH},
	})
	if service.Type != domain.WebServiceTypeHTTP || !reflect.DeepEqual(service.VisibleOn, []domain.WebServiceVisibleOn{domain.WebServiceVisibleOnMap, domain.WebServiceVisibleOnGraph}) {
		t.Fatalf("web service fields = type %q, visible on %v", service.Type, service.VisibleOn)
	}
	if visibleOnValues(nil) != nil {
		t.Fatal("visibleOnValues(nil) is non-nil")
	}
}
