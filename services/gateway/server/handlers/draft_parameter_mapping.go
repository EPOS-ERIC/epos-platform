package handlers

import (
	"time"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toClientParameterDraftMetadata(v gatewayapi.ParameterDraftMetadata) metadatalifecycleclient.ParameterDraftMetadata {
	return metadatalifecycleclient.ParameterDraftMetadata{
		Kind:           metadatalifecycleclient.ParameterKind(v.Kind),
		Variable:       v.Variable,
		Property:       v.Property,
		Label:          v.Label,
		Required:       v.Required,
		MultipleValues: v.MultipleValues,
		ReadOnly:       v.ReadOnly,
	}
}

func toClientParameterDraft(v gatewayapi.ParameterDraft) metadatalifecycleclient.ParameterDraft {
	switch {
	case v.IsStringParameterDraft():
		x := v.StringParameterDraft
		return metadatalifecycleclient.StringParameterDraftAsParameterDraft(&metadatalifecycleclient.StringParameterDraft{
			Kind:          "STRING",
			Metadata:      toClientParameterDraftMetadata(x.Metadata),
			Default:       *metadatalifecycleclient.NewNullableString(toClientNilStringPtr(x.Default)),
			AllowedValues: x.AllowedValues,
			Pattern:       x.Pattern,
		})
	case v.IsBooleanParameterDraft():
		x := v.BooleanParameterDraft
		return metadatalifecycleclient.BooleanParameterDraftAsParameterDraft(&metadatalifecycleclient.BooleanParameterDraft{
			Kind:     "BOOLEAN",
			Metadata: toClientParameterDraftMetadata(x.Metadata),
			Default:  *metadatalifecycleclient.NewNullableBool(toClientNilBool(x.Default)),
		})
	case v.IsIntegerParameterDraft():
		x := v.IntegerParameterDraft
		return metadatalifecycleclient.IntegerParameterDraftAsParameterDraft(&metadatalifecycleclient.IntegerParameterDraft{
			Kind:          "INTEGER",
			Metadata:      toClientParameterDraftMetadata(x.Metadata),
			Minimum:       *metadatalifecycleclient.NewNullableInt64(toClientNilInt64(x.Minimum)),
			Maximum:       *metadatalifecycleclient.NewNullableInt64(toClientNilInt64(x.Maximum)),
			Default:       *metadatalifecycleclient.NewNullableInt64(toClientNilInt64(x.Default)),
			AllowedValues: x.AllowedValues,
		})
	case v.IsFloatParameterDraft():
		x := v.FloatParameterDraft
		return metadatalifecycleclient.FloatParameterDraftAsParameterDraft(&metadatalifecycleclient.FloatParameterDraft{
			Kind:          "FLOAT",
			Metadata:      toClientParameterDraftMetadata(x.Metadata),
			Minimum:       *metadatalifecycleclient.NewNullableFloat64(toClientNilFloat64(x.Minimum)),
			Maximum:       *metadatalifecycleclient.NewNullableFloat64(toClientNilFloat64(x.Maximum)),
			Default:       *metadatalifecycleclient.NewNullableFloat64(toClientNilFloat64(x.Default)),
			AllowedValues: x.AllowedValues,
		})
	case v.IsDateTimeParameterDraft():
		x := v.DateTimeParameterDraft
		return metadatalifecycleclient.DateTimeParameterDraftAsParameterDraft(&metadatalifecycleclient.DateTimeParameterDraft{
			Kind:     "DATETIME",
			Metadata: toClientParameterDraftMetadata(x.Metadata),
			Minimum:  *metadatalifecycleclient.NewNullableTime(toClientNilTime(x.Minimum)),
			Maximum:  *metadatalifecycleclient.NewNullableTime(toClientNilTime(x.Maximum)),
			Default:  *metadatalifecycleclient.NewNullableTime(toClientNilTime(x.Default)),
			Pattern:  x.Pattern,
		})
	default:
		return metadatalifecycleclient.ParameterDraft{}
	}
}

func toClientNilStringPtr(v gatewayapi.NilString) *string {
	if v.Null {
		return nil
	}
	return &v.Value
}

func toClientNilBool(v gatewayapi.NilBool) *bool {
	if v.Null {
		return nil
	}
	return &v.Value
}

func toClientNilInt64(v gatewayapi.NilInt64) *int64 {
	if v.Null {
		return nil
	}
	return &v.Value
}

func toClientNilFloat64(v gatewayapi.NilFloat64) *float64 {
	if v.Null {
		return nil
	}
	return &v.Value
}

func toClientNilTime(v gatewayapi.NilDateTime) *time.Time {
	if v.Null {
		return nil
	}
	return &v.Value
}
