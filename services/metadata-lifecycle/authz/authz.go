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
	User(context.Context, string) (domain.AuthorizationUser, error)
	PermissionScope(context.Context, string) (domain.PermissionScope, error)
	IsGlobalAdmin(context.Context, string) (bool, error)
	HasGlobalPermission(context.Context, string, string) (bool, error)
	IsActiveMember(context.Context, string, uuid.UUID) (bool, error)
	HasAnyActiveMembership(context.Context, string) (bool, error)
	HasGroupPermission(context.Context, string, uuid.UUID, string) (bool, error)
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

func (a *Authorizer) CanReadUser(ctx context.Context, actorID, targetID string) (domain.Result, error) {
	if actorID == "" || targetID == "" {
		return domain.Result{Allowed: false, Reason: "invalid user"}, nil
	}

	actor, result, err := a.activeUser(ctx, actorID)
	if err != nil || !result.Allowed {
		return result, err
	}
	if actorID == targetID {
		return domain.Result{Allowed: true, Reason: "self"}, nil
	}

	return a.canManageUserDirectory(ctx, actor.ID)
}

// CanListUsers checks permissions required to manage the user directory.
func (a *Authorizer) CanListUsers(ctx context.Context, userID string) (domain.Result, error) {
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
	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}

	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}

	for _, permission := range []string{
		"global.membership.manage",
		"global.role.manage",
		"global.permission.manage",
	} {
		allowed, err := a.repository.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			return domain.Result{}, fmt.Errorf("check global permission: %w", err)
		}

		if allowed {
			return domain.Result{Allowed: true, Reason: permission}, nil
		}
	}

	return domain.Result{Allowed: false, Reason: "user directory permission required"}, nil
}

// AuthorizeActiveUser permits authenticated operations for active users.
func (a *Authorizer) AuthorizeActiveUser(ctx context.Context, userID string) (domain.Result, error) {
	if userID == "" {
		return domain.Result{Allowed: false, Reason: "invalid user"}, nil
	}

	_, result, err := a.activeUser(ctx, userID)

	return result, err
}

// CanListGroupMemberships permits group membership roster access.
func (a *Authorizer) CanListGroupMemberships(ctx context.Context, userID string, groupID uuid.UUID) (domain.Result, error) {
	if userID == "" || groupID == uuid.Nil {
		return domain.Result{Allowed: false, Reason: "invalid group access request"}, nil
	}

	_, result, err := a.activeUser(ctx, userID)
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
	for _, permission := range []string{
		"global.membership.manage",
		"global.role.manage",
	} {
		allowed, err := a.repository.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			return domain.Result{}, fmt.Errorf("check global permission: %w", err)
		}

		if allowed {
			return domain.Result{Allowed: true, Reason: permission}, nil
		}
	}

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}

	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	for _, permission := range []string{
		"group.membership.manage",
		"group.role.manage",
		"group.permission.manage",
	} {
		allowed, err := a.repository.HasGroupPermission(ctx, userID, groupID, permission)
		if err != nil {
			return domain.Result{}, fmt.Errorf("check group permission: %w", err)
		}

		if allowed {
			return domain.Result{Allowed: true, Reason: permission}, nil
		}
	}

	return domain.Result{Allowed: false, Reason: "group roster permission required"}, nil
}

// CanManageGroupMemberships permits membership acceptance, rejection, and revocation.
func (a *Authorizer) CanManageGroupMemberships(ctx context.Context, userID string, groupID uuid.UUID) (domain.Result, error) {
	return a.canManageGroupPermission(ctx, userID, groupID, "global.membership.manage", "group.membership.manage")
}

