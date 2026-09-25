// Package authz contains stored-resource access decisions.
package authz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type repository interface {
	ListPermissions(context.Context) ([]domain.Permission, error)
	User(context.Context, string) (domain.AuthorizationUser, error)
	IsGlobalAdmin(context.Context, string) (bool, error)
	HasGlobalPermission(context.Context, string, domain.GlobalPermission) (bool, error)
	IsActiveMember(context.Context, string, uuid.UUID) (bool, error)
	ListActiveUserMemberships(context.Context, string) ([]uuid.UUID, error)
	HasAnyActiveMembership(context.Context, string) (bool, error)
	HasGroupPermission(context.Context, string, uuid.UUID, domain.GroupPermission) (bool, error)
	ListGroupsWithAnyPermission(context.Context, string, []domain.GroupPermission) ([]uuid.UUID, error)
	HasGroupRole(context.Context, string, uuid.UUID, string) (bool, error)
	DatasetGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	SoftwareApplicationGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	SoftwareSourceCodeGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	IsDatasetPublished(context.Context, uuid.UUID) (bool, error)
	IsSoftwareApplicationPublished(context.Context, uuid.UUID) (bool, error)
	IsSoftwareSourceCodePublished(context.Context, uuid.UUID) (bool, error)
	DatasetVersionAuthz(context.Context, uuid.UUID, uuid.UUID) (domain.VersionAuthz, error)
	SoftwareApplicationVersionAuthz(context.Context, uuid.UUID, uuid.UUID) (domain.VersionAuthz, error)
	SoftwareSourceCodeVersionAuthz(context.Context, uuid.UUID, uuid.UUID) (domain.VersionAuthz, error)
}

type Authorizer struct {
	repository repository
	log        *slog.Logger
}

// New creates an authorizer backed by repository.
func New(log *slog.Logger, repository repository) *Authorizer {
	return &Authorizer{
		repository: repository,
		log:        log.With("component", "authz-service"),
	}
}

// ListPermissions returns the fixed authorization catalog.
func (a *Authorizer) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	permissions, err := a.repository.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}

	return permissions, nil
}

func (a *Authorizer) CanReadUser(ctx context.Context, actorID, targetID string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanReadUser", result.Allowed, result.Reason, err)
	}()

	if actorID == "" || targetID == "" {
		return domain.Result{Allowed: false, Reason: "invalid user"}, nil
	}

	actor, result, err := a.activeUser(ctx, actorID)
	if err != nil {
		return domain.Result{}, err
	}
	if !result.Allowed {
		// allow self status lookup for blocked users to allow the gateway to block any blocked user authorized requests
		if actorID == targetID && actor.Status == domain.UserStatusBlocked {
			return domain.Result{Allowed: true, Reason: "blocked self status lookup"}, nil
		}
		return result, nil
	}

	if actorID == targetID {
		return domain.Result{Allowed: true, Reason: "self"}, nil
	}

	return a.canManageUserDirectory(ctx, actor.ID)
}

// CanManageGlobalNotificationSubscriptions permits self-management and global notification administrators.
func (a *Authorizer) CanManageGlobalNotificationSubscriptions(ctx context.Context, actorID, targetID string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanManageGlobalNotificationSubscriptions", result.Allowed, result.Reason, err)
	}()

	if actorID == "" || targetID == "" {
		return domain.Result{Allowed: false, Reason: "invalid user"}, nil
	}
	if _, result, err = a.activeUser(ctx, actorID); err != nil || !result.Allowed {
		return result, err
	}
	if actorID == targetID {
		return domain.Result{Allowed: true, Reason: "self"}, nil
	}

	target, err := a.repository.User(ctx, targetID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("get notification subscription target: %w", err)
	}
	if target.Status != domain.UserStatusActive {
		return domain.Result{Allowed: false, Reason: "target user is not active"}, nil
	}
	admin, err := a.repository.IsGlobalAdmin(ctx, actorID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}
	allowed, err := a.repository.HasGlobalPermission(ctx, actorID, domain.GlobalPermissionNotificationManage)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global notification permission: %w", err)
	}
	if allowed {
		return domain.Result{Allowed: true, Reason: string(domain.GlobalPermissionNotificationManage)}, nil
	}
	return domain.Result{Allowed: false, Reason: "global notification permission required"}, nil
}

