// Package groups provides PostgreSQL persistence for groups.
package groups

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(db)}
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

func mapGroup(row sqlc.Group) groups.Group {
	return groups.Group{
		ID:          row.ID,
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
