// Package users provides PostgreSQL persistence for Metadata Lifecycle users.
package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{
		Queries: sqlc.New(db),
		db:      db,
	}
}

func (r *Repo) ListUsers(ctx context.Context) ([]domain.UserSummary, error) {
	rows, err := r.Queries.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]domain.UserSummary, 0, len(rows))
	for _, row := range rows {
		roles, err := r.GetUserGlobalRoles(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("get global roles for %s: %w", row.ID, err)
		}

		permissions, err := r.GetUserGlobalPermissions(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("get global permissions for %s: %w", row.ID, err)
		}

		user, err := toDomainUserSummary(row, roles, permissions)
		if err != nil {
			return nil, fmt.Errorf("construct user summary %s: %w", row.ID, err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *Repo) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	user, err := r.Queries.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return r.loadUser(ctx, user.ID, user.Email, user.GivenName, user.FamilyName, user.Status, user.CreatedAt, user.UpdatedAt)
}

func (r *Repo) loadUser(
	ctx context.Context,
	id,
	email,
	givenName,
	familyName string,
	status sqlc.UserStatus,
	createdAt,
	updatedAt time.Time,
) (domain.User, error) {
	globalRoles, err := r.GetUserGlobalRoles(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user global roles: %w", err)
	}

	globalPermissions, err := r.GetUserGlobalPermissions(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user global permissions: %w", err)
	}

	memberships, err := r.GetUserMemberships(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user memberships: %w", err)
	}

	roles, err := r.GetUserMembershipRoles(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user membership roles: %w", err)
	}

	permissions, err := r.GetUserMembershipPermissions(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user membership permissions: %w", err)
	}

	roleRequests, err := r.ListPendingUserGroupRoleRequests(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user pending role requests: %w", err)
	}

	rolesByGroup := make(map[uuid.UUID][]domain.GroupRole)
	for _, role := range roles {
		value, err := domain.NewGroupRole(role.RoleKey)
		if err != nil {
			return domain.User{}, fmt.Errorf("construct group role: %w", err)
		}
		rolesByGroup[role.GroupID] = append(rolesByGroup[role.GroupID], value)
	}

	permissionsByGroup := make(map[uuid.UUID][]string)
	for _, permission := range permissions {
		permissionsByGroup[permission.GroupID] = append(permissionsByGroup[permission.GroupID], permission.PermissionKey)
	}

	roleRequestsByGroup := make(map[uuid.UUID][]domain.RoleRequest)
	for _, request := range roleRequests {
		value, err := toDomainRoleRequest(request.ID, request.RoleKey, request.Status, request.RequestedAt)
		if err != nil {
			return domain.User{}, fmt.Errorf("construct role request %s: %w", request.ID, err)
		}
		roleRequestsByGroup[request.GroupID] = append(roleRequestsByGroup[request.GroupID], value)
	}

	userMemberships := make([]domain.Membership, 0, len(memberships))
	for _, membership := range memberships {
		value, err := domain.NewUserMembership(domain.Membership{
			Group: domain.GroupSummary{
				ID:   membership.GroupID,
				Name: membership.GroupName,
			},
			Roles:        rolesByGroup[membership.GroupID],
			Status:       domain.MembershipStatus(membership.Status),
			Permissions:  permissionsByGroup[membership.GroupID],
			RoleRequests: roleRequestsByGroup[membership.GroupID],
			CreatedAt:    membership.CreatedAt,
			UpdatedAt:    membership.UpdatedAt,
		})
		if err != nil {
			return domain.User{}, fmt.Errorf("construct membership for %s: %w", membership.GroupID, err)
		}
		userMemberships = append(userMemberships, value)
	}

	return toDomainUser(id, email, givenName, familyName, status, createdAt, updatedAt, globalRoles, globalPermissions, userMemberships)
}

func (r *Repo) CreateUser(ctx context.Context, claims domain.UserClaims) (domain.User, error) {
	user, err := r.Queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:         claims.ID,
		Email:      claims.Email,
		GivenName:  claims.GivenName,
		FamilyName: claims.FamilyName,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return domain.User{}, domain.ErrUserConflict
		}
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return toDomainUser(
		user.ID,
		user.Email,
		user.GivenName,
		user.FamilyName,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
		[]string{},
		[]string{},
		[]domain.Membership{},
	)
}

