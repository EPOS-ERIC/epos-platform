package clients

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

type MetadataLifecycleGRPC struct {
	userGRPC         metadatalifecyclev1.UserServiceClient
	groupGRPC        metadatalifecyclev1.GroupServiceClient
	categoryGRPC     metadatalifecyclev1.CategoryServiceClient
	contactPointGRPC metadatalifecyclev1.ContactPointServiceClient
	organizationGRPC metadatalifecyclev1.OrganizationServiceClient
}

func (c *MetadataLifecycleGRPC) GetUser(ctx context.Context, actorID, id string) (*metadatalifecyclev1.GetUserResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	user, err := c.userGRPC.GetUser(ctx, &metadatalifecyclev1.GetUserRequest{
		UserId: id,
	})
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle user", err)
	}

	return user, nil
}

func (c *MetadataLifecycleGRPC) ListUsers(ctx context.Context, actorID string) (*metadatalifecyclev1.ListUsersResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	users, err := c.userGRPC.ListUsers(ctx, &metadatalifecyclev1.ListUsersRequest{})
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle users", err)
	}

	return users, nil
}

func (c *MetadataLifecycleGRPC) ListGroups(ctx context.Context) (*metadatalifecyclev1.ListGroupsResponse, error) {
	groups, err := c.groupGRPC.ListGroups(ctx, &metadatalifecyclev1.ListGroupsRequest{})
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle groups", err)
	}

	return groups, nil
}

func (c *MetadataLifecycleGRPC) GetGroup(ctx context.Context, request *metadatalifecyclev1.GetGroupRequest) (*metadatalifecyclev1.GetGroupResponse, error) {
	group, err := c.groupGRPC.GetGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle group", err)
	}

	return group, nil
}

func (c *MetadataLifecycleGRPC) ListCategories(ctx context.Context, request *metadatalifecyclev1.ListCategoriesRequest) (*metadatalifecyclev1.ListCategoriesResponse, error) {
	response, err := c.categoryGRPC.ListCategories(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle categories", err)
	}

	return response, nil
}

func (c *MetadataLifecycleGRPC) GetCategory(ctx context.Context, request *metadatalifecyclev1.GetCategoryRequest) (*metadatalifecyclev1.GetCategoryResponse, error) {
	response, err := c.categoryGRPC.GetCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle category", err)
	}

	return response, nil
}

func (c *MetadataLifecycleGRPC) CreateCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateCategoryRequest) (*metadatalifecyclev1.CreateCategoryResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.categoryGRPC.CreateCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle category", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ReplaceCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceCategoryRequest) (*metadatalifecyclev1.ReplaceCategoryResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.categoryGRPC.ReplaceCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle category", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) DeleteCategory(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteCategoryRequest) (*metadatalifecyclev1.DeleteCategoryResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.categoryGRPC.DeleteCategory(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("delete metadata-lifecycle category", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ListContactPoints(ctx context.Context, actorID string, request *metadatalifecyclev1.ListContactPointsRequest) (*metadatalifecyclev1.ListContactPointsResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPointGRPC.ListContactPoints(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle contact points", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) GetContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.GetContactPointRequest) (*metadatalifecyclev1.GetContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPointGRPC.GetContactPoint(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) CreateContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateContactPointRequest) (*metadatalifecyclev1.CreateContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPointGRPC.CreateContactPoint(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ReplaceContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceContactPointRequest) (*metadatalifecyclev1.ReplaceContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPointGRPC.ReplaceContactPoint(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) DeleteContactPoint(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteContactPointRequest) (*metadatalifecyclev1.DeleteContactPointResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.contactPointGRPC.DeleteContactPoint(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("delete metadata-lifecycle contact point", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ListOrganizations(ctx context.Context, request *metadatalifecyclev1.ListOrganizationsRequest) (*metadatalifecyclev1.ListOrganizationsResponse, error) {
	response, err := c.organizationGRPC.ListOrganizations(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle organizations", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) GetOrganization(ctx context.Context, request *metadatalifecyclev1.GetOrganizationRequest) (*metadatalifecyclev1.GetOrganizationResponse, error) {
	response, err := c.organizationGRPC.GetOrganization(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle organization", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) CreateOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateOrganizationRequest) (*metadatalifecyclev1.CreateOrganizationResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.organizationGRPC.CreateOrganization(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle organization", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ReplaceOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceOrganizationRequest) (*metadatalifecyclev1.ReplaceOrganizationResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.organizationGRPC.ReplaceOrganization(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle organization", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) DeleteOrganization(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteOrganizationRequest) (*metadatalifecyclev1.DeleteOrganizationResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.organizationGRPC.DeleteOrganization(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("delete metadata-lifecycle organization", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) CreateGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateGroupRequest) (*metadatalifecyclev1.CreateGroupResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	group, err := c.groupGRPC.CreateGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle group", err)
	}

	return group, nil
}

func (c *MetadataLifecycleGRPC) ReplaceGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceGroupRequest) (*metadatalifecyclev1.ReplaceGroupResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	group, err := c.groupGRPC.ReplaceGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle group", err)
	}

	return group, nil
}

func (c *MetadataLifecycleGRPC) DeleteGroup(ctx context.Context, actorID string, request *metadatalifecyclev1.DeleteGroupRequest) (*metadatalifecyclev1.DeleteGroupResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	response, err := c.groupGRPC.DeleteGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("delete metadata-lifecycle group", err)
	}

	return response, nil
}