// CanManageGroupNotificationSubscriptions permits active members to manage themselves and group notification managers to manage others.
func (a *Authorizer) CanManageGroupNotificationSubscriptions(ctx context.Context, actorID, targetID string, groupID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanManageGroupNotificationSubscriptions", result.Allowed, result.Reason, err)
	}()

	if actorID == "" || targetID == "" || groupID == uuid.Nil {
		return domain.Result{Allowed: false, Reason: "invalid group notification request"}, nil
	}
	targetMember, err := a.repository.IsActiveMember(ctx, targetID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check target group membership: %w", err)
	}
	if !targetMember {
		return domain.Result{Allowed: false, Reason: "target user is not an active group member"}, nil
	}
	if actorID == targetID {
		return domain.Result{Allowed: true, Reason: "self"}, nil
	}

	if _, result, err = a.activeUser(ctx, actorID); err != nil || !result.Allowed {
		return result, err
	}
	admin, err := a.repository.IsGlobalAdmin(ctx, actorID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}
	globalAllowed, err := a.repository.HasGlobalPermission(ctx, actorID, domain.GlobalPermissionNotificationManage)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global notification permission: %w", err)
	}
	if globalAllowed {
		return domain.Result{Allowed: true, Reason: string(domain.GlobalPermissionNotificationManage)}, nil
	}
	groupAllowed, err := a.repository.HasGroupPermission(ctx, actorID, groupID, domain.GroupPermissionNotificationManage)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group notification permission: %w", err)
	}
	if groupAllowed {
		return domain.Result{Allowed: true, Reason: string(domain.GroupPermissionNotificationManage)}, nil
	}
	return domain.Result{Allowed: false, Reason: "group notification permission required"}, nil
}

// GroupsWithAnyPermissionForActiveMember lists groups where actor has any requested permission and target has an active membership.
func (a *Authorizer) GroupsWithAnyPermissionForActiveMember(ctx context.Context, actorID, targetID string, permissions []domain.GroupPermission) (groupIDs []uuid.UUID, err error) {
	defer func() {
		recordAuthzDecision(ctx, "GroupsWithAnyPermissionForActiveMember", len(groupIDs) > 0, "", err)
	}()

	_, result, err := a.activeUser(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("load actor: %w", err)
	}
	if !result.Allowed {
		return nil, nil
	}

	permissionGroups, err := a.repository.ListGroupsWithAnyPermission(ctx, actorID, permissions)
	if err != nil {
		return nil, fmt.Errorf("list actor permission groups: %w", err)
	}
	if len(permissionGroups) == 0 {
		return []uuid.UUID{}, nil
	}

	targetGroups, err := a.repository.ListActiveUserMemberships(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("list target active group memberships: %w", err)
	}

	activeTargetGroups := make(map[uuid.UUID]struct{}, len(targetGroups))
	for _, groupID := range targetGroups {
		activeTargetGroups[groupID] = struct{}{}
	}
	groupIDs = make([]uuid.UUID, 0, len(permissionGroups))
	for _, groupID := range permissionGroups {
		if _, active := activeTargetGroups[groupID]; active {
			groupIDs = append(groupIDs, groupID)
		}
	}

	return groupIDs, nil
}

// CanListUsers checks permissions required to manage the user directory.
func (a *Authorizer) CanListUsers(ctx context.Context, userID string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanListUsers", result.Allowed, result.Reason, err)
	}()

	if userID == "" {
		return domain.Result{Allowed: false, Reason: "invalid user"}, nil
	}

	actor, result, err := a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	return a.canManageUserDirectory(ctx, actor.ID)
}

func (a *Authorizer) canManageUserDirectory(ctx context.Context, userID string) (domain.Result, error) {
	for _, permission := range []domain.GlobalPermission{
		domain.GlobalPermissionMembershipManage,
		domain.GlobalPermissionRoleManage,
		domain.GlobalPermissionPermissionManage,
		domain.GlobalPermissionNotificationManage,
	} {
		allowed, err := a.repository.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			return domain.Result{}, fmt.Errorf("check global permission: %w", err)
		}

		if allowed {
			return domain.Result{Allowed: true, Reason: string(permission)}, nil
		}
	}

	return domain.Result{Allowed: false, Reason: "user directory permission required"}, nil
}

