package grpcapi

import (
	"context"
	"fmt"
	"slices"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const actorMetadataKey = "x-epos-user-id"

type UserServer struct {
	metadatalifecyclev1.UnimplementedUserServiceServer
	users *users.Service
}

func NewUser(users *users.Service) *UserServer {
	return &UserServer{
		users: users,
	}
}

func (u *UserServer) GetUser(ctx context.Context, request *metadatalifecyclev1.GetUserRequest) (*metadatalifecyclev1.GetUserResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := u.users.GetByID(ctx, actor, request.UserId)
	if err != nil {
		return nil, wrapHandlerError(err)
	}

	userStatus, err := toProtoUserStatus(user.Status)
	if err != nil {
		return nil, err
	}

	globalRoles, err := mapSliceErr(user.GlobalRoles, toProtoGlobalRole)
	if err != nil {
		return nil, err
	}

	memberships, err := mapSliceErr(user.Memberships, toProtoMembership)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.GetUserResponse{
		Id:                user.ID,
		Email:             user.Email,
		GivenName:         user.GivenName,
		FamilyName:        user.FamilyName,
		Status:            userStatus,
		GlobalRoles:       globalRoles,
		GlobalPermissions: slices.Clone(user.GlobalPermissions),
		CreatedAt:         timestamppb.New(user.CreatedAt),
		UpdatedAt:         timestamppb.New(user.UpdatedAt),
		Memberships:       memberships,
	}, nil
}

func (u *UserServer) ListUsers(ctx context.Context, _ *metadatalifecyclev1.ListUsersRequest) (*metadatalifecyclev1.ListUsersResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	users, err := u.users.List(ctx, actor)
	if err != nil {
		return nil, wrapHandlerError(err)
	}

	result, err := mapSliceErr(users, toProtoUserListItem)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.ListUsersResponse{Users: result}, nil
}

func (u *UserServer) GrantGlobalPermission(ctx context.Context, request *metadatalifecyclev1.GrantGlobalPermissionRequest) (*metadatalifecyclev1.GrantGlobalPermissionResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.users.GrantGlobalPermission(ctx, actor, request.GetUserId(), request.GetPermission()); err != nil {
		return nil, wrapHandlerError(err)
	}

	return &metadatalifecyclev1.GrantGlobalPermissionResponse{}, nil
}

func (u *UserServer) RevokeGlobalPermission(ctx context.Context, request *metadatalifecyclev1.RevokeGlobalPermissionRequest) (*metadatalifecyclev1.RevokeGlobalPermissionResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.users.RevokeGlobalPermission(ctx, actor, request.GetUserId(), request.GetPermission()); err != nil {
		return nil, wrapHandlerError(err)
	}

	return &metadatalifecyclev1.RevokeGlobalPermissionResponse{}, nil
}

func (u *UserServer) AssignGlobalRole(ctx context.Context, request *metadatalifecyclev1.AssignGlobalRoleRequest) (*metadatalifecyclev1.AssignGlobalRoleResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	role, err := toDomainGlobalRole(request.GetRole())
	if err != nil {
		return nil, err
	}
	if err := u.users.AssignGlobalRole(ctx, actor, request.GetUserId(), string(role)); err != nil {
		return nil, wrapHandlerError(err)
	}

	return &metadatalifecyclev1.AssignGlobalRoleResponse{}, nil
}

func (u *UserServer) RemoveGlobalRole(ctx context.Context, request *metadatalifecyclev1.RemoveGlobalRoleRequest) (*metadatalifecyclev1.RemoveGlobalRoleResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	role, err := toDomainGlobalRole(request.GetRole())
	if err != nil {
		return nil, err
	}
	if err := u.users.RemoveGlobalRole(ctx, actor, request.GetUserId(), string(role)); err != nil {
		return nil, wrapHandlerError(err)
	}

	return &metadatalifecyclev1.RemoveGlobalRoleResponse{}, nil
}

func (u *UserServer) BlockUser(ctx context.Context, request *metadatalifecyclev1.BlockUserRequest) (*metadatalifecyclev1.BlockUserResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.users.BlockUser(ctx, actor, request.GetUserId(), request.GetReason()); err != nil {
		return nil, wrapHandlerError(err)
	}

	return &metadatalifecyclev1.BlockUserResponse{}, nil
}

func (u *UserServer) UnblockUser(ctx context.Context, request *metadatalifecyclev1.UnblockUserRequest) (*metadatalifecyclev1.UnblockUserResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.users.UnblockUser(ctx, actor, request.GetUserId()); err != nil {
		return nil, wrapHandlerError(err)
	}

	return &metadatalifecyclev1.UnblockUserResponse{}, nil
}

func actorFromContext(ctx context.Context) (domain.Actor, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "actor required")
	}

	values := md.Get(actorMetadataKey)
	if len(values) != 1 {
		return "", status.Error(codes.Unauthenticated, "actor required")
	}

	actor, err := domain.NewActor(values[0])
	if err != nil {
		return "", status.Error(codes.InvalidArgument, "invalid actor")
	}

	return actor, nil
}

