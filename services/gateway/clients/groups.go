package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) CreateGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateGroupRequest) (*metadatalifecyclev1.CreateGroupResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.groups.CreateGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle group", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ReplaceGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceGroupRequest) (*metadatalifecyclev1.ReplaceGroupResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.groups.ReplaceGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle group", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) DeleteGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteGroupRequest) (*metadatalifecyclev1.DeleteGroupResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.groups.DeleteGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("delete metadata-lifecycle group", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ListGroupMemberships(ctx context.Context, actorID string, request *metadatalifecyclev1.ListGroupMembershipsRequest) (*metadatalifecyclev1.ListGroupMembershipsResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.access.ListGroupMemberships(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("list memberships", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ListCategories(ctx context.Context, request *metadatalifecyclev1.ListCategoriesRequest) (*metadatalifecyclev1.ListCategoriesResponse, error) {
	response, err := c.categories.ListCategories(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle categories", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) GetCategory(ctx context.Context, request *metadatalifecyclev1.GetCategoryRequest) (*metadatalifecyclev1.GetCategoryResponse, error) {
	response, err := c.categories.GetCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle category", err)
	}
	if response == nil || response.Category == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle category: response is empty")
	}
	return response, nil
}

func (c *MetadataLifecycle) CreateCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateCategoryRequest) (*metadatalifecyclev1.CreateCategoryResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.categories.CreateCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle category", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ReplaceCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceCategoryRequest) (*metadatalifecyclev1.ReplaceCategoryResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.categories.ReplaceCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle category", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) DeleteCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteCategoryRequest) (*metadatalifecyclev1.DeleteCategoryResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.categories.DeleteCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("delete metadata-lifecycle category", err)
	}
	return response, nil
}
