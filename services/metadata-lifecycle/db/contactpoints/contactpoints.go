// Package contactpoints provides PostgreSQL persistence for contact points.
package contactpoints

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
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

// Sync replaces the remote contact points using the caller's transaction.
func Sync(ctx context.Context, q *sqlc.Queries, values []domain.SyncedContactPoint) error {
	nids := make([]int64, 0, len(values))
	// immediate email uniqueness rejects reassignments within a snapshot; defer the check if Drupal starts moving emails between NIDs.
	for _, value := range values {
		if err := q.UpsertSyncedContactPoint(ctx, sqlc.UpsertSyncedContactPointParams{
			RemoteNid: &value.NID,
			Email:     value.Input.Email,
		}); err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperror.New(apperror.CodeConflict, fmt.Sprintf("Email conflicts with another contact point for Drupal person %d.", value.NID))
			}

			return fmt.Errorf("upsert contact point %d: %w", value.NID, err)
		}

		nids = append(nids, value.NID)
	}
	if err := q.DeleteAbsentSyncedContactPoints(ctx, nids); err != nil {
		return fmt.Errorf("delete absent contact points: %w", err)
	}

	return nil
}

func (r *Repo) List(ctx context.Context) ([]domain.ContactPoint, error) {
	rows, err := r.ListContactPoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contact points: %w", err)
	}

	result := make([]domain.ContactPoint, 0, len(rows))
	for _, row := range rows {
		contactPoint, err := toDomainContactPoint(row)
		if err != nil {
			return nil, fmt.Errorf("construct contact point %s: %w", row.ID, err)
		}
		result = append(result, contactPoint)
	}

	return result, nil
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (domain.ContactPoint, error) {
	row, err := r.GetContactPoint(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ContactPoint{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("get contact point: %w", err)
	}

	contactPoint, err := toDomainContactPoint(row)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("construct contact point %s: %w", row.ID, err)
	}

	return contactPoint, nil
}

func (r *Repo) Create(ctx context.Context, input domain.ContactPointInput) (domain.ContactPoint, error) {
	id := uuid.New()
	row, err := r.CreateContactPoint(ctx, toCreateContactPointParams(id, input))
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return domain.ContactPoint{}, apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("insert contact point: %w", err))
		}

		return domain.ContactPoint{}, fmt.Errorf("insert contact point: %w", err)
	}

	contactPoint, err := toDomainContactPoint(row)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("construct contact point %s: %w", row.ID, err)
	}

	return contactPoint, nil
}

func (r *Repo) Replace(ctx context.Context, id uuid.UUID, input domain.ContactPointInput) (domain.ContactPoint, error) {
	row, err := r.ReplaceContactPoint(ctx, toReplaceContactPointParams(id, input))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ContactPoint{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		// TODO: maybe we can create an helper to do this check and reuse in the various db packages?
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return domain.ContactPoint{}, apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("update contact point: %w", err))
		}

		return domain.ContactPoint{}, fmt.Errorf("update contact point: %w", err)
	}

	contactPoint, err := toDomainContactPoint(row)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("construct contact point %s: %w", row.ID, err)
	}

	return contactPoint, nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.DeleteContactPoint(ctx, id)
	if err != nil {
		return fmt.Errorf("delete contact point: %w", err)
	}
	if deleted == 0 {
		return apperror.Default(apperror.CodeNotFound)
	}

	return nil
}
