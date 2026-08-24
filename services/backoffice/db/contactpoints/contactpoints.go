// Package contactpoints provides PostgreSQL persistence for contact points.
package contactpoints

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(db)}
}

func (r *Repo) List(ctx context.Context) ([]contactpoints.ContactPoint, error) {
	rows, err := r.ListContactPoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contact points: %w", err)
	}

	result := make([]contactpoints.ContactPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapContactPoint(row))
	}

	return result, nil
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (contactpoints.ContactPoint, error) {
	row, err := r.GetContactPoint(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return contactpoints.ContactPoint{}, contactpoints.ErrNotFound
	}
	if err != nil {
		return contactpoints.ContactPoint{}, fmt.Errorf("get contact point: %w", err)
	}

	return mapContactPoint(row), nil
}

func mapContactPoint(row sqlc.ContactPoint) contactpoints.ContactPoint {
	return contactpoints.ContactPoint{
		ID:        row.ID,
		SourceIRI: row.SourceIri,
		Email:     row.Email,
		Role:      row.Role,
	}
}
