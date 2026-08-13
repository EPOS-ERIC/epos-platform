package software

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/softwarelifecycle"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

//nolint:dupl // Application and source code creation intentionally mirror typed SQLC operations.
func (r *ApplicationRepo) CreateDraft(
	ctx context.Context,
	actorID,
	groupID uuid.UUID,
	value metadata.SoftwareApplication,
) (softwarelifecycle.Application, error) {
	var result softwarelifecycle.Application
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		app, err := q.CreateLifecycleSoftwareApplication(ctx, sqlc.CreateLifecycleSoftwareApplicationParams{
			SourceIri: value.SourceIRI,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create application: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareApplicationVersion(
			ctx,
			ApplicationVersionParams(app.ID, actorID, value),
		)
		if err != nil {
			return fmt.Errorf("create application version: %w", err)
		}
		if err := PersistApplicationChildren(ctx, q, version.ID, value); err != nil {
			return err
		}
		result, err = applicationResult(ctx, q, app.ID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *ApplicationRepo) ReplaceDraft(
	ctx context.Context,
	appID,
	versionID uuid.UUID,
	revision int32,
	value metadata.SoftwareApplication,
) (softwarelifecycle.Application, error) {
	var result softwarelifecycle.Application
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		p := ApplicationVersionParams(appID, uuid.Nil, value)
		version, err := q.ReplaceLifecycleSoftwareApplicationDraft(ctx, sqlc.ReplaceLifecycleSoftwareApplicationDraftParams{
			Name:                  p.Name,
			Description:           p.Description,
			MainEntityOfPage:      p.MainEntityOfPage,
			License:               p.License,
			Version:               p.Version,
			Keywords:              p.Keywords,
			DownloadUrl:           p.DownloadUrl,
			Requirements:          p.Requirements,
			OperatingSystem:       p.OperatingSystem,
			ID:                    versionID,
			SoftwareApplicationID: appID,
			Revision:              revision,
		},
		)
		if err != nil {
			return fmt.Errorf("replace application draft: %w", err)
		}
		if err := q.DeleteLifecycleSoftwareApplicationVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("delete application draft children: %w", err)
		}
		if err := PersistApplicationChildren(ctx, q, version.ID, value); err != nil {
			return err
		}
		result, err = applicationResult(ctx, q, appID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *ApplicationRepo) Get(ctx context.Context, appID, versionID uuid.UUID) (softwarelifecycle.Application, error) {
	result, err := applicationResult(ctx, sqlc.New(r.db), appID, versionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return softwarelifecycle.Application{}, softwarelifecycle.ErrNotFound
	}
	return result, err
}

func (r *ApplicationRepo) Submit(
	ctx context.Context,
	actorID, appID, versionID uuid.UUID,
) (softwarelifecycle.Application, error) {
	return r.transition(ctx, appID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.SubmitLifecycleSoftwareApplicationVersion(ctx, sqlc.SubmitLifecycleSoftwareApplicationVersionParams{
			ActorID: &actorID,
			ID:      versionID,
		})
	})
}

func (r *ApplicationRepo) Withdraw(ctx context.Context, appID, versionID uuid.UUID) (softwarelifecycle.Application, error) {
	return r.transition(ctx, appID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.WithdrawLifecycleSoftwareApplicationVersion(ctx, versionID)
	})
}

func (r *ApplicationRepo) Reject(ctx context.Context, actorID, appID, versionID uuid.UUID, note string) (softwarelifecycle.Application, error) {
	return r.transition(ctx, appID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.RejectLifecycleSoftwareApplicationVersion(ctx, sqlc.RejectLifecycleSoftwareApplicationVersionParams{
			ActorID:       &actorID,
			RejectionNote: &note,
			ID:            versionID,
		})
	})
}

func (r *ApplicationRepo) Reopen(ctx context.Context, appID, versionID uuid.UUID) (softwarelifecycle.Application, error) {
	return r.transition(ctx, appID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.ReopenLifecycleSoftwareApplicationVersion(ctx, versionID)
	})
}