// AuthorizeActiveUser permits authenticated operations for active users.
func (a *Authorizer) AuthorizeActiveUser(ctx context.Context, userID string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "AuthorizeActiveUser", result.Allowed, result.Reason, err)
	}()

	if userID == "" {
		return domain.Result{Allowed: false, Reason: "invalid user"}, nil
	}

	_, result, err = a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	return domain.Result{Allowed: true, Reason: "active user"}, nil
}

// CanListGroupMemberships permits group membership roster access.
func (a *Authorizer) CanListGroupMemberships(ctx context.Context, userID string, groupID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanListGroupMemberships", result.Allowed, result.Reason, err)
	}()

	if userID == "" || groupID == uuid.Nil {
		return domain.Result{Allowed: false, Reason: "invalid group access request"}, nil
	}

	_, result, err = a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}

	// TODO: see if these kind of permission checks could use an helper or an ad hoc query to check for more than one permission
	for _, permission := range []domain.GlobalPermission{
		domain.GlobalPermissionMembershipManage,
		domain.GlobalPermissionRoleManage,
		domain.GlobalPermissionNotificationManage,
	} {
		allowed, err := a.repository.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			return domain.Result{}, fmt.Errorf("check global permission: %w", err)
		}

		if allowed {
			return domain.Result{Allowed: true, Reason: string(permission)}, nil
		}
	}

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}

	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	for _, permission := range []domain.GroupPermission{
		domain.GroupPermissionMembershipManage,
		domain.GroupPermissionRoleManage,
		domain.GroupPermissionPermissionManage,
		domain.GroupPermissionNotificationManage,
	} {
		allowed, err := a.repository.HasGroupPermission(ctx, userID, groupID, permission)
		if err != nil {
			return domain.Result{}, fmt.Errorf("check group permission: %w", err)
		}

		if allowed {
			return domain.Result{Allowed: true, Reason: string(permission)}, nil
		}
	}

	return domain.Result{Allowed: false, Reason: "group roster permission required"}, nil
}

// CanManageGroupMemberships permits membership acceptance, rejection, and revocation.
func (a *Authorizer) CanManageGroupMemberships(ctx context.Context, userID string, groupID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanManageGroupMemberships", result.Allowed, result.Reason, err)
	}()

	return a.canManageGroupPermission(ctx, userID, groupID, domain.GlobalPermissionMembershipManage, domain.GroupPermissionMembershipManage)
}

// CanRevokeGroupMembership applies the extra protection for GROUP_ADMIN memberships.
func (a *Authorizer) CanRevokeGroupMembership(ctx context.Context, userID string, groupID uuid.UUID, targetUserID string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanRevokeGroupMembership", result.Allowed, result.Reason, err)
	}()

	if userID == targetUserID {
		active, err := a.AuthorizeActiveUser(ctx, userID)
		if err != nil || !active.Allowed {
			return active, err
		}
		return domain.Result{Allowed: true, Reason: "self"}, nil
	}

	allowed, err := a.CanManageGroupMemberships(ctx, userID, groupID)
	if err != nil || !allowed.Allowed {
		return allowed, err
	}

	globalPermission, err := a.repository.HasGlobalPermission(ctx, userID, domain.GlobalPermissionMembershipManage)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global membership permission: %w", err)
	}

	if globalPermission {
		return allowed, nil
	}

	// TODO: use an enum for GROUP_ADMIN
	targetIsAdmin, err := a.repository.HasGroupRole(ctx, targetUserID, groupID, "GROUP_ADMIN")
	if err != nil {
		return domain.Result{}, fmt.Errorf("check target group role: %w", err)
	}

	if targetIsAdmin {
		return domain.Result{Allowed: false, Reason: "GROUP_ADMIN membership requires global management"}, nil
	}

	return allowed, nil
}

// CanRequestGroupRole permits active members to request a group role.
func (a *Authorizer) CanRequestGroupRole(ctx context.Context, userID string, groupID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanRequestGroupRole", result.Allowed, result.Reason, err)
	}()

	if userID == "" || groupID == uuid.Nil {
		return domain.Result{Allowed: false, Reason: "invalid group access request"}, nil
	}

	_, result, err = a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	return domain.Result{Allowed: true, Reason: "active group member"}, nil
}

