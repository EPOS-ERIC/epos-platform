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
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *Repo) WithdrawMembershipRequest(ctx context.Context, groupID uuid.UUID, userID string) error {
	rows, err := r.WithdrawGroupMembershipRequest(ctx, sqlc.WithdrawGroupMembershipRequestParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return fmt.Errorf("withdraw membership request: %w", err)
	}

	if rows == 0 {
		return groups.ErrNotFound
	}

	return nil
}

func (r *Repo) RevokeMembership(ctx context.Context, groupID uuid.UUID, userID string) error {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check group: %w", err)
	}

	_, err = r.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check user: %w", err)
	}

	rows, err := r.RevokeGroupMembership(ctx, sqlc.RevokeGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return fmt.Errorf("revoke membership: %w", err)
	}
	if rows == 0 {
		return groups.ErrNotFound
	}

	return nil
}

func (r *Repo) RequestRole(ctx context.Context, groupID uuid.UUID, userID, role string) (groups.RoleRequest, error) {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.RoleRequest{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.RoleRequest{}, fmt.Errorf("check group: %w", err)
	}

	_, err = r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.RoleRequest{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.RoleRequest{}, fmt.Errorf("check membership: %w", err)
	}

	request, err := r.CreateGroupRoleRequest(ctx, sqlc.CreateGroupRoleRequestParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: role,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.RoleRequest{}, groups.ErrConflict
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return groups.RoleRequest{}, groups.ErrConflict
	}
	if err != nil {
		return groups.RoleRequest{}, fmt.Errorf("create role request: %w", err)
	}

	return groups.RoleRequest{
		ID:          request.ID,
		Role:        request.RoleKey,
		Status:      string(request.Status),
		RequestedAt: request.RequestedAt,
	}, nil
}

// ApplyRoleRequestDecision accepts or rejects a pending role request and records the
// decision actor and time. Acceptance assigns the requested role atomically,
// then returns the target user's updated membership.
func (r *Repo) ApplyRoleRequestDecision(ctx context.Context, groupID, requestID uuid.UUID, status, actorID string) (groups.Membership, error) {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Membership{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("check group: %w", err)
	}

	var request sqlc.GroupRoleRequest
	decidedBy := actorID
	err = r.withTx(ctx, func(q *sqlc.Queries) error {
		request, err = q.ApplyRoleRequestDecision(ctx, sqlc.ApplyRoleRequestDecisionParams{
			Status:    sqlc.RoleRequestStatus(status),
			DecidedBy: &decidedBy,
			RequestID: requestID,
			GroupID:   groupID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return groups.ErrConflict
		}
		if err != nil {
			return fmt.Errorf("update role request: %w", err)
		}

		if status == "ACCEPTED" {
			if err := q.AssignGroupRoleFromRequest(ctx, sqlc.AssignGroupRoleFromRequestParams{
				AssignedBy: actorID,
				RequestID:  requestID,
				GroupID:    groupID,
			}); err != nil {
				return fmt.Errorf("assign requested role: %w", err)
			}
		}

		return nil
	})
	if errors.Is(err, groups.ErrConflict) {
		return groups.Membership{}, fmt.Errorf("role request conflict: %w", groups.ErrConflict)
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("decide role request transaction: %w", err)
	}

	membership, err := r.loadMembership(ctx, request.GroupID, request.UserID)
	if err != nil {
		return groups.Membership{}, fmt.Errorf("load decided role membership: %w", err)
	}

	return membership, nil
}

func (r *Repo) WithdrawRoleRequest(ctx context.Context, groupID, requestID uuid.UUID, userID string) error {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check group: %w", err)
	}

	rows, err := r.WithdrawGroupRoleRequest(ctx, sqlc.WithdrawGroupRoleRequestParams{
		RequestID: requestID,
		GroupID:   groupID,
		UserID:    userID,
	})
	if err != nil {
		return fmt.Errorf("withdraw role request: %w", err)
	}
	if rows == 0 {
		return groups.ErrNotFound
	}

	return nil
}

