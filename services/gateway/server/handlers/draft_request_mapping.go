package handlers

import (
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoCreateResource(groupID string, q gatewayapi.CreateResourceDraftRequest) (*metadatalifecyclev1.CreateResourceRequest, gatewayapi.WriteResourceType, error) {
	request := &metadatalifecyclev1.CreateResourceRequest{GroupId: groupID}
	switch {
	case q.IsCreateDatasetResourceRequest():
		v := q.CreateDatasetResourceRequest.Metadata
		metadata := &metadatalifecyclev1.DatasetInput{
			Title:         string(v.Title),
			Description:   string(v.Description),
			Keywords:      toProtoKeywords(v.Keywords),
			ContactPoints: uuidStrings(v.ContactPoints),
			Providers:     uuidStrings(v.Providers),
			Categories:    uuidStrings(v.Categories),
		}
		accrualPeriodicity, err := toProtoDatasetAccrualPeriodicity(v.AccrualPeriodicity)
		if err != nil {
			return nil, "", err
		}
		metadata.AccrualPeriodicity = accrualPeriodicity
		temporalRange, err := toProtoDatasetTemporalRange(v.TemporalCoverage, v.GeologicalTemporalCoverage)
		if err != nil {
			return nil, "", err
		}
		switch {
		case temporalRange.temporal != nil:
			metadata.TemporalRange = &metadatalifecyclev1.DatasetInput_TemporalCoverage{TemporalCoverage: temporalRange.temporal}
		case temporalRange.geological != nil:
			metadata.TemporalRange = &metadatalifecyclev1.DatasetInput_GeologicalTemporalCoverage{GeologicalTemporalCoverage: temporalRange.geological}
		}
		if v.QualityAnnotation.Set && !v.QualityAnnotation.Null {
			metadata.QualityAnnotation = new(v.QualityAnnotation.Value)
		}
		accessRight, err := toProtoDatasetAccessRight(v.AccessRight)
		if err != nil {
			return nil, "", err
		}
		metadata.AccessRight = accessRight
		if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
			metadata.SpatialCoverage = new(v.SpatialCoverage.Value)
		}
		if v.Created.Set && !v.Created.Null {
			metadata.Created = timestamppb.New(v.Created.Value)
		}
		if v.Updated.Set && !v.Updated.Null {
			metadata.Updated = timestamppb.New(v.Updated.Value)
		}
		for _, id := range v.PersistentIdentifiers {
			metadata.PersistentIdentifiers = append(metadata.PersistentIdentifiers, toProtoVersionedIdentifierInput(id))
		}
		for _, distribution := range v.Distributions {
			mapped, err := toProtoDistribution(distribution)
			if err != nil {
				return nil, "", err
			}
			metadata.Distributions = append(metadata.Distributions, mapped)
		}
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_Dataset{Dataset: metadata}
		return request, gatewayapi.WriteResourceTypeDataset, nil
	case q.IsCreateSoftwareApplicationResourceRequest():
		v := q.CreateSoftwareApplicationResourceRequest.Metadata
		metadata := &metadatalifecyclev1.SoftwareApplicationInput{
			Name:          string(v.Name),
			Description:   string(v.Description),
			Keywords:      toProtoKeywords(v.Keywords),
			Creators:      uuidStrings(v.Creators),
			Categories:    uuidStrings(v.Categories),
			ContactPoints: uuidStrings(v.ContactPoints),
		}
		if v.MainEntityOfPage.Set {
			metadata.MainEntityOfPage = new(v.MainEntityOfPage.Value.String())
		}
		if v.License.Set && !v.License.Null {
			metadata.License = new(v.License.Value)
		}
		if v.Version.Set && !v.Version.Null {
			metadata.Version = new(v.Version.Value)
		}
		if v.DownloadURL.Set {
			metadata.DownloadUrl = new(v.DownloadURL.Value.String())
		}
		if v.Requirements.Set && !v.Requirements.Null {
			metadata.Requirements = new(v.Requirements.Value)
		}
		if v.OperatingSystem.Set && !v.OperatingSystem.Null {
			metadata.OperatingSystem = new(v.OperatingSystem.Value)
		}
		for _, id := range v.Identifiers {
			metadata.Identifiers = append(metadata.Identifiers, toProtoVersionedIdentifierInput(id))
		}
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_SoftwareApplication{SoftwareApplication: metadata}
		return request, gatewayapi.WriteResourceTypeSoftwareApplication, nil
	case q.IsCreateSoftwareSourceCodeResourceRequest():
		v := q.CreateSoftwareSourceCodeResourceRequest.Metadata
		metadata := &metadatalifecyclev1.SoftwareSourceCodeInput{
			Name:          string(v.Name),
			Description:   string(v.Description),
			Keywords:      toProtoKeywords(v.Keywords),
			Creators:      uuidStrings(v.Creators),
			Categories:    uuidStrings(v.Categories),
			ContactPoints: uuidStrings(v.ContactPoints),
		}
		if v.ProgrammingLanguage.Set {
			metadata.ProgrammingLanguage = new(v.ProgrammingLanguage.Value)
		}
		if v.Repository.Set {
			metadata.Repository = new(v.Repository.Value.String())
		}
		if v.License.Set && !v.License.Null {
			metadata.License = new(v.License.Value)
		}
		if v.Version.Set && !v.Version.Null {
			metadata.Version = new(v.Version.Value)
		}
		if v.SoftwareRequirements.Set && !v.SoftwareRequirements.Null {
			metadata.SoftwareRequirements = new(v.SoftwareRequirements.Value)
		}
		if v.RuntimePlatform.Set && !v.RuntimePlatform.Null {
			metadata.RuntimePlatform = new(v.RuntimePlatform.Value)
		}
		if v.MainEntityOfPage.Set {
			metadata.MainEntityOfPage = new(v.MainEntityOfPage.Value.String())
		}
		for _, id := range v.Identifiers {
			metadata.Identifiers = append(metadata.Identifiers, toProtoVersionedIdentifierInput(id))
		}
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_SoftwareSourceCode{SoftwareSourceCode: metadata}
		return request, gatewayapi.WriteResourceTypeSoftwareSourceCode, nil
	default:
		return nil, "", errors.New("unsupported resource draft discriminator")
	}
}

