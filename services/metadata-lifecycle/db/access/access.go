// Package access provides PostgreSQL persistence for group access.
package access

import (
	"context"
	"errors"
	"fmt"

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
	return &Repo{Queries: sqlc.New(db), db: db}
}

func (r *Repo) ListMemberships(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMembership, error) {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAccessNotFound
	} else if err != nil {
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

	grantedPermissions, err := r.ListGroupMembershipGrantedPermissions(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group membership granted permissions: %w", err)
	}

	rolePermissions, err := r.ListGroupMembershipRolePermissions(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group membership role permissions: %w", err)
	}

	requests, err := r.ListPendingGroupRoleRequests(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list pending group role requests: %w", err)
	}

	rolesByUser := make(map[string][]domain.GroupRole)
	for _, role := range roles {
		value, err := domain.NewGroupRole(role.RoleKey)
		if err != nil {
			return nil, fmt.Errorf("construct group role: %w", err)
		}
		rolesByUser[role.UserID] = append(rolesByUser[role.UserID], value)
	}

	grantedPermissionsByUser := make(map[string][]domain.GroupPermission)
	for _, permission := range grantedPermissions {
		value, err := domain.NewGroupPermission(permission.PermissionKey)
		if err != nil {
			return nil, fmt.Errorf("construct granted permission: %w", err)
		}
		grantedPermissionsByUser[permission.UserID] = append(grantedPermissionsByUser[permission.UserID], value)
	}

	rolePermissionsByUser := make(map[string][]domain.GroupPermission)
	for _, permission := range rolePermissions {
		value, err := domain.NewGroupPermission(permission.PermissionKey)
		if err != nil {
			return nil, fmt.Errorf("construct role permission: %w", err)
		}
		rolePermissionsByUser[permission.UserID] = append(rolePermissionsByUser[permission.UserID], value)
	}

	requestsByUser := make(map[string][]domain.RoleRequest)
	for _, request := range requests {
		value, err := toDomainRoleRequest(request.ID, request.RoleKey, string(request.Status), request.RequestedAt)
		if err != nil {
			return nil, fmt.Errorf("construct role request %s: %w", request.ID, err)
		}
		requestsByUser[request.UserID] = append(requestsByUser[request.UserID], value)
	}

	result := make([]domain.GroupMembership, 0, len(rows))
	for _, row := range rows {
		membership, err := toDomainGroupMembership(
			domain.MembershipUser{
				ID:         row.User.ID,
				GivenName:  row.User.GivenName,
				FamilyName: row.User.FamilyName,
				Email:      row.User.Email,
			},
			row.GroupMembership,
			rolesByUser[row.GroupMembership.UserID],
			grantedPermissionsByUser[row.GroupMembership.UserID],
			rolePermissionsByUser[row.GroupMembership.UserID],
			requestsByUser[row.GroupMembership.UserID],
		)
		if err != nil {
			return nil, fmt.Errorf("construct membership for %s: %w", row.GroupMembership.UserID, err)
		}
		result = append(result, membership)
	}

	return result, nil
}

func (r *Repo) RequestMembership(ctx context.Context, groupID uuid.UUID, userID string, roles []domain.GroupRole) (domain.GroupMembership, error) {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return domain.GroupMembership{}, domain.ErrAccessNotFound
	} else if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("check group: %w", err)
	}

	user, err := r.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GroupMembership{}, domain.ErrAccessNotFound
	}
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("check user: %w", err)
	}

	var membership domain.GroupMembership
	err = dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
		row, err := q.RequestGroupMembership(ctx, sqlc.RequestGroupMembershipParams{
			UserID:  userID,
			GroupID: groupID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccessConflict
		}
		if err != nil {
			return fmt.Errorf("request membership: %w", err)
		}

		requests := make([]domain.RoleRequest, 0, len(roles))
		for _, role := range roles {
			request, err := q.CreatePendingGroupRoleRequest(ctx, sqlc.CreatePendingGroupRoleRequestParams{
				UserID:  userID,
				GroupID: groupID,
				RoleKey: string(role),
			})
			if err != nil {
				return fmt.Errorf("create role request: %w", err)
			}

			requestValue, err := toDomainRoleRequest(request.ID, request.RoleKey, string(request.Status), request.RequestedAt)
			if err != nil {
				return fmt.Errorf("construct role request %s: %w", request.ID, err)
			}
			requests = append(requests, requestValue)
		}

		membership, err = toDomainGroupMembership(
			domain.MembershipUser{
				ID:         user.ID,
				GivenName:  user.GivenName,
				FamilyName: user.FamilyName,
				Email:      user.Email,
			},
			row,
			nil,
			nil,
			nil,
			requests,
		)
		if err != nil {
			return fmt.Errorf("construct membership: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("request membership transaction: %w", err)
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
		return domain.ErrAccessNotFound
	}

	return nil
}

func (r *Repo) RevokeMembership(ctx context.Context, groupID uuid.UUID, userID string) error {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAccessNotFound
	} else if err != nil {
		return fmt.Errorf("check group: %w", err)
	}

	if _, err := r.GetUserByID(ctx, userID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAccessNotFound
	} else if err != nil {
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
		return domain.ErrAccessNotFound
	}

	return nil
}