// CanManageGroupRoles permits role-request decisions and standard direct group role changes.
func (a *Authorizer) CanManageGroupRoles(ctx context.Context, userID string, groupID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanManageGroupRoles", result.Allowed, result.Reason, err)
	}()

	return a.canManageGroupPermission(ctx, userID, groupID, domain.GlobalPermissionRoleManage, domain.GroupPermissionRoleManage)
}

// CanManageSpecificGroupRole applies role-specific restrictions for direct assignment and removal.
func (a *Authorizer) CanManageSpecificGroupRole(ctx context.Context, userID string, groupID uuid.UUID, role string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanManageSpecificGroupRole", result.Allowed, result.Reason, err)
	}()

	allowed, err := a.CanManageGroupRoles(ctx, userID, groupID)
	if err != nil || !allowed.Allowed || role != "GROUP_ADMIN" {
		return allowed, err
	}

	globalPermission, err := a.repository.HasGlobalPermission(ctx, userID, domain.GlobalPermissionRoleManage)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global role permission: %w", err)
	}
	if !globalPermission {
		return domain.Result{Allowed: false, Reason: "GROUP_ADMIN requires global role management"}, nil
	}

	return allowed, nil
}

func (a *Authorizer) canManageGroupPermission(ctx context.Context, userID string, groupID uuid.UUID, globalPermission domain.GlobalPermission, groupPermission domain.GroupPermission) (domain.Result, error) {
	if userID == "" || groupID == uuid.Nil {
		return domain.Result{Allowed: false, Reason: "invalid group access request"}, nil
	}

	_, result, err := a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	allowed, err := a.repository.HasGlobalPermission(ctx, userID, globalPermission)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global permission: %w", err)
	}
	if allowed {
		return domain.Result{Allowed: true, Reason: string(globalPermission)}, nil
	}

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	allowed, err = a.repository.HasGroupPermission(ctx, userID, groupID, groupPermission)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group permission: %w", err)
	}
	if !allowed {
		return domain.Result{Allowed: false, Reason: string(groupPermission) + " permission required"}, nil
	}

	return domain.Result{Allowed: true, Reason: string(groupPermission)}, nil
}

// CanReadDatasetVersion applies publication and stored-group membership rules to one version.
func (a *Authorizer) CanReadDatasetVersion(ctx context.Context, userID *string, datasetID, versionID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanReadDatasetVersion", result.Allowed, result.Reason, err)
	}()

	version, err := a.repository.DatasetVersionAuthz(ctx, datasetID, versionID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("load dataset version authorization: %w", err)
	}
	return a.authorizeVersionRead(ctx, userID, version.GroupID, version.Status)
}

// CanReadSoftwareApplicationVersion applies publication and stored-group membership rules to one version.
// TODO: see if we can generalize this function to a general "can read resource version"?
func (a *Authorizer) CanReadSoftwareApplicationVersion(ctx context.Context, userID *string, applicationID, versionID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanReadSoftwareApplicationVersion", result.Allowed, result.Reason, err)
	}()

	version, err := a.repository.SoftwareApplicationVersionAuthz(ctx, applicationID, versionID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("load software application version authorization: %w", err)
	}
	return a.authorizeVersionRead(ctx, userID, version.GroupID, version.Status)
}

// CanReadSoftwareSourceCodeVersion applies publication and stored-group membership rules to one version.
func (a *Authorizer) CanReadSoftwareSourceCodeVersion(ctx context.Context, userID *string, sourceCodeID, versionID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanReadSoftwareSourceCodeVersion", result.Allowed, result.Reason, err)
	}()

	version, err := a.repository.SoftwareSourceCodeVersionAuthz(ctx, sourceCodeID, versionID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("load software source code version authorization: %w", err)
	}
	return a.authorizeVersionRead(ctx, userID, version.GroupID, version.Status)
}

