// Package softwareapplications provides PostgreSQL persistence for software application lifecycle data.
package softwareapplications

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	resourcedb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/resource"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo persists software application lifecycle operations.
type Repo struct {
	db *pgxpool.Pool
}

// New creates a software application repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{db: pool}
}

func (r *Repo) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value domain.SoftwareApplicationInput) (domain.SoftwareApplicationVersion, error) {
	var result domain.SoftwareApplicationVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		invalidIndex, err := resourcedb.CategoriesBelongToGroup(ctx, q, groupID, value.CategoryIDs)
		if err != nil {
			return fmt.Errorf("validate application categories: %w", err)
		}
		if invalidIndex >= 0 {
			return apperror.Field(fmt.Sprintf("metadata.categories[%d]", invalidIndex), "Category must belong to this group and appear only once.", nil)
		}

		application, err := q.CreateLifecycleSoftwareApplication(ctx, sqlc.CreateLifecycleSoftwareApplicationParams{
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create application: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareApplicationVersion(ctx, ToCreateVersionParams(application.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create application version: %w", err)
		}
		if err := PersistChildren(ctx, q, version.ID, value); err != nil {
			return err
		}
		result, err = applicationResult(ctx, q, application.ID, version.ID)
		return err
	})
	return result, resourcedb.LifecycleError(err)
}

func (r *Repo) ReplaceDraft(ctx context.Context, applicationID, versionID uuid.UUID, value domain.SoftwareApplicationDraftInput) (domain.SoftwareApplicationVersion, error) {
	var result domain.SoftwareApplicationVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if len(value.Metadata.CategoryIDs) > 0 {
			application, err := q.GetLifecycleSoftwareApplication(ctx, applicationID)
			if err != nil {
				return fmt.Errorf("get application for category validation: %w", err)
			}
			invalidIndex, err := resourcedb.CategoriesBelongToGroup(ctx, q, application.GroupID, value.Metadata.CategoryIDs)
			if err != nil {
				return fmt.Errorf("validate application categories: %w", err)
			}
			if invalidIndex >= 0 {
				return apperror.Field(fmt.Sprintf("metadata.categories[%d]", invalidIndex), "Category must belong to this group and appear only once.", nil)
			}
		}

		version, err := q.ReplaceLifecycleSoftwareApplicationDraft(ctx, ToReplaceDraftParams(applicationID, versionID, value.Revision, value.Metadata))
		if err != nil {
			return fmt.Errorf("replace application draft: %w", err)
		}
		if err := q.DeleteLifecycleSoftwareApplicationVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("delete application draft children: %w", err)
		}
		if err := PersistChildren(ctx, q, version.ID, value.Metadata); err != nil {
			return err
		}
		result, err = applicationResult(ctx, q, applicationID, version.ID)
		return err
	})
	return result, resourcedb.LifecycleError(err)
}

func (r *Repo) Get(ctx context.Context, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	result, err := applicationResult(ctx, sqlc.New(r.db), applicationID, versionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SoftwareApplicationVersion{}, apperror.Default(apperror.CodeNotFound)
	}
	return result, err
}

func (r *Repo) List(ctx context.Context, applicationID uuid.UUID) ([]domain.ResourceVersionSummary, error) {
	rows, err := sqlc.New(r.db).ListLifecycleSoftwareApplicationVersionSummaries(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list application version summaries: %w", err)
	}

	result := make([]domain.ResourceVersionSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainSoftwareApplicationVersionSummary(row))
	}
	return result, nil
}

func (r *Repo) GetPublished(ctx context.Context, applicationID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	version, err := sqlc.New(r.db).GetPublishedLifecycleSoftwareApplicationVersion(ctx, applicationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SoftwareApplicationVersion{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.SoftwareApplicationVersion{}, fmt.Errorf("get published application version: %w", err)
	}
	return r.Get(ctx, applicationID, version.ID)
}

func (r *Repo) GetActive(ctx context.Context, applicationID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	version, err := sqlc.New(r.db).GetActiveLifecycleSoftwareApplicationVersion(ctx, applicationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SoftwareApplicationVersion{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.SoftwareApplicationVersion{}, fmt.Errorf("get active application version: %w", err)
	}
	return r.Get(ctx, applicationID, version.ID)
}

func (r *Repo) Submit(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	return r.transition(ctx, applicationID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.SubmitLifecycleSoftwareApplicationVersion(ctx, sqlc.SubmitLifecycleSoftwareApplicationVersionParams{
			ActorID: &actorID,
			ID:      versionID,
		})
	})
}

func (r *Repo) Withdraw(ctx context.Context, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	return r.transition(ctx, applicationID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.WithdrawLifecycleSoftwareApplicationVersion(ctx, versionID)
	})
}

func (r *Repo) Reject(ctx context.Context, actorID string, applicationID, versionID uuid.UUID, value domain.SoftwareApplicationRejectionInput) (domain.SoftwareApplicationVersion, error) {
	return r.transition(ctx, applicationID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.RejectLifecycleSoftwareApplicationVersion(ctx, sqlc.RejectLifecycleSoftwareApplicationVersionParams{
			ActorID:       &actorID,
			RejectionNote: &value.Note,
			ID:            versionID,
		})
	})
}

func (r *Repo) Reopen(ctx context.Context, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	return r.transition(ctx, applicationID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.ReopenLifecycleSoftwareApplicationVersion(ctx, versionID)
	})
}

