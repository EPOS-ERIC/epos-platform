// Package contactrequests provides contact-request target reads from PostgreSQL.
package contactrequests

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	queries *sqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{queries: sqlc.New(db)}
}

func (r *Repo) Resource(ctx context.Context, resourceType domain.ResourceType, id uuid.UUID) (string, []string, error) {
	row, err := r.queries.GetContactRequestResource(ctx, sqlc.GetContactRequestResourceParams{
		ResourceType: string(resourceType),
		ResourceID:   id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, apperror.Default(apperror.CodeNotFound)
	}

	if err != nil {
		return "", nil, fmt.Errorf("get contact request resource: %w", err)
	}

	return row.Title, row.Emails, nil
}

func (r *Repo) Distribution(ctx context.Context, id uuid.UUID) (string, []string, error) {
	row, err := r.queries.GetContactRequestDistribution(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, apperror.Default(apperror.CodeNotFound)
	}

	if err != nil {
		return "", nil, fmt.Errorf("get contact request distribution: %w", err)
	}

	return strings.TrimSpace(row.DatasetTitle + " — " + row.DistributionTitle), row.Emails, nil
}