func (r *ApplicationRepo) Publish(ctx context.Context, actorID, appID, versionID uuid.UUID) (softwarelifecycle.Application, error) {
	return r.transition(ctx, appID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.PublishLifecycleSoftwareApplicationVersion(ctx, sqlc.PublishLifecycleSoftwareApplicationVersionParams{
			ActorID:   &actorID,
			VersionID: versionID,
		})
	})
}

func (r *ApplicationRepo) Unpublish(ctx context.Context, actorID, appID, versionID uuid.UUID) (softwarelifecycle.Application, error) {
	return r.transition(ctx, appID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareApplicationVersion, error) {
		return q.UnpublishLifecycleSoftwareApplicationVersion(ctx, sqlc.UnpublishLifecycleSoftwareApplicationVersionParams{
			ActorID: &actorID,
			ID:      versionID,
		})
	})
}

func (r *ApplicationRepo) transition(
	ctx context.Context,
	appID, versionID uuid.UUID,
	fn func(*sqlc.Queries) (sqlc.SoftwareApplicationVersion, error),
) (softwarelifecycle.Application, error) {
	q := sqlc.New(r.db)
	if _, err := q.EnsureLifecycleSoftwareApplicationVersion(ctx, sqlc.EnsureLifecycleSoftwareApplicationVersionParams{
		VersionID:             versionID,
		SoftwareApplicationID: appID,
	},
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return softwarelifecycle.Application{}, softwarelifecycle.ErrNotFound
		}
		return softwarelifecycle.Application{}, fmt.Errorf("ensure application version: %w", err)
	}
	version, err := fn(q)
	if err != nil {
		return softwarelifecycle.Application{}, lifecycleError(err)
	}
	return applicationResult(ctx, q, appID, version.ID)
}

func (r *ApplicationRepo) Discard(ctx context.Context, appID, versionID uuid.UUID) error {
	q := sqlc.New(r.db)
	if _, err := q.EnsureLifecycleSoftwareApplicationVersion(ctx, sqlc.EnsureLifecycleSoftwareApplicationVersionParams{
		VersionID:             versionID,
		SoftwareApplicationID: appID,
	},
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return softwarelifecycle.ErrNotFound
		}
		return fmt.Errorf("ensure application version: %w", err)
	}
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		id, err := q.DiscardLifecycleSoftwareApplicationVersion(ctx, versionID)
		if err != nil {
			return fmt.Errorf("discard application version: %w", err)
		}
		return q.DeleteEmptyLifecycleSoftwareApplication(ctx, id)
	})
	return lifecycleError(err)
}

func (r *ApplicationRepo) DraftFromPublication(
	ctx context.Context,
	actorID, appID uuid.UUID,
) (softwarelifecycle.Application, error) {
	return r.clone(ctx, actorID, appID, appID, sqlc.ResourceStatusPUBLISHED, uuid.Nil, true)
}

func (r *ApplicationRepo) DraftFromLatestArchive(
	ctx context.Context,
	actorID, appID uuid.UUID,
) (softwarelifecycle.Application, error) {
	return r.clone(ctx, actorID, appID, appID, sqlc.ResourceStatusARCHIVED, uuid.Nil, true)
}

func (r *ApplicationRepo) CopyPublished(
	ctx context.Context,
	actorID, sourceID, groupID uuid.UUID,
	sourceIRI string,
) (softwarelifecycle.Application, error) {
	var targetID uuid.UUID
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		app, err := q.CreateLifecycleSoftwareApplication(ctx, sqlc.CreateLifecycleSoftwareApplicationParams{
			SourceIri: sourceIRI,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create application copy: %w", err)
		}
		targetID = app.ID
		source, err := q.GetPublishedLifecycleSoftwareApplicationVersion(ctx, sourceID)
		if err != nil {
			return fmt.Errorf("get published application version: %w", err)
		}
		_, err = q.CloneLifecycleSoftwareApplicationVersion(
			ctx,
			sqlc.CloneLifecycleSoftwareApplicationVersionParams{
				SourceVersionID:             source.ID,
				SourceStatus:                sqlc.ResourceStatusPUBLISHED,
				TargetSoftwareApplicationID: targetID,
				ActorID:                     actorID,
				TargetGroupID:               groupID,
				CopyCategories:              false,
			},
		)
		if err != nil {
			return fmt.Errorf("clone published application version: %w", err)
		}
		return nil
	})
	if err != nil {
		return softwarelifecycle.Application{}, lifecycleError(err)
	}
	active, err := sqlc.New(r.db).GetActiveLifecycleSoftwareApplicationVersion(ctx, targetID)
	if err != nil {
		return softwarelifecycle.Application{}, fmt.Errorf("get active application version: %w", err)
	}
	return r.Get(ctx, targetID, active.ID)
}

