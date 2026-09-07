package handlers

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoDatasetDraftMetadata(v gatewayapi.DatasetDraftMetadata) *metadatalifecyclev1.DatasetInput {
	r := &metadatalifecyclev1.DatasetInput{
		Title:            string(v.Title),
		ContactPoints:    uuidStrings(v.ContactPoints),
		Providers:        uuidStrings(v.Providers),
		Categories:       uuidStrings(v.Categories),
		Keywords:         toProtoKeywords(v.Keywords),
		TemporalCoverage: toProtoTemporal(v.TemporalCoverage),
	}
	if v.Description.Set {
		r.Description = new(string(v.Description.Value))
	}
	if v.AccrualPeriodicity.Set && !v.AccrualPeriodicity.Null {
		r.AccrualPeriodicity = new(v.AccrualPeriodicity.Value)
	}
	if v.QualityAnnotation.Set && !v.QualityAnnotation.Null {
		r.QualityAnnotation = new(v.QualityAnnotation.Value)
	}
	if v.AccessRight.Set && !v.AccessRight.Null {
		r.AccessRight = new(v.AccessRight.Value)
	}
	if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
		r.SpatialCoverage = new(v.SpatialCoverage.Value)
	}
	if v.Created.Set && !v.Created.Null {
		r.Created = timestamppb.New(v.Created.Value)
	}
	if v.Updated.Set && !v.Updated.Null {
		r.Updated = timestamppb.New(v.Updated.Value)
	}
	for _, id := range v.PersistentIdentifiers {
		r.PersistentIdentifiers = append(r.PersistentIdentifiers, &metadatalifecyclev1.IdentifierInput{
			Type:  string(id.Type),
			Value: string(id.Value),
		})
	}
	for _, distribution := range v.Distributions {
		r.Distributions = append(r.Distributions, toProtoDistributionDraft(distribution))
	}
	return r
}

func toProtoSoftwareApplicationDraftMetadata(v gatewayapi.SoftwareApplicationDraftMetadata) *metadatalifecyclev1.SoftwareApplicationInput {
	r := &metadatalifecyclev1.SoftwareApplicationInput{
		Name:          string(v.Name),
		Creators:      uuidStrings(v.Creators),
		Categories:    uuidStrings(v.Categories),
		ContactPoints: uuidStrings(v.ContactPoints),
		Description:   string(v.Description),
		Keywords:      toProtoKeywords(v.Keywords),
	}

	if v.MainEntityOfPage.Set {
		r.MainEntityOfPage = new(v.MainEntityOfPage.Value.String())
	}
	if v.License.Set && !v.License.Null {
		r.License = new(v.License.Value)
	}
	if v.Version.Set && !v.Version.Null {
		r.Version = new(v.Version.Value)
	}
	if v.DownloadURL.Set {
		r.DownloadUrl = new(v.DownloadURL.Value.String())
	}
	if v.Requirements.Set && !v.Requirements.Null {
		r.Requirements = new(v.Requirements.Value)
	}
	if v.OperatingSystem.Set && !v.OperatingSystem.Null {
		r.OperatingSystem = new(v.OperatingSystem.Value)
	}
	for _, id := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, &metadatalifecyclev1.IdentifierInput{
			Type:  string(id.Type),
			Value: string(id.Value),
		})
	}
	return r
}

func toProtoSoftwareSourceCodeDraftMetadata(v gatewayapi.SoftwareSourceCodeDraftMetadata) *metadatalifecyclev1.SoftwareSourceCodeInput {
	r := &metadatalifecyclev1.SoftwareSourceCodeInput{
		Name:          string(v.Name),
		Creators:      uuidStrings(v.Creators),
		Categories:    uuidStrings(v.Categories),
		ContactPoints: uuidStrings(v.ContactPoints),
		Description:   string(v.Description),
		Keywords:      toProtoKeywords(v.Keywords),
	}
	if v.ProgrammingLanguage.Set {
		r.ProgrammingLanguage = new(v.ProgrammingLanguage.Value)
	}
	if v.Repository.Set {
		r.Repository = new(v.Repository.Value.String())
	}
	if v.License.Set && !v.License.Null {
		r.License = new(v.License.Value)
	}
	if v.SoftwareRequirements.Set && !v.SoftwareRequirements.Null {
		r.SoftwareRequirements = new(v.SoftwareRequirements.Value)
	}
	if v.RuntimePlatform.Set && !v.RuntimePlatform.Null {
		r.RuntimePlatform = new(v.RuntimePlatform.Value)
	}
	if v.MainEntityOfPage.Set {
		r.MainEntityOfPage = new(v.MainEntityOfPage.Value.String())
	}
	if v.Version.Set && !v.Version.Null {
		r.Version = new(v.Version.Value)
	}
	for _, id := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, &metadatalifecyclev1.IdentifierInput{
			Type:  string(id.Type),
			Value: string(id.Value),
		})
	}
	return r
}
