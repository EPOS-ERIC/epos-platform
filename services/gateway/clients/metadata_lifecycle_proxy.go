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
	accessGRPC       metadatalifecyclev1.GroupAccessServiceClient
	resourceGRPC     metadatalifecyclev1.ResourceServiceClient
	discoveryGRPC    metadatalifecyclev1.DiscoveryServiceClient
	importGRPC       metadatalifecyclev1.ImportServiceClient
}

func (c *MetadataLifecycleGRPC) ImportTurtle(ctx context.Context, actorID string, request *metadatalifecyclev1.ImportTurtleRequest) (*metadatalifecyclev1.ImportTurtleResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.importGRPC.ImportTurtle(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("import metadata-lifecycle Turtle", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) CreateResource(ctx context.Context, actorID string, request *metadatalifecyclev1.CreateResourceRequest) (*metadatalifecyclev1.CreateResourceResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resourceGRPC.CreateResource(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) GetResourceVersion(ctx context.Context, actorID string, request *metadatalifecyclev1.GetResourceVersionRequest) (*metadatalifecyclev1.GetResourceVersionResponse, error) {
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}
	response, err := c.resourceGRPC.GetResourceVersion(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle resource version", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) SearchResources(ctx context.Context, actorID string, request *metadatalifecyclev1.SearchResourcesRequest) (*metadatalifecyclev1.SearchResourcesResponse, error) {
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}
	response, err := c.discoveryGRPC.SearchResources(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("search metadata-lifecycle resources", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ReplaceResourceDraft(ctx context.Context, actorID string, request *metadatalifecyclev1.ReplaceResourceDraftRequest) (*metadatalifecyclev1.ReplaceResourceDraftResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resourceGRPC.ReplaceResourceDraft(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("replace metadata-lifecycle resource draft", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) DiscardResourceDraft(ctx context.Context, actorID string, request *metadatalifecyclev1.DiscardResourceDraftRequest) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.resourceGRPC.DiscardResourceDraft(ctx, request)
	if err != nil {
		return wrapGRPCError("discard metadata-lifecycle resource draft", err)
	}
	return nil
}

func (c *MetadataLifecycleGRPC) SubmitResource(ctx context.Context, actorID string, request *metadatalifecyclev1.SubmitResourceRequest) (*metadatalifecyclev1.SubmitResourceResponse, error) {
	response, err := c.resourceGRPC.SubmitResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), request)
	if err != nil {
		return nil, wrapGRPCError("submit metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) WithdrawResource(ctx context.Context, actorID string, request *metadatalifecyclev1.WithdrawResourceRequest) (*metadatalifecyclev1.WithdrawResourceResponse, error) {
	response, err := c.resourceGRPC.WithdrawResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), request)
	if err != nil {
		return nil, wrapGRPCError("withdraw metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ReopenResource(ctx context.Context, actorID string, request *metadatalifecyclev1.ReopenResourceRequest) (*metadatalifecyclev1.ReopenResourceResponse, error) {
	response, err := c.resourceGRPC.ReopenResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), request)
	if err != nil {
		return nil, wrapGRPCError("reopen metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) PublishResource(ctx context.Context, actorID string, request *metadatalifecyclev1.PublishResourceRequest) (*metadatalifecyclev1.PublishResourceResponse, error) {
	response, err := c.resourceGRPC.PublishResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), request)
	if err != nil {
		return nil, wrapGRPCError("publish metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) UnpublishResource(ctx context.Context, actorID string, request *metadatalifecyclev1.UnpublishResourceRequest) (*metadatalifecyclev1.UnpublishResourceResponse, error) {
	response, err := c.resourceGRPC.UnpublishResource(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), request)
	if err != nil {
		return nil, wrapGRPCError("unpublish metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) RejectResource(ctx context.Context, actorID string, request *metadatalifecyclev1.RejectResourceRequest) (*metadatalifecyclev1.RejectResourceResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resourceGRPC.RejectResource(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("reject metadata-lifecycle resource", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) DraftResourceFromPublication(ctx context.Context, actorID string, request *metadatalifecyclev1.DraftResourceFromPublicationRequest) (*metadatalifecyclev1.DraftResourceFromPublicationResponse, error) {
	response, err := c.resourceGRPC.DraftResourceFromPublication(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle draft from publication", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) DraftResourceFromLatestArchive(ctx context.Context, actorID string, request *metadatalifecyclev1.DraftResourceFromLatestArchiveRequest) (*metadatalifecyclev1.DraftResourceFromLatestArchiveResponse, error) {
	response, err := c.resourceGRPC.DraftResourceFromLatestArchive(metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID), request)
	if err != nil {
		return nil, wrapGRPCError("create metadata-lifecycle draft from archive", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) CopyPublishedResource(ctx context.Context, actorID string, request *metadatalifecyclev1.CopyPublishedResourceRequest) (*metadatalifecyclev1.CopyPublishedResourceResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.resourceGRPC.CopyPublishedResource(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("copy metadata-lifecycle published resource", err)
	}
	return response, nil
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

func (c *MetadataLifecycleGRPC) ListGroupMemberships(ctx context.Context, actorID string, request *metadatalifecyclev1.ListGroupMembershipsRequest) (*metadatalifecyclev1.ListGroupMembershipsResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.ListGroupMemberships(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("list memberships", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) RequestMembership(ctx context.Context, actorID string, request *metadatalifecyclev1.RequestMembershipRequest) (*metadatalifecyclev1.RequestMembershipResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.RequestMembership(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("request membership", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) WithdrawMembership(ctx context.Context, actorID string, request *metadatalifecyclev1.WithdrawMembershipRequest) (*metadatalifecyclev1.WithdrawMembershipResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.WithdrawMembership(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("withdraw membership", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) DecideMembership(ctx context.Context, actorID string, request *metadatalifecyclev1.DecideMembershipRequest) (*metadatalifecyclev1.DecideMembershipResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.DecideMembership(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("decide membership", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) RevokeMembership(ctx context.Context, actorID string, request *metadatalifecyclev1.RevokeMembershipRequest) (*metadatalifecyclev1.RevokeMembershipResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.RevokeMembership(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("revoke membership", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) RequestRole(ctx context.Context, actorID string, request *metadatalifecyclev1.RequestRoleRequest) (*metadatalifecyclev1.RequestRoleResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.RequestRole(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("request role", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) ApplyRoleRequestDecision(ctx context.Context, actorID string, request *metadatalifecyclev1.ApplyRoleRequestDecisionRequest) (*metadatalifecyclev1.ApplyRoleRequestDecisionResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.ApplyRoleRequestDecision(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("decide role request", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) WithdrawRoleRequest(ctx context.Context, actorID string, request *metadatalifecyclev1.WithdrawRoleRequestRequest) (*metadatalifecyclev1.WithdrawRoleRequestResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.WithdrawRoleRequest(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("withdraw role request", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) AssignRole(ctx context.Context, actorID string, request *metadatalifecyclev1.AssignRoleRequest) (*metadatalifecyclev1.AssignRoleResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.AssignRole(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("assign role", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) RemoveRole(ctx context.Context, actorID string, request *metadatalifecyclev1.RemoveRoleRequest) (*metadatalifecyclev1.RemoveRoleResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.RemoveRole(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("remove role", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) GrantPermission(ctx context.Context, actorID string, request *metadatalifecyclev1.GrantPermissionRequest) (*metadatalifecyclev1.GrantPermissionResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.GrantPermission(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("grant permission", err)
	}
	return response, nil
}

func (c *MetadataLifecycleGRPC) RevokePermission(ctx context.Context, actorID string, request *metadatalifecyclev1.RevokePermissionRequest) (*metadatalifecyclev1.RevokePermissionResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.accessGRPC.RevokePermission(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("revoke permission", err)
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
