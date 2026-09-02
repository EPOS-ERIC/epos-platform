package grpcapi

import (
	"context"
	"errors"
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
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "actor required")
	}

	values := md.Get(actorMetadataKey)
	if len(values) != 1 {
		return nil, status.Error(codes.Unauthenticated, "actor required")
	}

	actor, err := domain.NewActor(values[0])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid actor")
	}

	user, err := u.users.GetByID(ctx, actor, request.UserId)
	if err != nil {
		return nil, toUserError(err)
	}

	userStatus, err := toProtoUserStatus(user.Status)
	if err != nil {
		return nil, status.Error(codes.Internal, "map user status")
	}

	globalRoles, err := mapSliceErr(user.GlobalRoles, toProtoGlobalRole)
	if err != nil {
		return nil, status.Error(codes.Internal, "map global roles")
	}

	memberships, err := mapSliceErr(user.Memberships, toProtoMembership)
	if err != nil {
		return nil, status.Error(codes.Internal, "map memberships")
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

func toUserError(err error) error {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, domain.ErrUserForbidden):
		return status.Error(codes.PermissionDenied, "user read is not permitted")
	default:
		return status.Error(codes.Internal, "get user")
	}
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
		Group: &metadatalifecyclev1.GroupReference{
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
