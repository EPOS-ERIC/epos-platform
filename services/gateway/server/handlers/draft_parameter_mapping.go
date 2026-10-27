package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoParameterDraft(v gatewayapi.ParameterDraft) *metadatalifecyclev1.ParameterInput {
	result := &metadatalifecyclev1.ParameterInput{}
	switch {
	case v.IsStringParameterDraft():
		x := v.StringParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Role = toProtoParameterRole(x.Role)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ParameterInput_String_{String_: &metadatalifecyclev1.StringParameter{
			Default:        protoOptionalString(x.Default),
			Enum:           toProtoStringEnumOptions(x.Enum),
			Pattern:        protoOptionalString(x.Pattern),
			MultipleValues: x.MultipleValues,
		}}
	case v.IsBooleanParameterDraft():
		x := v.BooleanParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Role = toProtoParameterRole(x.Role)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ParameterInput_Boolean{Boolean: &metadatalifecyclev1.BooleanParameter{Default: protoOptionalBool(x.Default)}}
	case v.IsIntegerParameterDraft():
		x := v.IntegerParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Role = toProtoParameterRole(x.Role)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ParameterInput_Integer{Integer: &metadatalifecyclev1.IntegerParameter{
			Minimum:        protoOptionalInt64(x.Minimum),
			Maximum:        protoOptionalInt64(x.Maximum),
			Default:        protoOptionalInt64(x.Default),
			Enum:           toProtoIntegerEnumOptions(x.Enum),
			MultipleValues: x.MultipleValues,
		}}
	case v.IsFloatParameterDraft():
		x := v.FloatParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Role = toProtoParameterRole(x.Role)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ParameterInput_Float{Float: &metadatalifecyclev1.FloatParameter{
			Minimum:        protoOptionalFloat(x.Minimum),
			Maximum:        protoOptionalFloat(x.Maximum),
			Default:        protoOptionalFloat(x.Default),
			Enum:           toProtoFloatEnumOptions(x.Enum),
			MultipleValues: x.MultipleValues,
		}}
	case v.IsDateTimeParameterDraft():
		x := v.DateTimeParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Role = toProtoParameterRole(x.Role)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ParameterInput_DateTime{DateTime: &metadatalifecyclev1.DateTimeParameter{
			Minimum: protoOptionalTime(x.Minimum),
			Maximum: protoOptionalTime(x.Maximum),
			Default: protoOptionalTime(x.Default),
			Layout:  protoOptionalString(x.Layout),
		}}
	case v.IsTemplateParameterDraft():
		x := v.TemplateParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Role = toProtoParameterRole(x.Role)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		template := &metadatalifecyclev1.TemplateParameterInput{Template: x.Template}
		for _, parameter := range x.Parameters {
			template.Parameters = append(template.Parameters, toProtoScalarParameterDraft(parameter))
		}
		result.Value = &metadatalifecyclev1.ParameterInput_Template{Template: template}
	default:
		return nil
	}
	return result
}

func toProtoScalarParameterDraft(v gatewayapi.ScalarParameterDraft) *metadatalifecyclev1.ScalarParameterInput {
	result := &metadatalifecyclev1.ScalarParameterInput{}
	switch {
	case v.IsStringScalarParameterDraft():
		x := v.StringScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameterInput_String_{String_: &metadatalifecyclev1.ScalarStringParameter{
			Default: protoOptionalString(x.Default),
			Enum:    toProtoStringEnumOptions(x.Enum),
			Pattern: protoOptionalString(x.Pattern),
		}}
	case v.IsBooleanScalarParameterDraft():
		x := v.BooleanScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameterInput_Boolean{Boolean: &metadatalifecyclev1.ScalarBooleanParameter{Default: protoOptionalBool(x.Default)}}
	case v.IsIntegerScalarParameterDraft():
		x := v.IntegerScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameterInput_Integer{Integer: &metadatalifecyclev1.ScalarIntegerParameter{
			Minimum: protoOptionalInt64(x.Minimum),
			Maximum: protoOptionalInt64(x.Maximum),
			Default: protoOptionalInt64(x.Default),
			Enum:    toProtoIntegerEnumOptions(x.Enum),
		}}
	case v.IsFloatScalarParameterDraft():
		x := v.FloatScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Required = x.Required
		result.Description = protoOptionalString(x.Description)
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameterInput_Float{Float: &metadatalifecyclev1.ScalarFloatParameter{
			Minimum: protoOptionalFloat(x.Minimum),
			Maximum: protoOptionalFloat(x.Maximum),
			Default: protoOptionalFloat(x.Default),
			Enum:    toProtoFloatEnumOptions(x.Enum),
		}}
	case v.IsDateTimeScalarParameterDraft():
		x := v.DateTimeScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameterInput_DateTime{DateTime: &metadatalifecyclev1.ScalarDateTimeParameter{
			Minimum: protoOptionalTime(x.Minimum),
			Maximum: protoOptionalTime(x.Maximum),
			Default: protoOptionalTime(x.Default),
			Layout:  protoOptionalString(x.Layout),
		}}
	}
	return result
}

