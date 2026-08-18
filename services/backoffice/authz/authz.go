// Package authz contains stored-resource access decisions.
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

var ErrNotFound = errors.New("authorization resource not found")

type UserStatus string

const (
	UserStatusActive  UserStatus = "ACTIVE"
	UserStatusBlocked UserStatus = "BLOCKED"
)

type PermissionScope string

const (
	PermissionScopeGlobal PermissionScope = "GLOBAL"
	PermissionScopeGroup  PermissionScope = "GROUP"
)

type ResourceStatus string

const (
	ResourceStatusDraft     ResourceStatus = "DRAFT"
	ResourceStatusSubmitted ResourceStatus = "SUBMITTED"
	ResourceStatusRejected  ResourceStatus = "REJECTED"
	ResourceStatusPublished ResourceStatus = "PUBLISHED"
	ResourceStatusArchived  ResourceStatus = "ARCHIVED"
)

type User struct {
	ID     uuid.UUID
	Status UserStatus
}

type VersionAuthz struct {
	ResourceID uuid.UUID
	GroupID    uuid.UUID
	Status     ResourceStatus
}

type repository interface {
	User(context.Context, uuid.UUID) (User, error)
	PermissionScope(context.Context, string) (PermissionScope, error)
	IsGlobalAdmin(context.Context, uuid.UUID) (bool, error)
	HasGlobalPermission(context.Context, uuid.UUID, string) (bool, error)
	IsActiveMember(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	HasAnyActiveMembership(context.Context, uuid.UUID) (bool, error)
	HasGroupPermission(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
	DatasetGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	SoftwareApplicationGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	SoftwareSourceCodeGroup(context.Context, uuid.UUID) (uuid.UUID, error)
	IsDatasetPublished(context.Context, uuid.UUID) (bool, error)
	IsSoftwareApplicationPublished(context.Context, uuid.UUID) (bool, error)
	IsSoftwareSourceCodePublished(context.Context, uuid.UUID) (bool, error)
	DatasetVersionAuthz(context.Context, uuid.UUID, uuid.UUID) (VersionAuthz, error)
	SoftwareApplicationVersionAuthz(context.Context, uuid.UUID, uuid.UUID) (VersionAuthz, error)
	SoftwareSourceCodeVersionAuthz(context.Context, uuid.UUID, uuid.UUID) (VersionAuthz, error)
}

// CanReadDatasetVersion applies publication and stored-group membership rules to one version.
func (a *Authorizer) CanReadDatasetVersion(ctx context.Context, userID *uuid.UUID, datasetID, versionID uuid.UUID) (Result, error) {
	version, err := a.repository.DatasetVersionAuthz(ctx, datasetID, versionID)
	if errors.Is(err, ErrNotFound) {
		return Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("load dataset version authorization: %w", err)
	}
	if version.Status == ResourceStatusPublished {
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

	member, err := a.repository.IsActiveMember(ctx, *userID, version.GroupID)
	if err != nil {
		return Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}
	return Result{Allowed: true, Reason: "active group member"}, nil
}

// CanReadSoftwareApplicationVersion applies publication and stored-group membership rules to one version.
func (a *Authorizer) CanReadSoftwareApplicationVersion(ctx context.Context, userID *uuid.UUID, applicationID, versionID uuid.UUID) (Result, error) {
	version, err := a.repository.SoftwareApplicationVersionAuthz(ctx, applicationID, versionID)
	if err != nil {
		return versionReadError("software application", err)
	}
	return a.authorizeVersionRead(ctx, userID, version.GroupID, version.Status)
}

// CanReadSoftwareSourceCodeVersion applies publication and stored-group membership rules to one version.
func (a *Authorizer) CanReadSoftwareSourceCodeVersion(ctx context.Context, userID *uuid.UUID, sourceCodeID, versionID uuid.UUID) (Result, error) {
	version, err := a.repository.SoftwareSourceCodeVersionAuthz(ctx, sourceCodeID, versionID)
	if err != nil {
		return versionReadError("software source code", err)
	}
	return a.authorizeVersionRead(ctx, userID, version.GroupID, version.Status)
}

func versionReadError(resource string, err error) (Result, error) {
	if errors.Is(err, ErrNotFound) {
		return Result{Allowed: false, Reason: "resource not found"}, nil
	}
	return Result{}, fmt.Errorf("load %s version authorization: %w", resource, err)
}

func (a *Authorizer) authorizeVersionRead(ctx context.Context, userID *uuid.UUID, groupID uuid.UUID, status ResourceStatus) (Result, error) {
	if status == ResourceStatusPublished {
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
	member, err := a.repository.IsActiveMember(ctx, *userID, groupID)
	if err != nil {
		return Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}
	return Result{Allowed: true, Reason: "active group member"}, nil
}

type Authorizer struct {
	repository repository
}

// New creates an authorizer backed by repository.
func New(repository repository) *Authorizer {
	return &Authorizer{repository: repository}
}

// AuthorizeAdmin permits active global administrators only.
func (a *Authorizer) AuthorizeAdmin(ctx context.Context, userID uuid.UUID) (Result, error) {
	_, result, err := a.activeUser(ctx, userID)
	if err != nil || !result.Allowed {
		return result, err
	}
	admin, err := a.repository.IsGlobalAdmin(ctx, userID)
	if err != nil {
		return Result{}, fmt.Errorf("check global admin: %w", err)
	}
	if !admin {
		return Result{Allowed: false, Reason: "global admin required"}, nil
	}
	return Result{Allowed: true, Reason: "global admin"}, nil
}

// AuthorizeGlobal evaluates a global permission without requiring membership.
func (a *Authorizer) AuthorizeGlobal(ctx context.Context, userID uuid.UUID, permissionKey string) (Result, error) {
	if err := a.requirePermissionScope(ctx, permissionKey, PermissionScopeGlobal); err != nil {
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

	allowed, err := a.repository.HasGlobalPermission(ctx, userID, permissionKey)
	if err != nil {
		return Result{}, fmt.Errorf("check global permission: %w", err)
	}
	if !allowed {
		return Result{Allowed: false, Reason: "permission not granted"}, nil
	}

	return Result{Allowed: true, Reason: "global permission"}, nil
}

// AuthorizeResource gets group from stored logical resource, never caller input.
func (a *Authorizer) AuthorizeResource(
	ctx context.Context,
	userID uuid.UUID,
	resourceType ResourceType,
	resourceID uuid.UUID,
	permissionKey string,
) (Result, error) {
	groupID, err := a.resourceGroup(ctx, resourceType, resourceID)
	if errors.Is(err, ErrNotFound) {
		return Result{Allowed: false, Reason: "resource not found"}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("load resource group: %w", err)
	}

	return a.AuthorizeGroup(ctx, userID, groupID, permissionKey)
}

// AuthorizeGroup handles resource creation before a logical resource exists.
func (a *Authorizer) AuthorizeGroup(ctx context.Context, userID, groupID uuid.UUID, permissionKey string) (Result, error) {
	if err := a.requirePermissionScope(ctx, permissionKey, PermissionScopeGroup); err != nil {
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

	member, err := a.repository.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return Result{}, fmt.Errorf("check group membership: %w", err)
	}
	if !member {
		return Result{Allowed: false, Reason: "user is not an active group member"}, nil
	}

	allowed, err := a.repository.HasGroupPermission(ctx, userID, groupID, permissionKey)
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
	if errors.Is(err, ErrNotFound) {
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

	member, err := a.repository.IsActiveMember(ctx, *userID, groupID)
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

func (a *Authorizer) activeUser(ctx context.Context, userID uuid.UUID) (User, Result, error) {
	user, err := a.repository.User(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return User{ID: uuid.Nil, Status: ""}, Result{Allowed: false, Reason: "user not found"}, nil
	}
	if err != nil {
		return User{}, Result{}, fmt.Errorf("load user: %w", err)
	}
	if user.Status != UserStatusActive {
		return User{ID: uuid.Nil, Status: ""}, Result{Allowed: false, Reason: "user is not active"}, nil
	}
	return user, Result{Allowed: true, Reason: ""}, nil
}

func (a *Authorizer) requirePermissionScope(ctx context.Context, permissionKey string, scope PermissionScope) error {
	actual, err := a.repository.PermissionScope(ctx, permissionKey)
	if errors.Is(err, ErrNotFound) {
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
		groupID, err := a.repository.DatasetGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get dataset group: %w", err)
		}

		return groupID, nil
	case ResourceTypeSoftwareApplication:
		groupID, err := a.repository.SoftwareApplicationGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get software application group: %w", err)
		}

		return groupID, nil
	case ResourceTypeSoftwareSourceCode:
		groupID, err := a.repository.SoftwareSourceCodeGroup(ctx, id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get software source code group: %w", err)
		}

		return groupID, nil
	default:
		return uuid.Nil, fmt.Errorf("unsupported resource type: %q", typ)
	}
}