// authorizeVersionRead loads actor facts needed to evaluate one non-public version.
func (a *Authorizer) authorizeVersionRead(ctx context.Context, userID *string, groupID uuid.UUID, status domain.Status) (domain.Result, error) {
	if status == domain.Published {
		return versionReadDecision(status, userID != nil, false, false, false, ""), nil
	}
	if userID == nil {
		return versionReadDecision(status, false, false, false, false, ""), nil
	}
	_, result, err := a.activeUser(ctx, *userID)
	if err != nil || !result.Allowed {
		return versionReadDecision(status, true, false, false, false, result.Reason), err
	}
	admin, err := a.repository.IsGlobalAdmin(ctx, *userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return versionReadDecision(status, true, true, true, false, result.Reason), nil
	}
	member, err := a.repository.IsActiveMember(ctx, *userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	return versionReadDecision(status, true, true, false, member, result.Reason), nil
}

// versionReadDecision applies the shared version visibility rule to loaded facts.
func versionReadDecision(status domain.Status, authenticated, active, admin, member bool, inactiveReason string) domain.Result {
	if status == domain.Published {
		return domain.Result{Allowed: true, Reason: "public published resource"}
	}
	if !authenticated {
		return domain.Result{Allowed: false, Reason: "authentication required"}
	}
	if !active {
		return domain.Result{Allowed: false, Reason: inactiveReason}
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}
	}
	return domain.Result{Allowed: true, Reason: "active group member"}
}

// AuthorizeAdmin permits active global administrators only.
func (a *Authorizer) AuthorizeAdmin(ctx context.Context, userID string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "AuthorizeAdmin", result.Allowed, result.Reason, err)
	}()

	_, result, err = a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}
	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if !admin {
		return domain.Result{Allowed: false, Reason: "global admin required"}, nil
	}
	return domain.Result{Allowed: true, Reason: "global admin"}, nil
}

// AuthorizeGlobal evaluates a global permission without requiring membership.
func (a *Authorizer) AuthorizeGlobal(ctx context.Context, userID string, permission domain.GlobalPermission) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "AuthorizeGlobal", result.Allowed, result.Reason, err)
	}()

	_, result, err = a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	allowed, err := a.repository.HasGlobalPermission(ctx, userID, permission)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global permission: %w", err)
	}
	if !allowed {
		return domain.Result{Allowed: false, Reason: "permission not granted"}, nil
	}

	return domain.Result{Allowed: true, Reason: "global permission"}, nil
}

// AuthorizeResource gets group from stored logical resource, never caller input.
func (a *Authorizer) AuthorizeResource(
	ctx context.Context,
	userID string,
	resourceType domain.ResourceType,
	resourceID uuid.UUID,
	permission domain.GroupPermission,
) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "AuthorizeResource", result.Allowed, result.Reason, err)
	}()

	groupID, err := a.resourceGroup(ctx, resourceType, resourceID)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return domain.Result{}, fmt.Errorf("load resource group: %w", err)
	}

	return a.AuthorizeGroup(ctx, userID, groupID, permission)
}

// AuthorizeGroup handles resource creation before a logical resource exists.
func (a *Authorizer) AuthorizeGroup(ctx context.Context, userID string, groupID uuid.UUID, permission domain.GroupPermission) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "AuthorizeGroup", result.Allowed, result.Reason, err)
	}()

	_, result, err = a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	if globalPermission, ok := globalPermissionForGroupPermission(permission); ok {
		allowed, err := a.repository.HasGlobalPermission(ctx, userID, globalPermission)
		if err != nil {
			return domain.Result{}, fmt.Errorf("check global permission: %w", err)
		}
		if allowed {
			return domain.Result{Allowed: true, Reason: string(globalPermission)}, nil
		}
	}

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	allowed, err := a.repository.HasGroupPermission(ctx, userID, groupID, permission)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group permission: %w", err)
	}
	if !allowed {
		return domain.Result{Allowed: false, Reason: "permission not granted"}, nil
	}

	return domain.Result{Allowed: true, Reason: "group permission"}, nil
}

