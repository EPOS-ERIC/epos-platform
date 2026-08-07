package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var membershipsLog = logging.Get("memberships-service")

type Membership struct {
	UserID    string           `json:"userId"`
	GroupID   string           `json:"groupId"`
	Role      MembershipRole   `json:"role"`
	Status    MembershipStatus `json:"status"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type Memberships struct {
	db MembershipRepository
}

func NewMemberships(database MembershipRepository) *Memberships {
	return &Memberships{
		db: database,
	}
}

type UpsertMembershipParams struct {
	UserID, GroupID uuid.UUID
	Role            MembershipRole
	Status          MembershipStatus
}

func (s *Memberships) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]Membership, error) {
	if _, err := s.db.GetGroupByID(ctx, groupID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		membershipsLog.ErrorContext(ctx, "get group failed", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("get group by id: %w", err)
	}

	rows, err := s.db.ListMembershipsByGroupID(ctx, groupID)
	if err != nil {
		membershipsLog.ErrorContext(ctx, "list memberships failed", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("list memberships by group: %w", err)
	}

	items := make([]Membership, len(rows))

	for i, row := range rows {
		items[i] = Membership{UserID: row.UserID.String(), GroupID: row.GroupID.String(), Role: NormalizeMembershipRole(string(row.Role)), Status: NormalizeMembershipStatus(string(row.Status)), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	}

	return items, nil
}

func (s *Memberships) Upsert(ctx context.Context, params UpsertMembershipParams) (Membership, error) {
	params.Role = NormalizeMembershipRole(string(params.Role))
	params.Status = NormalizeMembershipStatus(string(params.Status))

	if params.Role == "" || params.Status == "" || !params.Role.Valid() || !params.Status.Valid() {
		return Membership{}, ErrInvalid
	}

	if _, err := s.db.GetUserByID(ctx, params.UserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Membership{}, ErrNotFound
		}
		membershipsLog.ErrorContext(ctx, "get user failed", "user_id", params.UserID, "error", err)
		return Membership{}, fmt.Errorf("get user by id: %w", err)
	}

	if _, err := s.db.GetGroupByID(ctx, params.GroupID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Membership{}, ErrNotFound
		}
		membershipsLog.ErrorContext(ctx, "get group failed", "group_id", params.GroupID, "error", err)
		return Membership{}, fmt.Errorf("get group by id: %w", err)
	}

	row, err := s.db.UpsertMembership(ctx, sqlc.UpsertMembershipParams{
		UserID:  params.UserID,
		GroupID: params.GroupID,
		Role:    sqlc.MembershipRole(params.Role),
		Status:  sqlc.MembershipStatus(params.Status),
	})
	if err != nil {
		membershipsLog.ErrorContext(ctx, "upsert membership failed", "group_id", params.GroupID, "user_id", params.UserID, "error", err)
		return Membership{}, fmt.Errorf("upsert membership: %w", err)
	}

	return Membership{UserID: row.UserID.String(), GroupID: row.GroupID.String(), Role: NormalizeMembershipRole(string(row.Role)), Status: NormalizeMembershipStatus(string(row.Status)), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (s *Memberships) Delete(ctx context.Context, userID, groupID uuid.UUID) error {
	if _, err := s.db.GetMembershipByIDs(ctx, sqlc.GetMembershipByIDsParams{UserID: userID, GroupID: groupID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		membershipsLog.ErrorContext(ctx, "get membership failed", "group_id", groupID, "user_id", userID, "error", err)
		return fmt.Errorf("get membership: %w", err)
	}

	if err := s.db.DeleteMembership(ctx, sqlc.DeleteMembershipParams{UserID: userID, GroupID: groupID}); err != nil {
		membershipsLog.ErrorContext(ctx, "delete membership failed", "group_id", groupID, "user_id", userID, "error", err)
		return fmt.Errorf("delete membership: %w", err)
	}

	return nil
}

func (s *Memberships) IsActiveMember(ctx context.Context, userID, groupID uuid.UUID) (bool, error) {
	result, err := s.db.IsActiveMember(ctx, sqlc.IsActiveMemberParams{UserID: userID, GroupID: groupID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		membershipsLog.ErrorContext(ctx, "check active membership failed", "group_id", groupID, "user_id", userID, "error", err)
		return false, fmt.Errorf("check active membership: %w", err)
	}

	return result, nil
}
