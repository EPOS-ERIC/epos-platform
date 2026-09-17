// Package softwaresourcecodes provides PostgreSQL persistence for software source-code lifecycle data.
package softwaresourcecodes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	resourcedb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/resource"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo persists software source-code lifecycle operations.
type Repo struct {
	db *pgxpool.Pool
}

// New creates a software source-code repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{db: pool}
}

func (r *Repo) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value domain.SoftwareSourceCodeInput) (domain.SoftwareSourceCodeVersion, error) {
	var result domain.SoftwareSourceCodeVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		valid, err := resourcedb.CategoriesBelongToGroup(ctx, q, groupID, value.CategoryIDs)
		if err != nil {
			return fmt.Errorf("validate source code categories: %w", err)
		}
		if !valid {
			return domain.ErrSoftwareSourceCodeInvalid
		}

		sourceCode, err := q.CreateLifecycleSoftwareSourceCode(ctx, sqlc.CreateLifecycleSoftwareSourceCodeParams{
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create source code: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareSourceCodeVersion(ctx, ToCreateVersionParams(sourceCode.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create source code version: %w", err)
		}
		if err := PersistChildren(ctx, q, version.ID, value); err != nil {
			return err
		}
		result, err = sourceCodeResult(ctx, q, sourceCode.ID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *Repo) ReplaceDraft(ctx context.Context, sourceCodeID, versionID uuid.UUID, value domain.SoftwareSourceCodeDraftInput) (domain.SoftwareSourceCodeVersion, error) {
	var result domain.SoftwareSourceCodeVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if len(value.Metadata.CategoryIDs) > 0 {
			sourceCode, err := q.GetLifecycleSoftwareSourceCode(ctx, sourceCodeID)
			if err != nil {
				return fmt.Errorf("get source code for category validation: %w", err)
			}
			valid, err := resourcedb.CategoriesBelongToGroup(ctx, q, sourceCode.GroupID, value.Metadata.CategoryIDs)
			if err != nil {
				return fmt.Errorf("validate source code categories: %w", err)
			}
			if !valid {
				return domain.ErrSoftwareSourceCodeInvalid
			}
		}

		version, err := q.ReplaceLifecycleSoftwareSourceCodeDraft(ctx, ToReplaceDraftParams(sourceCodeID, versionID, value.Revision, value.Metadata))
		if err != nil {
			return fmt.Errorf("replace source code draft: %w", err)
		}
		if err := q.DeleteLifecycleSoftwareSourceCodeVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("delete source code draft children: %w", err)
		}
		if err := PersistChildren(ctx, q, version.ID, value.Metadata); err != nil {
			return err
		}
		result, err = sourceCodeResult(ctx, q, sourceCodeID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *Repo) Get(ctx context.Context, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	result, err := sourceCodeResult(ctx, sqlc.New(r.db), sourceCodeID, versionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SoftwareSourceCodeVersion{}, domain.ErrSoftwareSourceCodeNotFound
	}
	return result, err
}

func (r *Repo) List(ctx context.Context, sourceCodeID uuid.UUID) ([]domain.ResourceVersionSummary, error) {
	rows, err := sqlc.New(r.db).ListLifecycleSoftwareSourceCodeVersionSummaries(ctx, sourceCodeID)
	if err != nil {
		return nil, fmt.Errorf("list source code version summaries: %w", err)
	}

	result := make([]domain.ResourceVersionSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainSoftwareSourceCodeVersionSummary(row))
	}
	return result, nil
}

func (r *Repo) GetPublished(ctx context.Context, sourceCodeID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	version, err := sqlc.New(r.db).GetPublishedLifecycleSoftwareSourceCodeVersion(ctx, sourceCodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SoftwareSourceCodeVersion{}, domain.ErrSoftwareSourceCodeNotFound
	}
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, fmt.Errorf("get published source code version: %w", err)
	}
	return r.Get(ctx, sourceCodeID, version.ID)
}

func (r *Repo) GetActive(ctx context.Context, sourceCodeID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	version, err := sqlc.New(r.db).GetActiveLifecycleSoftwareSourceCodeVersion(ctx, sourceCodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SoftwareSourceCodeVersion{}, domain.ErrSoftwareSourceCodeNotFound
	}
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, fmt.Errorf("get active source code version: %w", err)
	}
	return r.Get(ctx, sourceCodeID, version.ID)
}

func (r *Repo) Submit(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.SubmitLifecycleSoftwareSourceCodeVersion(ctx, sqlc.SubmitLifecycleSoftwareSourceCodeVersionParams{
			ActorID: &actorID,
			ID:      versionID,
		})
	})
}

