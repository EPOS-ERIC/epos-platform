// Package authz provides PostgreSQL persistence for authorization decisions.
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{
		Queries: sqlc.New(db),
	}
}

func (r *Repo) User(ctx context.Context, userID string) (domain.AuthorizationUser, error) {
	user, err := r.GetAuthorizationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthorizationUser{}, domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return domain.AuthorizationUser{}, fmt.Errorf("get authorization user: %w", err)
	}
	result, err := toDomainUser(user)
	if err != nil {
		return domain.AuthorizationUser{}, fmt.Errorf("construct authorization user: %w", err)
	}

	return result, nil
}

func (r *Repo) PermissionScope(ctx context.Context, key string) (domain.PermissionScope, error) {
	scope, err := r.GetPermissionScope(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get permission scope: %w", err)
	}
	result, err := toDomainPermissionScope(scope)
	if err != nil {
		return "", fmt.Errorf("construct permission scope: %w", err)
	}

	return result, nil
}

func (r *Repo) HasGlobalPermission(ctx context.Context, userID, permissionKey string) (bool, error) {
	allowed, err := r.Queries.HasGlobalPermission(ctx, sqlc.HasGlobalPermissionParams{
		UserID:        userID,
		PermissionKey: permissionKey,
	})
	if err != nil {
		return false, fmt.Errorf("check global permission: %w", err)
	}

	return allowed, nil
}

func (r *Repo) IsActiveMember(ctx context.Context, userID string, groupID uuid.UUID) (bool, error) {
	active, err := r.Queries.IsActiveMember(ctx, sqlc.IsActiveMemberParams{
		UserID:  userID,
		GroupID: groupID,
	})
	if err != nil {
		return false, fmt.Errorf("check active membership: %w", err)
	}

	return active, nil
}

// ActiveMemberGroupIDs returns candidate groups where user has active membership.
func (r *Repo) ActiveMemberGroupIDs(ctx context.Context, userID string, groupIDs []uuid.UUID) ([]uuid.UUID, error) {
	result, err := r.ListActiveMemberGroupIDs(ctx, sqlc.ListActiveMemberGroupIDsParams{
		UserID:   userID,
		GroupIds: groupIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list active member groups: %w", err)
	}

	return result, nil
}

func (r *Repo) HasGroupRole(ctx context.Context, userID string, groupID uuid.UUID, roleKey string) (bool, error) {
	hasRole, err := r.Queries.HasGroupRole(ctx, sqlc.HasGroupRoleParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: roleKey,
	})
	if err != nil {
		return false, fmt.Errorf("check group role: %w", err)
	}

	return hasRole, nil
}

func (r *Repo) HasGroupPermission(ctx context.Context, userID string, groupID uuid.UUID, permissionKey string) (bool, error) {
	allowed, err := r.Queries.HasGroupPermission(ctx, sqlc.HasGroupPermissionParams{
		UserID:        userID,
		GroupID:       groupID,
		PermissionKey: permissionKey,
	})
	if err != nil {
		return false, fmt.Errorf("check group permission: %w", err)
	}

	return allowed, nil
}

func (r *Repo) DatasetVersionAuthz(ctx context.Context, datasetID, versionID uuid.UUID) (domain.VersionAuthz, error) {
	version, err := r.GetDatasetVersionAuthorization(ctx, sqlc.GetDatasetVersionAuthorizationParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.VersionAuthz{}, domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("get dataset version authorization: %w", err)
	}
	result, err := toDomainVersionAuthz(version.DatasetID, version.GroupID, version.Status)
	if err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("construct dataset version authorization: %w", err)
	}

	return result, nil
}

func (r *Repo) SoftwareApplicationVersionAuthz(ctx context.Context, applicationID, versionID uuid.UUID) (domain.VersionAuthz, error) {
	version, err := r.GetSoftwareApplicationVersionAuthorization(ctx, sqlc.GetSoftwareApplicationVersionAuthorizationParams{
		SoftwareApplicationID: applicationID,
		VersionID:             versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.VersionAuthz{}, domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("get software application version authorization: %w", err)
	}
	result, err := toDomainVersionAuthz(version.SoftwareApplicationID, version.GroupID, version.Status)
	if err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("construct software application version authorization: %w", err)
	}

	return result, nil
}

func (r *Repo) SoftwareSourceCodeVersionAuthz(ctx context.Context, sourceCodeID, versionID uuid.UUID) (domain.VersionAuthz, error) {
	version, err := r.GetSoftwareSourceCodeVersionAuthorization(ctx, sqlc.GetSoftwareSourceCodeVersionAuthorizationParams{
		SoftwareSourceCodeID: sourceCodeID,
		VersionID:            versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.VersionAuthz{}, domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("get software source code version authorization: %w", err)
	}
	result, err := toDomainVersionAuthz(version.SoftwareSourceCodeID, version.GroupID, version.Status)
	if err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("construct software source code version authorization: %w", err)
	}

	return result, nil
}

func (r *Repo) DatasetGroup(ctx context.Context, resourceID uuid.UUID) (uuid.UUID, error) {
	resource, err := r.GetDatasetGroup(ctx, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get dataset group: %w", err)
	}

	return resource, nil
}

func (r *Repo) SoftwareApplicationGroup(ctx context.Context, resourceID uuid.UUID) (uuid.UUID, error) {
	resource, err := r.GetSoftwareApplicationGroup(ctx, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get software application group: %w", err)
	}

	return resource, nil
}

func (r *Repo) SoftwareSourceCodeGroup(ctx context.Context, resourceID uuid.UUID) (uuid.UUID, error) {
	resource, err := r.GetSoftwareSourceCodeGroup(ctx, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, domain.ErrAuthorizationNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get software source code group: %w", err)
	}

	return resource, nil
}
