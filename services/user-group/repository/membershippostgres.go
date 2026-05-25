package repository

import (
	"database/sql"
	"errors"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

type MembershipPostgresRepository struct {
	db *sql.DB
}

func NewMembershipPostgresRepository(db *sql.DB) *MembershipPostgresRepository {
	return &MembershipPostgresRepository{db: db}
}

func (r *MembershipPostgresRepository) ListByGroupID(groupID string) []domain.Membership {
	rows, err := r.db.Query(`SELECT user_id::text, group_id::text, role, status, created_at, updated_at FROM group_memberships WHERE group_id = $1 ORDER BY created_at DESC`, groupID)
	if err != nil {
		return []domain.Membership{}
	}
	defer rows.Close()

	items := []domain.Membership{}
	for rows.Next() {
		var m domain.Membership
		if err := rows.Scan(&m.UserID, &m.GroupID, &m.Role, &m.Status, &m.CreatedAt, &m.UpdatedAt); err == nil {
			items = append(items, m)
		}
	}

	return items
}

func (r *MembershipPostgresRepository) Upsert(userID, groupID, role, status string) (domain.Membership, error) {
	if !isAllowedMembershipRole(role) {
		return domain.Membership{}, ErrMembershipInvalidRole
	}
	if !isAllowedMembershipStatus(status) {
		return domain.Membership{}, ErrMembershipInvalidStatus
	}

	_, err := r.db.Exec(`
		INSERT INTO group_memberships (user_id, group_id, role, status)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET role = EXCLUDED.role, status = EXCLUDED.status, updated_at = now()
	`, userID, groupID, role, status)
	if err != nil {
		return domain.Membership{}, mapDBError(err)
	}

	return r.getOne(userID, groupID)
}

func (r *MembershipPostgresRepository) Delete(userID, groupID string) error {
	res, err := r.db.Exec(`DELETE FROM group_memberships WHERE user_id = $1 AND group_id = $2`, userID, groupID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrMembershipNotFound
	}

	return nil
}

func (r *MembershipPostgresRepository) IsActiveMember(userID, groupID string) (bool, error) {
	var status string
	err := r.db.QueryRow(`SELECT status FROM group_memberships WHERE user_id = $1 AND group_id = $2`, userID, groupID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return status == "ACTIVE", nil
}

func (r *MembershipPostgresRepository) getOne(userID, groupID string) (domain.Membership, error) {
	var m domain.Membership
	err := r.db.QueryRow(`SELECT user_id::text, group_id::text, role, status, created_at, updated_at FROM group_memberships WHERE user_id = $1 AND group_id = $2`, userID, groupID).
		Scan(&m.UserID, &m.GroupID, &m.Role, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Membership{}, ErrMembershipNotFound
		}
		return domain.Membership{}, err
	}

	return m, nil
}
