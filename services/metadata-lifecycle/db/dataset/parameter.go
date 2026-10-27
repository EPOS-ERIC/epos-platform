package dataset

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/types"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type parameterOwner struct {
	webServiceID                *uuid.UUID
	wfsServiceID                *uuid.UUID
	wmsGetMapServiceID          *uuid.UUID
	wmsGetFeatureInfoServiceID  *uuid.UUID
	wmtsGetTileServiceID        *uuid.UUID
	wmtsGetFeatureInfoServiceID *uuid.UUID
	webServiceBodyID            *uuid.UUID
	templateParameterID         *uuid.UUID
}

func parameterParams(
	value domain.ParameterInput,
	owner parameterOwner,
	property *string,
	role *domain.WebServiceParameterRole,
	location *domain.ParameterLocation,
	explode bool,
) (sqlc.AddLifecycleParameterParams, error) {
	if value.Schema == nil {
		return sqlc.AddLifecycleParameterParams{}, fmt.Errorf("parameter schema is required")
	}

	params := sqlc.AddLifecycleParameterParams{
		WebServiceID:                owner.webServiceID,
		WfsServiceID:                owner.wfsServiceID,
		WmsGetMapServiceID:          owner.wmsGetMapServiceID,
		WmsGetFeatureInfoServiceID:  owner.wmsGetFeatureInfoServiceID,
		WmtsGetTileServiceID:        owner.wmtsGetTileServiceID,
		WmtsGetFeatureInfoServiceID: owner.wmtsGetFeatureInfoServiceID,
		WebServiceBodyID:            owner.webServiceBodyID,
		TemplateParameterID:         owner.templateParameterID,
		Position:                    int32(value.Position),
		Name:                        value.Name,
		Label:                       value.Label,
		Description:                 value.Description,
		Required:                    value.Required,
		ReadOnly:                    value.ReadOnly,
		Property:                    property,
		Explode:                     explode,
	}
	if role != nil {
		params.Role = new(sqlc.WebServiceParameterRole(*role))
	}
	if location != nil {
		converted := sqlc.WebServiceParameterLocation(*location)
		params.Location = &converted
	}

	params.SchemaKind = sqlc.ParameterSchemaKind(value.Schema.Kind())
	return params, nil
}

func persistParameter(
	ctx context.Context,
	q *sqlc.Queries,
	value domain.ParameterInput,
	owner parameterOwner,
	property *string,
	role *domain.WebServiceParameterRole,
	location *domain.ParameterLocation,
	explode bool,
) (sqlc.Parameter, error) {
	params, err := parameterParams(value, owner, property, role, location, explode)
	if err != nil {
		return sqlc.Parameter{}, err
	}
	created, err := q.AddLifecycleParameter(ctx, params)
	if err != nil {
		return sqlc.Parameter{}, fmt.Errorf("add parameter: %w", err)
	}

	var schemaErr error
	switch schema := value.Schema.(type) {
	case domain.StringSchema:
		schemaErr = q.AddLifecycleStringParameterSchema(ctx, sqlc.AddLifecycleStringParameterSchemaParams{
			ParameterID:    created.ID,
			Pattern:        schema.Pattern,
			DefaultValue:   schema.Default,
			Enum:           toStringEnumOptions(schema.Enum),
			MultipleValues: schema.MultipleValues,
		})
	case domain.BooleanSchema:
		schemaErr = q.AddLifecycleBooleanParameterSchema(ctx, sqlc.AddLifecycleBooleanParameterSchemaParams{
			ParameterID:  created.ID,
			DefaultValue: schema.Default,
		})
	case domain.IntegerSchema:
		schemaErr = q.AddLifecycleIntegerParameterSchema(ctx, sqlc.AddLifecycleIntegerParameterSchemaParams{
			ParameterID:    created.ID,
			Minimum:        schema.Minimum,
			Maximum:        schema.Maximum,
			DefaultValue:   schema.Default,
			Enum:           toIntegerEnumOptions(schema.Enum),
			MultipleValues: schema.MultipleValues,
		})
	case domain.FloatSchema:
		schemaErr = q.AddLifecycleFloatParameterSchema(ctx, sqlc.AddLifecycleFloatParameterSchemaParams{
			ParameterID:    created.ID,
			Minimum:        schema.Minimum,
			Maximum:        schema.Maximum,
			DefaultValue:   schema.Default,
			Enum:           toFloatEnumOptions(schema.Enum),
			MultipleValues: schema.MultipleValues,
		})
	case domain.DateTimeSchema:
		schemaErr = q.AddLifecycleDatetimeParameterSchema(ctx, sqlc.AddLifecycleDatetimeParameterSchemaParams{
			ParameterID:  created.ID,
			Minimum:      schema.Minimum,
			Maximum:      schema.Maximum,
			DefaultValue: schema.Default,
			Layout:       schema.Layout,
		})
	case domain.TemplateSchemaInput:
		schemaErr = q.AddLifecycleParameterTemplate(ctx, sqlc.AddLifecycleParameterTemplateParams{
			ParameterID: created.ID,
			Template:    schema.Template,
		})
		if schemaErr != nil {
			break
		}
		for _, child := range schema.Parameters {
			if _, err := persistParameter(ctx, q, child, parameterOwner{templateParameterID: &created.ID}, nil, nil, nil, false); err != nil {
				return sqlc.Parameter{}, fmt.Errorf("add template parameter: %w", err)
			}
		}
	default:
		return sqlc.Parameter{}, fmt.Errorf("unsupported parameter schema %T", value.Schema)
	}
	if schemaErr != nil {
		return sqlc.Parameter{}, fmt.Errorf("add parameter schema: %w", schemaErr)
	}
	return created, nil
}

