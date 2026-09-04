package handlers

import (
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toClientDatasetDraftMetadata(v gatewayapi.DatasetDraftMetadata) metadatalifecycleclient.DatasetDraftMetadata {
	r := metadatalifecycleclient.DatasetDraftMetadata{Title: string(v.Title)}
	if v.Description.Set {
		r.SetDescription(v.Description.Value)
	}
	if v.Keywords.Set {
		r.SetKeywords(v.Keywords.Value)
	}
	if v.AccrualPeriodicity.Set && !v.AccrualPeriodicity.Null {
		r.SetAccrualPeriodicity(v.AccrualPeriodicity.Value)
	}
	if v.QualityAnnotation.Set && !v.QualityAnnotation.Null {
		r.SetQualityAnnotation(v.QualityAnnotation.Value)
	}
	if v.AccessRight.Set && !v.AccessRight.Null {
		r.SetAccessRight(v.AccessRight.Value)
	}
	if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
		r.SetSpatialCoverage(v.SpatialCoverage.Value)
	}
	if x := toClientTemporal(v.TemporalCoverage); x != nil {
		r.SetTemporalCoverage(*x)
	}
	if v.Created.Set && !v.Created.Null {
		r.SetCreated(v.Created.Value)
	}
	if v.Updated.Set && !v.Updated.Null {
		r.SetUpdated(v.Updated.Value)
	}
	for _, x := range v.Distributions {
		r.Distributions = append(r.Distributions, toClientDistributionDraft(x))
	}
	for _, x := range v.PersistentIdentifiers {
		r.PersistentIdentifiers = append(r.PersistentIdentifiers, metadatalifecycleclient.IdentifierDraft{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	for _, x := range v.Providers {
		r.Providers = append(r.Providers, x.String())
	}
	for _, x := range v.Categories {
		r.Categories = append(r.Categories, x.String())
	}
	return r
}

func toClientSoftwareApplicationDraftMetadata(v gatewayapi.SoftwareApplicationDraftMetadata) metadatalifecycleclient.SoftwareApplicationDraftMetadata {
	r := metadatalifecycleclient.SoftwareApplicationDraftMetadata{
		Name:        string(v.Name),
		Description: &v.Description.Value,
	}
	if v.Keywords.Set {
		r.SetKeywords(v.Keywords.Value)
	}
	if v.License.Set && !v.License.Null {
		r.SetLicense(v.License.Value)
	}
	if v.Version.Set && !v.Version.Null {
		r.SetVersion(v.Version.Value)
	}
	if v.DownloadURL.Set {
		r.SetDownloadUrl(v.DownloadURL.Value.String())
	}
	if v.Requirements.Set && !v.Requirements.Null {
		r.SetRequirements(v.Requirements.Value)
	}
	if v.OperatingSystem.Set && !v.OperatingSystem.Null {
		r.SetOperatingSystem(v.OperatingSystem.Value)
	}
	if v.MainEntityOfPage.Set {
		r.SetMainEntityOfPage(v.MainEntityOfPage.Value.String())
	}
	for _, x := range v.Creators {
		r.Creators = append(r.Creators, x.String())
	}
	for _, x := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, metadatalifecycleclient.IdentifierDraft{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	for _, x := range v.Categories {
		r.Categories = append(r.Categories, x.String())
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	return r
}

func toClientSoftwareSourceCodeDraftMetadata(v gatewayapi.SoftwareSourceCodeDraftMetadata) metadatalifecycleclient.SoftwareSourceCodeDraftMetadata {
	r := metadatalifecycleclient.SoftwareSourceCodeDraftMetadata{
		Name:        string(v.Name),
		Description: &v.Description.Value,
	}
	if v.Keywords.Set {
		r.SetKeywords(v.Keywords.Value)
	}
	if v.Repository.Set {
		r.SetRepository(v.Repository.Value.String())
	}
	if v.ProgrammingLanguage.Set {
		r.SetProgrammingLanguage(v.ProgrammingLanguage.Value)
	}
	if v.License.Set && !v.License.Null {
		r.SetLicense(v.License.Value)
	}
	if v.SoftwareRequirements.Set && !v.SoftwareRequirements.Null {
		r.SetSoftwareRequirements(v.SoftwareRequirements.Value)
	}
	if v.RuntimePlatform.Set && !v.RuntimePlatform.Null {
		r.SetRuntimePlatform(v.RuntimePlatform.Value)
	}
	if v.MainEntityOfPage.Set {
		r.SetMainEntityOfPage(v.MainEntityOfPage.Value.String())
	}
	if v.Version.Set && !v.Version.Null {
		r.SetVersion(v.Version.Value)
	}
	for _, x := range v.Creators {
		r.Creators = append(r.Creators, x.String())
	}
	for _, x := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, metadatalifecycleclient.IdentifierDraft{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	for _, x := range v.Categories {
		r.Categories = append(r.Categories, x.String())
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	return r
}
