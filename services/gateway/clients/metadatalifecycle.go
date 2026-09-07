// Package clients provides outbound service clients for the gateway.
package clients

import (
	"context"
	"errors"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TODO: clean up this package, create a sub package metadatalifecycle and in there split the grpc functions into logical files

type MetadataLifecycle struct {
	users         metadatalifecyclev1.UserServiceClient
	groups        metadatalifecyclev1.GroupServiceClient
	categories    metadatalifecyclev1.CategoryServiceClient
	contactPoints metadatalifecyclev1.ContactPointServiceClient
	organizations metadatalifecyclev1.OrganizationServiceClient
	access        metadatalifecyclev1.GroupAccessServiceClient
	resources     metadatalifecyclev1.ResourceServiceClient
	discovery     metadatalifecyclev1.DiscoveryServiceClient
	import_       metadatalifecyclev1.ImportServiceClient
	identity      metadatalifecyclev1.IdentityServiceClient
}

const actorMetadataKey = "x-epos-user-id"

func NewMetadataLifecycle(baseURL string) (*MetadataLifecycle, error) {
	conn, err := grpc.NewClient(
		baseURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("create metadata-lifecycle gRPC client: %w", err)
	}

	client := &MetadataLifecycle{
		users:         metadatalifecyclev1.NewUserServiceClient(conn),
		groups:        metadatalifecyclev1.NewGroupServiceClient(conn),
		categories:    metadatalifecyclev1.NewCategoryServiceClient(conn),
		contactPoints: metadatalifecyclev1.NewContactPointServiceClient(conn),
		organizations: metadatalifecyclev1.NewOrganizationServiceClient(conn),
		access:        metadatalifecyclev1.NewGroupAccessServiceClient(conn),
		resources:     metadatalifecyclev1.NewResourceServiceClient(conn),
		discovery:     metadatalifecyclev1.NewDiscoveryServiceClient(conn),
		import_:       metadatalifecyclev1.NewImportServiceClient(conn),
		identity:      metadatalifecyclev1.NewIdentityServiceClient(conn),
	}

	return client, nil
}

func wrapGRPCError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}

	switch status.Code(err) {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return fmt.Errorf("%s: %w", operation, domain.ErrBadRequest)
	case codes.NotFound:
		return fmt.Errorf("%s: %w", operation, domain.ErrNotFound)
	case codes.AlreadyExists, codes.Aborted:
		return fmt.Errorf("%s: %w", operation, domain.ErrConflict)
	case codes.PermissionDenied:
		return fmt.Errorf("%s: %w", operation, domain.ErrForbidden)
	case codes.Unauthenticated:
		return fmt.Errorf("%s: %w", operation, domain.ErrUnauthorized)
	case codes.DeadlineExceeded, codes.Canceled:
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	default:
		// Internal, Unavailable, Unknown, ResourceExhausted, Unimplemented.
		// Gateway handlers already turn unknown upstream failures into 503.
		return fmt.Errorf("%s: %w", operation, err)
	}
}

func (c *MetadataLifecycle) ListGroups(ctx context.Context) (*metadatalifecyclev1.ListGroupsResponse, error) {
	response, err := c.groups.ListGroups(ctx, &metadatalifecyclev1.ListGroupsRequest{})
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle groups", err)
	}

	return response, nil
}

func (c *MetadataLifecycle) ListUsers(ctx context.Context, actorID string) (*metadatalifecyclev1.ListUsersResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	users, err := c.users.ListUsers(ctx, &metadatalifecyclev1.ListUsersRequest{})
	if err != nil {
		return nil, wrapGRPCError("list metadata-lifecycle users", err)
	}

	return users, nil
}

func (c *MetadataLifecycle) GetGroup(ctx context.Context, request *metadatalifecyclev1.GetGroupRequest) (*metadatalifecyclev1.GetGroupResponse, error) {
	response, err := c.groups.GetGroup(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle group", err)
	}

	if response == nil || response.Group == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle group: response is empty")
	}

	return response, nil
}

func (c *MetadataLifecycle) WithdrawMembershipRequest(ctx context.Context, groupID, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.access.WithdrawMembership(ctx, &metadatalifecyclev1.WithdrawMembershipRequest{
		GroupId: groupID,
	})
	if err != nil {
		return wrapGRPCError("withdraw membership", err)
	}
	return nil
}

func (c *MetadataLifecycle) RequestMembership(ctx context.Context, groupID, actorID string, roles []metadatalifecyclev1.GroupRole) (*metadatalifecyclev1.RequestMembershipResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	response, err := c.access.RequestMembership(ctx, &metadatalifecyclev1.RequestMembershipRequest{
		GroupId: groupID,
		Roles:   roles,
	})
	if err != nil {
		return nil, wrapGRPCError("request membership", err)
	}

	return response, nil
}

func (c *MetadataLifecycle) DecideMembership(ctx context.Context, groupID, userID string, status metadatalifecyclev1.MembershipStatus, actorID string) (*metadatalifecyclev1.DecideMembershipResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	response, err := c.access.DecideMembership(ctx, &metadatalifecyclev1.DecideMembershipRequest{
		GroupId: groupID,
		UserId:  userID,
		Status:  status,
	})
	if err != nil {
		return nil, wrapGRPCError("decide membership", err)
	}

	return response, nil
}

func (c *MetadataLifecycle) RevokeMembership(ctx context.Context, groupID, userID, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.access.RevokeMembership(ctx, &metadatalifecyclev1.RevokeMembershipRequest{
		GroupId: groupID,
		UserId:  userID,
	})
	if err != nil {
		return wrapGRPCError("revoke membership", err)
	}
	return nil
}

