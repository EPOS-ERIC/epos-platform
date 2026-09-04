package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
)

func (c *MetadataLifecycle) ListContactPoints(ctx context.Context, actorID string) (*metadatalifecyclev1.ListContactPointsResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.ListContactPoints(ctx, actorID, &metadatalifecyclev1.ListContactPointsRequest{})
}

func (c *MetadataLifecycle) GetContactPoint(ctx context.Context, actorID, id string) (*metadatalifecyclev1.GetContactPointResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.GetContactPoint(ctx, actorID, &metadatalifecyclev1.GetContactPointRequest{ContactPointId: id})
}

func (c *MetadataLifecycle) CreateContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateContactPointRequest) (*metadatalifecyclev1.CreateContactPointResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.CreateContactPoint(ctx, actorID, request)
}

func (c *MetadataLifecycle) ReplaceContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceContactPointRequest) (*metadatalifecyclev1.ReplaceContactPointResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.ReplaceContactPoint(ctx, actorID, request)
}

func (c *MetadataLifecycle) DeleteContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteContactPointRequest) (*metadatalifecyclev1.DeleteContactPointResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.DeleteContactPoint(ctx, actorID, request)
}

func (c *MetadataLifecycle) ListOrganizations(ctx context.Context) (*metadatalifecyclev1.ListOrganizationsResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.ListOrganizations(ctx, &metadatalifecyclev1.ListOrganizationsRequest{})
}

func (c *MetadataLifecycle) GetOrganization(ctx context.Context, id string) (*metadatalifecyclev1.GetOrganizationResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.GetOrganization(ctx, &metadatalifecyclev1.GetOrganizationRequest{OrganizationId: id})
}

func (c *MetadataLifecycle) CreateOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateOrganizationRequest) (*metadatalifecyclev1.CreateOrganizationResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.CreateOrganization(ctx, actorID, request)
}

func (c *MetadataLifecycle) ReplaceOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceOrganizationRequest) (*metadatalifecyclev1.ReplaceOrganizationResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.ReplaceOrganization(ctx, actorID, request)
}

func (c *MetadataLifecycle) DeleteOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteOrganizationRequest) (*metadatalifecyclev1.DeleteOrganizationResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.DeleteOrganization(ctx, actorID, request)
}
