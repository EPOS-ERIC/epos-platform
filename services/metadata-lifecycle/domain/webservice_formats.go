package domain

import "fmt"

type NativeFormat struct {
	OutputFormat string
	Label        string
}

// NativeFormats derives capabilities from validated web-service metadata.
func (value WebService) NativeFormats() []NativeFormat {
	if value.Configuration == nil {
		return nil
	}
	return value.Configuration.NativeFormats()
}

func (value WFSConfiguration) NativeFormats() []NativeFormat {
	result := make([]NativeFormat, len(value.OutputFormats))
	for i, format := range value.OutputFormats {
		result[i] = NativeFormat{OutputFormat: format, Label: format}
	}
	return result
}

func (WMSConfiguration) NativeFormats() []NativeFormat {
	// PNG is a platform map requirement; the client checks provider support.
	return []NativeFormat{{OutputFormat: "image/png", Label: "image/png"}}
}

func (WMTSConfiguration) NativeFormats() []NativeFormat {
	// PNG is a platform map requirement; the client checks provider support.
	return []NativeFormat{{OutputFormat: "image/png", Label: "image/png"}}
}

func (value GenericWebServiceConfiguration) NativeFormats() []NativeFormat {
	var formatSchema *StringSchema
	if parameter := value.formatParameter(); parameter != nil {
		schema := parameter.Schema.(StringSchema)
		formatSchema = &schema
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

func (value GenericWebServiceConfiguration) formatParameter() *Parameter {
	for _, parameter := range value.Parameters {
		if parameter.Role != nil && *parameter.Role == WebServiceParameterRoleFormat {
			return &parameter.Parameter
		}
		if template, ok := parameter.Schema.(TemplateSchema); ok {
			for _, child := range template.Parameters {
				if child.Role != nil && *child.Role == WebServiceParameterRoleFormat {
					return &child
				}
			}
		}
	}
	if value.Body != nil {
		for _, parameter := range value.Body.Parameters {
			if parameter.Role != nil && *parameter.Role == WebServiceParameterRoleFormat {
				return &parameter
			}
		}
	}
	return nil
}

func validateWebServiceParameterRoles(parameters []WebServiceParameterInput, field string, body []ParameterInput) error {
	seen := map[WebServiceParameterRole]struct{}{}
	for i, parameter := range parameters {
		target := fmt.Sprintf("%s[%d]", field, i)
		if err := validateParameterRole(parameter.ParameterInput, target, seen); err != nil {
			return err
		}
		if template, ok := parameter.Schema.(TemplateSchemaInput); ok {
			for j, child := range template.Parameters {
				if err := validateParameterRole(child, fmt.Sprintf("%s.parameters[%d]", target, j), seen); err != nil {
					return err
				}
			}
		}
	}
	for i, parameter := range body {
		if err := validateParameterRole(parameter, fmt.Sprintf("body_parameters[%d]", i), seen); err != nil {
			return err
		}
	}
	return nil
}

func validateParameterRole(parameter ParameterInput, target string, seen map[WebServiceParameterRole]struct{}) error {
	if parameter.Role == nil {
		return nil
	}
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
	return nil
}