func toProtoUserListItem(value domain.UserSummary) (*metadatalifecyclev1.UserListItem, error) {
	status, err := toProtoUserStatus(value.Status)
	if err != nil {
		return nil, err
	}

	globalRoles, err := mapSliceErr(value.GlobalRoles, toProtoGlobalRole)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.UserListItem{
		Id:                value.ID,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            status,
		GlobalRoles:       globalRoles,
		GlobalPermissions: slices.Clone(value.GlobalPermissions),
		CreatedAt:         timestamppb.New(value.CreatedAt),
		UpdatedAt:         timestamppb.New(value.UpdatedAt),
	}, nil
}

func toProtoMembership(value domain.Membership) (*metadatalifecyclev1.UserMembership, error) {
	roles, err := mapSliceErr(value.Roles, toProtoGroupRole)
	if err != nil {
		return nil, err
	}

	status, err := toProtoMembershipStatus(value.Status)
	if err != nil {
		return nil, err
	}

	roleRequests, err := mapSliceErr(value.RoleRequests, toProtoRoleRequest)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.UserMembership{
		Group: &metadatalifecyclev1.EntityReference{
			Id:   value.Group.ID.String(),
			Name: value.Group.Name,
		},
		GroupRoles:       roles,
		Status:           status,
		GroupPermissions: slices.Clone(value.Permissions),
		RoleRequests:     roleRequests,
		CreatedAt:        timestamppb.New(value.CreatedAt),
		UpdatedAt:        timestamppb.New(value.UpdatedAt),
	}, nil
}

func toProtoRoleRequest(value domain.RoleRequest) (*metadatalifecyclev1.RoleRequest, error) {
	role, err := toProtoGroupRole(value.Role)
	if err != nil {
		return nil, err
	}

	status, err := toProtoRoleRequestStatus(value.Status)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.RoleRequest{
		Id:          value.ID.String(),
		GroupRole:   role,
		Status:      status,
		RequestedAt: timestamppb.New(value.RequestedAt),
	}, nil
}

// TODO: split this file/package correctly, we don't want all these mapping functions in the same file

func toProtoRoleRequestStatus(value domain.RoleRequestStatus) (metadatalifecyclev1.RoleRequestStatus, error) {
	switch value {
	case domain.RoleRequestPending:
		return metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_PENDING, nil
	case domain.RoleRequestAccepted:
		return metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_ACCEPTED, nil
	case domain.RoleRequestRejected:
		return metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_REJECTED, nil
	default:
		return 0, fmt.Errorf("unknown role request status: %q", value)
	}
}

func toProtoGroupRole(value domain.GroupRole) (metadatalifecyclev1.GroupRole, error) {
	switch value {
	case domain.GroupRoleEditor:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR, nil
	case domain.GroupRoleReviewer:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER, nil
	case domain.GroupRoleAdmin:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_ADMIN, nil
	default:
		return 0, fmt.Errorf("unknown group role: %q", value)
	}
}

func toProtoMembershipStatus(value domain.MembershipStatus) (metadatalifecyclev1.MembershipStatus, error) {
	switch value {
	case domain.MembershipPending:
		return metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_PENDING, nil
	case domain.MembershipActive:
		return metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_ACTIVE, nil
	case domain.MembershipRejected:
		return metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_REJECTED, nil
	default:
		return 0, fmt.Errorf("unknown membership status: %q", value)
	}
}

func toProtoUserStatus(value domain.UserStatus) (metadatalifecyclev1.UserStatus, error) {
	switch value {
	case domain.UserStatusActive:
		return metadatalifecyclev1.UserStatus_USER_STATUS_ACTIVE, nil
	case domain.UserStatusBlocked:
		return metadatalifecyclev1.UserStatus_USER_STATUS_BLOCKED, nil
	default:
		return 0, fmt.Errorf("unknown user status: %q", value)
	}
}

func toProtoGlobalRole(value domain.GlobalRole) (metadatalifecyclev1.GlobalRole, error) {
	switch value {
	case domain.GlobalRoleAdmin:
		return metadatalifecyclev1.GlobalRole_GLOBAL_ROLE_ADMIN, nil
	default:
		return 0, fmt.Errorf("unknown global role: %q", value)
	}
}

func toDomainGlobalRole(value metadatalifecyclev1.GlobalRole) (domain.GlobalRole, error) {
	switch value {
	case metadatalifecyclev1.GlobalRole_GLOBAL_ROLE_ADMIN:
		return domain.GlobalRoleAdmin, nil
	default:
		return "", domain.ErrInvalidUser
	}
}

func mapSliceErr[T, K any](values []T, f func(T) (K, error)) ([]K, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]K, len(values))
	for i, value := range values {
		mapped, err := f(value)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
	}
	return result, nil
}