func (r *Repo) Withdraw(ctx context.Context, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.WithdrawLifecycleSoftwareSourceCodeVersion(ctx, versionID)
	})
}

func (r *Repo) Reject(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID, value domain.SoftwareSourceCodeRejectionInput) (domain.SoftwareSourceCodeVersion, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.RejectLifecycleSoftwareSourceCodeVersion(ctx, sqlc.RejectLifecycleSoftwareSourceCodeVersionParams{
			ActorID:       &actorID,
			RejectionNote: &value.Note,
			ID:            versionID,
		})
	})
}

func (r *Repo) Reopen(ctx context.Context, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.ReopenLifecycleSoftwareSourceCodeVersion(ctx, versionID)
	})
}

func (r *Repo) Publish(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		if err := q.ArchivePublishedLifecycleSoftwareSourceCodeVersion(ctx, sqlc.ArchivePublishedLifecycleSoftwareSourceCodeVersionParams{
			ActorID:              &actorID,
			SoftwareSourceCodeID: sourceCodeID,
		}); err != nil {
			return sqlc.SoftwareSourceCodeVersion{}, fmt.Errorf("archive published source code version: %w", err)
		}
		return q.PublishLifecycleSoftwareSourceCodeVersion(ctx, sqlc.PublishLifecycleSoftwareSourceCodeVersionParams{
			ActorID:   &actorID,
			VersionID: versionID,
		})
	})
}

func (r *Repo) Unpublish(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.UnpublishLifecycleSoftwareSourceCodeVersion(ctx, sqlc.UnpublishLifecycleSoftwareSourceCodeVersionParams{
			ActorID: &actorID,
			ID:      versionID,
		})
	})
}

func (r *Repo) transition(
	ctx context.Context,
	sourceCodeID, versionID uuid.UUID,
	fn func(*sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error),
) (domain.SoftwareSourceCodeVersion, error) {
	var result domain.SoftwareSourceCodeVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if _, err := q.EnsureLifecycleSoftwareSourceCodeVersion(ctx, sqlc.EnsureLifecycleSoftwareSourceCodeVersionParams{
			VersionID:            versionID,
			SoftwareSourceCodeID: sourceCodeID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrSoftwareSourceCodeNotFound
			}
			return fmt.Errorf("ensure source code version: %w", err)
		}
		version, err := fn(q)
		if err != nil {
			return lifecycleError(err)
		}
		result, err = sourceCodeResult(ctx, q, sourceCodeID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *Repo) Discard(ctx context.Context, sourceCodeID, versionID uuid.UUID) error {
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if _, err := q.EnsureLifecycleSoftwareSourceCodeVersion(ctx, sqlc.EnsureLifecycleSoftwareSourceCodeVersionParams{
			VersionID:            versionID,
			SoftwareSourceCodeID: sourceCodeID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrSoftwareSourceCodeNotFound
			}
			return fmt.Errorf("ensure source code version: %w", err)
		}
		id, err := q.DiscardLifecycleSoftwareSourceCodeVersion(ctx, versionID)
		if err != nil {
			return fmt.Errorf("discard source code version: %w", err)
		}
		return q.DeleteEmptyLifecycleSoftwareSourceCode(ctx, id)
	})
	return lifecycleError(err)
}

