package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/google/uuid"
)

func (c *MetadataLifecycle) CreateResourceDraft(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateResourceRequest) (*metadatalifecyclev1.CreateResourceResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC resource client is not configured")
	}
	return c.GRPC.CreateResource(ctx, actorID, request)
}

func (c *MetadataLifecycle) GetResourceVersion(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.GetResourceVersionResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC resource client is not configured")
	}
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.GetResourceVersion(ctx, actorID, &metadatalifecyclev1.GetResourceVersionRequest{
		ResourceType: typ,
		ResourceId:   resourceID,
		VersionId:    versionID,
	})
}

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
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC resource client is not configured")
	}
	if request == nil {
		return nil, fmt.Errorf("resource draft request is required")
	}
	request.ResourceId = resourceID
	request.VersionId = versionID
	return c.GRPC.ReplaceResourceDraft(ctx, actorID, request)
}

func (c *MetadataLifecycle) DiscardResourceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string) error {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return err
	}
	return c.GRPC.DiscardResourceDraft(ctx, actorID, &metadatalifecyclev1.DiscardResourceDraftRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
}

func (c *MetadataLifecycle) SubmitResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.SubmitResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.SubmitResource(ctx, actorID, &metadatalifecyclev1.SubmitResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
}

func (c *MetadataLifecycle) WithdrawResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.WithdrawResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.WithdrawResource(ctx, actorID, &metadatalifecyclev1.WithdrawResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
}

func (c *MetadataLifecycle) RejectResource(ctx context.Context, actorID, resourceType, resourceID, versionID, note string) (*metadatalifecyclev1.RejectResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.RejectResource(ctx, actorID, &metadatalifecyclev1.RejectResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID, Note: note})
}

func (c *MetadataLifecycle) ReopenResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.ReopenResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.ReopenResource(ctx, actorID, &metadatalifecyclev1.ReopenResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
}

func (c *MetadataLifecycle) PublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.PublishResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.PublishResource(ctx, actorID, &metadatalifecyclev1.PublishResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
}

func (c *MetadataLifecycle) UnpublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (*metadatalifecyclev1.UnpublishResourceResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.UnpublishResource(ctx, actorID, &metadatalifecyclev1.UnpublishResourceRequest{ResourceType: typ, ResourceId: resourceID, VersionId: versionID})
}

func (c *MetadataLifecycle) CreateResourceDraftFromPublication(ctx context.Context, actorID, resourceType, resourceID string) (*metadatalifecyclev1.DraftResourceFromPublicationResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.DraftResourceFromPublication(ctx, actorID, &metadatalifecyclev1.DraftResourceFromPublicationRequest{ResourceType: typ, ResourceId: resourceID})
}

func (c *MetadataLifecycle) CreateResourceDraftFromArchive(ctx context.Context, actorID, resourceType, resourceID string) (*metadatalifecyclev1.DraftResourceFromLatestArchiveResponse, error) {
	typ, err := resourceTypeProto(resourceType)
	if err != nil {
		return nil, err
	}
	return c.GRPC.DraftResourceFromLatestArchive(ctx, actorID, &metadatalifecyclev1.DraftResourceFromLatestArchiveRequest{ResourceType: typ, ResourceId: resourceID})
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
	return c.GRPC.CopyPublishedResource(ctx, actorID, &metadatalifecyclev1.CopyPublishedResourceRequest{ResourceType: typ, ResourceId: resourceID, GroupId: group.String()})
}
