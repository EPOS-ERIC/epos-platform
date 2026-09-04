package handlers

import (
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func toProtoCreateResource(groupID string, q gatewayapi.CreateResourceDraftRequest) (*metadatalifecyclev1.CreateResourceRequest, string, error) {
	request := &metadatalifecyclev1.CreateResourceRequest{GroupId: groupID}
	switch {
	case q.IsCreateDatasetResourceRequest():
		v := q.CreateDatasetResourceRequest.Metadata
		metadata := &metadatalifecyclev1.DatasetMetadata{Title: string(v.Title), Description: v.Description.Value, Keywords: v.Keywords.Value}
		if v.AccrualPeriodicity.Set && !v.AccrualPeriodicity.Null {
			metadata.AccrualPeriodicity = new(v.AccrualPeriodicity.Value)
		}
		if v.QualityAnnotation.Set && !v.QualityAnnotation.Null {
			metadata.QualityAnnotation = new(v.QualityAnnotation.Value)
		}
		if v.AccessRight.Set && !v.AccessRight.Null {
			metadata.AccessRight = new(v.AccessRight.Value)
		}
		if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
			metadata.SpatialCoverage = new(v.SpatialCoverage.Value)
		}
		metadata.TemporalCoverage = toProtoTemporal(v.TemporalCoverage)
		if v.Created.Set && !v.Created.Null {
			metadata.Created = timestampPtr(v.Created.Value)
		}
		if v.Updated.Set && !v.Updated.Null {
			metadata.Updated = timestampPtr(v.Updated.Value)
		}
		metadata.ContactPoints = uuidStrings(v.ContactPoints)
		metadata.Providers = uuidStrings(v.Providers)
		metadata.Categories = uuidStrings(v.Categories)
		for _, id := range v.PersistentIdentifiers {
			metadata.PersistentIdentifiers = append(metadata.PersistentIdentifiers, &metadatalifecyclev1.Identifier{Type: string(id.Type), Value: string(id.Value)})
		}
		for _, distribution := range v.Distributions {
			metadata.Distributions = append(metadata.Distributions, toProtoDistribution(distribution))
		}
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_Dataset{Dataset: metadata}
		return request, "dataset", nil
	case q.IsCreateSoftwareApplicationResourceRequest():
		v := q.CreateSoftwareApplicationResourceRequest.Metadata
		metadata := &metadatalifecyclev1.SoftwareApplicationMetadata{Name: string(v.Name), Description: v.Description.Value, Keywords: v.Keywords.Value, Creators: uuidStrings(v.Creators), Categories: uuidStrings(v.Categories), ContactPoints: uuidStrings(v.ContactPoints)}
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
			metadata.Identifiers = append(metadata.Identifiers, &metadatalifecyclev1.Identifier{Type: string(id.Type), Value: string(id.Value)})
		}
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_SoftwareApplication{SoftwareApplication: metadata}
		return request, "software-application", nil
	case q.IsCreateSoftwareSourceCodeResourceRequest():
		v := q.CreateSoftwareSourceCodeResourceRequest.Metadata
		metadata := &metadatalifecyclev1.SoftwareSourceCodeMetadata{Name: string(v.Name), Description: v.Description.Value, Keywords: v.Keywords.Value, Creators: uuidStrings(v.Creators), Categories: uuidStrings(v.Categories), ContactPoints: uuidStrings(v.ContactPoints)}
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
			metadata.Identifiers = append(metadata.Identifiers, &metadatalifecyclev1.Identifier{Type: string(id.Type), Value: string(id.Value)})
		}
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_SoftwareSourceCode{SoftwareSourceCode: metadata}
		return request, "software-source-code", nil
	default:
		return nil, "", errors.New("unsupported resource draft discriminator")
	}
}

func toProtoReplaceResourceDraft(q gatewayapi.ReplaceResourceDraftRequest) (string, *metadatalifecyclev1.ReplaceResourceDraftRequest, error) {
	request := &metadatalifecyclev1.ReplaceResourceDraftRequest{}
	switch {
	case q.IsReplaceDatasetDraftRequest():
		request.Metadata = &metadatalifecyclev1.ReplaceResourceDraftRequest_Dataset{Dataset: toProtoDatasetDraftMetadata(q.ReplaceDatasetDraftRequest.Metadata)}
		request.Revision = q.ReplaceDatasetDraftRequest.Revision
		return "dataset", request, nil
	case q.IsReplaceSoftwareApplicationDraftRequest():
		request.Metadata = &metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareApplication{SoftwareApplication: toProtoSoftwareApplicationDraftMetadata(q.ReplaceSoftwareApplicationDraftRequest.Metadata)}
		request.Revision = q.ReplaceSoftwareApplicationDraftRequest.Revision
		return "software-application", request, nil
	case q.IsReplaceSoftwareSourceCodeDraftRequest():
		request.Metadata = &metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareSourceCode{SoftwareSourceCode: toProtoSoftwareSourceCodeDraftMetadata(q.ReplaceSoftwareSourceCodeDraftRequest.Metadata)}
		request.Revision = q.ReplaceSoftwareSourceCodeDraftRequest.Revision
		return "software-source-code", request, nil
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
