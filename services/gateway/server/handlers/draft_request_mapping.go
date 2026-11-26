package handlers

import (
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func toProtoCreateResource(groupID string, q gatewayapi.CreateResourceDraftRequest) (*metadatalifecyclev1.CreateResourceRequest, gatewayapi.WriteResourceType, error) {
	request := &metadatalifecyclev1.CreateResourceRequest{GroupId: groupID}
	switch {
	case q.IsCreateDatasetResourceRequest():
		metadata, err := toProtoDatasetDraftMetadata(q.CreateDatasetResourceRequest.Metadata)
		if err != nil {
			return nil, "", err
		}
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_Dataset{Dataset: metadata}
		return request, gatewayapi.WriteResourceTypeDataset, nil
	case q.IsCreateSoftwareApplicationResourceRequest():
		metadata := toProtoSoftwareApplicationDraftMetadata(q.CreateSoftwareApplicationResourceRequest.Metadata)
		request.Metadata = &metadatalifecyclev1.CreateResourceRequest_SoftwareApplication{SoftwareApplication: metadata}
		return request, gatewayapi.WriteResourceTypeSoftwareApplication, nil
	case q.IsCreateSoftwareSourceCodeResourceRequest():
		metadata := toProtoSoftwareSourceCodeDraftMetadata(q.CreateSoftwareSourceCodeResourceRequest.Metadata)
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