func (c *MetadataLifecycle) RequestRole(ctx context.Context, groupID string, role metadatalifecyclev1.GroupRole, actorID string) (*metadatalifecyclev1.RequestRoleResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.access.RequestRole(ctx, &metadatalifecyclev1.RequestRoleRequest{
		GroupId: groupID,
		Role:    role,
	})
	if err != nil {
		return nil, wrapGRPCError("request role", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ApplyRoleRequestDecision(ctx context.Context, groupID, requestID string, status metadatalifecyclev1.RoleRequestStatus, actorID string) (*metadatalifecyclev1.ApplyRoleRequestDecisionResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.access.ApplyRoleRequestDecision(ctx, &metadatalifecyclev1.ApplyRoleRequestDecisionRequest{
		GroupId:   groupID,
		RequestId: requestID,
		Status:    status,
	})
	if err != nil {
		return nil, wrapGRPCError("decide role request", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) WithdrawRoleRequest(ctx context.Context, groupID, requestID, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.access.WithdrawRoleRequest(ctx, &metadatalifecyclev1.WithdrawRoleRequestRequest{
		GroupId:   groupID,
		RequestId: requestID,
	})
	if err != nil {
		return wrapGRPCError("withdraw role request", err)
	}
	return nil
}

func (c *MetadataLifecycle) AssignGroupRole(ctx context.Context, groupID, userID string, role metadatalifecyclev1.GroupRole, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.access.AssignRole(ctx, &metadatalifecyclev1.AssignRoleRequest{
		GroupId: groupID,
		UserId:  userID,
		Role:    role,
	})
	if err != nil {
		return wrapGRPCError("assign role", err)
	}
	return nil
}

func (c *MetadataLifecycle) RemoveGroupRole(ctx context.Context, groupID, userID string, role metadatalifecyclev1.GroupRole, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.access.RemoveRole(ctx, &metadatalifecyclev1.RemoveRoleRequest{
		GroupId: groupID,
		UserId:  userID,
		Role:    role,
	})
	if err != nil {
		return wrapGRPCError("remove role", err)
	}
	return nil
}

func (c *MetadataLifecycle) GrantGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.access.GrantPermission(ctx, &metadatalifecyclev1.GrantPermissionRequest{
		GroupId:    groupID,
		UserId:     userID,
		Permission: permission,
	})
	if err != nil {
		return wrapGRPCError("grant permission", err)
	}
	return nil
}

func (c *MetadataLifecycle) RevokeGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.access.RevokePermission(ctx, &metadatalifecyclev1.RevokePermissionRequest{
		GroupId:    groupID,
		UserId:     userID,
		Permission: permission,
	})
	if err != nil {
		return wrapGRPCError("revoke permission", err)
	}
	return nil
}

func (c *MetadataLifecycle) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.users.RevokeGlobalPermission(ctx, &metadatalifecyclev1.RevokeGlobalPermissionRequest{
		UserId:     userID,
		Permission: permission,
	})
	if err != nil {
		return wrapGRPCError("revoke metadata-lifecycle global permission", err)
	}
	return nil
}

func (c *MetadataLifecycle) GrantGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.users.GrantGlobalPermission(ctx, &metadatalifecyclev1.GrantGlobalPermissionRequest{
		UserId:     userID,
		Permission: permission,
	})
	if err != nil {
		return wrapGRPCError("grant metadata-lifecycle global permission", err)
	}
	return nil
}

func (c *MetadataLifecycle) AssignGlobalRole(ctx context.Context, actorID, userID string, role metadatalifecyclev1.GlobalRole) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.users.AssignGlobalRole(ctx, &metadatalifecyclev1.AssignGlobalRoleRequest{
		UserId: userID,
		Role:   role,
	})
	if err != nil {
		return wrapGRPCError("assign metadata-lifecycle global role", err)
	}
	return nil
}

func (c *MetadataLifecycle) RemoveGlobalRole(ctx context.Context, actorID, userID string, role metadatalifecyclev1.GlobalRole) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.users.RemoveGlobalRole(ctx, &metadatalifecyclev1.RemoveGlobalRoleRequest{
		UserId: userID,
		Role:   role,
	})
	if err != nil {
		return wrapGRPCError("remove metadata-lifecycle global role", err)
	}
	return nil
}

func (c *MetadataLifecycle) BlockUser(ctx context.Context, actorID, userID, reason string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.users.BlockUser(ctx, &metadatalifecyclev1.BlockUserRequest{
		UserId: userID,
		Reason: reason,
	})
	if err != nil {
		return wrapGRPCError("block metadata-lifecycle user", err)
	}
	return nil
}

func (c *MetadataLifecycle) UnblockUser(ctx context.Context, actorID, userID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.users.UnblockUser(ctx, &metadatalifecyclev1.UnblockUserRequest{
		UserId: userID,
	})
	if err != nil {
		return wrapGRPCError("unblock metadata-lifecycle user", err)
	}
	return nil
}

func (c *MetadataLifecycle) ImportTurtle(ctx context.Context, actorID string, request *metadatalifecyclev1.ImportTurtleRequest) (*metadatalifecyclev1.ImportTurtleResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.import_.ImportTurtle(ctx, request)
	if err != nil {
		return nil, wrapGRPCError("import metadata-lifecycle Turtle", err)
	}
	return response, nil
}
