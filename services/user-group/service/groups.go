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

type Group struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsSystem    bool      `json:"isSystem"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Groups struct {
	db *db.PostgresDB
}

func NewGroups(database *db.PostgresDB) *Groups {
	return &Groups{
		db: database,
	}
}

type CreateGroupParams struct {
	Slug        string
	Name        string
	Description string
	IsSystem    bool
}

type UpdateGroupParams struct {
	ID          uuid.UUID
	Slug        string
	Name        string
	Description string
	IsSystem    bool
}

type PatchGroupParams struct {
	Slug        *string
	Name        *string
	Description *string
	IsSystem    *bool
}

func (s *Groups) List(ctx context.Context) ([]Group, error) {
	rows, err := s.db.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	items := make([]Group, len(rows))

	for i, row := range rows {
		items[i] = toGroup(row.ID, row.Slug, row.Name, row.Description, row.IsSystem, row.CreatedAt, row.UpdatedAt)
	}

	return items, nil
}

func (s *Groups) GetByID(ctx context.Context, id uuid.UUID) (Group, error) {
	row, err := s.db.GetGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Group{}, ErrNotFound
		}
		return Group{}, fmt.Errorf("get group by id: %w", err)
	}

	return toGroup(row.ID, row.Slug, row.Name, row.Description, row.IsSystem, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Groups) Create(ctx context.Context, params CreateGroupParams) (Group, error) {
	params.Slug = strings.TrimSpace(params.Slug)
	params.Name = strings.TrimSpace(params.Name)

	if params.Slug == "" || params.Name == "" {
		return Group{}, ErrInvalid
	}

	row, err := s.db.CreateGroup(ctx, sqlc.CreateGroupParams{
		Slug:        params.Slug,
		Name:        params.Name,
		Description: nullString(params.Description),
		IsSystem:    params.IsSystem,
	})
	if err != nil {
		if isUniqueViolation(err, "groups_slug_key") {
			return Group{}, ErrConflict
		}
		return Group{}, fmt.Errorf("create group: %w", err)
	}

	return toGroup(row.ID, row.Slug, row.Name, row.Description, row.IsSystem, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Groups) Replace(ctx context.Context, params UpdateGroupParams) (Group, error) {
	params.Slug = strings.TrimSpace(params.Slug)
	params.Name = strings.TrimSpace(params.Name)

	if params.Slug == "" || params.Name == "" {
		return Group{}, ErrInvalid
	}

	row, err := s.db.UpdateGroup(ctx, sqlc.UpdateGroupParams{
		ID:          params.ID,
		Slug:        params.Slug,
		Name:        params.Name,
		Description: nullString(params.Description),
		IsSystem:    params.IsSystem,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Group{}, ErrNotFound
		}
		if isUniqueViolation(err, "groups_slug_key") {
			return Group{}, ErrConflict
		}
		return Group{}, fmt.Errorf("update group: %w", err)
	}

	return toGroup(row.ID, row.Slug, row.Name, row.Description, row.IsSystem, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Groups) Patch(ctx context.Context, id uuid.UUID, params PatchGroupParams) (Group, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return Group{}, err
	}

	if params.Slug != nil {
		trimmed := strings.TrimSpace(*params.Slug)
		if trimmed == "" {
			return Group{}, ErrInvalid
		}
		current.Slug = trimmed
	}
	if params.Name != nil {
		trimmed := strings.TrimSpace(*params.Name)
		if trimmed == "" {
			return Group{}, ErrInvalid
		}
		current.Name = trimmed
	}
	if params.Description != nil {
		current.Description = strings.TrimSpace(*params.Description)
	}
	if params.IsSystem != nil {
		current.IsSystem = *params.IsSystem
	}

	return s.Replace(ctx, UpdateGroupParams{
		ID:          id,
		Slug:        current.Slug,
		Name:        current.Name,
		Description: current.Description,
		IsSystem:    current.IsSystem,
	})
}

func (s *Groups) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}

	if err := s.db.DeleteGroup(ctx, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	return nil
}

func toGroup(id uuid.UUID, slug, name, description string, isSystem bool, createdAt, updatedAt time.Time) Group {
	return Group{
		ID:          id.String(),
		Slug:        slug,
		Name:        name,
		Description: description,
		IsSystem:    isSystem,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