func globalPermissionForGroupPermission(permission domain.GroupPermission) (domain.GlobalPermission, bool) {
	switch permission {
	case domain.GroupPermissionResourceCreate:
		return domain.GlobalPermissionResourceCreate, true
	case domain.GroupPermissionResourceEdit:
		return domain.GlobalPermissionResourceEdit, true
	case domain.GroupPermissionResourceSubmit:
		return domain.GlobalPermissionResourceSubmit, true
	case domain.GroupPermissionResourceWithdraw:
		return domain.GlobalPermissionResourceWithdraw, true
	case domain.GroupPermissionResourceReopen:
		return domain.GlobalPermissionResourceReopen, true
	case domain.GroupPermissionResourceDiscard:
		return domain.GlobalPermissionResourceDiscard, true
	case domain.GroupPermissionResourceReject:
		return domain.GlobalPermissionResourceReject, true
	case domain.GroupPermissionResourcePublish:
		return domain.GlobalPermissionResourcePublish, true
	case domain.GroupPermissionResourceUnpublish:
		return domain.GlobalPermissionResourceUnpublish, true
	case domain.GroupPermissionMembershipManage:
		return domain.GlobalPermissionMembershipManage, true
	case domain.GroupPermissionRoleManage:
		return domain.GlobalPermissionRoleManage, true
	case domain.GroupPermissionPermissionManage:
		return domain.GlobalPermissionPermissionManage, true
	case domain.GroupPermissionNotificationManage:
		return domain.GlobalPermissionNotificationManage, true
	case domain.GroupPermissionCategoryManage:
		return domain.GlobalPermissionCategoryManage, true
	default:
		return "", false
	}
}

// CanReadResource permits public reads of published resources. Non-public
// resources require an active member in stored resource group or global admin.
func (a *Authorizer) CanReadResource(ctx context.Context, userID *string, resourceType domain.ResourceType, resourceID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanReadResource", result.Allowed, result.Reason, err)
	}()

	groupID, err := a.resourceGroup(ctx, resourceType, resourceID)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return domain.Result{}, fmt.Errorf("load resource group: %w", err)
	}

	published, err := a.resourcePublished(ctx, resourceType, resourceID)
	if err != nil {
		return domain.Result{}, err
	}
	if published {
		return domain.Result{Allowed: true, Reason: "public published resource"}, nil
	}
	if userID == nil {
		return domain.Result{Allowed: false, Reason: "authentication required"}, nil
	}

	return a.canReadResourceGroup(ctx, *userID, groupID)
}

// CanReadResourceVersions requires access to the resource's owning group.
func (a *Authorizer) CanReadResourceVersions(ctx context.Context, userID string, resourceType domain.ResourceType, resourceID uuid.UUID) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanReadResourceVersions", result.Allowed, result.Reason, err)
	}()

	groupID, err := a.resourceGroup(ctx, resourceType, resourceID)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return domain.Result{}, fmt.Errorf("load resource group: %w", err)
	}

	return a.canReadResourceGroup(ctx, userID, groupID)
}

// canReadResourceGroup permits active users who are global admins or active group members.
func (a *Authorizer) canReadResourceGroup(ctx context.Context, userID string, groupID uuid.UUID) (domain.Result, error) {
	result, admin, err := a.activeGlobalAdmin(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	return domain.Result{Allowed: true, Reason: "active group member"}, nil
}

func (a *Authorizer) activeGlobalAdmin(ctx context.Context, userID string) (domain.Result, bool, error) {
	_, result, err := a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, false, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return domain.Result{}, false, fmt.Errorf("check global admin: %w", err)
	}

	return result, admin, nil
}

// CanReadContactPoint protects email data while allowing active members in any group.
func (a *Authorizer) CanReadContactPoint(ctx context.Context, userID string) (result domain.Result, err error) {
	defer func() {
		recordAuthzDecision(ctx, "CanReadContactPoint", result.Allowed, result.Reason, err)
	}()

	_, result, err = a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.HasAnyActiveMembership(ctx, userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check active memberships: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user has no active group membership"}, nil
	}

	return domain.Result{Allowed: true, Reason: "active group member"}, nil
}

