package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
)

func (c *MetadataLifecycle) CreateGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateGroupRequest) (*metadatalifecyclev1.CreateGroupResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	response, err := c.GRPC.CreateGroup(ctx, actorID, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *MetadataLifecycle) ReplaceGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceGroupRequest) (*metadatalifecyclev1.ReplaceGroupResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	response, err := c.GRPC.ReplaceGroup(ctx, actorID, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *MetadataLifecycle) DeleteGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteGroupRequest) (*metadatalifecyclev1.DeleteGroupResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	response, err := c.GRPC.DeleteGroup(ctx, actorID, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *MetadataLifecycle) ListGroupMemberships(ctx context.Context, actorID string, request *metadatalifecyclev1.ListGroupMembershipsRequest) (*metadatalifecyclev1.ListGroupMembershipsResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.ListGroupMemberships(ctx, actorID, request)
}

func (c *MetadataLifecycle) ListCategories(ctx context.Context, request *metadatalifecyclev1.ListCategoriesRequest) (*metadatalifecyclev1.ListCategoriesResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	response, err := c.GRPC.ListCategories(ctx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *MetadataLifecycle) GetCategory(ctx context.Context, request *metadatalifecyclev1.GetCategoryRequest) (*metadatalifecyclev1.GetCategoryResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	response, err := c.GRPC.GetCategory(ctx, request)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Category == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle category: response is empty")
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateCategoryRequest) (*metadatalifecyclev1.CreateCategoryResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.CreateCategory(ctx, actorID, request)
}

func (c *MetadataLifecycle) ReplaceCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceCategoryRequest) (*metadatalifecyclev1.ReplaceCategoryResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.ReplaceCategory(ctx, actorID, request)
}

func (c *MetadataLifecycle) DeleteCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteCategoryRequest) (*metadatalifecyclev1.DeleteCategoryResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.DeleteCategory(ctx, actorID, request)
}
