package handlers

import (
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toProtoDatasetDraftMetadata(v gatewayapi.DatasetDraftMetadata) (*metadatalifecyclev1.DatasetInput, error) {
	r := &metadatalifecyclev1.DatasetInput{
		Title:                     v.Title,
		ContactPoints:             uuidStrings(v.ContactPoints),
		Providers:                 uuidStrings(v.Providers),
		Categories:                uuidStrings(v.Categories),
		Keywords:                  toProtoKeywords(v.Keywords),
		ConformsTo:                toClientStringPtr(v.ConformsTo),
		Provenance:                toClientStringPtr(v.Provenance),
		Version:                   toClientStringPtr(v.Version),
		QualityAnnotation:         toProtoOptURIString(v.QualityAnnotation),
		SpatialCoverage:           toClientStringPtr(v.SpatialCoverage),
		SpatialResolutionInMeters: protoOptionalFloat(v.SpatialResolutionInMeters),
		TemporalResolution:        protoOptionalFloat(v.TemporalResolution),
		Created:                   protoOptionalTime(v.Created),
		Updated:                   protoOptionalTime(v.Updated),
	}
	if v.TemporalCoverage.Set && v.GeologicalTemporalCoverage.Set {
		return nil, errors.New("dataset temporal coverage fields are mutually exclusive")
	}
	switch {
	case v.TemporalCoverage.Set:
		r.TemporalRange = &metadatalifecyclev1.DatasetInput_TemporalCoverage{TemporalCoverage: toProtoTemporal(v.TemporalCoverage)}
	case v.GeologicalTemporalCoverage.Set:
		r.TemporalRange = &metadatalifecyclev1.DatasetInput_GeologicalTemporalCoverage{GeologicalTemporalCoverage: toProtoGeologicalTemporal(v.GeologicalTemporalCoverage.Value)}
	}
	r.Description = v.Description
	var err error
	r.Type, err = toProtoDatasetType(v.Type)
	if err != nil {
		return nil, err
	}
	accrualPeriodicity, err := toProtoDatasetAccrualPeriodicity(v.AccrualPeriodicity)
	if err != nil {
		return nil, err
	}
	r.AccrualPeriodicity = accrualPeriodicity
	accessRight, err := toProtoDatasetAccessRight(v.AccessRight)
	if err != nil {
		return nil, err
	}
	r.AccessRight = accessRight
	for _, id := range v.PersistentIdentifiers {
		r.PersistentIdentifiers = append(r.PersistentIdentifiers, toProtoVersionedIdentifierInput(id))
	}
	for _, distribution := range v.Distributions {
		mapped, err := toProtoDistribution(distribution)
		if err != nil {
			return nil, err
		}
		r.Distributions = append(r.Distributions, mapped)
	}
	return r, nil
}

func toProtoSoftwareApplicationDraftMetadata(v gatewayapi.SoftwareApplicationDraftMetadata) *metadatalifecyclev1.SoftwareApplicationInput {
	r := &metadatalifecyclev1.SoftwareApplicationInput{
		Name:          v.Name,
		Creators:      uuidStrings(v.Creators),
		Categories:    uuidStrings(v.Categories),
		ContactPoints: uuidStrings(v.ContactPoints),
		Description:   v.Description,
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
		r.Identifiers = append(r.Identifiers, toProtoVersionedIdentifierInput(id))
	}
	return r
}

func toProtoSoftwareSourceCodeDraftMetadata(v gatewayapi.SoftwareSourceCodeDraftMetadata) *metadatalifecyclev1.SoftwareSourceCodeInput {
	r := &metadatalifecyclev1.SoftwareSourceCodeInput{
		Name:          v.Name,
		Creators:      uuidStrings(v.Creators),
		Categories:    uuidStrings(v.Categories),
		ContactPoints: uuidStrings(v.ContactPoints),
		Description:   v.Description,
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
		r.Identifiers = append(r.Identifiers, toProtoVersionedIdentifierInput(id))
	}
	return r
}

func toProtoVersionedIdentifierInput(value gatewayapi.VersionedIdentifierInput) *metadatalifecyclev1.VersionedIdentifierInput {
	return &metadatalifecyclev1.VersionedIdentifierInput{
		ResourceId: protoOptionalUUID(value.ResourceID),
		Type:       value.Type,
		Value:      value.Value,
	}
}
