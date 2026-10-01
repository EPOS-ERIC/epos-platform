// Package groups provides PostgreSQL persistence for groups.
package groups

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

func (r *Repo) List(ctx context.Context) ([]domain.Group, error) {
	rows, err := r.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	result := make([]domain.Group, 0, len(rows))
	for _, row := range rows {
		group, err := toDomainGroup(row)
		if err != nil {
			return nil, fmt.Errorf("construct group %s: %w", row.ID, err)
		}
		result = append(result, group)
	}

	return result, nil
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (domain.Group, error) {
	group, err := r.GetGroupByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Group{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.Group{}, fmt.Errorf("get group by id: %w", err)
	}

	result, err := toDomainGroup(group)
	if err != nil {
		return domain.Group{}, fmt.Errorf("construct group %s: %w", group.ID, err)
	}

	return result, nil
}

func (r *Repo) Create(ctx context.Context, input domain.GroupInput) (domain.Group, error) {
	id := uuid.New()
	row, err := r.CreateGroup(ctx, toCreateGroupParams(id, input))
	if err != nil {
		pgErr, ok := errors.AsType[*pgconn.PgError](err)
		if ok && pgErr.Code == "23505" {
			return domain.Group{}, apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("insert group: %w", err))
		}

		return domain.Group{}, fmt.Errorf("insert group: %w", err)
	}

	group, err := toDomainGroup(row)
	if err != nil {
		return domain.Group{}, fmt.Errorf("construct group %s: %w", row.ID, err)
	}

	return group, nil
}

func (r *Repo) Replace(ctx context.Context, id uuid.UUID, input domain.GroupInput) (domain.Group, error) {
	row, err := r.ReplaceGroup(ctx, toReplaceGroupParams(id, input))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Group{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		pgErr, ok := errors.AsType[*pgconn.PgError](err)
		if ok && pgErr.Code == "23505" {
			return domain.Group{}, apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("update group: %w", err))
		}

		return domain.Group{}, fmt.Errorf("update group: %w", err)
	}

	group, err := toDomainGroup(row)
	if err != nil {
		return domain.Group{}, fmt.Errorf("construct group %s: %w", row.ID, err)
	}

	return group, nil
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
		return apperror.Default(apperror.CodeNotFound)
	case "CONFLICT":
		return apperror.Default(apperror.CodeConflict)
	default:
		return fmt.Errorf("unexpected group deletion outcome %q", outcome)
	}
}
