package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) ProvisionUser(ctx context.Context, identity domain.ActorIdentity) error {
	_, err := c.identity.ProvisionUser(ctx, &metadatalifecyclev1.ProvisionUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	})
	if err != nil {
		return c.wrapError(ctx, "provision metadata-lifecycle user", err)
	}

	return nil
}

func (c *MetadataLifecycle) GetUser(ctx context.Context, actorID, id string) (domain.User, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	user, err := c.users.GetUser(ctx, &metadatalifecyclev1.GetUserRequest{UserId: id})
	if err != nil {
		return domain.User{}, c.wrapError(ctx, "get metadata-lifecycle user", err)
	}

	status, err := fromProtoUserStatus(user.Status)
	if err != nil {
		return domain.User{}, fmt.Errorf("map user status: %w", err)
	}

	globalRoles, err := mapSliceErr(user.GlobalRoles, fromProtoGlobalRole)
	if err != nil {
		return domain.User{}, err
	}

	globalGrantedPermissions, err := mapSliceErr(user.GlobalGrantedPermissions, fromProtoGlobalPermission)
	if err != nil {
		return domain.User{}, err
	}

	globalRolePermissions, err := mapSliceErr(user.GlobalRolePermissions, fromProtoGlobalPermission)
	if err != nil {
		return domain.User{}, err
	}

	memberships, err := mapSliceErr(user.Memberships, c.mapMembership)
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:                       user.Id,
		Email:                    user.Email,
		GivenName:                user.GivenName,
		FamilyName:               user.FamilyName,
		Status:                   status,
		BlockReason:              user.BlockReason,
		GlobalRoles:              globalRoles,
		GlobalGrantedPermissions: globalGrantedPermissions,
		GlobalRolePermissions:    globalRolePermissions,
		CreatedAt:                user.GetCreatedAt().AsTime(),
		UpdatedAt:                user.GetUpdatedAt().AsTime(),
		Memberships:              memberships,
	}, nil
}

func (c *MetadataLifecycle) mapMembership(membership *metadatalifecyclev1.UserMembership) (domain.Membership, error) {
	status, err := fromProtoMembershipStatus(membership.Status)
	if err != nil {
		return domain.Membership{}, fmt.Errorf("map membership status: %w", err)
	}

	groupRoles, err := mapSliceErr(membership.GroupRoles, fromProtoGroupRole)
	if err != nil {
		return domain.Membership{}, err
	}

	roleRequests, err := mapSliceErr(membership.RoleRequests, c.mapRoleRequest)
	if err != nil {
		return domain.Membership{}, err
	}

	grantedPermissions, err := mapSliceErr(membership.GrantedPermissions, fromProtoGroupPermission)
	if err != nil {
		return domain.Membership{}, err
	}

	rolePermissions, err := mapSliceErr(membership.RolePermissions, fromProtoGroupPermission)
	if err != nil {
		return domain.Membership{}, err
	}

	return domain.Membership{
		Group: domain.GroupSummary{
			ID:   membership.Group.Id,
			Name: membership.Group.Name,
		},
		Roles:              groupRoles,
		Status:             status,
		GrantedPermissions: grantedPermissions,
		RolePermissions:    rolePermissions,
		RoleRequests:       roleRequests,
		CreatedAt:          membership.GetCreatedAt().AsTime(),
		UpdatedAt:          membership.GetUpdatedAt().AsTime(),
	}, nil
}

func (c *MetadataLifecycle) mapRoleRequest(request *metadatalifecyclev1.RoleRequest) (domain.RoleRequest, error) {
	status, err := fromProtoRoleRequestStatus(request.Status)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("map role request status: %w", err)
	}
	role, err := fromProtoGroupRole(request.GroupRole)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("map role request role: %w", err)
	}
	return domain.RoleRequest{
		ID:          request.Id,
		Role:        role,
		Status:      status,
		RequestedAt: request.GetRequestedAt().AsTime(),
	}, nil
}

func fromProtoUserStatus(value metadatalifecyclev1.UserStatus) (domain.UserStatus, error) {
	switch value {
	case metadatalifecyclev1.UserStatus_USER_STATUS_ACTIVE:
		return domain.UserStatusActive, nil
	case metadatalifecyclev1.UserStatus_USER_STATUS_BLOCKED:
		return domain.UserStatusBlocked, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown user status: %v", value)
	}
}

func fromProtoGlobalRole(value metadatalifecyclev1.GlobalRole) (domain.GlobalRole, error) {
	switch value {
	case metadatalifecyclev1.GlobalRole_GLOBAL_ROLE_ADMIN:
		return domain.GlobalRoleAdmin, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown global role: %v", value)
	}
}