type parameterSchemas struct {
	string     sqlc.StringParameterSchema
	stringOK   bool
	boolean    sqlc.BooleanParameterSchema
	booleanOK  bool
	integer    sqlc.IntegerParameterSchema
	integerOK  bool
	float      sqlc.FloatParameterSchema
	floatOK    bool
	datetime   sqlc.DatetimeParameterSchema
	datetimeOK bool
}

func toStringEnumOptions(values []domain.StringEnumOption) []types.StringEnumOption {
	if values == nil {
		return nil
	}
	options := make([]types.StringEnumOption, len(values))
	for i, value := range values {
		options[i] = types.StringEnumOption(value)
	}
	return options
}

func fromStringEnumOptions(values []types.StringEnumOption) []domain.StringEnumOption {
	if values == nil {
		return nil
	}
	options := make([]domain.StringEnumOption, len(values))
	for i, value := range values {
		options[i] = domain.StringEnumOption(value)
	}
	return options
}

func toIntegerEnumOptions(values []domain.IntegerEnumOption) []types.IntegerEnumOption {
	if values == nil {
		return nil
	}
	options := make([]types.IntegerEnumOption, len(values))
	for i, value := range values {
		options[i] = types.IntegerEnumOption(value)
	}
	return options
}

func fromIntegerEnumOptions(values []types.IntegerEnumOption) []domain.IntegerEnumOption {
	if values == nil {
		return nil
	}
	options := make([]domain.IntegerEnumOption, len(values))
	for i, value := range values {
		options[i] = domain.IntegerEnumOption(value)
	}
	return options
}

func toFloatEnumOptions(values []domain.FloatEnumOption) []types.FloatEnumOption {
	if values == nil {
		return nil
	}
	options := make([]types.FloatEnumOption, len(values))
	for i, value := range values {
		options[i] = types.FloatEnumOption(value)
	}
	return options
}

func fromFloatEnumOptions(values []types.FloatEnumOption) []domain.FloatEnumOption {
	if values == nil {
		return nil
	}
	options := make([]domain.FloatEnumOption, len(values))
	for i, value := range values {
		options[i] = domain.FloatEnumOption(value)
	}
	return options
}

func readParameterSchemas(ctx context.Context, q *sqlc.Queries, parameterIDs []uuid.UUID) (map[uuid.UUID]parameterSchemas, error) {
	result := make(map[uuid.UUID]parameterSchemas, len(parameterIDs))
	strings, err := q.ListLifecycleStringParameterSchemas(ctx, parameterIDs)
	if err != nil {
		return nil, fmt.Errorf("list string parameter schemas: %w", err)
	}
	for _, row := range strings {
		schemas := result[row.StringParameterSchema.ParameterID]
		schemas.string = row.StringParameterSchema
		schemas.stringOK = true
		result[row.StringParameterSchema.ParameterID] = schemas
	}

	booleans, err := q.ListLifecycleBooleanParameterSchemas(ctx, parameterIDs)
	if err != nil {
		return nil, fmt.Errorf("list boolean parameter schemas: %w", err)
	}
	for _, row := range booleans {
		schemas := result[row.BooleanParameterSchema.ParameterID]
		schemas.boolean = row.BooleanParameterSchema
		schemas.booleanOK = true
		result[row.BooleanParameterSchema.ParameterID] = schemas
	}

	integers, err := q.ListLifecycleIntegerParameterSchemas(ctx, parameterIDs)
	if err != nil {
		return nil, fmt.Errorf("list integer parameter schemas: %w", err)
	}
	for _, row := range integers {
		schemas := result[row.IntegerParameterSchema.ParameterID]
		schemas.integer = row.IntegerParameterSchema
		schemas.integerOK = true
		result[row.IntegerParameterSchema.ParameterID] = schemas
	}

	floats, err := q.ListLifecycleFloatParameterSchemas(ctx, parameterIDs)
	if err != nil {
		return nil, fmt.Errorf("list float parameter schemas: %w", err)
	}
	for _, row := range floats {
		schemas := result[row.FloatParameterSchema.ParameterID]
		schemas.float = row.FloatParameterSchema
		schemas.floatOK = true
		result[row.FloatParameterSchema.ParameterID] = schemas
	}

	datetimes, err := q.ListLifecycleDatetimeParameterSchemas(ctx, parameterIDs)
	if err != nil {
		return nil, fmt.Errorf("list datetime parameter schemas: %w", err)
	}
	for _, row := range datetimes {
		schemas := result[row.DatetimeParameterSchema.ParameterID]
		schemas.datetime = row.DatetimeParameterSchema
		schemas.datetimeOK = true
		result[row.DatetimeParameterSchema.ParameterID] = schemas
	}
	return result, nil
}