func toProtoReplaceResourceDraft(q gatewayapi.ReplaceResourceDraftRequest) (gatewayapi.WriteResourceType, *metadatalifecyclev1.ReplaceResourceDraftRequest, error) {
	request := &metadatalifecyclev1.ReplaceResourceDraftRequest{}
	switch {
	case q.IsReplaceDatasetDraftRequest():
		metadata, err := toProtoDatasetDraftMetadata(q.ReplaceDatasetDraftRequest.Metadata)
		if err != nil {
			return "", nil, err
		}
		request.Metadata = &metadatalifecyclev1.ReplaceResourceDraftRequest_Dataset{
			Dataset: metadata,
		}
		request.Revision = q.ReplaceDatasetDraftRequest.Revision
		return gatewayapi.WriteResourceTypeDataset, request, nil
	case q.IsReplaceSoftwareApplicationDraftRequest():
		request.Metadata = &metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareApplication{
			SoftwareApplication: toProtoSoftwareApplicationDraftMetadata(q.ReplaceSoftwareApplicationDraftRequest.Metadata),
		}
		request.Revision = q.ReplaceSoftwareApplicationDraftRequest.Revision
		return gatewayapi.WriteResourceTypeSoftwareApplication, request, nil
	case q.IsReplaceSoftwareSourceCodeDraftRequest():
		request.Metadata = &metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareSourceCode{
			SoftwareSourceCode: toProtoSoftwareSourceCodeDraftMetadata(q.ReplaceSoftwareSourceCodeDraftRequest.Metadata),
		}
		request.Revision = q.ReplaceSoftwareSourceCodeDraftRequest.Revision
		return gatewayapi.WriteResourceTypeSoftwareSourceCode, request, nil
	default:
		return "", nil, errors.New("unsupported resource draft discriminator")
	}
}

func uuidStrings(values []uuid.UUID) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.String())
	}
	return result
}