func (r *Repo) DraftFromPublication(ctx context.Context, actorID string, sourceCodeID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	return r.clone(ctx, actorID, sourceCodeID, sourceCodeID, sqlc.ResourceStatusPUBLISHED, uuid.Nil, true)
}

func (r *Repo) DraftFromLatestArchive(ctx context.Context, actorID string, sourceCodeID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	return r.clone(ctx, actorID, sourceCodeID, sourceCodeID, sqlc.ResourceStatusARCHIVED, uuid.Nil, true)
}

func (r *Repo) CopyPublished(ctx context.Context, actorID string, sourceSourceCodeID, targetGroupID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	var result domain.SoftwareSourceCodeVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		source, err := q.GetPublishedLifecycleSoftwareSourceCodeVersion(ctx, sourceSourceCodeID)
		if err != nil {
			return fmt.Errorf("get published source code version: %w", err)
		}
		sourceCode, err := q.CreateLifecycleSoftwareSourceCode(ctx, sqlc.CreateLifecycleSoftwareSourceCodeParams{
			GroupID:   targetGroupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create source code copy: %w", err)
		}
		version, err := q.CloneLifecycleSoftwareSourceCodeVersion(ctx, sqlc.CloneLifecycleSoftwareSourceCodeVersionParams{
			SourceVersionID:            source.ID,
			SourceStatus:               sqlc.ResourceStatusPUBLISHED,
			TargetSoftwareSourceCodeID: sourceCode.ID,
			ActorID:                    actorID,
			TargetGroupID:              targetGroupID,
			CopyCategories:             false,
		})
		if err != nil {
			return fmt.Errorf("clone published source code version: %w", err)
		}
		result, err = sourceCodeResult(ctx, q, sourceCode.ID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *Repo) clone(
	ctx context.Context,
	actorID string,
	sourceID, targetID uuid.UUID,
	status sqlc.ResourceStatus,
	groupID uuid.UUID,
	copyCategories bool,
) (domain.SoftwareSourceCodeVersion, error) {
	var versionID uuid.UUID
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		var source sqlc.SoftwareSourceCodeVersion
		var err error
		if status == sqlc.ResourceStatusPUBLISHED {
			source, err = q.GetPublishedLifecycleSoftwareSourceCodeVersion(ctx, sourceID)
		} else {
			source, err = q.GetLatestArchivedLifecycleSoftwareSourceCodeVersion(ctx, sourceID)
		}
		if err != nil {
			return fmt.Errorf("get source source code version: %w", err)
		}
		if groupID == uuid.Nil {
			logical, err := q.GetLifecycleSoftwareSourceCode(ctx, targetID)
			if err != nil {
				return fmt.Errorf("get target source code: %w", err)
			}
			groupID = logical.GroupID
		}
		version, err := q.CloneLifecycleSoftwareSourceCodeVersion(ctx, sqlc.CloneLifecycleSoftwareSourceCodeVersionParams{
			SourceVersionID:            source.ID,
			SourceStatus:               status,
			TargetSoftwareSourceCodeID: targetID,
			ActorID:                    actorID,
			TargetGroupID:              groupID,
			CopyCategories:             copyCategories,
		})
		if err != nil {
			return fmt.Errorf("clone source code version: %w", err)
		}
		versionID = version.ID
		return nil
	})
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, lifecycleError(err)
	}
	return r.Get(ctx, targetID, versionID)
}

func lifecycleError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSoftwareSourceCodeConflict
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23514":
			return fmt.Errorf("database validation: %w", domain.ErrSoftwareSourceCodeInvalid)
		case "23001", "23505", "23P01", "P0001":
			return fmt.Errorf("database invariant %q: %w", pgErr.ConstraintName, domain.ErrSoftwareSourceCodeConflict)
		default:
			if strings.HasPrefix(pgErr.Code, "23") {
				return fmt.Errorf("database integrity error %q: %w", pgErr.Code, domain.ErrSoftwareSourceCodeInternal)
			}
		}
	}
	return err
}