// CanRevokeGroupMembership applies the extra protection for GROUP_ADMIN memberships.
func (a *Authorizer) CanRevokeGroupMembership(ctx context.Context, userID string, groupID uuid.UUID, targetUserID string) (domain.Result, error) {
	allowed, err := a.CanManageGroupMemberships(ctx, userID, groupID)
	if err != nil || !allowed.Allowed {
		return allowed, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}

	globalPermission, err := a.repository.HasGlobalPermission(ctx, userID, "global.membership.manage")
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global membership permission: %w", err)
	}

	if admin || globalPermission {
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
func (a *Authorizer) CanRequestGroupRole(ctx context.Context, userID string, groupID uuid.UUID) (domain.Result, error) {
	if userID == "" || groupID == uuid.Nil {
		return domain.Result{Allowed: false, Reason: "invalid group access request"}, nil
	}

	_, result, err := a.activeUser(ctx, userID)
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
func (a *Authorizer) CanManageGroupRoles(ctx context.Context, userID string, groupID uuid.UUID) (domain.Result, error) {
	return a.canManageGroupPermission(ctx, userID, groupID, "global.role.manage", "group.role.manage")
}

// CanManageSpecificGroupRole applies role-specific restrictions for direct assignment and removal.
func (a *Authorizer) CanManageSpecificGroupRole(ctx context.Context, userID string, groupID uuid.UUID, role string) (domain.Result, error) {
	allowed, err := a.CanManageGroupRoles(ctx, userID, groupID)
	if err != nil || !allowed.Allowed || role != "GROUP_ADMIN" {
		return allowed, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return allowed, nil
	}

	globalPermission, err := a.repository.HasGlobalPermission(ctx, userID, "global.role.manage")
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global role permission: %w", err)
	}
	if !globalPermission {
		return domain.Result{Allowed: false, Reason: "GROUP_ADMIN requires global role management"}, nil
	}

	return allowed, nil
}

func (a *Authorizer) canManageGroupPermission(ctx context.Context, userID string, groupID uuid.UUID, globalPermission, groupPermission string) (domain.Result, error) {
	if userID == "" || groupID == uuid.Nil {
		return domain.Result{Allowed: false, Reason: "invalid group access request"}, nil
	}

	_, result, err := a.activeUser(ctx, userID)
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

	allowed, err := a.repository.HasGlobalPermission(ctx, userID, globalPermission)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global permission: %w", err)
	}
	if allowed {
		return domain.Result{Allowed: true, Reason: globalPermission}, nil
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
		return domain.Result{Allowed: false, Reason: groupPermission + " permission required"}, nil
	}

	return domain.Result{Allowed: true, Reason: groupPermission}, nil
}

// CanReadDatasetVersion applies publication and stored-group membership rules to one version.
func (a *Authorizer) CanReadDatasetVersion(ctx context.Context, userID *string, datasetID, versionID uuid.UUID) (domain.Result, error) {
	version, err := a.repository.DatasetVersionAuthz(ctx, datasetID, versionID)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return domain.Result{}, fmt.Errorf("load dataset version authorization: %w", err)
	}
	if version.Status == domain.Published {
		return domain.Result{Allowed: true, Reason: "public published resource"}, nil
	}
	if userID == nil {
		return domain.Result{Allowed: false, Reason: "authentication required"}, nil
	}

	_, result, err := a.activeUser(ctx, *userID)
	if err != nil || !result.Allowed {
		return result, err
	}
	admin, err := a.repository.IsGlobalAdmin(ctx, *userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.IsActiveMember(ctx, *userID, version.GroupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}
	return domain.Result{Allowed: true, Reason: "active group member"}, nil
}

// CanReadSoftwareApplicationVersion applies publication and stored-group membership rules to one version.
// TODO: see if we can generalize this function to a general "can read resource version"?
func (a *Authorizer) CanReadSoftwareApplicationVersion(ctx context.Context, userID *string, applicationID, versionID uuid.UUID) (domain.Result, error) {
	version, err := a.repository.SoftwareApplicationVersionAuthz(ctx, applicationID, versionID)
	if err != nil {
		return versionReadError("software application", err)
	}
	return a.authorizeVersionRead(ctx, userID, version.GroupID, version.Status)
}

// CanReadSoftwareSourceCodeVersion applies publication and stored-group membership rules to one version.
func (a *Authorizer) CanReadSoftwareSourceCodeVersion(ctx context.Context, userID *string, sourceCodeID, versionID uuid.UUID) (domain.Result, error) {
	version, err := a.repository.SoftwareSourceCodeVersionAuthz(ctx, sourceCodeID, versionID)
	if err != nil {
		return versionReadError("software source code", err)
	}
	return a.authorizeVersionRead(ctx, userID, version.GroupID, version.Status)
}

func versionReadError(resource string, err error) (domain.Result, error) {
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.Result{Allowed: false, Reason: "resource not found"}, nil
	}
	return domain.Result{}, fmt.Errorf("load %s version authorization: %w", resource, err)
}