//nolint:dupl // Application and source code cloning intentionally mirror typed SQLC operations.
func (r *ApplicationRepo) clone(
	ctx context.Context,
	actorID, sourceID, targetID uuid.UUID,
	status sqlc.ResourceStatus,
	groupID uuid.UUID,
	copyCategories bool,
) (softwarelifecycle.Application, error) {
	var versionID uuid.UUID
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
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
			app, err := q.GetLifecycleSoftwareApplication(ctx, targetID)
			if err != nil {
				return fmt.Errorf("get target application: %w", err)
			}
			groupID = app.GroupID
		}
		row, err := q.CloneLifecycleSoftwareApplicationVersion(ctx, sqlc.CloneLifecycleSoftwareApplicationVersionParams{
			SourceVersionID:             source.ID,
			SourceStatus:                status,
			TargetSoftwareApplicationID: targetID,
			ActorID:                     actorID,
			TargetGroupID:               groupID,
			CopyCategories:              copyCategories,
		},
		)
		if err != nil {
			return fmt.Errorf("clone application version: %w", err)
		}
		versionID = row.ID
		return nil
	})
	if err != nil {
		return softwarelifecycle.Application{}, lifecycleError(err)
	}
	return r.Get(ctx, targetID, versionID)
}

func ApplicationVersionParams(
	id, actorID uuid.UUID,
	value metadata.SoftwareApplication,
) sqlc.CreateLifecycleSoftwareApplicationVersionParams {
	return sqlc.CreateLifecycleSoftwareApplicationVersionParams{
		SoftwareApplicationID: id,
		Name:                  value.Name,
		Description:           value.Description,
		MainEntityOfPage:      value.MainEntityOfPage,
		License:               value.License,
		Version:               value.Version,
		Keywords:              value.Keywords,
		DownloadUrl:           value.DownloadURL,
		Requirements:          value.Requirements,
		OperatingSystem:       value.OperatingSystem,
		CreatedBy:             actorID,
	}
}

func ReplaceApplicationVersionParams(
	id, applicationID uuid.UUID,
	revision int32,
	value metadata.SoftwareApplication,
) sqlc.ReplaceLifecycleSoftwareApplicationDraftParams {
	return sqlc.ReplaceLifecycleSoftwareApplicationDraftParams{
		Name:                  value.Name,
		Description:           value.Description,
		MainEntityOfPage:      value.MainEntityOfPage,
		License:               value.License,
		Version:               value.Version,
		Keywords:              value.Keywords,
		DownloadUrl:           value.DownloadURL,
		Requirements:          value.Requirements,
		OperatingSystem:       value.OperatingSystem,
		ID:                    id,
		SoftwareApplicationID: applicationID,
		Revision:              revision,
	}
}

//nolint:dupl // Application and source code children require parallel typed SQLC operations.
func PersistApplicationChildren(
	ctx context.Context,
	q *sqlc.Queries,
	versionID uuid.UUID,
	value metadata.SoftwareApplication,
) error {
	for _, item := range value.Identifiers {
		if err := q.AddLifecycleSoftwareApplicationIdentifier(
			ctx,
			sqlc.AddLifecycleSoftwareApplicationIdentifierParams{
				SourceIri:                    item.SourceIRI,
				Type:                         string(item.Type),
				Value:                        item.Value,
				SoftwareApplicationVersionID: &versionID,
			},
		); err != nil {
			return fmt.Errorf("add application identifier: %w", err)
		}
	}
	for _, item := range value.Creators {
		if err := q.AddLifecycleSoftwareApplicationCreator(
			ctx,
			sqlc.AddLifecycleSoftwareApplicationCreatorParams{
				SoftwareApplicationVersionID: versionID,
				OrganizationID:               item.ID,
			},
		); err != nil {
			return fmt.Errorf("add application creator: %w", err)
		}
	}
	for _, item := range value.ContactPoints {
		if err := q.AddLifecycleSoftwareApplicationContactPoint(
			ctx,
			sqlc.AddLifecycleSoftwareApplicationContactPointParams{
				SoftwareApplicationVersionID: versionID,
				ContactPointID:               item.ID,
			},
		); err != nil {
			return fmt.Errorf("add application contact point: %w", err)
		}
	}
	for _, item := range value.Categories {
		if err := q.AddLifecycleSoftwareApplicationCategory(
			ctx,
			sqlc.AddLifecycleSoftwareApplicationCategoryParams{
				SoftwareApplicationVersionID: versionID,
				CategoryID:                   item.ID,
			},
		); err != nil {
			return fmt.Errorf("add application category: %w", err)
		}
	}
	return nil
}

