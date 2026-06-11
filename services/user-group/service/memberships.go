package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/db/sqlc"
	"github.com/google/uuid"
)

type Membership struct {
	UserID    string    `json:"userId"`
	GroupID   string    `json:"groupId"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Memberships struct {
	db *db.PostgresDB
}

func NewMemberships(database *db.PostgresDB) *Memberships {
	return &Memberships{
		db: database,
	}
}

type UpsertMembershipParams struct {
	UserID, GroupID uuid.UUID
	Role, Status    string
}

func (s *Memberships) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]Membership, error) {
	rows, err := s.db.ListMembershipsByGroupID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list memberships by group: %w", err)
	}

	items := make([]Membership, len(rows))

	for i, row := range rows {
		items[i] = Membership{UserID: row.UserID.String(), GroupID: row.GroupID.String(), Role: row.Role, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	}

	return items, nil
}

func (s *Memberships) Upsert(ctx context.Context, params UpsertMembershipParams) (Membership, error) {
	params.Role = strings.ToUpper(strings.TrimSpace(params.Role))
	params.Status = strings.ToUpper(strings.TrimSpace(params.Status))

	if params.Role == "" || params.Status == "" || !isAllowedMembershipRole(params.Role) || !isAllowedMembershipStatus(params.Status) {
		return Membership{}, ErrInvalid
	}

	row, err := s.db.UpsertMembership(ctx, sqlc.UpsertMembershipParams{
		UserID:  params.UserID,
		GroupID: params.GroupID,
		Role:    params.Role,
		Status:  params.Status,
	})
	if err != nil {
		return Membership{}, fmt.Errorf("upsert membership: %w", err)
	}

	return Membership{UserID: row.UserID.String(), GroupID: row.GroupID.String(), Role: row.Role, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (s *Memberships) Delete(ctx context.Context, userID, groupID uuid.UUID) error {
	if _, err := s.db.GetMembershipByIDs(ctx, sqlc.GetMembershipByIDsParams{UserID: userID, GroupID: groupID}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("get membership: %w", err)
	}

	if err := s.db.DeleteMembership(ctx, sqlc.DeleteMembershipParams{UserID: userID, GroupID: groupID}); err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}

	return nil
}

func (s *Memberships) IsActiveMember(ctx context.Context, userID, groupID uuid.UUID) (bool, error) {
	result, err := s.db.IsActiveMember(ctx, sqlc.IsActiveMemberParams{UserID: userID, GroupID: groupID})
	if err != nil {
		return false, fmt.Errorf("check active membership: %w", err)
	}

	return result, nil
}
