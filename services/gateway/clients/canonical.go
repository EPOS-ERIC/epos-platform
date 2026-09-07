package clients

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) ListContactPoints(ctx context.Context, actorID string) (*metadatalifecyclev1.ListContactPointsResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPoints.ListContactPoints(ctx, &metadatalifecyclev1.ListContactPointsRequest{})
	if err != nil {
		return nil, c.wrapError(ctx, "list metadata-lifecycle contact points", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) GetContactPoint(ctx context.Context, actorID, id string) (*metadatalifecyclev1.GetContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPoints.GetContactPoint(ctx, &metadatalifecyclev1.GetContactPointRequest{ContactPointId: id})
	if err != nil {
		return nil, c.wrapError(ctx, "get metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateContactPointRequest) (*metadatalifecyclev1.CreateContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPoints.CreateContactPoint(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "create metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ReplaceContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceContactPointRequest) (*metadatalifecyclev1.ReplaceContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPoints.ReplaceContactPoint(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "replace metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) DeleteContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteContactPointRequest) (*metadatalifecyclev1.DeleteContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPoints.DeleteContactPoint(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "delete metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ListOrganizations(ctx context.Context) (*metadatalifecyclev1.ListOrganizationsResponse, error) {
	response, err := c.organizations.ListOrganizations(ctx, &metadatalifecyclev1.ListOrganizationsRequest{})
	if err != nil {
		return nil, c.wrapError(ctx, "list metadata-lifecycle organizations", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) GetOrganization(ctx context.Context, id string) (*metadatalifecyclev1.GetOrganizationResponse, error) {
	response, err := c.organizations.GetOrganization(ctx, &metadatalifecyclev1.GetOrganizationRequest{OrganizationId: id})
	if err != nil {
		return nil, c.wrapError(ctx, "get metadata-lifecycle organization", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateOrganizationRequest) (*metadatalifecyclev1.CreateOrganizationResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.organizations.CreateOrganization(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "create metadata-lifecycle organization", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ReplaceOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceOrganizationRequest) (*metadatalifecyclev1.ReplaceOrganizationResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.organizations.ReplaceOrganization(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "replace metadata-lifecycle organization", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) DeleteOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteOrganizationRequest) (*metadatalifecyclev1.DeleteOrganizationResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.organizations.DeleteOrganization(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "delete metadata-lifecycle organization", err)
	}
	return response, nil
}