//nolint:dupl // Bidirectional enum mapping intentionally mirrors protobuf values.
func toProtoGlobalPermission(value domain.GlobalPermission) (metadatalifecyclev1.GlobalPermission, error) {
	switch value {
	case domain.GlobalPermissionMembershipManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_MEMBERSHIP_MANAGE, nil
	case domain.GlobalPermissionRoleManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_ROLE_MANAGE, nil
	case domain.GlobalPermissionPermissionManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_PERMISSION_MANAGE, nil
	case domain.GlobalPermissionNotificationManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_NOTIFICATION_MANAGE, nil
	case domain.GlobalPermissionOrganizationManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_ORGANIZATION_MANAGE, nil
	case domain.GlobalPermissionContactPointManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONTACT_POINT_MANAGE, nil
	case domain.GlobalPermissionCategorySchemeManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CATEGORY_SCHEME_MANAGE, nil
	case domain.GlobalPermissionResourceCreate:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_CREATE, nil
	case domain.GlobalPermissionResourceEdit:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_EDIT, nil
	case domain.GlobalPermissionResourceSubmit:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_SUBMIT, nil
	case domain.GlobalPermissionResourceWithdraw:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_WITHDRAW, nil
	case domain.GlobalPermissionResourceReopen:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_REOPEN, nil
	case domain.GlobalPermissionResourceDiscard:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_DISCARD, nil
	case domain.GlobalPermissionResourceReject:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_REJECT, nil
	case domain.GlobalPermissionResourcePublish:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_PUBLISH, nil
	case domain.GlobalPermissionResourceUnpublish:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_UNPUBLISH, nil
	case domain.GlobalPermissionCategoryManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CATEGORY_MANAGE, nil
	case domain.GlobalPermissionConfigurationManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONFIGURATION_MANAGE, nil
	case domain.GlobalPermissionConverterManage:
		return metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONVERTER_MANAGE, nil
	default:
		return 0, fmt.Errorf("invalid global permission %q", value)
	}
}

//nolint:dupl // Bidirectional enum mapping intentionally mirrors protobuf values.
func fromProtoGlobalPermission(value metadatalifecyclev1.GlobalPermission) (domain.GlobalPermission, error) {
	switch value {
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_MEMBERSHIP_MANAGE:
		return domain.GlobalPermissionMembershipManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_ROLE_MANAGE:
		return domain.GlobalPermissionRoleManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_PERMISSION_MANAGE:
		return domain.GlobalPermissionPermissionManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_NOTIFICATION_MANAGE:
		return domain.GlobalPermissionNotificationManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_ORGANIZATION_MANAGE:
		return domain.GlobalPermissionOrganizationManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONTACT_POINT_MANAGE:
		return domain.GlobalPermissionContactPointManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CATEGORY_SCHEME_MANAGE:
		return domain.GlobalPermissionCategorySchemeManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_CREATE:
		return domain.GlobalPermissionResourceCreate, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_EDIT:
		return domain.GlobalPermissionResourceEdit, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_SUBMIT:
		return domain.GlobalPermissionResourceSubmit, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_WITHDRAW:
		return domain.GlobalPermissionResourceWithdraw, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_REOPEN:
		return domain.GlobalPermissionResourceReopen, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_DISCARD:
		return domain.GlobalPermissionResourceDiscard, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_REJECT:
		return domain.GlobalPermissionResourceReject, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_PUBLISH:
		return domain.GlobalPermissionResourcePublish, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_UNPUBLISH:
		return domain.GlobalPermissionResourceUnpublish, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CATEGORY_MANAGE:
		return domain.GlobalPermissionCategoryManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONFIGURATION_MANAGE:
		return domain.GlobalPermissionConfigurationManage, nil
	case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONVERTER_MANAGE:
		return domain.GlobalPermissionConverterManage, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown global permission: %v", value)
	}
}

func fromProtoGroupRole(value metadatalifecyclev1.GroupRole) (domain.GroupRole, error) {
	switch value {
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR:
		return domain.GroupRoleEditor, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER:
		return domain.GroupRoleReviewer, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_ADMIN:
		return domain.GroupRoleAdmin, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown group role: %v", value)
	}
}

//nolint:dupl // Bidirectional enum mapping intentionally mirrors protobuf values.
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
		return 0, fmt.Errorf("invalid group permission %q", value)
	}
}

//nolint:dupl // Bidirectional enum mapping intentionally mirrors protobuf values.
func fromProtoGroupPermission(value metadatalifecyclev1.GroupPermission) (domain.GroupPermission, error) {
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
		return "", fmt.Errorf("metadata-lifecycle returned unknown group permission: %v", value)
	}
}

func fromProtoMembershipStatus(value metadatalifecyclev1.MembershipStatus) (domain.MembershipStatus, error) {
	switch value {
	case metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_PENDING:
		return domain.MembershipStatusPending, nil
	case metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_ACTIVE:
		return domain.MembershipStatusActive, nil
	case metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_REJECTED:
		return domain.MembershipStatusRejected, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown membership status: %v", value)
	}
}

func fromProtoRoleRequestStatus(value metadatalifecyclev1.RoleRequestStatus) (domain.RoleRequestStatus, error) {
	switch value {
	case metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_PENDING:
		return domain.RoleRequestStatusPending, nil
	case metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_ACCEPTED:
		return domain.RoleRequestStatusAccepted, nil
	case metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_REJECTED:
		return domain.RoleRequestStatusRejected, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown role request status: %v", value)
	}
}
