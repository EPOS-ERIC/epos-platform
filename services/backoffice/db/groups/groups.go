// Package groups provides PostgreSQL persistence for groups.
package groups

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/dbtx"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/groups"
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
	return &Repo{Queries: sqlc.New(db), db: db}
}

func (r *Repo) List(ctx context.Context) ([]groups.Group, error) {
	rows, err := r.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	result := make([]groups.Group, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapGroup(row))
	}

	return result, nil
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (groups.Group, error) {
	group, err := r.GetGroupByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Group{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Group{}, fmt.Errorf("get group by id: %w", err)
	}

	return mapGroup(group), nil
}

func (r *Repo) Create(ctx context.Context, group groups.Group) (groups.Group, error) {
	row, err := r.CreateGroup(ctx, sqlc.CreateGroupParams{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		SourceIri:   group.SourceIRI,
		Concept:     sqlc.TopLevelConcept(group.Concept),
		Label:       group.Label,
		Logo:        group.Logo,
		Page:        group.Page,
		Color:       group.Color,
		Position:    group.Position,
	})
	if err != nil {
		pgErr, ok := errors.AsType[*pgconn.PgError](err)
		if ok && pgErr.Code == "23505" {
			return groups.Group{}, groups.ErrConflict
		}

		return groups.Group{}, fmt.Errorf("insert group: %w", err)
	}

	return mapGroup(row), nil
}

