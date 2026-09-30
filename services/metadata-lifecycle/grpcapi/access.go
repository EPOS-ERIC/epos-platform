package grpcapi

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	accessservice "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/access"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AccessServer struct {
	metadatalifecyclev1.UnimplementedGroupAccessServiceServer
	access *accessservice.Service
}

func NewAccess(access *accessservice.Service) *AccessServer {
	return &AccessServer{access: access}
}

func (s *AccessServer) ListGroupMemberships(ctx context.Context, request *metadatalifecyclev1.ListGroupMembershipsRequest) (*metadatalifecyclev1.ListGroupMembershipsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	values, err := s.access.ListMemberships(ctx, actor, groupID)
	if err != nil {
		return nil, err
	}
	result := make([]*metadatalifecyclev1.GroupMembership, len(values))
	for i, value := range values {
		result[i], err = toProtoGroupMembership(value)
		if err != nil {
			return nil, err
		}
	}
	return &metadatalifecyclev1.ListGroupMembershipsResponse{Memberships: result}, nil
}

func (s *AccessServer) RequestMembership(ctx context.Context, request *metadatalifecyclev1.RequestMembershipRequest) (*metadatalifecyclev1.RequestMembershipResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}

	roles := make([]domain.GroupRole, len(request.Roles))
	seen := make(map[domain.GroupRole]struct{}, len(request.Roles))
	for i, role := range request.Roles {
		parsedRole, err := toDomainGroupRole(role)
		if err != nil {
			return nil, err
		}

		parsedRole, err = domain.NewRequestedGroupRole(string(parsedRole))
		if err != nil {
			return nil, err
		}

		if _, ok := seen[parsedRole]; ok {
			return nil, apperror.Default(apperror.CodeValidationFailed)
		}

		seen[parsedRole] = struct{}{}
		roles[i] = parsedRole
	}

	value, notifications, err := s.access.RequestMembership(ctx, actor, groupID, roles)
	if err != nil {
		return nil, err
	}

	membership, err := toProtoGroupMembership(value)
	if err != nil {
		return nil, err
	}

	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.RequestMembershipResponse{
		Membership:    membership,
		Notifications: protoNotifications,
	}, nil
}

func (s *AccessServer) WithdrawMembership(ctx context.Context, request *metadatalifecyclev1.WithdrawMembershipRequest) (*metadatalifecyclev1.WithdrawMembershipResponse, error) {
	err := s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		return s.access.WithdrawMembershipRequest(ctx, actor, groupID)
	})

	return &metadatalifecyclev1.WithdrawMembershipResponse{}, err
}

func (s *AccessServer) DecideMembership(ctx context.Context, request *metadatalifecyclev1.DecideMembershipRequest) (*metadatalifecyclev1.DecideMembershipResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}

	decision, err := toDomainDecision(request.Decision)
	if err != nil {
		return nil, err
	}

	value, notifications, err := s.access.DecideMembership(ctx, actor, groupID, request.UserId, decision)
	if err != nil {
		return nil, err
	}

	membership, err := toProtoGroupMembership(value)
	if err != nil {
		return nil, err
	}

	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.DecideMembershipResponse{
		Membership:    membership,
		Notifications: protoNotifications,
	}, nil
}

func (s *AccessServer) RevokeMembership(ctx context.Context, request *metadatalifecyclev1.RevokeMembershipRequest) (*metadatalifecyclev1.RevokeMembershipResponse, error) {
	err := s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		return s.access.RevokeMembership(ctx, actor, groupID, request.UserId)
	})
	return &metadatalifecyclev1.RevokeMembershipResponse{}, err
}

func (s *AccessServer) RequestRole(ctx context.Context, request *metadatalifecyclev1.RequestRoleRequest) (*metadatalifecyclev1.RequestRoleResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	role, err := toDomainGroupRole(request.Role)
	if err == nil {
		role, err = domain.NewRequestedGroupRole(string(role))
	}
	if err != nil {
		return nil, err
	}
	value, notifications, err := s.access.RequestRole(ctx, actor, groupID, role)
	if err != nil {
		return nil, err
	}
	roleRequest, err := toProtoRoleRequest(value)
	if err != nil {
		return nil, err
	}
	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.RequestRoleResponse{Request: roleRequest, Notifications: protoNotifications}, nil
}