// AuthorizeDiscovery applies discovery visibility once and returns candidates safe to project.
// It loads the actor's active group memberships and filters candidates in memory.
func (a *Authorizer) AuthorizeDiscovery(ctx context.Context, actor *domain.Actor, candidates []domain.DiscoveryCandidate) (authorization domain.DiscoveryAuthorization, err error) {
	defer func() {
		recordAuthzDiscovery(ctx, actor != nil, len(candidates), len(authorization.AllowedCandidates), err)
	}()

	if len(candidates) == 0 {
		return domain.DiscoveryAuthorization{
			AllowedCandidates: []domain.DiscoveryCandidate{},
			CreatedByGroupIDs: map[uuid.UUID]struct{}{},
		}, nil
	}

	// Reject unsupported resource types up front.
	for _, c := range candidates {
		switch c.Type {
		case domain.Dataset, domain.SoftwareApplication, domain.SoftwareSourceCode:
		default:
			return domain.DiscoveryAuthorization{}, fmt.Errorf("unsupported resource type %q: %w", c.Type, domain.ErrInvalidDiscovery)
		}
	}

	// Resolve the acting user, if any; anonymous or inactive actors stay nil.
	var user *domain.AuthorizationUser
	if actor != nil {
		loaded, userResult, err := a.activeUser(ctx, string(*actor))
		if err != nil {
			return domain.DiscoveryAuthorization{}, fmt.Errorf("load discovery user: %w", err)
		}
		if userResult.Allowed {
			user = &loaded
		}
	}

	visibleGroupIDs := map[uuid.UUID]struct{}{}

	// Find the groups where the user is an active member to protect fields like "created_by".
	switch {
	case user == nil:
		// No active user: nothing to look up, nothing to grant beyond defaults.

	case user.GlobalAdmin:
		// Admins bypass membership checks entirely; expose every candidate's group.
		for _, c := range candidates {
			visibleGroupIDs[c.GroupID] = struct{}{}
		}

	default:
		// Regular member: load this user's active group memberships once.
		memberships, err := a.repository.ListActiveUserMemberships(ctx, string(*actor))
		if err != nil {
			return domain.DiscoveryAuthorization{}, fmt.Errorf("load discovery group memberships: %w", err)
		}
		for _, g := range memberships {
			visibleGroupIDs[g] = struct{}{}
		}
	}

	// Filter candidates down to what's actually visible to the actor
	allowed := make([]domain.DiscoveryCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c.Status == domain.Published {
			allowed = append(allowed, c)
			continue
		}
		if user == nil {
			continue
		}
		if user.GlobalAdmin {
			allowed = append(allowed, c)
			continue
		}
		if _, member := visibleGroupIDs[c.GroupID]; member {
			allowed = append(allowed, c)
		}
	}

	return domain.DiscoveryAuthorization{
		AllowedCandidates: allowed,
		CreatedByGroupIDs: visibleGroupIDs,
	}, nil
}

func (a *Authorizer) resourcePublished(ctx context.Context, typ domain.ResourceType, id uuid.UUID) (bool, error) {
	switch typ {
	case domain.Dataset:
		published, err := a.repository.IsDatasetPublished(ctx, id)
		if err != nil {
			return false, fmt.Errorf("check dataset publication: %w", err)
		}
		return published, nil
	case domain.SoftwareApplication:
		published, err := a.repository.IsSoftwareApplicationPublished(ctx, id)
		if err != nil {
			return false, fmt.Errorf("check software application publication: %w", err)
		}
		return published, nil
	case domain.SoftwareSourceCode:
		published, err := a.repository.IsSoftwareSourceCodePublished(ctx, id)
		if err != nil {
			return false, fmt.Errorf("check software source code publication: %w", err)
		}
		return published, nil
	default:
		return false, fmt.Errorf("unsupported resource type: %q", typ)
	}
}

func (a *Authorizer) activeUser(ctx context.Context, userID string) (domain.AuthorizationUser, domain.Result, error) {
	user, err := a.repository.User(ctx, userID)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.AuthorizationUser{ID: "", Status: ""}, domain.Result{Allowed: false, Reason: "user not found"}, nil
	}
	if err != nil {
		return domain.AuthorizationUser{}, domain.Result{}, fmt.Errorf("load user: %w", err)
	}
	if user.Status != domain.UserStatusActive {
		return user, domain.Result{Allowed: false, Reason: "user is not active"}, nil
	}
	return user, domain.Result{Allowed: true, Reason: ""}, nil
}

func (a *Authorizer) resourceGroup(ctx context.Context, typ domain.ResourceType, id uuid.UUID) (uuid.UUID, error) {
	switch typ {
	case domain.Dataset:
		groupID, err := a.repository.DatasetGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get dataset group: %w", err)
		}

		return groupID, nil
	case domain.SoftwareApplication:
		groupID, err := a.repository.SoftwareApplicationGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get software application group: %w", err)
		}

		return groupID, nil
	case domain.SoftwareSourceCode:
		groupID, err := a.repository.SoftwareSourceCodeGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get software source code group: %w", err)
		}

		return groupID, nil
	default:
		return uuid.Nil, fmt.Errorf("unsupported resource type: %q", typ)
	}
}