func (r *Repo) RequestRole(ctx context.Context, groupID uuid.UUID, userID string, role domain.GroupRole) (domain.RoleRequest, error) {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return domain.RoleRequest{}, domain.ErrAccessNotFound
	} else if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("check group: %w", err)
	}

	if _, err := r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return domain.RoleRequest{}, domain.ErrAccessNotFound
	} else if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("check membership: %w", err)
	}

	request, err := r.CreateGroupRoleRequest(ctx, sqlc.CreateGroupRoleRequestParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: string(role),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RoleRequest{}, domain.ErrAccessConflict
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return domain.RoleRequest{}, domain.ErrAccessConflict
	}
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("create role request: %w", err)
	}

	result, err := toDomainRoleRequest(request.ID, request.RoleKey, string(request.Status), request.RequestedAt)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("construct role request %s: %w", request.ID, err)
	}

	return result, nil
}

// DecideRoleRequest accepts or rejects a pending role request and records the
// decision actor and time. Acceptance assigns the requested role atomically,
// then returns the target user's updated membership.
func (r *Repo) DecideRoleRequest(ctx context.Context, groupID, requestID uuid.UUID, decision domain.Decision, actorID string) (domain.RoleRequestDecision, error) {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return domain.RoleRequestDecision{}, domain.ErrAccessNotFound
	} else if err != nil {
		return domain.RoleRequestDecision{}, fmt.Errorf("check group: %w", err)
	}

	var request sqlc.GroupRoleRequest
	err := dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
		value, err := q.DecideRoleRequest(ctx, sqlc.DecideRoleRequestParams{
			Decision:  string(decision),
			DecidedBy: &actorID,
			RequestID: requestID,
			GroupID:   groupID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccessConflict
		}
		if err != nil {
			return fmt.Errorf("update role request: %w", err)
		}
		request = value

		if decision == domain.DecisionAccept {
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
	if err != nil {
		return domain.RoleRequestDecision{}, fmt.Errorf("decide role request transaction: %w", err)
	}

	membership, err := r.loadMembership(ctx, request.GroupID, request.UserID)
	if err != nil {
		return domain.RoleRequestDecision{}, fmt.Errorf("load decided role membership: %w", err)
	}

	roleRequest, err := toDomainRoleRequest(request.ID, request.RoleKey, string(request.Status), request.RequestedAt)
	if err != nil {
		return domain.RoleRequestDecision{}, fmt.Errorf("construct decided role request %s: %w", request.ID, err)
	}
	return domain.RoleRequestDecision{Request: roleRequest, Membership: membership}, nil
}

func (r *Repo) WithdrawRoleRequest(ctx context.Context, groupID, requestID uuid.UUID, userID string) error {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAccessNotFound
	} else if err != nil {
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
		return domain.ErrAccessNotFound
	}

	return nil
}

func (r *Repo) AssignRole(ctx context.Context, groupID uuid.UUID, userID string, role domain.GroupRole, actorID string) (bool, error) {
	exists, err := r.checkRoleTarget(ctx, groupID, userID, role)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	rows, err := r.AssignGroupRole(ctx, sqlc.AssignGroupRoleParams{
		UserID:     userID,
		GroupID:    groupID,
		RoleKey:    string(role),
		AssignedBy: actorID,
	})
	if err != nil {
		return false, fmt.Errorf("assign role: %w", err)
	}
	if rows == 0 {
		return false, domain.ErrAccessConflict
	}

	return true, nil
}