func (a *Authorizer) authorizeVersionRead(ctx context.Context, userID *string, groupID uuid.UUID, status domain.Status) (domain.Result, error) {
	if status == domain.Published {
		return domain.Result{Allowed: true, Reason: "public published resource"}, nil
	}
	if userID == nil {
		return domain.Result{Allowed: false, Reason: "authentication required"}, nil
	}
	_, result, err := a.activeUser(ctx, *userID)
	if err != nil || !result.Allowed {
		return result, err
	}
	admin, err := a.repository.IsGlobalAdmin(ctx, *userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}
	member, err := a.repository.IsActiveMember(ctx, *userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}
	return domain.Result{Allowed: true, Reason: "active group member"}, nil
}

// AuthorizeAdmin permits active global administrators only.
func (a *Authorizer) AuthorizeAdmin(ctx context.Context, userID string) (domain.Result, error) {
	_, result, err := a.activeUser(ctx, userID)
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
func (a *Authorizer) AuthorizeGlobal(ctx context.Context, userID, permissionKey string) (domain.Result, error) {
	if err := a.requirePermissionScope(ctx, permissionKey, domain.PermissionScopeGlobal); err != nil {
		return domain.Result{}, err
	}

	_, result, err := a.activeUser(ctx, userID)
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

	allowed, err := a.repository.HasGlobalPermission(ctx, userID, permissionKey)
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
	permissionKey string,
) (domain.Result, error) {
	groupID, err := a.resourceGroup(ctx, resourceType, resourceID)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return domain.Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return domain.Result{}, fmt.Errorf("load resource group: %w", err)
	}

	return a.AuthorizeGroup(ctx, userID, groupID, permissionKey)
}

// AuthorizeGroup handles resource creation before a logical resource exists.
func (a *Authorizer) AuthorizeGroup(ctx context.Context, userID string, groupID uuid.UUID, permissionKey string) (domain.Result, error) {
	if err := a.requirePermissionScope(ctx, permissionKey, domain.PermissionScopeGroup); err != nil {
		return domain.Result{}, err
	}

	_, result, err := a.activeUser(ctx, userID)
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

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	allowed, err := a.repository.HasGroupPermission(ctx, userID, groupID, permissionKey)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group permission: %w", err)
	}
	if !allowed {
		return domain.Result{Allowed: false, Reason: "permission not granted"}, nil
	}

	return domain.Result{Allowed: true, Reason: "group permission"}, nil
}

// CanReadResource permits public reads of published resources. Non-public
// resources require an active member in stored resource group or global admin.
func (a *Authorizer) CanReadResource(ctx context.Context, userID *string, resourceType domain.ResourceType, resourceID uuid.UUID) (domain.Result, error) {
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

	_, result, err := a.activeUser(ctx, *userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, *userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.IsActiveMember(ctx, *userID, groupID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}
	return domain.Result{Allowed: true, Reason: "active group member"}, nil
}

// CanReadContactPoint protects email data while allowing active members in any group.
func (a *Authorizer) CanReadContactPoint(ctx context.Context, userID string) (domain.Result, error) {
	_, result, err := a.activeUser(ctx, userID)
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

// CanReadDiscoveryCreatedBy allows creator identity exposure to active users
// with membership in at least one group.
func (a *Authorizer) CanReadDiscoveryCreatedBy(ctx context.Context, userID *string) (domain.Result, error) {
	if userID == nil || *userID == "" {
		return domain.Result{Allowed: false, Reason: "authentication required"}, nil
	}

	_, result, err := a.activeUser(ctx, *userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	isAdmin, err := a.repository.IsGlobalAdmin(ctx, *userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check global admin: %w", err)
	}

	if isAdmin {
		return domain.Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.HasAnyActiveMembership(ctx, *userID)
	if err != nil {
		return domain.Result{}, fmt.Errorf("check active memberships: %w", err)
	}
	if !member {
		return domain.Result{Allowed: false, Reason: "user has no active group membership"}, nil
	}

	return domain.Result{Allowed: true, Reason: "active group member"}, nil
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
		return domain.AuthorizationUser{ID: "", Status: ""}, domain.Result{Allowed: false, Reason: "user is not active"}, nil
	}
	return user, domain.Result{Allowed: true, Reason: ""}, nil
}

func (a *Authorizer) requirePermissionScope(ctx context.Context, permissionKey string, scope domain.PermissionScope) error {
	actual, err := a.repository.PermissionScope(ctx, permissionKey)
	if errors.Is(err, domain.ErrAuthorizationNotFound) {
		return fmt.Errorf("unknown permission: %s", permissionKey)
	}
	if err != nil {
		return fmt.Errorf("get permission scope: %w", err)
	}
	if actual != scope {
		return fmt.Errorf("permission %s is not %s scoped", permissionKey, scope)
	}
	return nil
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
