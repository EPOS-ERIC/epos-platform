// Package authorization contains stored-resource access decisions.
package authorization

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ResourceType string

const (
	ResourceTypeDataset             ResourceType = "dataset"
	ResourceTypeSoftwareApplication ResourceType = "software_application"
	ResourceTypeSoftwareSourceCode  ResourceType = "software_source_code"
)

type Result struct {
	Allowed bool
	Reason  string
}

type repository interface {
	GetAuthorizationUser(context.Context, uuid.UUID) (sqlc.GetAuthorizationUserRow, error)
	GetPermissionScope(context.Context, string) (sqlc.PermissionScope, error)
	IsGlobalAdmin(context.Context, uuid.UUID) (bool, error)
	HasGlobalPermission(context.Context, sqlc.HasGlobalPermissionParams) (bool, error)
	IsActiveMember(context.Context, sqlc.IsActiveMemberParams) (bool, error)
	HasAnyActiveMembership(context.Context, uuid.UUID) (bool, error)
	HasGroupPermission(context.Context, sqlc.HasGroupPermissionParams) (bool, error)
	GetDatasetGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	GetSoftwareApplicationGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	GetSoftwareSourceCodeGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	IsDatasetPublished(context.Context, uuid.UUID) (bool, error)
	IsSoftwareApplicationPublished(context.Context, uuid.UUID) (bool, error)
	IsSoftwareSourceCodePublished(context.Context, uuid.UUID) (bool, error)
}

type Authorizer struct {
	repository repository
}

func New(repository repository) *Authorizer {
	return &Authorizer{repository: repository}
}

// AuthorizeGlobal evaluates a global permission without requiring membership.
func (a *Authorizer) AuthorizeGlobal(ctx context.Context, userID uuid.UUID, permissionKey string) (Result, error) {
	if err := a.requirePermissionScope(ctx, permissionKey, sqlc.PermissionScopeGLOBAL); err != nil {
		return Result{}, err
	}

	_, result, err := a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return Result{Allowed: true, Reason: "global admin"}, nil
	}

	allowed, err := a.repository.HasGlobalPermission(ctx, sqlc.HasGlobalPermissionParams{UserID: userID, PermissionKey: permissionKey})
	if err != nil {
		return Result{}, fmt.Errorf("check global permission: %w", err)
	}
	if !allowed {
		return Result{Allowed: false, Reason: "permission not granted"}, nil
	}

	return Result{Allowed: true, Reason: "global permission"}, nil
}

// AuthorizeResource gets group from stored logical resource, never caller input.
func (a *Authorizer) AuthorizeResource(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID, permissionKey string) (Result, error) {
	groupID, err := a.resourceGroup(ctx, resourceType, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("load resource group: %w", err)
	}

	return a.AuthorizeGroup(ctx, userID, groupID, permissionKey)
}

