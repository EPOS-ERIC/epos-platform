// Package authz provides PostgreSQL persistence for authorization decisions.
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repo struct {
	*sqlc.Queries
}

func (r *Repo) User(ctx context.Context, userID uuid.UUID) (authz.User, error) {
	user, err := r.GetAuthorizationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return authz.User{}, authz.ErrNotFound
	}
	if err != nil {
		return authz.User{}, fmt.Errorf("get authorization user: %w", err)
	}
	return authz.User{ID: user.ID, Status: authz.UserStatus(user.Status)}, nil
}

func (r *Repo) PermissionScope(ctx context.Context, key string) (authz.PermissionScope, error) {
	scope, err := r.GetPermissionScope(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", authz.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get permission scope: %w", err)
	}
	return authz.PermissionScope(scope), nil
}

func (r *Repo) HasGlobalPermission(ctx context.Context, userID uuid.UUID, permissionKey string) (bool, error) {
	allowed, err := r.Queries.HasGlobalPermission(ctx, sqlc.HasGlobalPermissionParams{
		UserID:        userID,
		PermissionKey: permissionKey,
	})
	return allowed, wrap("check global permission", err)
}

func (r *Repo) IsActiveMember(ctx context.Context, userID, groupID uuid.UUID) (bool, error) {
	active, err := r.Queries.IsActiveMember(ctx, sqlc.IsActiveMemberParams{
		UserID:  userID,
		GroupID: groupID,
	})
	return active, wrap("check active membership", err)
}

func (r *Repo) HasGroupPermission(ctx context.Context, userID, groupID uuid.UUID, permissionKey string) (bool, error) {
	allowed, err := r.Queries.HasGroupPermission(ctx, sqlc.HasGroupPermissionParams{
		UserID:        userID,
		GroupID:       groupID,
		PermissionKey: permissionKey,
	})
	return allowed, wrap("check group permission", err)
}

func (r *Repo) DatasetVersionAuthz(ctx context.Context, datasetID, versionID uuid.UUID) (authz.VersionAuthz, error) {
	version, err := r.GetDatasetVersionAuthorization(ctx, sqlc.GetDatasetVersionAuthorizationParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return authz.VersionAuthz{}, authz.ErrNotFound
	}
	if err != nil {
		return authz.VersionAuthz{}, fmt.Errorf("get dataset version authorization: %w", err)
	}
	return authz.VersionAuthz{
		ResourceID: version.DatasetID,
		GroupID:    version.GroupID,
		Status:     authz.ResourceStatus(version.Status),
	}, nil
}

func (r *Repo) SoftwareApplicationVersionAuthz(ctx context.Context, applicationID, versionID uuid.UUID) (authz.VersionAuthz, error) {
	version, err := r.GetSoftwareApplicationVersionAuthorization(ctx, sqlc.GetSoftwareApplicationVersionAuthorizationParams{
		SoftwareApplicationID: applicationID,
		VersionID:             versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return authz.VersionAuthz{}, authz.ErrNotFound
	}
	if err != nil {
		return authz.VersionAuthz{}, fmt.Errorf("get software application version authorization: %w", err)
	}
	return authz.VersionAuthz{
		ResourceID: version.SoftwareApplicationID,
		GroupID:    version.GroupID,
		Status:     authz.ResourceStatus(version.Status),
	}, nil
}

func (r *Repo) SoftwareSourceCodeVersionAuthz(ctx context.Context, sourceCodeID, versionID uuid.UUID) (authz.VersionAuthz, error) {
	version, err := r.GetSoftwareSourceCodeVersionAuthorization(ctx, sqlc.GetSoftwareSourceCodeVersionAuthorizationParams{
		SoftwareSourceCodeID: sourceCodeID,
		VersionID:            versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return authz.VersionAuthz{}, authz.ErrNotFound
	}
	if err != nil {
		return authz.VersionAuthz{}, fmt.Errorf("get software source code version authorization: %w", err)
	}
	return authz.VersionAuthz{
		ResourceID: version.SoftwareSourceCodeID,
		GroupID:    version.GroupID,
		Status:     authz.ResourceStatus(version.Status),
	}, nil
}

func (r *Repo) DatasetGroup(ctx context.Context, resourceID uuid.UUID) (uuid.UUID, error) {
	resource, err := r.GetDatasetGroup(ctx, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, authz.ErrNotFound
	}
	return resource, nil
}

func (r *Repo) SoftwareApplicationGroup(ctx context.Context, resourceID uuid.UUID) (uuid.UUID, error) {
	resource, err := r.GetSoftwareApplicationGroup(ctx, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, authz.ErrNotFound
	}
	return resource, nil
}

func (r *Repo) SoftwareSourceCodeGroup(ctx context.Context, resourceID uuid.UUID) (uuid.UUID, error) {
	resource, err := r.GetSoftwareSourceCodeGroup(ctx, resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, authz.ErrNotFound
	}
	return resource, nil
}

func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