func (r *Repo) RemoveRole(ctx context.Context, groupID uuid.UUID, userID string, role domain.GroupRole) error {
	if _, err := r.checkRoleTarget(ctx, groupID, userID, role); err != nil {
		return err
	}

	if err := r.RemoveGroupRole(ctx, sqlc.RemoveGroupRoleParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: string(role),
	}); err != nil {
		return fmt.Errorf("remove role: %w", err)
	}

	return nil
}

func (r *Repo) GrantPermission(ctx context.Context, groupID uuid.UUID, userID string, permission domain.GroupPermission, actorID string) (bool, error) {
	if err := r.checkPermissionTarget(ctx, groupID, userID, permission); err != nil {
		return false, err
	}

	rows, err := r.GrantGroupPermission(ctx, sqlc.GrantGroupPermissionParams{
		UserID:        userID,
		GroupID:       groupID,
		PermissionKey: string(permission),
		GrantedBy:     actorID,
	})
	if err != nil {
		return false, fmt.Errorf("grant permission: %w", err)
	}

	return rows > 0, nil
}

func (r *Repo) RevokePermission(ctx context.Context, groupID uuid.UUID, userID string, permission domain.GroupPermission) error {
	if err := r.checkPermissionTarget(ctx, groupID, userID, permission); err != nil {
		return err
	}

	outcome, err := r.RevokeGroupPermission(ctx, sqlc.RevokeGroupPermissionParams{
		UserID:        userID,
		GroupID:       groupID,
		PermissionKey: string(permission),
	})
	if err != nil {
		return fmt.Errorf("revoke permission: %w", err)
	}

	switch outcome {
	case "GRANTED", "ABSENT":
		return nil
	case "ROLE":
		return domain.ErrAccessConflict
	default:
		return fmt.Errorf("revoke permission: unexpected outcome %q", outcome)
	}
}

func (r *Repo) checkPermissionTarget(ctx context.Context, groupID uuid.UUID, userID string, permission domain.GroupPermission) error {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAccessNotFound
	} else if err != nil {
		return fmt.Errorf("check group: %w", err)
	}

	if _, err := r.GetUserByID(ctx, userID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAccessNotFound
	} else if err != nil {
		return fmt.Errorf("check user: %w", err)
	}

	membership, err := r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAccessNotFound
	}
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if membership.GroupMembership.Status != sqlc.MembershipStatusACTIVE {
		return domain.ErrAccessConflict
	}

	scope, err := r.GetPermissionScope(ctx, string(permission))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidAccess
	}
	if err != nil {
		return fmt.Errorf("check permission: %w", err)
	}
	if scope != sqlc.PermissionScopeGROUP {
		return domain.ErrInvalidAccess
	}

	return nil
}

// checkRoleTarget validates the target membership and pending-request state.
// It reports whether the requested role is already assigned.
func (r *Repo) checkRoleTarget(ctx context.Context, groupID uuid.UUID, userID string, role domain.GroupRole) (bool, error) {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrAccessNotFound
	} else if err != nil {
		return false, fmt.Errorf("check group: %w", err)
	}

	if _, err := r.GetUserByID(ctx, userID); errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrAccessNotFound
	} else if err != nil {
		return false, fmt.Errorf("check user: %w", err)
	}

	membership, err := r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrAccessNotFound
	}
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	if membership.GroupMembership.Status != sqlc.MembershipStatusACTIVE {
		return false, domain.ErrAccessConflict
	}

	_, err = r.GetPendingGroupRoleRequest(ctx, sqlc.GetPendingGroupRoleRequestParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: string(role),
	})
	if err == nil {
		return false, domain.ErrAccessConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("check pending role request: %w", err)
	}

	exists, err := r.GroupMembershipRoleExists(ctx, sqlc.GroupMembershipRoleExistsParams{
		UserID:  userID,
		GroupID: groupID,
		RoleKey: string(role),
	})
	if err != nil {
		return false, fmt.Errorf("check group role: %w", err)
	}
	if exists {
		return true, nil
	}

	return false, nil
}