func protoOptionalString(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func toProtoStringEnumOptions(values []gatewayapi.StringEnumOption) []*metadatalifecyclev1.StringEnumOption {
	if values == nil {
		return nil
	}
	result := make([]*metadatalifecyclev1.StringEnumOption, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.StringEnumOption{Value: value.Value, Label: protoOptionalEnumLabel(value.Label)}
	}
	return result
}

func toProtoIntegerEnumOptions(values []gatewayapi.IntegerEnumOption) []*metadatalifecyclev1.IntegerEnumOption {
	if values == nil {
		return nil
	}
	result := make([]*metadatalifecyclev1.IntegerEnumOption, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.IntegerEnumOption{Value: value.Value, Label: protoOptionalEnumLabel(value.Label)}
	}
	return result
}

func toProtoFloatEnumOptions(values []gatewayapi.FloatEnumOption) []*metadatalifecyclev1.FloatEnumOption {
	if values == nil {
		return nil
	}
	result := make([]*metadatalifecyclev1.FloatEnumOption, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.FloatEnumOption{Value: value.Value, Label: protoOptionalEnumLabel(value.Label)}
	}
	return result
}

func protoOptionalEnumLabel(value gatewayapi.OptString) *string {
	if !value.Set {
		return nil
	}
	return new(value.Value)
}

func toAPIStringEnumOptions(values []*metadatalifecyclev1.StringEnumOption) ([]gatewayapi.StringEnumOption, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]gatewayapi.StringEnumOption, len(values))
	for i, value := range values {
		if value == nil {
			return nil, fmt.Errorf("upstream string enum option %d is nil", i)
		}
		result[i] = gatewayapi.StringEnumOption{Value: value.Value, Label: apiEnumLabel(value.Label)}
	}
	return result, nil
}

func toAPIIntegerEnumOptions(values []*metadatalifecyclev1.IntegerEnumOption) ([]gatewayapi.IntegerEnumOption, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]gatewayapi.IntegerEnumOption, len(values))
	for i, value := range values {
		if value == nil {
			return nil, fmt.Errorf("upstream integer enum option %d is nil", i)
		}
		result[i] = gatewayapi.IntegerEnumOption{Value: value.Value, Label: apiEnumLabel(value.Label)}
	}
	return result, nil
}

func toAPIFloatEnumOptions(values []*metadatalifecyclev1.FloatEnumOption) ([]gatewayapi.FloatEnumOption, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]gatewayapi.FloatEnumOption, len(values))
	for i, value := range values {
		if value == nil {
			return nil, fmt.Errorf("upstream float enum option %d is nil", i)
		}
		result[i] = gatewayapi.FloatEnumOption{Value: value.Value, Label: apiEnumLabel(value.Label)}
	}
	return result, nil
}

func apiEnumLabel(value *string) gatewayapi.OptString {
	if value == nil {
		return gatewayapi.OptString{}
	}
	return gatewayapi.NewOptString(*value)
}

func toProtoParameterLocation(value gatewayapi.ParameterLocation) metadatalifecyclev1.WebServiceParameterLocation {
	switch value {
	case gatewayapi.ParameterLocationPATH:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_PATH
	case gatewayapi.ParameterLocationQUERY:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_QUERY
	case gatewayapi.ParameterLocationHEADER:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_HEADER
	default:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_UNSPECIFIED
	}
}

func toProtoParameterRole(value gatewayapi.OptNilParameterRole) metadatalifecyclev1.WebServiceParameterRole {
	role, ok := value.Get()
	if !ok {
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_UNSPECIFIED
	}
	switch role {
	case gatewayapi.ParameterRoleFORMAT:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_FORMAT
	case gatewayapi.ParameterRoleSTARTDATE:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_START_DATE
	case gatewayapi.ParameterRoleENDDATE:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_END_DATE
	case gatewayapi.ParameterRoleMINLON:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LON
	case gatewayapi.ParameterRoleMAXLON:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LON
	case gatewayapi.ParameterRoleMINLAT:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LAT
	case gatewayapi.ParameterRoleMAXLAT:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LAT
	default:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_UNSPECIFIED
	}
}

func protoOptionalBool(v gatewayapi.OptNilBool) *bool {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func protoOptionalInt64(v gatewayapi.OptNilInt64) *int64 {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func protoOptionalFloat(v gatewayapi.OptNilFloat64) *float64 {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func protoOptionalTime(v gatewayapi.OptNilDateTime) *timestamppb.Timestamp {
	if !v.Set || v.Null {
		return nil
	}
	return timestamppb.New(v.Value)
}