func (s *AccessServer) DecideRoleRequest(ctx context.Context, request *metadatalifecyclev1.DecideRoleRequestRequest) (*metadatalifecyclev1.DecideRoleRequestResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	requestID, err := parseAccessID(request.RequestId, "request id")
	if err != nil {
		return nil, err
	}
	decision, err := toDomainDecision(request.Decision)
	if err != nil {
		return nil, err
	}
	value, notifications, err := s.access.DecideRoleRequest(ctx, actor, groupID, requestID, decision)
	if err != nil {
		return nil, err
	}
	membership, err := toProtoGroupMembership(value)
	if err != nil {
		return nil, err
	}
	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DecideRoleRequestResponse{Membership: membership, Notifications: protoNotifications}, nil
}

func (s *AccessServer) WithdrawRoleRequest(ctx context.Context, request *metadatalifecyclev1.WithdrawRoleRequestRequest) (*metadatalifecyclev1.WithdrawRoleRequestResponse, error) {
	err := s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		requestID, err := parseAccessID(request.RequestId, "request id")
		if err != nil {
			return err
		}
		return s.access.WithdrawRoleRequest(ctx, actor, groupID, requestID)
	})
	return &metadatalifecyclev1.WithdrawRoleRequestResponse{}, err
}

func (s *AccessServer) AssignRole(ctx context.Context, request *metadatalifecyclev1.AssignRoleRequest) (*metadatalifecyclev1.AssignRoleResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	role, err := toDomainGroupRole(request.Role)
	if err != nil {
		return nil, err
	}
	notifications, err := s.access.AssignRole(ctx, actor, groupID, request.UserId, role)
	if err != nil {
		return nil, err
	}

	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.AssignRoleResponse{Notifications: protoNotifications}, nil
}

func (s *AccessServer) RemoveRole(ctx context.Context, request *metadatalifecyclev1.RemoveRoleRequest) (*metadatalifecyclev1.RemoveRoleResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	role, err := toDomainGroupRole(request.Role)
	if err != nil {
		return nil, err
	}
	if err := s.access.RemoveRole(ctx, actor, groupID, request.UserId, role); err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.RemoveRoleResponse{}, nil
}

func (s *AccessServer) GrantPermission(ctx context.Context, request *metadatalifecyclev1.GrantPermissionRequest) (*metadatalifecyclev1.GrantPermissionResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	permission, err := toDomainGroupPermission(request.GetPermission())
	if err != nil {
		return nil, err
	}
	notifications, err := s.access.GrantPermission(ctx, actor, groupID, request.UserId, permission)
	if err != nil {
		return nil, err
	}
	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.GrantPermissionResponse{Notifications: protoNotifications}, nil
}

func (s *AccessServer) RevokePermission(ctx context.Context, request *metadatalifecyclev1.RevokePermissionRequest) (*metadatalifecyclev1.RevokePermissionResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	permission, err := toDomainGroupPermission(request.GetPermission())
	if err != nil {
		return nil, err
	}
	if err := s.access.RevokePermission(ctx, actor, groupID, request.UserId, permission); err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.RevokePermissionResponse{}, nil
}

func (s *AccessServer) runActor(ctx context.Context, f func(domain.Actor) error) error {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return err
	}
	return f(actor)
}

func parseAccessID(value, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.New(apperror.CodeValidationFailed, fmt.Sprintf("invalid %s", name))
	}
	return id, nil
}

