// Package groups provides PostgreSQL persistence for groups.
package groups

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/groups"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repo struct {
	*sqlc.Queries
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
