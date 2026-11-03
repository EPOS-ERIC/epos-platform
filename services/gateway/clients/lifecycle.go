package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
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

// GetResourceVersion returns the unchanged response and fields needed by gateway orchestration.
func (c *MetadataLifecycle) GetResourceVersion(ctx context.Context, actorID string, resourceType domain.ResourceType, resourceID, versionID uuid.UUID) (*metadatalifecyclev1.GetResourceVersionResponse, domain.ResourceVersion, error) {
	kind, err := toProtoResourceType(resourceType)
	if err != nil {
		return nil, domain.ResourceVersion{}, err
	}

	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}

	response, err := c.resources.GetResourceVersion(ctx, &metadatalifecyclev1.GetResourceVersionRequest{
		ResourceType: kind,
		ResourceId:   resourceID.String(),
		VersionId:    versionID.String(),
	})
	if err != nil {
		return nil, domain.ResourceVersion{}, c.wrapError(ctx, "get metadata-lifecycle resource version", err)
	}

	result, err := fromProtoResourceVersion(response.GetResource())
	if err != nil {
		return nil, domain.ResourceVersion{}, err
	}

	return response, result, nil
}

func (c *MetadataLifecycle) PrepareDistributionDownload(ctx context.Context, actorID string, distributionID uuid.UUID) (domain.DownloadTarget, error) {
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}

	response, err := c.resources.PrepareDistributionDownload(ctx, &metadatalifecyclev1.PrepareDistributionDownloadRequest{
		DistributionId: distributionID.String(),
	})
	if err != nil {
		return domain.DownloadTarget{}, c.wrapError(ctx, "prepare distribution download", err)
	}

	return fromProtoPreparedDistributionDownload(response)
}

func (c *MetadataLifecycle) DistributionFormats(ctx context.Context, actorID string, distributionID uuid.UUID) ([]string, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	response, err := c.resources.GetDistributionFormats(ctx, &metadatalifecyclev1.GetDistributionFormatsRequest{DistributionId: distributionID.String()})
	if err != nil {
		return nil, c.wrapError(ctx, "get distribution formats", err)
	}

	return response.GetFormats(), nil
}

func (c *MetadataLifecycle) CheckDistributionRead(ctx context.Context, actorID string, distributionID uuid.UUID) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.resources.CheckDistributionRead(ctx, &metadatalifecyclev1.CheckDistributionReadRequest{DistributionId: distributionID.String()})
	if err != nil {
		return c.wrapError(ctx, "check distribution read", err)
	}

	return nil
}

func (c *MetadataLifecycle) PrepareDistributionAccess(ctx context.Context, actorID string, request domain.AccessRequest) (domain.PreparedRequest, error) {
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}

	input, err := toProtoDistributionAccess(request)
	if err != nil {
		return domain.PreparedRequest{}, err
	}
	response, err := c.resources.PrepareDistributionAccess(ctx, input)
	if err != nil {
		return domain.PreparedRequest{}, c.wrapError(ctx, "prepare distribution access", err)
	}

	return fromProtoPreparedDistributionAccess(response)
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
