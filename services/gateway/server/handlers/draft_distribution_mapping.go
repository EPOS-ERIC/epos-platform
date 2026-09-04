package handlers

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoTemporal(v gatewayapi.OptTemporalCoverage) *metadatalifecyclev1.Temporal {
	if !v.Set {
		return nil
	}
	r := &metadatalifecyclev1.Temporal{}
	if v.Value.Start.Set && !v.Value.Start.Null {
		r.Start = timestamppb.New(v.Value.Start.Value)
	}
	if v.Value.End.Set && !v.Value.End.Null {
		r.End = timestamppb.New(v.Value.End.Value)
	}
	return r
}

func toProtoDocumentation(v gatewayapi.OptDocumentation) *metadatalifecyclev1.Documentation {
	if !v.Set {
		return nil
	}
	r := &metadatalifecyclev1.Documentation{Uri: string(v.Value.URI)}
	if v.Value.Title.Set && !v.Value.Title.Null {
		r.Title = new(v.Value.Title.Value)
	}
	if v.Value.Description.Set && !v.Value.Description.Null {
		r.Description = new(v.Value.Description.Value)
	}
	return r
}

func toProtoWebServiceDraft(v gatewayapi.WebServiceDraft) *metadatalifecyclev1.WebServiceDraft {
	r := &metadatalifecyclev1.WebServiceDraft{Name: string(v.Name), Description: v.Description.Value, TemplateUrl: string(v.TemplateURL), Format: string(v.Format), TemporalCoverage: toProtoTemporal(v.TemporalCoverage), Documentation: toProtoDocumentation(v.Documentation), ContactPoints: uuidStrings(v.ContactPoints)}
	if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
		r.SpatialCoverage = new(v.SpatialCoverage.Value)
	}
	if v.Provider.Set {
		r.Provider = new(v.Provider.Value.String())
	}
	for _, parameter := range v.Parameters {
		r.Parameters = append(r.Parameters, toProtoParameterDraft(parameter))
	}
	return r
}

func toProtoDistributionDraft(v gatewayapi.DistributionDraft) *metadatalifecyclev1.DistributionDraft {
	title, description, license, issued, modified, byteSize := distributionFields(v)
	r := &metadatalifecyclev1.DistributionDraft{Type: string(v.Type), Title: title, Description: description, License: license, Issued: issued, Modified: modified, ByteSize: byteSize}
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		r.Delivery = &metadatalifecyclev1.DistributionDraft_WebService{WebService: toProtoWebServiceDraft(x.WebService)}
		return r
	}
	x := v.DownloadableFileDistributionDraft
	r.Delivery = &metadatalifecyclev1.DistributionDraft_DownloadableFile{DownloadableFile: &metadatalifecyclev1.DownloadableFile{Url: string(x.DownloadableFile.URL), Format: string(x.DownloadableFile.Format)}}
	return r
}

func toProtoDistribution(v gatewayapi.DistributionDraft) *metadatalifecyclev1.Distribution {
	title, description, license, issued, modified, byteSize := distributionFields(v)
	r := &metadatalifecyclev1.Distribution{Type: string(v.Type), Title: title, Description: description, License: license, Issued: issued, Modified: modified, ByteSize: byteSize}
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		r.Delivery = &metadatalifecyclev1.Distribution_WebService{WebService: toProtoWebService(x.WebService)}
		return r
	}

	x := v.DownloadableFileDistributionDraft
	r.Delivery = &metadatalifecyclev1.Distribution_DownloadableFile{DownloadableFile: &metadatalifecyclev1.DownloadableFile{Url: string(x.DownloadableFile.URL), Format: string(x.DownloadableFile.Format)}}
	return r
}

func distributionFields(v gatewayapi.DistributionDraft) (string, string, *string, *timestamppb.Timestamp, *timestamppb.Timestamp, *int64) {
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		return x.Title, x.Description.Value, protoOptionalString(x.License), protoOptionalTime(x.Issued), protoOptionalTime(x.Modified), protoOptionalInt64(x.ByteSize)
	}
	x := v.DownloadableFileDistributionDraft
	return x.Title, x.Description.Value, protoOptionalString(x.License), protoOptionalTime(x.Issued), protoOptionalTime(x.Modified), protoOptionalInt64(x.ByteSize)
}