func (r *Repo) GrantGlobalPermission(ctx context.Context, userID, permission, grantedBy string) error {
	_, err := r.GetAuthorizationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}

	scope, err := r.GetPermissionScope(ctx, permission)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidUser
	}
	if err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if scope != sqlc.PermissionScopeGLOBAL {
		return domain.ErrInvalidUser
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
		return domain.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}

	scope, err := r.GetPermissionScope(ctx, permission)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidUser
	}
	if err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if scope != sqlc.PermissionScopeGLOBAL {
		return domain.ErrInvalidUser
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

func (r *Repo) AssignGlobalRole(ctx context.Context, userID, assignedBy, role string) error {
	if role != "ADMIN" {
		return domain.ErrInvalidUser
	}

	if _, err := r.GetAuthorizationUser(ctx, userID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	} else if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}

	if r.db == nil {
		return fmt.Errorf("identity database pool is not configured")
	}

	err := dbtx.Run(ctx, r.db, r.Queries, func(queries *sqlc.Queries) error {
		if err := queries.LockGlobalAdminChanges(ctx); err != nil {
			return fmt.Errorf("lock global role changes: %w", err)
		}
		if err := queries.AssignGlobalAdmin(ctx, sqlc.AssignGlobalAdminParams{
			UserID:     userID,
			AssignedBy: &assignedBy,
		}); err != nil {
			return fmt.Errorf("assign global admin role: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("assign global role transaction: %w", err)
	}
	return nil
}

func (r *Repo) RemoveGlobalRole(ctx context.Context, userID, role string) error {
	if role != "ADMIN" {
		return domain.ErrInvalidUser
	}

	if _, err := r.GetAuthorizationUser(ctx, userID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	} else if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}

	if r.db == nil {
		return fmt.Errorf("identity database pool is not configured")
	}

	err := dbtx.Run(ctx, r.db, r.Queries, func(queries *sqlc.Queries) error {
		if err := queries.LockGlobalAdminChanges(ctx); err != nil {
			return fmt.Errorf("lock global role changes: %w", err)
		}
		rows, err := queries.RemoveGlobalAdmin(ctx, userID)
		if err != nil {
			return fmt.Errorf("remove global admin role: %w", err)
		}
		if rows > 0 {
			admins, err := queries.CountActiveGlobalAdmins(ctx)
			if err != nil {
				return fmt.Errorf("count active global admins: %w", err)
			}
			if admins == 0 {
				return domain.ErrUserConflict
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("remove global role transaction: %w", err)
	}
	return nil
}

func (r *Repo) BlockUser(ctx context.Context, actorID, userID, reason string) error {
	if r.db == nil {
		return fmt.Errorf("identity database pool is not configured")
	}

	err := dbtx.Run(ctx, r.db, r.Queries, func(queries *sqlc.Queries) error {
		if err := queries.LockGlobalAdminChanges(ctx); err != nil {
			return fmt.Errorf("lock global admin changes: %w", err)
		}

		_, err := queries.GetAuthorizationUser(ctx, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrUserNotFound
		}
		if err != nil {
			return fmt.Errorf("check target user: %w", err)
		}

		changed, err := queries.BlockUser(ctx, sqlc.BlockUserParams{
			Reason:    &reason,
			BlockedBy: &actorID,
			UserID:    userID,
		})
		if err != nil {
			return fmt.Errorf("block user: %w", err)
		}
		if changed == 0 {
			return nil
		}

		admins, err := queries.CountActiveGlobalAdmins(ctx)
		if err != nil {
			return fmt.Errorf("count active global admins: %w", err)
		}
		if admins == 0 {
			return domain.ErrUserConflict
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("block user transaction: %w", err)
	}

	return nil
}

func (r *Repo) UnblockUser(ctx context.Context, userID string) error {
	if r.db == nil {
		return fmt.Errorf("identity database pool is not configured")
	}

	err := dbtx.Run(ctx, r.db, r.Queries, func(queries *sqlc.Queries) error {
		if err := queries.LockGlobalAdminChanges(ctx); err != nil {
			return fmt.Errorf("lock global admin changes: %w", err)
		}

		_, err := queries.GetAuthorizationUser(ctx, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrUserNotFound
		}
		if err != nil {
			return fmt.Errorf("check target user: %w", err)
		}

		if _, err := queries.UnblockUser(ctx, userID); err != nil {
			return fmt.Errorf("unblock user: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("unblock user transaction: %w", err)
	}

	return nil
}
