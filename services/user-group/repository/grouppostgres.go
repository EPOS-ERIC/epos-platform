package repository

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

type GroupPostgresRepository struct {
	db *sql.DB
}

func NewGroupPostgresRepository(db *sql.DB) *GroupPostgresRepository {
	return &GroupPostgresRepository{db: db}
}

func (r *GroupPostgresRepository) List() []domain.Group {
	rows, err := r.db.Query(`SELECT id::text, slug, name, COALESCE(description,''), is_system, created_at, updated_at FROM groups ORDER BY created_at DESC`)
	if err != nil {
		return []domain.Group{}
	}
	defer rows.Close()
    groups := []domain.Group{}
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.Slug, &g.Name, &g.Description, &g.IsSystem, &g.CreatedAt, &g.UpdatedAt); err == nil {
			groups = append(groups, g)
		}
	}
	return groups
}

func (r *GroupPostgresRepository) GetByID(id string) (domain.Group, error) {
	return r.getOne(`SELECT id::text, slug, name, COALESCE(description,''), is_system, created_at, updated_at FROM groups WHERE id = $1`, id)
}

func (r *GroupPostgresRepository) Create(slug, name, description string, isSystem bool) (domain.Group, error) {
	return r.getOne(`INSERT INTO groups (slug, name, description, is_system) VALUES ($1,$2,$3,$4) RETURNING id::text, slug, name, COALESCE(description,''), is_system, created_at, updated_at`, slug, name, nullIfEmpty(description), isSystem)
}

func (r *GroupPostgresRepository) Replace(id, slug, name, description string, isSystem bool) (domain.Group, error) {
	res, err := r.db.Exec(`UPDATE groups SET slug=$2, name=$3, description=$4, is_system=$5, updated_at=now() WHERE id=$1`, id, slug, name, nullIfEmpty(description), isSystem)
	if err != nil {
		return domain.Group{}, mapDBError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.Group{}, ErrGroupNotFound
	}
	return r.GetByID(id)
}

func (r *GroupPostgresRepository) Patch(id string, slug, name, description *string, isSystem *bool) (domain.Group, error) {
	g, err := r.GetByID(id)
	if err != nil {
		return domain.Group{}, err
	}
	if slug != nil { g.Slug = *slug }
	if name != nil { g.Name = *name }
	if description != nil { g.Description = *description }
	if isSystem != nil { g.IsSystem = *isSystem }
	return r.Replace(id, g.Slug, g.Name, g.Description, g.IsSystem)
}

func (r *GroupPostgresRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func (r *GroupPostgresRepository) getOne(q string, args ...any) (domain.Group, error) {
	var g domain.Group
	err := r.db.QueryRow(q, args...).Scan(&g.ID, &g.Slug, &g.Name, &g.Description, &g.IsSystem, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Group{}, ErrGroupNotFound
		}
		return domain.Group{}, mapDBError(err)
	}
	return g, nil
}

func mapDBError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "groups_slug_key") {
		return ErrGroupSlugExists
	}
	if strings.Contains(msg, "users_auth_identifier_key") {
		return ErrUserAuthIDExists
	}
	return err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
