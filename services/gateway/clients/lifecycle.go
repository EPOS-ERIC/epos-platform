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
		return nil, c.wrapError(ctx, "create metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) GetResourceVersion(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID string) (*metadatalifecyclev1.GetResourceVersionResponse, error) {
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}
	response, err := c.resources.GetResourceVersion(ctx, &metadatalifecyclev1.GetResourceVersionRequest{
		ResourceType: resourceType,
		ResourceId:   resourceID,
		VersionId:    versionID,
	})
	if err != nil {
		return nil, c.wrapError(ctx, "get metadata-lifecycle resource version", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ListResourceVersions(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID string) (*metadatalifecyclev1.ListResourceVersionsResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resources.ListResourceVersions(ctx, &metadatalifecyclev1.ListResourceVersionsRequest{
		ResourceType: resourceType,
		ResourceId:   resourceID,
	})
	if err != nil {
		return nil, c.wrapError(ctx, "list metadata-lifecycle resource versions", err)
	}
	return response, nil
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
		return nil, c.wrapError(ctx, "replace metadata-lifecycle resource draft", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) DiscardResourceDraft(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.resources.DiscardResourceDraft(ctx, &metadatalifecyclev1.DiscardResourceDraftRequest{
		ResourceType: resourceType,
		ResourceId:   resourceID,
		VersionId:    versionID,
	})
	if err != nil {
		return c.wrapError(ctx, "discard metadata-lifecycle resource draft", err)
	}
	return nil
}

func (c *MetadataLifecycle) SubmitResource(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID string) (*metadatalifecyclev1.SubmitResourceResponse, error) {
	response, err := c.resources.SubmitResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.SubmitResourceRequest{ResourceType: resourceType, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, c.wrapError(ctx, "submit metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) WithdrawResource(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID string) (*metadatalifecyclev1.WithdrawResourceResponse, error) {
	response, err := c.resources.WithdrawResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.WithdrawResourceRequest{ResourceType: resourceType, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, c.wrapError(ctx, "withdraw metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) RejectResource(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID, note string) (*metadatalifecyclev1.RejectResourceResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resources.RejectResource(ctx, &metadatalifecyclev1.RejectResourceRequest{ResourceType: resourceType, ResourceId: resourceID, VersionId: versionID, Note: note})
	if err != nil {
		return nil, c.wrapError(ctx, "reject metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ReopenResource(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID string) (*metadatalifecyclev1.ReopenResourceResponse, error) {
	response, err := c.resources.ReopenResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.ReopenResourceRequest{ResourceType: resourceType, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, c.wrapError(ctx, "reopen metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) PublishResource(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID string) (*metadatalifecyclev1.PublishResourceResponse, error) {
	response, err := c.resources.PublishResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.PublishResourceRequest{ResourceType: resourceType, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, c.wrapError(ctx, "publish metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) UnpublishResource(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID, versionID string) (*metadatalifecyclev1.UnpublishResourceResponse, error) {
	response, err := c.resources.UnpublishResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.UnpublishResourceRequest{ResourceType: resourceType, ResourceId: resourceID, VersionId: versionID})
	if err != nil {
		return nil, c.wrapError(ctx, "unpublish metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateResourceDraftFromPublication(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID string) (*metadatalifecyclev1.DraftResourceFromPublicationResponse, error) {
	response, err := c.resources.DraftResourceFromPublication(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.DraftResourceFromPublicationRequest{ResourceType: resourceType, ResourceId: resourceID})
	if err != nil {
		return nil, c.wrapError(ctx, "create metadata-lifecycle draft from publication", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateResourceDraftFromArchive(ctx context.Context, actorID string, resourceType metadatalifecyclev1.ResourceType, resourceID string) (*metadatalifecyclev1.DraftResourceFromLatestArchiveResponse, error) {
	response, err := c.resources.DraftResourceFromLatestArchive(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), &metadatalifecyclev1.DraftResourceFromLatestArchiveRequest{ResourceType: resourceType, ResourceId: resourceID})
	if err != nil {
		return nil, c.wrapError(ctx, "create metadata-lifecycle draft from archive", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CopyResource(ctx context.Context, actorID, groupID string, resourceType metadatalifecyclev1.ResourceType, resourceID string) (*metadatalifecyclev1.CopyPublishedResourceResponse, error) {
	group, err := uuid.Parse(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group id: %w", err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resources.CopyPublishedResource(ctx, &metadatalifecyclev1.CopyPublishedResourceRequest{ResourceType: resourceType, ResourceId: resourceID, GroupId: group.String()})
	if err != nil {
		return nil, c.wrapError(ctx, "copy metadata-lifecycle published resource", err)
	}
	return response, nil
}
