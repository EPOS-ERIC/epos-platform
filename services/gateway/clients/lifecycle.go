package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) CreateResourceDraft(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateResourceRequest) (*metadatalifecyclev1.CreateResourceResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resources.CreateResource(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) GetResourceVersion(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.GetResourceVersionResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}
	response, err := c.resources.GetResourceVersion(ctx, &metadatalifecyclev1.GetResourceVersionRequest{
		ResourceType: typ,
		ResourceId:   resourceID,
		VersionId:    versionID,
	})
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle resource version", err)
	}
	return response, nil
}

// TODO: move this into the hanlers instead and make the functions in this package accept the metadatalifecyclev1 ResourceType instead so that we have a typed switch
func resourceTypeProto(value string) (metadatalifecyclev1.ResourceType, error) {
	switch value {
	case "dataset":
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET, nil
	case "software-application":
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION, nil
	case "software-source-code":
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE, nil
	default:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_UNSPECIFIED, fmt.Errorf("invalid resource type %q", value)
	}
}

func (c *MetadataLifecycle) ReplaceResourceDraft(ctx context.Context, actorID, resourceID, versionID string, request *metadatalifecyclev1.ReplaceResourceDraftRequest) (*metadatalifecyclev1.ReplaceResourceDraftResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("resource draft request is required")
	}
	request.ResourceId = resourceID
	request.VersionId = versionID
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resources.ReplaceResourceDraft(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle resource draft", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) DiscardResourceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string) error {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err = c.resources.DiscardResourceDraft(ctx, &metadatalifecyclev1.DiscardResourceDraftRequest{
		ResourceType: typ,
		ResourceId:   resourceID,
		VersionId:    versionID,
	})
	if err != nil {
		return wrapGRPCError("discard metadata-lifecycle resource draft", err)
	}
	return nil
}

func (c *MetadataLifecycle) SubmitResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.SubmitResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	response, err := c.resources.SubmitResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.SubmitResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, wrapGRPCError("submit metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) WithdrawResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.WithdrawResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	response, err := c.resources.WithdrawResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.WithdrawResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, wrapGRPCError("withdraw metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) RejectResource(ctx context.Context, actorID, resourceType, resourceID, versionID, note string) (*metadatalifecyclev1.RejectResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resources.RejectResource(ctx, &metadatalifecyclev1.RejectResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID, Note: note})
	if err != nil {
		return nil, wrapGRPCError("reject metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ReopenResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.ReopenResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	response, err := c.resources.ReopenResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.ReopenResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, wrapGRPCError("reopen metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) PublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.PublishResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	response, err := c.resources.PublishResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.PublishResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, wrapGRPCError("publish metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) UnpublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.UnpublishResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	response, err := c.resources.UnpublishResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.UnpublishResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, wrapGRPCError("unpublish metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateResourceDraftFromPublication(ctx context.Context, actorID, resourceType, resourceID string) (*metadatalifecyclev1.DraftResourceFromPublicationResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	response, err := c.resources.DraftResourceFromPublication(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.DraftResourceFromPublicationRequest{ResourceType: typ, ResourceId: resourceID})
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle draft from publication", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateResourceDraftFromArchive(ctx context.Context, actorID, resourceType, resourceID string) (*metadatalifecyclev1.DraftResourceFromLatestArchiveResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	response, err := c.resources.DraftResourceFromLatestArchive(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.DraftResourceFromLatestArchiveRequest{ResourceType: typ, ResourceId: resourceID})
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle draft from archive", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CopyResource(ctx context.Context, actorID, groupID, resourceType, resourceID string) (*metadatalifecyclev1.CopyPublishedResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	group, err := uuid.Parse(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group id: %w", err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resources.CopyPublishedResource(ctx, &metadatalifecyclev1.CopyPublishedResourceRequest{ResourceType: typ, ResourceId: resourceID, GroupId: group.String()})
	if err != nil {
		return nil, wrapGRPCError("copy metadata-lifecycle published resource", err)
	}
	return response, nil
}
