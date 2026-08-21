// Package identity provides PostgreSQL persistence for external identity resolution.
package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	identitydomain "github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repo struct {
	*sqlc.Queries
}

func (r *Repo) ListUsers(ctx context.Context) ([]identitydomain.UserSummary, error) {
	rows, err := r.Queries.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]identitydomain.UserSummary, 0, len(rows))
	for _, row := range rows {
		roles, err := r.GetUserGlobalRoles(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("get global roles for %s: %w", row.ID, err)
		}

		permissions, err := r.GetUserGlobalPermissions(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("get global permissions for %s: %w", row.ID, err)
		}

		users = append(users, identitydomain.UserSummary{
			ID:                row.ID,
			Email:             row.Email,
			GivenName:         row.GivenName,
			FamilyName:        row.FamilyName,
			Status:            string(row.Status),
			GlobalRoles:       roles,
			GlobalPermissions: permissions,
		})
	}

	return users, nil
}

func (r *Repo) GetUserByID(ctx context.Context, userID string) (identitydomain.User, error) {
	user, err := r.Queries.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.User{}, identitydomain.ErrNotFound
	}
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return r.mapUser(ctx, user.ID, user.Email, user.GivenName, user.FamilyName, user.Status, user.CreatedAt, user.UpdatedAt)
}

func (r *Repo) mapUser(
	ctx context.Context,
	id,
	email,
	givenName,
	familyName string,
	status sqlc.UserStatus,
	createdAt,
	updatedAt time.Time,
) (identitydomain.User, error) {
	globalRoles, err := r.GetUserGlobalRoles(ctx, id)
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user global roles: %w", err)
	}

	globalPermissions, err := r.GetUserGlobalPermissions(ctx, id)
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user global permissions: %w", err)
	}

	memberships, err := r.GetUserMemberships(ctx, id)
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user memberships: %w", err)
	}

	roles, err := r.GetUserMembershipRoles(ctx, id)
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user membership roles: %w", err)
	}

	permissions, err := r.GetUserMembershipPermissions(ctx, id)
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user membership permissions: %w", err)
	}

	roleRequests, err := r.ListPendingUserGroupRoleRequests(ctx, id)
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user pending role requests: %w", err)
	}

	rolesByGroup := make(map[uuid.UUID][]string)
	for _, role := range roles {
		rolesByGroup[role.GroupID] = append(rolesByGroup[role.GroupID], role.RoleKey)
	}

	permissionsByGroup := make(map[uuid.UUID][]string)
	for _, permission := range permissions {
		permissionsByGroup[permission.GroupID] = append(permissionsByGroup[permission.GroupID], permission.PermissionKey)
	}

	roleRequestsByGroup := make(map[uuid.UUID][]identitydomain.RoleRequest)
	for _, request := range roleRequests {
		roleRequestsByGroup[request.GroupID] = append(roleRequestsByGroup[request.GroupID], identitydomain.RoleRequest{
			ID:          request.ID,
			GroupID:     request.GroupID,
			Role:        request.RoleKey,
			Status:      string(request.Status),
			RequestedAt: request.RequestedAt,
		})
	}

	userMemberships := make([]identitydomain.Membership, 0, len(memberships))
	for _, membership := range memberships {
		userMemberships = append(userMemberships, identitydomain.Membership{
			GroupID:      membership.GroupID,
			GroupName:    membership.GroupName,
			Roles:        rolesByGroup[membership.GroupID],
			Status:       string(membership.Status),
			Permissions:  permissionsByGroup[membership.GroupID],
			RoleRequests: append([]identitydomain.RoleRequest{}, roleRequestsByGroup[membership.GroupID]...),
			CreatedAt:    membership.CreatedAt,
			UpdatedAt:    membership.UpdatedAt,
		})
	}

	return identitydomain.User{
		ID:                id,
		Email:             email,
		GivenName:         givenName,
		FamilyName:        familyName,
		Status:            string(status),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		GlobalRoles:       append([]string{}, globalRoles...),
		GlobalPermissions: append([]string{}, globalPermissions...),
		Memberships:       userMemberships,
	}, nil
}

func (r *Repo) CreateUser(ctx context.Context, claims identitydomain.Claims) (identitydomain.User, error) {
	user, err := r.Queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:         claims.ID,
		Email:      claims.Email,
		GivenName:  claims.GivenName,
		FamilyName: claims.FamilyName,
	})
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("create user: %w", err)
	}

	return mapUser(user.ID, user.Email, user.GivenName, user.FamilyName, user.Status, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repo) GrantGlobalPermission(ctx context.Context, userID, permission, grantedBy string) error {
	_, err := r.GetAuthorizationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}

	scope, err := r.GetPermissionScope(ctx, permission)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.ErrInvalid
	}
	if err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if scope != sqlc.PermissionScopeGLOBAL {
		return identitydomain.ErrInvalid
	}

	err = r.Queries.GrantGlobalPermission(ctx, sqlc.GrantGlobalPermissionParams{
		UserID:        userID,
		PermissionKey: permission,
		GrantedBy:     grantedBy,
	})
	if err != nil {
		return fmt.Errorf("grant global permission: %w", err)
	}

	return nil
}

func (r *Repo) RevokeGlobalPermission(ctx context.Context, userID, permission string) error {
	_, err := r.GetAuthorizationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}

	scope, err := r.GetPermissionScope(ctx, permission)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.ErrInvalid
	}
	if err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if scope != sqlc.PermissionScopeGLOBAL {
		return identitydomain.ErrInvalid
	}

	err = r.Queries.RevokeGlobalPermission(ctx, sqlc.RevokeGlobalPermissionParams{
		UserID:        userID,
		PermissionKey: permission,
	})
	if err != nil {
		return fmt.Errorf("revoke global permission: %w", err)
	}

	return nil
}

func mapUser(id, email, givenName, familyName string, status sqlc.UserStatus, createdAt, updatedAt time.Time) identitydomain.User {
	return identitydomain.User{
		ID:                id,
		Email:             email,
		GivenName:         givenName,
		FamilyName:        familyName,
		Status:            string(status),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		GlobalRoles:       []string{},
		GlobalPermissions: []string{},
		Memberships:       []identitydomain.Membership{},
	}
}