// DecideMembership changes a pending membership according to the requested decision.
// Rejection also marks its pending role requests as rejected in the same transaction.
func (r *Repo) DecideMembership(ctx context.Context, groupID uuid.UUID, userID string, decision domain.Decision, actorID string) (domain.GroupMembership, error) {
	if _, err := r.GetGroupByID(ctx, groupID); errors.Is(err, pgx.ErrNoRows) {
		return domain.GroupMembership{}, domain.ErrAccessNotFound
	} else if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("check group: %w", err)
	}

	if _, err := r.GetUserByID(ctx, userID); errors.Is(err, pgx.ErrNoRows) {
		return domain.GroupMembership{}, domain.ErrAccessNotFound
	} else if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("check user: %w", err)
	}

	var membership sqlc.GroupMembership
	err := dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
		row, err := q.DecideGroupMembership(ctx, sqlc.DecideGroupMembershipParams{
			Decision: string(decision),
			GroupID:  groupID,
			UserID:   userID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccessConflict
		}
		if err != nil {
			return fmt.Errorf("update membership: %w", err)
		}

		membership = row

		if decision == domain.DecisionReject {
			if err := q.RejectPendingGroupRoleRequests(ctx, sqlc.RejectPendingGroupRoleRequestsParams{
				DecidedBy: &actorID,
				GroupID:   groupID,
				UserID:    userID,
			}); err != nil {
				return fmt.Errorf("reject pending role requests: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide membership transaction: %w", err)
	}

	result, err := r.loadMembership(ctx, membership.GroupID, membership.UserID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("load decided membership: %w", err)
	}

	return result, nil
}

func (r *Repo) loadMembership(ctx context.Context, groupID uuid.UUID, userID string) (domain.GroupMembership, error) {
	row, err := r.GetGroupMembership(ctx, sqlc.GetGroupMembershipParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("get membership: %w", err)
	}

	roles, err := r.ListGroupMembershipRoles(ctx, groupID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("list membership roles: %w", err)
	}

	grantedPermissions, err := r.ListGroupMembershipGrantedPermissions(ctx, groupID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("list membership granted permissions: %w", err)
	}

	rolePermissions, err := r.ListGroupMembershipRolePermissions(ctx, groupID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("list membership role permissions: %w", err)
	}

	requests, err := r.ListPendingGroupRoleRequests(ctx, groupID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("list pending role requests: %w", err)
	}

	memberRoles := make([]domain.GroupRole, 0)
	for _, role := range roles {
		if role.UserID != userID {
			continue
		}
		value, err := domain.NewGroupRole(role.RoleKey)
		if err != nil {
			return domain.GroupMembership{}, fmt.Errorf("construct group role: %w", err)
		}
		memberRoles = append(memberRoles, value)
	}

	memberGrantedPermissions := make([]domain.GroupPermission, 0)
	for _, permission := range grantedPermissions {
		if permission.UserID == userID {
			value, err := domain.NewGroupPermission(permission.PermissionKey)
			if err != nil {
				return domain.GroupMembership{}, fmt.Errorf("construct granted permission: %w", err)
			}
			memberGrantedPermissions = append(memberGrantedPermissions, value)
		}
	}

	memberRolePermissions := make([]domain.GroupPermission, 0)
	for _, permission := range rolePermissions {
		if permission.UserID == userID {
			value, err := domain.NewGroupPermission(permission.PermissionKey)
			if err != nil {
				return domain.GroupMembership{}, fmt.Errorf("construct role permission: %w", err)
			}
			memberRolePermissions = append(memberRolePermissions, value)
		}
	}

	memberRequests := make([]domain.RoleRequest, 0)
	for _, request := range requests {
		if request.UserID != userID {
			continue
		}
		value, err := toDomainRoleRequest(request.ID, request.RoleKey, string(request.Status), request.RequestedAt)
		if err != nil {
			return domain.GroupMembership{}, fmt.Errorf("construct role request %s: %w", request.ID, err)
		}
		memberRequests = append(memberRequests, value)
	}

	membership, err := toDomainGroupMembership(
		domain.MembershipUser{
			ID:         row.User.ID,
			GivenName:  row.User.GivenName,
			FamilyName: row.User.FamilyName,
			Email:      row.User.Email,
		},
		row.GroupMembership,
		memberRoles,
		memberGrantedPermissions,
		memberRolePermissions,
		memberRequests,
	)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("construct membership: %w", err)
	}

	return membership, nil
}