// AuthorizeGroup handles resource creation before a logical resource exists.
func (a *Authorizer) AuthorizeGroup(ctx context.Context, userID, groupID uuid.UUID, permissionKey string) (Result, error) {
	if err := a.requirePermissionScope(ctx, permissionKey, sqlc.PermissionScopeGROUP); err != nil {
		return Result{}, err
	}

	_, result, err := a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.IsActiveMember(ctx, sqlc.IsActiveMemberParams{UserID: userID, GroupID: groupID})
	if err != nil {
		return Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	allowed, err := a.repository.HasGroupPermission(ctx, sqlc.HasGroupPermissionParams{UserID: userID, GroupID: groupID, PermissionKey: permissionKey})
	if err != nil {
		return Result{}, fmt.Errorf("check group permission: %w", err)
	}
	if !allowed {
		return Result{Allowed: false, Reason: "permission not granted"}, nil
	}

	return Result{Allowed: true, Reason: "group permission"}, nil
}

// CanReadResource permits public reads of published resources. Non-public
// resources require an active member in stored resource group or global admin.
func (a *Authorizer) CanReadResource(ctx context.Context, userID *uuid.UUID, resourceType ResourceType, resourceID uuid.UUID) (Result, error) {
	groupID, err := a.resourceGroup(ctx, resourceType, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("load resource group: %w", err)
	}

	published, err := a.resourcePublished(ctx, resourceType, resourceID)
	if err != nil {
		return Result{}, err
	}
	if published {
		return Result{Allowed: true, Reason: "public published resource"}, nil
	}
	if userID == nil {
		return Result{Allowed: false, Reason: "authentication required"}, nil
	}

	_, result, err := a.activeUser(ctx, *userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, *userID)
	if err != nil {
		return Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.IsActiveMember(ctx, sqlc.IsActiveMemberParams{UserID: *userID, GroupID: groupID})
	if err != nil {
		return Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}
	return Result{Allowed: true, Reason: "active group member"}, nil
}

// CanReadContactPoint protects email data while allowing active members in any group.
func (a *Authorizer) CanReadContactPoint(ctx context.Context, userID uuid.UUID) (Result, error) {
	_, result, err := a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}

	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if admin {
		return Result{Allowed: true, Reason: "global admin"}, nil
	}

	member, err := a.repository.HasAnyActiveMembership(ctx, userID)
	if err != nil {
		return Result{}, fmt.Errorf("check active memberships: %w", err)
	}
	if !member {
		return Result{Allowed: false, Reason: "user has no active group membership"}, nil
	}

	return Result{Allowed: true, Reason: "active group member"}, nil
}

func (a *Authorizer) resourcePublished(ctx context.Context, typ ResourceType, id uuid.UUID) (bool, error) {
	switch typ {
	case ResourceTypeDataset:
		published, err := a.repository.IsDatasetPublished(ctx, id)
		if err != nil {
			return false, fmt.Errorf("check dataset publication: %w", err)
		}
		return published, nil
	case ResourceTypeSoftwareApplication:
		published, err := a.repository.IsSoftwareApplicationPublished(ctx, id)
		if err != nil {
			return false, fmt.Errorf("check software application publication: %w", err)
		}
		return published, nil
	case ResourceTypeSoftwareSourceCode:
		published, err := a.repository.IsSoftwareSourceCodePublished(ctx, id)
		if err != nil {
			return false, fmt.Errorf("check software source code publication: %w", err)
		}
		return published, nil
	default:
		return false, fmt.Errorf("unsupported resource type: %q", typ)
	}
}

func (a *Authorizer) activeUser(ctx context.Context, userID uuid.UUID) (sqlc.GetAuthorizationUserRow, Result, error) {
	user, err := a.repository.GetAuthorizationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.GetAuthorizationUserRow{ID: uuid.Nil, Status: ""}, Result{Allowed: false, Reason: "user not found"}, nil
	}
	if err != nil {
		return sqlc.GetAuthorizationUserRow{}, Result{}, fmt.Errorf("load user: %w", err)
	}
	if user.Status != sqlc.UserStatusACTIVE {
		return sqlc.GetAuthorizationUserRow{ID: uuid.Nil, Status: ""}, Result{Allowed: false, Reason: "user is not active"}, nil
	}
	return user, Result{Allowed: true, Reason: ""}, nil
}

func (a *Authorizer) requirePermissionScope(ctx context.Context, permissionKey string, scope sqlc.PermissionScope) error {
	actual, err := a.repository.GetPermissionScope(ctx, permissionKey)
	if errors.Is(err, pgx.ErrNoRows) {
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

func (a *Authorizer) resourceGroup(ctx context.Context, typ ResourceType, id uuid.UUID) (uuid.UUID, error) {
	switch typ {
	case ResourceTypeDataset:
		groupID, err := a.repository.GetDatasetGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get dataset group: %w", err)
		}

		return groupID, nil
	case ResourceTypeSoftwareApplication:
		groupID, err := a.repository.GetSoftwareApplicationGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get software application group: %w", err)
		}

		return groupID, nil
	case ResourceTypeSoftwareSourceCode:
		groupID, err := a.repository.GetSoftwareSourceCodeGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get software source code group: %w", err)
		}

		return groupID, nil
	default:
		return uuid.Nil, fmt.Errorf("unsupported resource type: %q", typ)
	}
}
