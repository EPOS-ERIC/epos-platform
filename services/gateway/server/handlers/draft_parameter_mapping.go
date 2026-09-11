package handlers

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoParameterDraft(v gatewayapi.ParameterDraft) *metadatalifecyclev1.Parameter {
	result := &metadatalifecyclev1.Parameter{}
	switch {
	case v.IsStringParameterDraft():
		x := v.StringParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.Parameter_String_{String_: &metadatalifecyclev1.StringParameter{
			Default:        protoOptionalString(x.Default),
			Enum:           x.Enum,
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
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.Parameter_Boolean{Boolean: &metadatalifecyclev1.BooleanParameter{Default: protoOptionalBool(x.Default)}}
	case v.IsIntegerParameterDraft():
		x := v.IntegerParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.Property = protoOptionalString(x.Property)
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.Parameter_Integer{Integer: &metadatalifecyclev1.IntegerParameter{
			Minimum:        protoOptionalInt64(x.Minimum),
			Maximum:        protoOptionalInt64(x.Maximum),
			Default:        protoOptionalInt64(x.Default),
			Enum:           x.Enum,
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
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.Parameter_Float{Float: &metadatalifecyclev1.FloatParameter{
			Minimum:        protoOptionalFloat(x.Minimum),
			Maximum:        protoOptionalFloat(x.Maximum),
			Default:        protoOptionalFloat(x.Default),
			Enum:           x.Enum,
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
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.Parameter_DateTime{DateTime: &metadatalifecyclev1.DateTimeParameter{
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
		result.Location = toProtoParameterLocation(x.Location)
		result.Explode = x.Explode
		result.ReadOnly = x.ReadOnly
		template := &metadatalifecyclev1.TemplateParameter{Template: x.Template}
		for _, parameter := range x.Parameters {
			template.Parameters = append(template.Parameters, toProtoScalarParameterDraft(parameter))
		}
		result.Value = &metadatalifecyclev1.Parameter_Template{Template: template}
	default:
		return nil
	}
	return result
}

func toProtoScalarParameterDraft(v gatewayapi.ScalarParameterDraft) *metadatalifecyclev1.ScalarParameter {
	result := &metadatalifecyclev1.ScalarParameter{}
	switch {
	case v.IsStringScalarParameterDraft():
		x := v.StringScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameter_String_{String_: &metadatalifecyclev1.ScalarStringParameter{
			Default: protoOptionalString(x.Default),
			Enum:    x.Enum,
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
		result.Value = &metadatalifecyclev1.ScalarParameter_Boolean{Boolean: &metadatalifecyclev1.ScalarBooleanParameter{Default: protoOptionalBool(x.Default)}}
	case v.IsIntegerScalarParameterDraft():
		x := v.IntegerScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameter_Integer{Integer: &metadatalifecyclev1.ScalarIntegerParameter{
			Minimum: protoOptionalInt64(x.Minimum),
			Maximum: protoOptionalInt64(x.Maximum),
			Default: protoOptionalInt64(x.Default),
			Enum:    x.Enum,
		}}
	case v.IsFloatScalarParameterDraft():
		x := v.FloatScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Required = x.Required
		result.Description = protoOptionalString(x.Description)
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameter_Float{Float: &metadatalifecyclev1.ScalarFloatParameter{
			Minimum: protoOptionalFloat(x.Minimum),
			Maximum: protoOptionalFloat(x.Maximum),
			Default: protoOptionalFloat(x.Default),
			Enum:    x.Enum,
		}}
	case v.IsDateTimeScalarParameterDraft():
		x := v.DateTimeScalarParameterDraft
		result.Position = x.Position
		result.Name = string(x.Name)
		result.Label = string(x.Label)
		result.Description = protoOptionalString(x.Description)
		result.Required = x.Required
		result.ReadOnly = x.ReadOnly
		result.Value = &metadatalifecyclev1.ScalarParameter_DateTime{DateTime: &metadatalifecyclev1.ScalarDateTimeParameter{
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
