package domain

import "fmt"

// TODO: use a dedicated column for "role" of a parameter
const encodingFormatProperty = "schema:encodingFormat"

type NativeFormat struct {
	OutputFormat string
	Label        string
}

// NativeFormats derives capabilities from validated web-service metadata.
func (value WebService) NativeFormats() []NativeFormat {
	var formatSchema *StringSchema
	for _, parameter := range value.Parameters {
		if parameter.Property != nil && *parameter.Property == encodingFormatProperty {
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

func validateFormatParameter(parameters []WebServiceParameterInput) error {
	found := false
	for i, parameter := range parameters {
		if parameter.Property == nil || *parameter.Property != encodingFormatProperty {
			continue
		}
		target := fmt.Sprintf("parameters[%d]", i)
		if found {
			return invalidField(target+".property", "Only one encoding-format parameter is allowed.", nil)
		}
		schema, ok := parameter.Schema.(StringSchema)
		if !ok {
			return invalidField(target+".kind", "Encoding-format parameter must be STRING.", nil)
		}
		if schema.MultipleValues {
			return invalidField(target+".multiple_values", "Encoding-format parameter must be single-value.", nil)
		}
		if !parameter.ReadOnly {
			return invalidField(target+".read_only", "Encoding-format parameter must be read only.", nil)
		}
		if schema.Default == nil {
			return invalidField(target+".default", "Encoding-format parameter requires a default.", nil)
		}
		if _, err := parseStringSchema(schema); err != nil {
			return prefixOrField(err, target, target+".default", "Encoding-format parameter definition is invalid.")
		}
		found = true
	}
	return nil
}