func domainParameter(value sqlc.Parameter, template *sqlc.ParameterTemplate, children []sqlc.Parameter, schemas map[uuid.UUID]parameterSchemas) (domain.Parameter, error) {
	if value.SchemaKind != sqlc.ParameterSchemaKindTEMPLATE && len(children) > 0 {
		return domain.Parameter{}, fmt.Errorf("non-template parameter has child parameters")
	}
	parameterSchema := schemas[value.ID]
	var schema domain.ParameterSchema
	switch value.SchemaKind {
	case sqlc.ParameterSchemaKindSTRING:
		if !parameterSchema.stringOK {
			return domain.Parameter{}, fmt.Errorf("string parameter schema is missing")
		}
		schema = domain.StringSchema{
			Pattern:        parameterSchema.string.Pattern,
			Default:        parameterSchema.string.DefaultValue,
			Enum:           fromStringEnumOptions(parameterSchema.string.Enum),
			MultipleValues: parameterSchema.string.MultipleValues,
		}
	case sqlc.ParameterSchemaKindBOOLEAN:
		if !parameterSchema.booleanOK {
			return domain.Parameter{}, fmt.Errorf("boolean parameter schema is missing")
		}
		schema = domain.BooleanSchema{Default: parameterSchema.boolean.DefaultValue}
	case sqlc.ParameterSchemaKindINTEGER:
		if !parameterSchema.integerOK {
			return domain.Parameter{}, fmt.Errorf("integer parameter schema is missing")
		}
		schema = domain.IntegerSchema{
			Minimum:        parameterSchema.integer.Minimum,
			Maximum:        parameterSchema.integer.Maximum,
			Default:        parameterSchema.integer.DefaultValue,
			Enum:           fromIntegerEnumOptions(parameterSchema.integer.Enum),
			MultipleValues: parameterSchema.integer.MultipleValues,
		}
	case sqlc.ParameterSchemaKindFLOAT:
		if !parameterSchema.floatOK {
			return domain.Parameter{}, fmt.Errorf("float parameter schema is missing")
		}
		schema = domain.FloatSchema{
			Minimum:        parameterSchema.float.Minimum,
			Maximum:        parameterSchema.float.Maximum,
			Default:        parameterSchema.float.DefaultValue,
			Enum:           fromFloatEnumOptions(parameterSchema.float.Enum),
			MultipleValues: parameterSchema.float.MultipleValues,
		}
	case sqlc.ParameterSchemaKindDATETIME:
		if !parameterSchema.datetimeOK {
			return domain.Parameter{}, fmt.Errorf("datetime parameter schema is missing")
		}
		schema = domain.DateTimeSchema{
			Minimum: parameterSchema.datetime.Minimum,
			Maximum: parameterSchema.datetime.Maximum,
			Default: parameterSchema.datetime.DefaultValue,
			Layout:  parameterSchema.datetime.Layout,
		}
	case sqlc.ParameterSchemaKindTEMPLATE:
		if template == nil {
			return domain.Parameter{}, fmt.Errorf("parameter template is missing")
		}
		parameters := make([]domain.Parameter, 0, len(children))
		for _, child := range children {
			parameter, err := domainParameter(child, nil, nil, schemas)
			if err != nil {
				return domain.Parameter{}, fmt.Errorf("decode template parameter: %w", err)
			}
			parameters = append(parameters, parameter)
		}
		schema = domain.TemplateSchema{Template: template.Template, Parameters: parameters}
	default:
		return domain.Parameter{}, fmt.Errorf("unsupported parameter schema kind %q", value.SchemaKind)
	}
	return domain.Parameter{
		Position:    int(value.Position),
		Name:        value.Name,
		Label:       value.Label,
		Description: value.Description,
		Required:    value.Required,
		ReadOnly:    value.ReadOnly,
		Schema:      schema,
		ID:          value.ID,
	}, nil
}
