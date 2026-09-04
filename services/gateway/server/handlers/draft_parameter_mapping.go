package handlers

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoParameterDraft(v gatewayapi.ParameterDraft) *metadatalifecyclev1.ParameterDraft {
	switch {
	case v.IsStringParameterDraft():
		x := v.StringParameterDraft
		return &metadatalifecyclev1.ParameterDraft{Metadata: protoParameterMetadata(x.Metadata), Value: &metadatalifecyclev1.ParameterDraft_String_{String_: &metadatalifecyclev1.StringParameterDraft{Default: protoString(x.Default), AllowedValues: x.AllowedValues, Pattern: x.Pattern}}}
	case v.IsBooleanParameterDraft():
		x := v.BooleanParameterDraft
		return &metadatalifecyclev1.ParameterDraft{Metadata: protoParameterMetadata(x.Metadata), Value: &metadatalifecyclev1.ParameterDraft_Boolean{Boolean: &metadatalifecyclev1.BooleanParameterDraft{Default: protoBool(x.Default)}}}
	case v.IsIntegerParameterDraft():
		x := v.IntegerParameterDraft
		return &metadatalifecyclev1.ParameterDraft{Metadata: protoParameterMetadata(x.Metadata), Value: &metadatalifecyclev1.ParameterDraft_Integer{Integer: &metadatalifecyclev1.IntegerParameterDraft{Minimum: protoInt64(x.Minimum), Maximum: protoInt64(x.Maximum), Default: protoInt64(x.Default), AllowedValues: x.AllowedValues}}}
	case v.IsFloatParameterDraft():
		x := v.FloatParameterDraft
		return &metadatalifecyclev1.ParameterDraft{Metadata: protoParameterMetadata(x.Metadata), Value: &metadatalifecyclev1.ParameterDraft_Float{Float: &metadatalifecyclev1.FloatParameterDraft{Minimum: protoFloat(x.Minimum), Maximum: protoFloat(x.Maximum), Default: protoFloat(x.Default), AllowedValues: x.AllowedValues}}}
	case v.IsDateTimeParameterDraft():
		x := v.DateTimeParameterDraft
		return &metadatalifecyclev1.ParameterDraft{Metadata: protoParameterMetadata(x.Metadata), Value: &metadatalifecyclev1.ParameterDraft_DateTime{DateTime: &metadatalifecyclev1.DateTimeParameterDraft{Minimum: protoTime(x.Minimum), Maximum: protoTime(x.Maximum), Default: protoTime(x.Default), Pattern: x.Pattern}}}
	default:
		return nil
	}
}

func protoParameterMetadata(v gatewayapi.ParameterDraftMetadata) *metadatalifecyclev1.ParameterDraftMetadata {
	return &metadatalifecyclev1.ParameterDraftMetadata{Variable: v.Variable, Property: v.Property, Label: v.Label, Required: v.Required, MultipleValues: v.MultipleValues, ReadOnly: v.ReadOnly}
}

func protoString(v gatewayapi.NilString) *string {
	if v.Null {
		return nil
	}
	return new(v.Value)
}

func protoBool(v gatewayapi.NilBool) *bool {
	if v.Null {
		return nil
	}
	return new(v.Value)
}

func protoInt64(v gatewayapi.NilInt64) *int64 {
	if v.Null {
		return nil
	}
	return new(v.Value)
}

func protoFloat(v gatewayapi.NilFloat64) *float64 {
	if v.Null {
		return nil
	}
	return new(v.Value)
}

func protoTime(v gatewayapi.NilDateTime) *timestamppb.Timestamp {
	if v.Null {
		return nil
	}
	return timestamppb.New(v.Value)
}
