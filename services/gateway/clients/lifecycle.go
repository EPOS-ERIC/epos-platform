package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func (c *MetadataLifecycle) CreateResource(ctx context.Context, actorID, groupID, resourceType string, input domain.CreateResourceInput) (domain.ResourceVersion, error) {
	request, err := toClientCreateResourceRequest(resourceType, input.Metadata)
	if err != nil {
		return domain.ResourceVersion{}, err
	}

	record, response, err := c.resources.CreateResource(ctx, groupID).
		XEPOSUserID(actorID).
		CreateResourceRequest(request).Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}

	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) GetResourceVersion(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	c.log.Debug("get resource version", "resourceType", resourceType, "resourceID", resourceID, "versionID", versionID)

	request := c.resources.GetResourceVersion(ctx, resourceType, resourceID, versionID)
	if actorID != "" {
		request = request.XEPOSUserID(actorID)
	}

	c.log.Debug("get resource version", "request", request)

	record, response, err := request.Execute()
	if err != nil {
		c.log.Debug("get resource version", "error", err)
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}

	c.log.Debug("get resource version", "response", record)

	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) ReplaceResourceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.ReplaceResourceDraftInput) (domain.ResourceVersion, error) {
	request, err := toClientReplaceResourceDraftRequest(input)
	if err != nil {
		return domain.ResourceVersion{}, err
	}

	record, response, err := c.resources.ReplaceResourceDraft(ctx, resourceID, versionID).
		XEPOSUserID(actorID).
		ReplaceResourceDraftRequest(request).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}

	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) DiscardResourceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string) error {
	response, err := c.resources.DiscardResourceDraft(ctx, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	return lifecycleError(response, err)
}

func (c *MetadataLifecycle) SubmitResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.SubmitResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) WithdrawResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.WithdrawResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) RejectResource(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.RejectResourceInput) (domain.ResourceVersion, error) {
	record, response, err := c.resources.RejectResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		RejectRequest(toClientRejectRequest(input)).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) ReopenResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.ReopenResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) PublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.PublishResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) UnpublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.UnpublishResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) CreateResourceDraftFromPublication(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.CreateResourceDraftFromPublication(ctx, resourceType, resourceID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) CreateResourceDraftFromArchive(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.CreateResourceDraftFromArchive(ctx, resourceType, resourceID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func (c *MetadataLifecycle) CopyResource(ctx context.Context, actorID, groupID, resourceType, resourceID string, input domain.CopyResourceInput) (domain.ResourceVersion, error) {
	record, response, err := c.resources.CopyResource(ctx, resourceType, resourceID).
		XEPOSUserID(actorID).
		CopyResourceRequest(toClientCopyResourceRequest(groupID, input)).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return toDomainResourceVersion(record)
}

func lifecycleError(response *http.Response, err error) error {
	if err == nil {
		return nil
	}
	if response != nil {
		if response.Body != nil {
			_ = response.Body.Close()
		}
		return &domain.UpstreamError{StatusCode: response.StatusCode, WrappedError: err}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("metadata-lifecycle lifecycle request: %w", domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("metadata-lifecycle lifecycle request: %w", err)
}
