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
	if parameter := value.parameterWithRole(WebServiceParameterRoleFormat); parameter != nil {
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

func (value GenericWebServiceConfiguration) parameterWithRole(role WebServiceParameterRole) *Parameter {
	for i := range value.Parameters {
		parameter := &value.Parameters[i]
		if parameter.Role != nil && *parameter.Role == role {
			return &parameter.Parameter
		}
		if template, ok := parameter.Schema.(TemplateSchema); ok {
			for j := range template.Parameters {
				if template.Parameters[j].Role != nil && *template.Parameters[j].Role == role {
					return &template.Parameters[j]
				}
			}
		}
	}
	if value.Body != nil {
		for i := range value.Body.Parameters {
			if value.Body.Parameters[i].Role != nil && *value.Body.Parameters[i].Role == role {
				return &value.Body.Parameters[i]
			}
		}
	}
	return nil
}

func validateWebServiceParameterRoles(serviceType WebServiceType, parameters []WebServiceParameterInput, field string, body []ParameterInput) error {
	seen := map[WebServiceParameterRole]struct{}{}
	for i, parameter := range parameters {
		target := fmt.Sprintf("%s[%d]", field, i)
		if err := validateParameterRole(serviceType, parameter.ParameterInput, target, seen); err != nil {
			return err
		}
		if template, ok := parameter.Schema.(TemplateSchemaInput); ok {
			for j, child := range template.Parameters {
				if err := validateParameterRole(serviceType, child, fmt.Sprintf("%s.parameters[%d]", target, j), seen); err != nil {
					return err
				}
			}
		}
	}
	for i, parameter := range body {
		if err := validateParameterRole(serviceType, parameter, fmt.Sprintf("body_parameters[%d]", i), seen); err != nil {
			return err
		}
	}
	if serviceType == WebServiceTypeDynamic {
		for _, role := range []WebServiceParameterRole{WebServiceParameterRoleLongitude, WebServiceParameterRoleLatitude} {
			if _, exists := seen[role]; !exists {
				return invalidField(field, "DYNAMIC services require exactly one LONGITUDE and one LATITUDE parameter.", nil)
			}
		}
	}
	return nil
}

func validateParameterRole(serviceType WebServiceType, parameter ParameterInput, target string, seen map[WebServiceParameterRole]struct{}) error {
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
	case WebServiceParameterRoleLongitude, WebServiceParameterRoleLatitude:
		if serviceType != WebServiceTypeDynamic {
			return invalidField(target+".role", "LONGITUDE and LATITUDE roles are only supported on DYNAMIC services.", nil)
		}
		schema, ok := parameter.Schema.(FloatSchema)
		if !ok {
			return invalidField(target+".kind", "LONGITUDE and LATITUDE roles require a FLOAT parameter.", nil)
		}
		if !parameter.Required {
			return invalidField(target+".required", "DYNAMIC coordinate parameters must be required.", nil)
		}
		if parameter.ReadOnly {
			return invalidField(target+".read_only", "DYNAMIC coordinate parameters must be editable.", nil)
		}
		if schema.MultipleValues {
			return invalidField(target+".multiple_values", "DYNAMIC coordinate parameters must have a single value.", nil)
		}
		if schema.Default != nil {
			return invalidField(target+".default", "DYNAMIC coordinate parameters must not have a default.", nil)
		}
		if len(schema.Enum) != 0 {
			return invalidField(target+".enum", "DYNAMIC coordinate parameters must not define an enum.", nil)
		}
	default:
		return invalidField(target+".role", "Role is not supported.", nil)
	}
	return nil
}