// DecideMembership changes a pending membership to ACTIVE or REJECTED.
// Rejection also marks its pending role requests as rejected in the same transaction.
// TODO: use a better name like ApplyMembershipDecision
func (r *Repo) DecideMembership(ctx context.Context, groupID uuid.UUID, userID, status, actorID string) (groups.Membership, error) {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Membership{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("check group: %w", err)
	}

	_, err = r.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Membership{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("check user: %w", err)
	}

	var membership sqlc.GroupMembership
	err = r.withTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.DecideGroupMembership(ctx, sqlc.DecideGroupMembershipParams{
			// TODO: maybe we should see if we can use a custom type for enums? global domain package that converts or something
			Status:  sqlc.MembershipStatus(status),
			GroupID: groupID,
			UserID:  userID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return groups.ErrConflict
		}
		if err != nil {
			return fmt.Errorf("update membership: %w", err)
		}

		membership = row

		if status == "REJECTED" {
			decidedBy := actorID
			err = q.RejectPendingGroupRoleRequests(ctx, sqlc.RejectPendingGroupRoleRequestsParams{
				DecidedBy: &decidedBy,
				GroupID:   groupID,
				UserID:    userID,
			})
			if err != nil {
				return fmt.Errorf("reject pending role requests: %w", err)
			}
		}

		return nil
	})
	if errors.Is(err, groups.ErrConflict) {
		return groups.Membership{}, err
	}
	if err != nil {
		return groups.Membership{}, fmt.Errorf("decide membership transaction: %w", err)
	}

	result, err := r.loadMembership(ctx, membership.GroupID, membership.UserID)
	if err != nil {
		return groups.Membership{}, fmt.Errorf("load decided membership: %w", err)
	}

	return result, nil
}

func (r *Repo) loadMembership(ctx context.Context, groupID uuid.UUID, userID string) (groups.Membership, error) {
	row, err := r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return groups.Membership{}, fmt.Errorf("get membership: %w", err)
	}

	roles, err := r.ListGroupMembershipRoles(ctx, groupID)
	if err != nil {
		return groups.Membership{}, fmt.Errorf("list membership roles: %w", err)
	}

	permissions, err := r.ListGroupMembershipPermissions(ctx, groupID)
	if err != nil {
		return groups.Membership{}, fmt.Errorf("list membership permissions: %w", err)
	}

	requests, err := r.ListPendingGroupRoleRequests(ctx, groupID)
	if err != nil {
		return groups.Membership{}, fmt.Errorf("list pending role requests: %w", err)
	}

	result := groups.Membership{
		User: groups.MembershipUser{
			ID:         row.User.ID,
			GivenName:  row.User.GivenName,
			FamilyName: row.User.FamilyName,
			Email:      row.User.Email,
		},
		Status:       string(row.GroupMembership.Status),
		Roles:        []string{},
		Permissions:  []string{},
		RoleRequests: []groups.RoleRequest{},
		CreatedAt:    row.GroupMembership.CreatedAt,
		UpdatedAt:    row.GroupMembership.UpdatedAt,
	}

	for _, role := range roles {
		if role.GroupMembershipRole.UserID == userID {
			result.Roles = append(result.Roles, role.GroupMembershipRole.RoleKey)
		}
	}

	for _, permission := range permissions {
		if permission.GroupPermissionGrant.UserID == userID {
			result.Permissions = append(result.Permissions, permission.GroupPermissionGrant.PermissionKey)
		}
	}

	for _, request := range requests {
		if request.GroupRoleRequest.UserID == userID {
			result.RoleRequests = append(result.RoleRequests, groups.RoleRequest{
				ID:          request.GroupRoleRequest.ID,
				Role:        request.GroupRoleRequest.RoleKey,
				Status:      string(request.GroupRoleRequest.Status),
				RequestedAt: request.GroupRoleRequest.RequestedAt,
			})
		}
	}

	return result, nil
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
