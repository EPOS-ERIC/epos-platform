package handlers

import (
	"errors"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toClientCreateResource(q gatewayapi.CreateResourceDraftRequest) (string, metadatalifecycleclient.CreateResourceRequest, error) {
	switch {
	case q.IsCreateDatasetResourceRequest():
		return "dataset", metadatalifecycleclient.CreateDatasetResourceRequestAsCreateResourceRequest(&metadatalifecycleclient.CreateDatasetResourceRequest{
			ResourceType: "dataset",
			Metadata:     toClientDatasetDraftMetadata(q.CreateDatasetResourceRequest.Metadata),
		}), nil
	case q.IsCreateSoftwareApplicationResourceRequest():
		return "software-application", metadatalifecycleclient.CreateSoftwareApplicationResourceRequestAsCreateResourceRequest(&metadatalifecycleclient.CreateSoftwareApplicationResourceRequest{
			ResourceType: "software-application",
			Metadata:     toClientSoftwareApplicationDraftMetadata(q.CreateSoftwareApplicationResourceRequest.Metadata),
		}), nil
	case q.IsCreateSoftwareSourceCodeResourceRequest():
		return "software-source-code", metadatalifecycleclient.CreateSoftwareSourceCodeResourceRequestAsCreateResourceRequest(&metadatalifecycleclient.CreateSoftwareSourceCodeResourceRequest{
			ResourceType: "software-source-code",
			Metadata:     toClientSoftwareSourceCodeDraftMetadata(q.CreateSoftwareSourceCodeResourceRequest.Metadata),
		}), nil
	}
	return "", metadatalifecycleclient.CreateResourceRequest{}, errors.New("unsupported resource draft discriminator")
}

func toClientReplaceResourceDraft(q gatewayapi.ReplaceResourceDraftRequest) (string, metadatalifecycleclient.ReplaceResourceDraftRequest, error) {
	switch {
	case q.IsReplaceDatasetDraftRequest():
		x := q.ReplaceDatasetDraftRequest
		return "dataset", metadatalifecycleclient.ReplaceResourceDraftRequest{
			ReplaceDraftRequest: &metadatalifecycleclient.ReplaceDraftRequest{
				ResourceType: "dataset",
				Revision:     x.Revision,
				Metadata:     toClientDatasetDraftMetadata(x.Metadata),
			},
		}, nil
	case q.IsReplaceSoftwareApplicationDraftRequest():
		x := q.ReplaceSoftwareApplicationDraftRequest
		return "software-application", metadatalifecycleclient.ReplaceResourceDraftRequest{
			ReplaceSoftwareApplicationDraftRequest: &metadatalifecycleclient.ReplaceSoftwareApplicationDraftRequest{
				ResourceType: "software-application",
				Revision:     x.Revision,
				Metadata:     toClientSoftwareApplicationDraftMetadata(x.Metadata),
			},
		}, nil
	case q.IsReplaceSoftwareSourceCodeDraftRequest():
		x := q.ReplaceSoftwareSourceCodeDraftRequest
		return "software-source-code", metadatalifecycleclient.ReplaceResourceDraftRequest{
			ReplaceSoftwareSourceCodeDraftRequest: &metadatalifecycleclient.ReplaceSoftwareSourceCodeDraftRequest{
				ResourceType: "software-source-code",
				Revision:     x.Revision,
				Metadata:     toClientSoftwareSourceCodeDraftMetadata(x.Metadata),
			},
		}, nil
	}
	return "", metadatalifecycleclient.ReplaceResourceDraftRequest{}, errors.New("unsupported resource draft discriminator")
}