func (r *Repo) Publish(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	return r.transition(ctx, applicationID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		if err := q.ArchivePublishedLifecycleSoftwareApplicationVersion(ctx, sqlc.ArchivePublishedLifecycleSoftwareApplicationVersionParams{
			ActorID:               &actorID,
			SoftwareApplicationID: applicationID,
		}); err != nil {
			return sqlc.SoftwareApplicationVersion{}, fmt.Errorf("archive published application version: %w", err)
		}
		return q.PublishLifecycleSoftwareApplicationVersion(ctx, sqlc.PublishLifecycleSoftwareApplicationVersionParams{
			ActorID:   &actorID,
			VersionID: versionID,
		})
	})
}

func (r *Repo) Unpublish(ctx context.Context, actorID string, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	return r.transition(ctx, applicationID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.UnpublishLifecycleSoftwareApplicationVersion(ctx, sqlc.UnpublishLifecycleSoftwareApplicationVersionParams{
			ActorID: &actorID,
			ID:      versionID,
		})
	})
}

func (r *Repo) transition(
	ctx context.Context,
	applicationID, versionID uuid.UUID,
	fn func(*sqlc.Queries) (sqlc.SoftwareApplicationVersion, error),
) (domain.SoftwareApplicationVersion, error) {
	var result domain.SoftwareApplicationVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if _, err := q.EnsureLifecycleSoftwareApplicationVersion(ctx, sqlc.EnsureLifecycleSoftwareApplicationVersionParams{
			VersionID:             versionID,
			SoftwareApplicationID: applicationID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperror.Default(apperror.CodeNotFound)
			}
			return fmt.Errorf("ensure application version: %w", err)
		}
		version, err := fn(q)
		if err != nil {
			return resourcedb.LifecycleError(err)
		}
		result, err = applicationResult(ctx, q, applicationID, version.ID)
		return err
	})
	return result, resourcedb.LifecycleError(err)
}

func (r *Repo) Discard(ctx context.Context, applicationID, versionID uuid.UUID) error {
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if _, err := q.EnsureLifecycleSoftwareApplicationVersion(ctx, sqlc.EnsureLifecycleSoftwareApplicationVersionParams{
			VersionID:             versionID,
			SoftwareApplicationID: applicationID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperror.Default(apperror.CodeNotFound)
			}
			return fmt.Errorf("ensure application version: %w", err)
		}
		id, err := q.DiscardLifecycleSoftwareApplicationVersion(ctx, versionID)
		if err != nil {
			return fmt.Errorf("discard application version: %w", err)
		}
		return q.DeleteEmptyLifecycleSoftwareApplication(ctx, id)
	})
	return resourcedb.LifecycleError(err)
}

func (r *Repo) DraftFromPublication(ctx context.Context, actorID string, applicationID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	return r.clone(ctx, actorID, applicationID, applicationID, sqlc.ResourceStatusPUBLISHED, uuid.Nil, true)
}

func (r *Repo) DraftFromLatestArchive(ctx context.Context, actorID string, applicationID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	return r.clone(ctx, actorID, applicationID, applicationID, sqlc.ResourceStatusARCHIVED, uuid.Nil, true)
}

func (r *Repo) CopyPublished(ctx context.Context, actorID string, sourceApplicationID, targetGroupID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	var result domain.SoftwareApplicationVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		source, err := q.GetPublishedLifecycleSoftwareApplicationVersion(ctx, sourceApplicationID)
		if err != nil {
			return fmt.Errorf("get published application version: %w", err)
		}
		application, err := q.CreateLifecycleSoftwareApplication(ctx, sqlc.CreateLifecycleSoftwareApplicationParams{
			GroupID:   targetGroupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create application copy: %w", err)
		}
		version, err := q.CloneLifecycleSoftwareApplicationVersion(ctx, sqlc.CloneLifecycleSoftwareApplicationVersionParams{
			SourceVersionID:             source.ID,
			SourceStatus:                sqlc.ResourceStatusPUBLISHED,
			TargetSoftwareApplicationID: application.ID,
			ActorID:                     actorID,
			TargetGroupID:               targetGroupID,
			CopyCategories:              false,
		})
		if err != nil {
			return fmt.Errorf("clone published application version: %w", err)
		}
		result, err = applicationResult(ctx, q, application.ID, version.ID)
		return err
	})
	return result, resourcedb.LifecycleError(err)
}

func (r *Repo) clone(
	ctx context.Context,
	actorID string,
	sourceID, targetID uuid.UUID,
	status sqlc.ResourceStatus,
	groupID uuid.UUID,
	copyCategories bool,
) (domain.SoftwareApplicationVersion, error) {
	var versionID uuid.UUID
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		var source sqlc.SoftwareApplicationVersion
		var err error
		if status == sqlc.ResourceStatusPUBLISHED {
			source, err = q.GetPublishedLifecycleSoftwareApplicationVersion(ctx, sourceID)
		} else {
			source, err = q.GetLatestArchivedLifecycleSoftwareApplicationVersion(ctx, sourceID)
		}
		if err != nil {
			return fmt.Errorf("get source application version: %w", err)
		}
		if groupID == uuid.Nil {
			application, err := q.GetLifecycleSoftwareApplication(ctx, targetID)
			if err != nil {
				return fmt.Errorf("get target application: %w", err)
			}
			groupID = application.GroupID
		}
		version, err := q.CloneLifecycleSoftwareApplicationVersion(ctx, sqlc.CloneLifecycleSoftwareApplicationVersionParams{
			SourceVersionID:             source.ID,
			SourceStatus:                status,
			TargetSoftwareApplicationID: targetID,
			ActorID:                     actorID,
			TargetGroupID:               groupID,
			CopyCategories:              copyCategories,
		})
		if err != nil {
			return fmt.Errorf("clone application version: %w", err)
		}
		versionID = version.ID
		return nil
	})
	if err != nil {
		return domain.SoftwareApplicationVersion{}, resourcedb.LifecycleError(err)
	}
	return r.Get(ctx, targetID, versionID)
}
