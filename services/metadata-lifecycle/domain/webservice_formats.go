package domain

import "fmt"

type NativeFormat struct {
	OutputFormat string
	Label        string
}

// NativeFormats derives capabilities from validated web-service metadata.
func (value WebService) NativeFormats() []NativeFormat {
	var formatSchema *StringSchema
	for _, parameter := range value.Parameters {
		if parameter.Role != nil && *parameter.Role == WebServiceParameterRoleFormat {
			schema := parameter.Schema.(StringSchema)
			formatSchema = &schema
			break
		}
	}

	if formatSchema == nil {
		return []NativeFormat{{OutputFormat: value.Format, Label: value.Format}}
	}

	if len(formatSchema.Enum) == 0 {
		return []NativeFormat{{OutputFormat: *formatSchema.Default, Label: *formatSchema.Default}}
	}

	result := make([]NativeFormat, 0, len(formatSchema.Enum))
	for _, option := range formatSchema.Enum {
		label := option.Value
		if option.Label != nil {
			label = *option.Label
		}
		result = append(result, NativeFormat{
			OutputFormat: option.Value,
			Label:        label,
		})
	}

	return result
}

func validateWebServiceParameterRoles(parameters []WebServiceParameterInput) error {
	seen := make(map[WebServiceParameterRole]struct{})
	for i, parameter := range parameters {
		if parameter.Role == nil {
			continue
		}
		target := fmt.Sprintf("parameters[%d]", i)
		role := *parameter.Role
		if _, exists := seen[role]; exists {
			return invalidField(target+".role", "Only one parameter per role is allowed.", nil)
		}
		seen[role] = struct{}{}
		if parameter.Schema.Kind() == ParameterSchemaKindTemplate {
			return invalidField(target+".role", "Roles are only supported on non-template parameters.", nil)
		}
		switch role {
		case WebServiceParameterRoleFormat:
			schema, ok := parameter.Schema.(StringSchema)
			if !ok {
				return invalidField(target+".kind", "FORMAT role requires a STRING parameter.", nil)
			}
			if schema.MultipleValues {
				return invalidField(target+".multiple_values", "FORMAT role requires a single-value parameter.", nil)
			}
			if !parameter.ReadOnly {
				return invalidField(target+".read_only", "FORMAT role requires a read-only parameter.", nil)
			}
			if schema.Default == nil {
				return invalidField(target+".default", "FORMAT role requires a default.", nil)
			}
		case WebServiceParameterRoleStartDate, WebServiceParameterRoleEndDate:
			if _, ok := parameter.Schema.(DateTimeSchema); !ok {
				return invalidField(target+".kind", "Date roles require a DATETIME parameter.", nil)
			}
		case WebServiceParameterRoleMinLon, WebServiceParameterRoleMaxLon,
			WebServiceParameterRoleMinLat, WebServiceParameterRoleMaxLat:
			switch parameter.Schema.(type) {
			case IntegerSchema, FloatSchema:
				if parameter.Schema.AllowsMultipleValues() {
					return invalidField(target+".multiple_values", "Coordinate roles require a single-value parameter.", nil)
				}
			default:
				return invalidField(target+".kind", "Coordinate roles require an INTEGER or FLOAT parameter.", nil)
			}
		default:
			return invalidField(target+".role", "Role is not supported.", nil)
		}
	}
	return nil
}
