// Package groups provides PostgreSQL persistence for groups.
package groups

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/groups"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var log = logging.Get("groups-db")

type Repo struct {
	*sqlc.Queries
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(db), db: db}
}

func (r *Repo) List(ctx context.Context) ([]groups.Group, error) {
	rows, err := r.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	result := make([]groups.Group, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapGroup(row.Group))
	}

	return result, nil
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (groups.Group, error) {
	row, err := r.GetGroupByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Group{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Group{}, fmt.Errorf("get group by id: %w", err)
	}

	return mapGroup(row.Group), nil
}

func (r *Repo) ListMemberships(ctx context.Context, groupID uuid.UUID) ([]groups.Membership, error) {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, groups.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check group: %w", err)
	}

	rows, err := r.ListGroupMemberships(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group memberships: %w", err)
	}

	roles, err := r.ListGroupMembershipRoles(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group membership roles: %w", err)
	}

	permissions, err := r.ListGroupMembershipPermissions(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group membership permissions: %w", err)
	}

	requests, err := r.ListPendingGroupRoleRequests(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list pending group role requests: %w", err)
	}

	rolesByUser := make(map[string][]string)
	for _, role := range roles {
		rolesByUser[role.GroupMembershipRole.UserID] = append(rolesByUser[role.GroupMembershipRole.UserID], role.GroupMembershipRole.RoleKey)
	}

	permissionsByUser := make(map[string][]string)
	for _, permission := range permissions {
		permissionsByUser[permission.GroupPermissionGrant.UserID] = append(permissionsByUser[permission.GroupPermissionGrant.UserID], permission.GroupPermissionGrant.PermissionKey)
	}

	requestsByUser := make(map[string][]groups.RoleRequest)
	for _, request := range requests {
		requestsByUser[request.GroupRoleRequest.UserID] = append(requestsByUser[request.GroupRoleRequest.UserID], groups.RoleRequest{
			ID:          request.GroupRoleRequest.ID,
			Role:        request.GroupRoleRequest.RoleKey,
			Status:      string(request.GroupRoleRequest.Status),
			RequestedAt: request.GroupRoleRequest.RequestedAt,
		})
	}

	result := make([]groups.Membership, 0, len(rows))
	for _, row := range rows {
		result = append(result, groups.Membership{
			User: groups.MembershipUser{
				ID:         row.User.ID,
				GivenName:  row.User.GivenName,
				FamilyName: row.User.FamilyName,
				Email:      row.User.Email,
			},
			Status:       string(row.GroupMembership.Status),
			Roles:        append([]string{}, rolesByUser[row.GroupMembership.UserID]...),
			Permissions:  append([]string{}, permissionsByUser[row.GroupMembership.UserID]...),
			RoleRequests: append([]groups.RoleRequest{}, requestsByUser[row.GroupMembership.UserID]...),
			CreatedAt:    row.GroupMembership.CreatedAt,
			UpdatedAt:    row.GroupMembership.UpdatedAt,
		})
	}

	return result, nil
}

func (r *Repo) RequestMembership(ctx context.Context, groupID uuid.UUID, userID string, roles []string) (groups.Membership, error) {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Membership{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("check group: %w", err)
	}

	user, err := r.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Membership{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("check user: %w", err)
	}

	var membership groups.Membership
	err = r.withTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.RequestGroupMembership(ctx, sqlc.RequestGroupMembershipParams{
			UserID:  userID,
			GroupID: groupID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return groups.ErrConflict
		}
		if err != nil {
			return fmt.Errorf("request membership: %w", err)
		}

		requests := make([]groups.RoleRequest, 0, len(roles))
		for _, role := range roles {
			request, err := q.CreateGroupRoleRequest(ctx, sqlc.CreateGroupRoleRequestParams{
				UserID:  userID,
				GroupID: groupID,
				RoleKey: role,
			})
			if err != nil {
				return fmt.Errorf("create role request: %w", err)
			}
			requests = append(requests, groups.RoleRequest{
				ID:          request.ID,
				Role:        request.RoleKey,
				Status:      string(request.Status),
				RequestedAt: request.RequestedAt,
			})
		}

		membership = groups.Membership{
			User: groups.MembershipUser{
				ID:         user.ID,
				GivenName:  user.GivenName,
				FamilyName: user.FamilyName,
				Email:      user.Email,
			},
			Status:       string(row.Status),
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
			RoleRequests: requests,
			Roles:        []string{},
			Permissions:  []string{},
		}

		return nil
	})
	if errors.Is(err, groups.ErrConflict) {
		return groups.Membership{}, err
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("request membership transaction: %w", err)
	}

	return membership, nil
}

func (r *Repo) withTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.ErrorContext(ctx, "rollback transaction failed", "error", err)
		}
	}()

	if err := fn(sqlc.New(tx)); err != nil {
		return fmt.Errorf("execute transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func mapGroup(row sqlc.Group) groups.Group {
	return groups.Group{
		ID:          row.ID,
		SourceIRI:   row.SourceIri,
		Name:        row.Name,
		Description: row.Description,
		Label:       row.Label,
		Logo:        row.Logo,
		Page:        row.Page,
		Color:       row.Color,
		Position:    row.Position,
		Concept:     string(row.Concept),
	}
}