func protoOptionalString(v gatewayapi.OptNilString) *string {
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

func protoOptionalInt64(v gatewayapi.OptNilInt64) *int64 {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func toProtoWebService(v gatewayapi.WebServiceDraft) *metadatalifecyclev1.WebService {
	r := &metadatalifecyclev1.WebService{Name: string(v.Name), Description: v.Description.Value, TemplateUrl: string(v.TemplateURL), Format: string(v.Format), TemporalCoverage: toProtoTemporal(v.TemporalCoverage), Documentation: toProtoDocumentation(v.Documentation), ContactPoints: uuidStrings(v.ContactPoints)}
	if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
		r.SpatialCoverage = new(v.SpatialCoverage.Value)
	}
	if v.Provider.Set {
		r.Provider = new(v.Provider.Value.String())
	}
	for _, parameter := range v.Parameters {
		r.Parameters = append(r.Parameters, toProtoParameter(parameter))
	}
	return r
}

func toProtoParameter(v gatewayapi.ParameterDraft) *metadatalifecyclev1.Parameter {
	valueMetadata := vMetadata(v)
	metadata := &metadatalifecyclev1.ParameterMetadata{Variable: valueMetadata.Variable, Property: valueMetadata.Property, Label: valueMetadata.Label, Required: valueMetadata.Required, MultipleValues: valueMetadata.MultipleValues, ReadOnly: valueMetadata.ReadOnly, Kind: string(valueMetadata.Kind)}
	result := &metadatalifecyclev1.Parameter{Metadata: metadata}
	switch {
	case v.IsStringParameterDraft():
		x := v.StringParameterDraft
		result.Value = &metadatalifecyclev1.Parameter_String_{String_: &metadatalifecyclev1.StringParameter{Default: protoString(x.Default), AllowedValues: x.AllowedValues, Pattern: x.Pattern}}
	case v.IsBooleanParameterDraft():
		x := v.BooleanParameterDraft
		result.Value = &metadatalifecyclev1.Parameter_Boolean{Boolean: &metadatalifecyclev1.BooleanParameter{Default: protoBool(x.Default)}}
	case v.IsIntegerParameterDraft():
		x := v.IntegerParameterDraft
		result.Value = &metadatalifecyclev1.Parameter_Integer{Integer: &metadatalifecyclev1.IntegerParameter{Minimum: protoInt64(x.Minimum), Maximum: protoInt64(x.Maximum), Default: protoInt64(x.Default), AllowedValues: x.AllowedValues}}
	case v.IsFloatParameterDraft():
		x := v.FloatParameterDraft
		result.Value = &metadatalifecyclev1.Parameter_Float{Float: &metadatalifecyclev1.FloatParameter{Minimum: protoFloat(x.Minimum), Maximum: protoFloat(x.Maximum), Default: protoFloat(x.Default), AllowedValues: x.AllowedValues}}
	case v.IsDateTimeParameterDraft():
		x := v.DateTimeParameterDraft
		result.Value = &metadatalifecyclev1.Parameter_DateTime{DateTime: &metadatalifecyclev1.DateTimeParameter{Minimum: protoTime(x.Minimum), Maximum: protoTime(x.Maximum), Default: protoTime(x.Default), Pattern: x.Pattern}}
	default:
		return nil
	}
	return result
}

func vMetadata(v gatewayapi.ParameterDraft) gatewayapi.ParameterDraftMetadata {
	switch {
	case v.IsStringParameterDraft():
		return v.StringParameterDraft.Metadata
	case v.IsBooleanParameterDraft():
		return v.BooleanParameterDraft.Metadata
	case v.IsIntegerParameterDraft():
		return v.IntegerParameterDraft.Metadata
	case v.IsFloatParameterDraft():
		return v.FloatParameterDraft.Metadata
	case v.IsDateTimeParameterDraft():
		return v.DateTimeParameterDraft.Metadata
	default:
		return gatewayapi.ParameterDraftMetadata{}
	}
}