func (r *Repo) Replace(ctx context.Context, id uuid.UUID, request groups.ReplaceRequest) (groups.Group, error) {
	row, err := r.ReplaceGroup(ctx, sqlc.ReplaceGroupParams{
		ID:          id,
		Name:        request.Name,
		Description: request.Description,
		Label:       request.Label,
		Logo:        request.Logo,
		Page:        request.Page,
		Color:       request.Color,
		Position:    request.Position,
		Concept:     sqlc.TopLevelConcept(request.Concept),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.Group{}, groups.ErrNotFound
	}
	if err != nil {
		pgErr, ok := errors.AsType[*pgconn.PgError](err)
		if ok && pgErr.Code == "23505" {
			return groups.Group{}, groups.ErrConflict
		}

		return groups.Group{}, fmt.Errorf("update group: %w", err)
	}

	return mapGroup(row), nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	outcome, err := r.DeleteGroup(ctx, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	switch outcome {
	case "DELETED":
		return nil
	case "NOT_FOUND":
		return groups.ErrNotFound
	case "CONFLICT":
		return groups.ErrConflict
	default:
		return fmt.Errorf("unexpected group deletion outcome %q", outcome)
	}
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
		rolesByUser[role.UserID] = append(rolesByUser[role.UserID], role.RoleKey)
	}

	permissionsByUser := make(map[string][]string)
	for _, permission := range permissions {
		permissionsByUser[permission.UserID] = append(permissionsByUser[permission.UserID], permission.PermissionKey)
	}

	requestsByUser := make(map[string][]groups.RoleRequest)
	for _, request := range requests {
		requestsByUser[request.UserID] = append(requestsByUser[request.UserID], groups.RoleRequest{
			ID:          request.ID,
			Role:        request.RoleKey,
			Status:      string(request.Status),
			RequestedAt: request.RequestedAt,
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
	err = dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
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
			request, err := q.CreatePendingGroupRoleRequest(ctx, sqlc.CreatePendingGroupRoleRequestParams{
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
		return groups.Membership{}, fmt.Errorf("request membership transaction: %w", err)
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
	err = dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
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

func (r *Repo) AssignRole(ctx context.Context, groupID uuid.UUID, userID, role, actorID string) error {
	exists, err := r.checkRoleTarget(ctx, groupID, userID, role)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	rows, err := r.AssignGroupRole(ctx, sqlc.AssignGroupRoleParams{
		UserID:     userID,
		GroupID:    groupID,
		RoleKey:    role,
		AssignedBy: actorID,
	})
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	if rows == 0 {
		return groups.ErrConflict
	}

	return nil
}

func (r *Repo) RemoveRole(ctx context.Context, groupID uuid.UUID, userID, role string) error {
	_, err := r.checkRoleTarget(ctx, groupID, userID, role)
	if err != nil {
		return err
	}

	if _, err := r.RemoveGroupRole(ctx, sqlc.RemoveGroupRoleParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: role,
	}); err != nil {
		return fmt.Errorf("remove role: %w", err)
	}

	return nil
}

func (r *Repo) GrantPermission(ctx context.Context, groupID uuid.UUID, userID, permission, actorID string) error {
	if err := r.checkPermissionTarget(ctx, groupID, userID, permission); err != nil {
		return err
	}

	if err := r.GrantGroupPermission(ctx, sqlc.GrantGroupPermissionParams{
		UserID:        userID,
		GroupID:       groupID,
		PermissionKey: permission,
		GrantedBy:     actorID,
	}); err != nil {
		return fmt.Errorf("grant permission: %w", err)
	}

	return nil
}

func (r *Repo) RevokePermission(ctx context.Context, groupID uuid.UUID, userID, permission string) error {
	if err := r.checkPermissionTarget(ctx, groupID, userID, permission); err != nil {
		return err
	}

	if err := r.RevokeGroupPermission(ctx, sqlc.RevokeGroupPermissionParams{
		UserID:        userID,
		GroupID:       groupID,
		PermissionKey: permission,
	}); err != nil {
		return fmt.Errorf("revoke permission: %w", err)
	}

	return nil
}

func (r *Repo) checkPermissionTarget(ctx context.Context, groupID uuid.UUID, userID, permission string) error {
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

	membership, err := r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return groups.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if membership.GroupMembership.Status != sqlc.MembershipStatusACTIVE {
		return groups.ErrConflict
	}

	scope, err := r.GetPermissionScope(ctx, permission)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return groups.ErrInvalid
		}
		return fmt.Errorf("check permission: %w", err)
	}
	if scope != sqlc.PermissionScopeGROUP {
		return groups.ErrInvalid
	}

	return nil
}

// checkRoleTarget validates the target membership and pending-request state.
// It reports whether the requested role is already assigned.
func (r *Repo) checkRoleTarget(ctx context.Context, groupID uuid.UUID, userID, role string) (bool, error) {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, groups.ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("check group: %w", err)
	}

	_, err = r.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, groups.ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("check user: %w", err)
	}

	membership, err := r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, groups.ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	if membership.GroupMembership.Status != sqlc.MembershipStatusACTIVE {
		return false, groups.ErrConflict
	}

	_, err = r.GetPendingGroupRoleRequest(ctx, sqlc.GetPendingGroupRoleRequestParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: role,
	})
	if err == nil {
		return false, groups.ErrConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("check pending role request: %w", err)
	}

	exists, err := r.GroupMembershipRoleExists(ctx, sqlc.GroupMembershipRoleExistsParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: role,
	})
	if err != nil {
		return false, fmt.Errorf("check group role: %w", err)
	}
	if exists {
		return true, nil
	}

	return false, nil
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
	err = dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
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
		return groups.Membership{}, fmt.Errorf("decide membership transaction: %w", err)
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
		if role.UserID == userID {
			result.Roles = append(result.Roles, role.RoleKey)
		}
	}

	for _, permission := range permissions {
		if permission.UserID == userID {
			result.Permissions = append(result.Permissions, permission.PermissionKey)
		}
	}

	for _, request := range requests {
		if request.UserID == userID {
			result.RoleRequests = append(result.RoleRequests, groups.RoleRequest{
				ID:          request.ID,
				Role:        request.RoleKey,
				Status:      string(request.Status),
				RequestedAt: request.RequestedAt,
			})
		}
	}

	return result, nil
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