func applicationResult(ctx context.Context, q *sqlc.Queries, appID, versionID uuid.UUID) (softwarelifecycle.Application, error) {
	row, err := q.GetLifecycleSoftwareApplicationVersion(ctx, sqlc.GetLifecycleSoftwareApplicationVersionParams{
		VersionID:             versionID,
		SoftwareApplicationID: appID,
	},
	)
	if err != nil {
		return softwarelifecycle.Application{}, fmt.Errorf("get application version: %w", err)
	}
	ids, err := q.ListLifecycleSoftwareApplicationIdentifiers(ctx, &versionID)
	if err != nil {
		return softwarelifecycle.Application{}, fmt.Errorf("list application identifiers: %w", err)
	}
	creators, err := q.ListLifecycleSoftwareApplicationCreators(ctx, versionID)
	if err != nil {
		return softwarelifecycle.Application{}, fmt.Errorf("list application creators: %w", err)
	}
	contacts, err := q.ListLifecycleSoftwareApplicationContactPoints(ctx, versionID)
	if err != nil {
		return softwarelifecycle.Application{}, fmt.Errorf("list application contact points: %w", err)
	}
	categories, err := q.ListLifecycleSoftwareApplicationCategories(ctx, versionID)
	if err != nil {
		return softwarelifecycle.Application{}, fmt.Errorf("list application categories: %w", err)
	}
	value := metadata.SoftwareApplication{
		ID:               row.SoftwareApplicationVersion.ID,
		SourceIRI:        row.SoftwareApplication.SourceIri,
		Name:             row.SoftwareApplicationVersion.Name,
		Description:      row.SoftwareApplicationVersion.Description,
		MainEntityOfPage: row.SoftwareApplicationVersion.MainEntityOfPage,
		License:          row.SoftwareApplicationVersion.License,
		Version:          row.SoftwareApplicationVersion.Version,
		Keywords:         row.SoftwareApplicationVersion.Keywords,
		DownloadURL:      row.SoftwareApplicationVersion.DownloadUrl,
		Requirements:     row.SoftwareApplicationVersion.Requirements,
		OperatingSystem:  row.SoftwareApplicationVersion.OperatingSystem,
		Creators:         nil,
		Identifiers:      nil,
		ContactPoints:    nil,
		Categories:       nil,
	}
	for _, item := range ids {
		value.Identifiers = append(value.Identifiers, identifier(item.Identifier))
	}
	for _, item := range creators {
		creator, err := organizationWithContactPoints(ctx, q, item)
		if err != nil {
			return softwarelifecycle.Application{}, err
		}
		value.Creators = append(value.Creators, creator)
	}
	for _, item := range contacts {
		value.ContactPoints = append(value.ContactPoints, contactPoint(item.ContactPoint))
	}
	for _, item := range categories {
		cat := item.Category
		value.Categories = append(value.Categories, category(
			cat.ID,
			cat.SourceIri,
			cat.Name,
			item.Group,
		))
	}
	version := row.SoftwareApplicationVersion
	return softwarelifecycle.Application{
		ApplicationID: version.SoftwareApplicationID,
		VersionID:     version.ID,
		Revision:      version.Revision,
		Status:        metadata.DatasetStatus(version.Status),
		CreatedBy:     version.CreatedBy,
		Group:         categoryScheme(row.Group),
		Metadata:      value,
	}, nil
}
