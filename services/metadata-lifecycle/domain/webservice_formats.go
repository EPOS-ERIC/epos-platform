package domain

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

type NativeFormat struct {
	OutputFormat string
	Label        string
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
				return apperror.Field(field, "DYNAMIC services require exactly one LONGITUDE and one LATITUDE parameter.", nil)
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
		return apperror.Field(target+".role", "Only one parameter per role is allowed.", nil)
	}
	seen[role] = struct{}{}
	if parameter.Schema.Kind() == ParameterSchemaKindTemplate {
		return apperror.Field(target+".role", "Roles are only supported on non-template parameters.", nil)
	}
	switch role {
	case WebServiceParameterRoleFormat:
		schema, ok := parameter.Schema.(StringSchema)
		if !ok {
			return apperror.Field(target+".kind", "FORMAT role requires a STRING parameter.", nil)
		}
		if schema.MultipleValues {
			return apperror.Field(target+".multiple_values", "FORMAT role requires a single-value parameter.", nil)
		}
		if !parameter.ReadOnly {
			return apperror.Field(target+".read_only", "FORMAT role requires a read-only parameter.", nil)
		}
		if len(schema.Default) == 0 {
			return apperror.Field(target+".default", "FORMAT role requires a default.", nil)
		}
	case WebServiceParameterRoleStartDate, WebServiceParameterRoleEndDate:
		if _, ok := parameter.Schema.(DateTimeSchema); !ok {
			return apperror.Field(target+".kind", "Date roles require a DATETIME parameter.", nil)
		}
	case WebServiceParameterRoleMinLon, WebServiceParameterRoleMaxLon,
		WebServiceParameterRoleMinLat, WebServiceParameterRoleMaxLat:
		switch parameter.Schema.(type) {
		case IntegerSchema, FloatSchema:
			if parameter.Schema.AllowsMultipleValues() {
				return apperror.Field(target+".multiple_values", "Coordinate roles require a single-value parameter.", nil)
			}
		default:
			return apperror.Field(target+".kind", "Coordinate roles require an INTEGER or FLOAT parameter.", nil)
		}
	case WebServiceParameterRoleLongitude, WebServiceParameterRoleLatitude:
		if serviceType != WebServiceTypeDynamic {
			return apperror.Field(target+".role", "LONGITUDE and LATITUDE roles are only supported on DYNAMIC services.", nil)
		}
		schema, ok := parameter.Schema.(FloatSchema)
		if !ok {
			return apperror.Field(target+".kind", "LONGITUDE and LATITUDE roles require a FLOAT parameter.", nil)
		}
		if !parameter.Required {
			return apperror.Field(target+".required", "DYNAMIC coordinate parameters must be required.", nil)
		}
		if parameter.ReadOnly {
			return apperror.Field(target+".read_only", "DYNAMIC coordinate parameters must be editable.", nil)
		}
		if schema.MultipleValues {
			return apperror.Field(target+".multiple_values", "DYNAMIC coordinate parameters must have a single value.", nil)
		}
		if len(schema.Default) != 0 {
			return apperror.Field(target+".default", "DYNAMIC coordinate parameters must not have a default.", nil)
		}
		if len(schema.Enum) != 0 {
			return apperror.Field(target+".enum", "DYNAMIC coordinate parameters must not define an enum.", nil)
		}
	default:
		return apperror.Field(target+".role", "Role is not supported.", nil)
	}
	return nil
}
