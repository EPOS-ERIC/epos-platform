package clients

import (
	"context"
	"fmt"
	"slices"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type MetadataLifecycleIdentity struct {
	identityGRPC metadatalifecyclev1.IdentityServiceClient
	proxy        *MetadataLifecycleGRPC
}

func (c *MetadataLifecycleIdentity) ProvisionUser(ctx context.Context, identity ActorIdentity) error {
	_, err := c.identityGRPC.ProvisionUser(ctx, &metadatalifecyclev1.ProvisionUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	})
	if err != nil {
		return wrapGRPCError("provision metadata-lifecycle user", err)
	}

	return nil
}

func (c *MetadataLifecycleIdentity) GetUserForAuth(ctx context.Context, actorID, id string) (domain.User, error) {
	user, err := c.proxy.GetUser(ctx, actorID, id)
	if err != nil {
		return domain.User{}, err
	}

	status, err := fromProtoUserStatus(user.Status)
	if err != nil {
		return domain.User{}, fmt.Errorf("TODO: %w", err)
	}

	globalRoles, err := mapSliceErr(user.GlobalRoles, fromProtoGlobalRole)
	if err != nil {
		return domain.User{}, err
	}

	memberships, err := mapSliceErr(user.Memberships, c.mapMembership)
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:                user.Id,
		Email:             user.Email,
		GivenName:         user.GivenName,
		FamilyName:        user.FamilyName,
		Status:            status,
		GlobalRoles:       globalRoles,
		GlobalPermissions: slices.Clone(user.GlobalPermissions),
		CreatedAt:         user.GetCreatedAt().AsTime(),
		UpdatedAt:         user.GetUpdatedAt().AsTime(),
		Memberships:       memberships,
	}, nil
}

func (c *MetadataLifecycleIdentity) mapMembership(membership *metadatalifecyclev1.UserMembership) (domain.Membership, error) {
	status, err := fromProtoMembershipStatus(membership.Status)
	if err != nil {
		return domain.Membership{}, fmt.Errorf("TODO: %w", err)
	}

	groupRoles, err := mapSliceErr(membership.GroupRoles, fromProtoGroupRole)
	if err != nil {
		return domain.Membership{}, err
	}

	roleRequests, err := mapSliceErr(membership.RoleRequests, c.mapRoleRequest)
	if err != nil {
		return domain.Membership{}, err
	}

	return domain.Membership{
		Group: domain.GroupSummary{
			ID:   membership.Group.Id,
			Name: membership.Group.Name,
		},
		Roles:            groupRoles,
		Status:           status,
		GroupPermissions: slices.Clone(membership.GroupPermissions),
		RoleRequests:     roleRequests,
		CreatedAt:        membership.GetCreatedAt().AsTime(),
		UpdatedAt:        membership.GetUpdatedAt().AsTime(),
	}, nil
}

func (c *MetadataLifecycleIdentity) mapRoleRequest(request *metadatalifecyclev1.RoleRequest) (domain.RoleRequest, error) {
	status, err := fromProtoRoleRequestStatus(request.Status)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("TODO: %w", err)
	}
	role, err := fromProtoGroupRole(request.GroupRole)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("TODO: %w", err)
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
