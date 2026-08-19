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

func (r *Repo) GetUserByAuthIdentifier(ctx context.Context, authIdentifier string) (identitydomain.User, error) {
	user, err := r.Queries.GetUserByAuthIdentifier(ctx, authIdentifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.User{}, identitydomain.ErrNotFound
	}
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user by auth identifier: %w", err)
	}
	return r.mapUser(ctx, user.ID, user.AuthIdentifier, user.Email, user.GivenName, user.FamilyName, user.Status, user.CreatedAt, user.UpdatedAt)
}

func (r *Repo) GetUserByID(ctx context.Context, userID uuid.UUID) (identitydomain.User, error) {
	user, err := r.Queries.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.User{}, identitydomain.ErrNotFound
	}
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return r.mapUser(ctx, user.ID, user.AuthIdentifier, user.Email, user.GivenName, user.FamilyName, user.Status, user.CreatedAt, user.UpdatedAt)
}

func (r *Repo) mapUser(
	ctx context.Context,
	id uuid.UUID,
	authIdentifier,
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

	rolesByGroup := make(map[uuid.UUID][]string)
	for _, role := range roles {
		rolesByGroup[role.GroupID] = append(rolesByGroup[role.GroupID], role.RoleKey)
	}
	permissionsByGroup := make(map[uuid.UUID][]string)
	for _, permission := range permissions {
		permissionsByGroup[permission.GroupID] = append(permissionsByGroup[permission.GroupID], permission.PermissionKey)
	}
	userMemberships := make([]identitydomain.Membership, 0, len(memberships))
	for _, membership := range memberships {
		userMemberships = append(userMemberships, identitydomain.Membership{
			GroupID:     membership.GroupID,
			GroupName:   membership.GroupName,
			Roles:       rolesByGroup[membership.GroupID],
			Status:      string(membership.Status),
			Permissions: permissionsByGroup[membership.GroupID],
			CreatedAt:   membership.CreatedAt,
			UpdatedAt:   membership.UpdatedAt,
		})
	}

	return identitydomain.User{
		ID:                id,
		AuthIdentifier:    authIdentifier,
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
		AuthIdentifier: claims.AuthIdentifier,
		Email:          claims.Email,
		GivenName:      claims.GivenName,
		FamilyName:     claims.FamilyName,
	})
	if err != nil {
		return identitydomain.User{}, fmt.Errorf("create user: %w", err)
	}
	return mapUser(user.ID, user.AuthIdentifier, user.Email, user.GivenName, user.FamilyName, user.Status, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repo) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	if err := r.Queries.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func mapUser(id uuid.UUID, authIdentifier, email, givenName, familyName string, status sqlc.UserStatus, createdAt, updatedAt time.Time) identitydomain.User {
	return identitydomain.User{
		ID:                id,
		AuthIdentifier:    authIdentifier,
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