func toProtoGroupMembership(value domain.GroupMembership) (*metadatalifecyclev1.GroupMembership, error) {
	roles := make([]metadatalifecyclev1.GroupRole, len(value.Roles))
	for i, role := range value.Roles {
		roles[i], _ = toProtoGroupRole(role)
	}
	requests := make([]*metadatalifecyclev1.RoleRequest, len(value.RoleRequests))
	for i, request := range value.RoleRequests {
		mapped, _ := toProtoRoleRequest(request)
		requests[i] = mapped
	}
	status, err := toProtoMembershipStatus(value.Status)
	if err != nil {
		return nil, err
	}
	grantedPermissions, err := mapSliceErr(value.GrantedPermissions, toProtoGroupPermission)
	if err != nil {
		return nil, err
	}
	rolePermissions, err := mapSliceErr(value.RolePermissions, toProtoGroupPermission)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.GroupMembership{
		User: &metadatalifecyclev1.MembershipUser{
			Id:         value.User.ID,
			GivenName:  value.User.GivenName,
			FamilyName: value.User.FamilyName,
			Email:      value.User.Email,
		},
		Status:             status,
		Roles:              roles,
		GrantedPermissions: grantedPermissions,
		RolePermissions:    rolePermissions,
		RoleRequests:       requests,
		CreatedAt:          timestamppb.New(value.CreatedAt),
		UpdatedAt:          timestamppb.New(value.UpdatedAt),
	}, nil
}

//nolint:dupl // Enum conversion functions intentionally use explicit one-to-one mappings.
func toProtoGroupPermission(value domain.GroupPermission) (metadatalifecyclev1.GroupPermission, error) {
	switch value {
	case domain.GroupPermissionResourceCreate:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_CREATE, nil
	case domain.GroupPermissionResourceEdit:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_EDIT, nil
	case domain.GroupPermissionResourceSubmit:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_SUBMIT, nil
	case domain.GroupPermissionResourceWithdraw:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_WITHDRAW, nil
	case domain.GroupPermissionResourceReopen:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_REOPEN, nil
	case domain.GroupPermissionResourceDiscard:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_DISCARD, nil
	case domain.GroupPermissionResourceReject:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_REJECT, nil
	case domain.GroupPermissionResourcePublish:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_PUBLISH, nil
	case domain.GroupPermissionResourceUnpublish:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_UNPUBLISH, nil
	case domain.GroupPermissionMembershipManage:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_MEMBERSHIP_MANAGE, nil
	case domain.GroupPermissionRoleManage:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_ROLE_MANAGE, nil
	case domain.GroupPermissionPermissionManage:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_PERMISSION_MANAGE, nil
	case domain.GroupPermissionNotificationManage:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_NOTIFICATION_MANAGE, nil
	case domain.GroupPermissionCategoryManage:
		return metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_CATEGORY_MANAGE, nil
	default:
		return 0, fmt.Errorf("unknown group permission: %q", value)
	}
}

func toDomainGroupPermission(value metadatalifecyclev1.GroupPermission) (domain.GroupPermission, error) {
	switch value {
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_CREATE:
		return domain.GroupPermissionResourceCreate, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_EDIT:
		return domain.GroupPermissionResourceEdit, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_SUBMIT:
		return domain.GroupPermissionResourceSubmit, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_WITHDRAW:
		return domain.GroupPermissionResourceWithdraw, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_REOPEN:
		return domain.GroupPermissionResourceReopen, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_DISCARD:
		return domain.GroupPermissionResourceDiscard, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_REJECT:
		return domain.GroupPermissionResourceReject, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_PUBLISH:
		return domain.GroupPermissionResourcePublish, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_UNPUBLISH:
		return domain.GroupPermissionResourceUnpublish, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_MEMBERSHIP_MANAGE:
		return domain.GroupPermissionMembershipManage, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_ROLE_MANAGE:
		return domain.GroupPermissionRoleManage, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_PERMISSION_MANAGE:
		return domain.GroupPermissionPermissionManage, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_NOTIFICATION_MANAGE:
		return domain.GroupPermissionNotificationManage, nil
	case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_CATEGORY_MANAGE:
		return domain.GroupPermissionCategoryManage, nil
	default:
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
}

func toDomainGroupRole(value metadatalifecyclev1.GroupRole) (domain.GroupRole, error) {
	switch value {
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR:
		return domain.GroupRoleEditor, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER:
		return domain.GroupRoleReviewer, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_ADMIN:
		return domain.GroupRoleAdmin, nil
	default:
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
}

func toDomainDecision(value metadatalifecyclev1.Decision) (domain.Decision, error) {
	switch value {
	case metadatalifecyclev1.Decision_DECISION_ACCEPT:
		return domain.DecisionAccept, nil
	case metadatalifecyclev1.Decision_DECISION_REJECT:
		return domain.DecisionReject, nil
	default:
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
}
