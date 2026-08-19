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

//nolint:dupl // Source code and application creation intentionally mirror typed SQLC operations.
func (r *SourceCodeRepo) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value metadata.SoftwareSourceCode) (softwarelifecycle.SourceCode, error) {
	var result softwarelifecycle.SourceCode
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		logical, err := q.CreateLifecycleSoftwareSourceCode(ctx, sqlc.CreateLifecycleSoftwareSourceCodeParams{SourceIri: value.SourceIRI, GroupID: groupID, CreatedBy: actorID})
		if err != nil {
			return fmt.Errorf("create source code: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareSourceCodeVersion(ctx, SourceCodeVersionParams(logical.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create source code version: %w", err)
		}
		if err := PersistSourceCodeChildren(ctx, q, version.ID, value); err != nil {
			return err
		}
		result, err = sourceCodeResult(ctx, q, logical.ID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *SourceCodeRepo) ReplaceDraft(ctx context.Context, sourceCodeID, versionID uuid.UUID, revision int32, value metadata.SoftwareSourceCode) (softwarelifecycle.SourceCode, error) {
	var result softwarelifecycle.SourceCode
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		p := SourceCodeVersionParams(sourceCodeID, "", value)
		version, err := q.ReplaceLifecycleSoftwareSourceCodeDraft(ctx, sqlc.ReplaceLifecycleSoftwareSourceCodeDraftParams{
			Name: p.Name, Description: p.Description, ProgrammingLanguage: p.ProgrammingLanguage, Keywords: p.Keywords,
			Repository: p.Repository, License: p.License, SoftwareRequirements: p.SoftwareRequirements,
			RuntimePlatform: p.RuntimePlatform, MainEntityOfPage: p.MainEntityOfPage, Version: p.Version,
			ID: versionID, SoftwareSourceCodeID: sourceCodeID, Revision: revision,
		})
		if err != nil {
			return fmt.Errorf("replace source code draft: %w", err)
		}
		if err := q.DeleteLifecycleSoftwareSourceCodeVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("delete source code draft children: %w", err)
		}
		if err := PersistSourceCodeChildren(ctx, q, version.ID, value); err != nil {
			return err
		}
		result, err = sourceCodeResult(ctx, q, sourceCodeID, version.ID)
		return err
	})
	return result, lifecycleError(err)
}

func (r *SourceCodeRepo) Get(ctx context.Context, sourceCodeID, versionID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	result, err := sourceCodeResult(ctx, sqlc.New(r.db), sourceCodeID, versionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return softwarelifecycle.SourceCode{}, softwarelifecycle.ErrNotFound
	}
	return result, err
}

func (r *SourceCodeRepo) transition(ctx context.Context, sourceCodeID, versionID uuid.UUID, fn func(*sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error)) (softwarelifecycle.SourceCode, error) {
	q := sqlc.New(r.db)
	if _, err := q.EnsureLifecycleSoftwareSourceCodeVersion(ctx, sqlc.EnsureLifecycleSoftwareSourceCodeVersionParams{VersionID: versionID, SoftwareSourceCodeID: sourceCodeID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return softwarelifecycle.SourceCode{}, softwarelifecycle.ErrNotFound
		}
		return softwarelifecycle.SourceCode{}, fmt.Errorf("ensure source code version: %w", err)
	}
	version, err := fn(q)
	if err != nil {
		return softwarelifecycle.SourceCode{}, lifecycleError(err)
	}
	return sourceCodeResult(ctx, q, sourceCodeID, version.ID)
}

func (r *SourceCodeRepo) Submit(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.SubmitLifecycleSoftwareSourceCodeVersion(ctx, sqlc.SubmitLifecycleSoftwareSourceCodeVersionParams{ActorID: &actorID, ID: versionID})
	})
}

func (r *SourceCodeRepo) Withdraw(ctx context.Context, sourceCodeID, versionID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.WithdrawLifecycleSoftwareSourceCodeVersion(ctx, versionID)
	})
}

func (r *SourceCodeRepo) Reject(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID, note string) (softwarelifecycle.SourceCode, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.RejectLifecycleSoftwareSourceCodeVersion(ctx, sqlc.RejectLifecycleSoftwareSourceCodeVersionParams{ActorID: &actorID, RejectionNote: &note, ID: versionID})
	})
}

func (r *SourceCodeRepo) Reopen(ctx context.Context, sourceCodeID, versionID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.ReopenLifecycleSoftwareSourceCodeVersion(ctx, versionID)
	})
}

func (r *SourceCodeRepo) Publish(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.PublishLifecycleSoftwareSourceCodeVersion(ctx, sqlc.PublishLifecycleSoftwareSourceCodeVersionParams{ActorID: &actorID, VersionID: versionID})
	})
}

func (r *SourceCodeRepo) Unpublish(ctx context.Context, actorID string, sourceCodeID, versionID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	return r.transition(ctx, sourceCodeID, versionID, func(q *sqlc.Queries) (sqlc.SoftwareSourceCodeVersion, error) {
		return q.UnpublishLifecycleSoftwareSourceCodeVersion(ctx, sqlc.UnpublishLifecycleSoftwareSourceCodeVersionParams{ActorID: &actorID, ID: versionID})
	})
}

func (r *SourceCodeRepo) Discard(ctx context.Context, sourceCodeID, versionID uuid.UUID) error {
	q := sqlc.New(r.db)
	if _, err := q.EnsureLifecycleSoftwareSourceCodeVersion(ctx, sqlc.EnsureLifecycleSoftwareSourceCodeVersionParams{VersionID: versionID, SoftwareSourceCodeID: sourceCodeID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return softwarelifecycle.ErrNotFound
		}
		return fmt.Errorf("ensure source code version: %w", err)
	}
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		id, err := q.DiscardLifecycleSoftwareSourceCodeVersion(ctx, versionID)
		if err != nil {
			return fmt.Errorf("discard source code version: %w", err)
		}
		return q.DeleteEmptyLifecycleSoftwareSourceCode(ctx, id)
	})
	return lifecycleError(err)
}

func (r *SourceCodeRepo) DraftFromPublication(ctx context.Context, actorID string, sourceCodeID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	return r.clone(ctx, actorID, sourceCodeID, sourceCodeID, sqlc.ResourceStatusPUBLISHED, uuid.Nil, true)
}

func (r *SourceCodeRepo) DraftFromLatestArchive(ctx context.Context, actorID string, sourceCodeID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	return r.clone(ctx, actorID, sourceCodeID, sourceCodeID, sqlc.ResourceStatusARCHIVED, uuid.Nil, true)
}

func (r *SourceCodeRepo) CopyPublished(ctx context.Context, actorID string, sourceID, groupID uuid.UUID, sourceIRI string) (softwarelifecycle.SourceCode, error) {
	var targetID, versionID uuid.UUID
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
		logical, err := q.CreateLifecycleSoftwareSourceCode(ctx, sqlc.CreateLifecycleSoftwareSourceCodeParams{SourceIri: sourceIRI, GroupID: groupID, CreatedBy: actorID})
		if err != nil {
			return fmt.Errorf("create source code copy: %w", err)
		}
		targetID = logical.ID
		source, err := q.GetPublishedLifecycleSoftwareSourceCodeVersion(ctx, sourceID)
		if err != nil {
			return fmt.Errorf("get published source code version: %w", err)
		}
		row, err := q.CloneLifecycleSoftwareSourceCodeVersion(ctx, sqlc.CloneLifecycleSoftwareSourceCodeVersionParams{SourceVersionID: source.ID, SourceStatus: sqlc.ResourceStatusPUBLISHED, TargetSoftwareSourceCodeID: targetID, ActorID: actorID, TargetGroupID: groupID, CopyCategories: false})
		if err != nil {
			return fmt.Errorf("clone published source code version: %w", err)
		}
		versionID = row.ID
		return nil
	})
	if err != nil {
		return softwarelifecycle.SourceCode{}, lifecycleError(err)
	}
	return r.Get(ctx, targetID, versionID)
}

//nolint:dupl // Source code and application cloning intentionally mirror typed SQLC operations.
func (r *SourceCodeRepo) clone(ctx context.Context, actorID string, sourceID, targetID uuid.UUID, status sqlc.ResourceStatus, groupID uuid.UUID, copyCategories bool) (softwarelifecycle.SourceCode, error) {
	var versionID uuid.UUID
	err := withTx(ctx, r.db, func(q *sqlc.Queries) error {
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
		row, err := q.CloneLifecycleSoftwareSourceCodeVersion(ctx, sqlc.CloneLifecycleSoftwareSourceCodeVersionParams{SourceVersionID: source.ID, SourceStatus: status, TargetSoftwareSourceCodeID: targetID, ActorID: actorID, TargetGroupID: groupID, CopyCategories: copyCategories})
		if err != nil {
			return fmt.Errorf("clone source code version: %w", err)
		}
		versionID = row.ID
		return nil
	})
	if err != nil {
		return softwarelifecycle.SourceCode{}, lifecycleError(err)
	}
	return r.Get(ctx, targetID, versionID)
}

func SourceCodeVersionParams(id uuid.UUID, actorID string, value metadata.SoftwareSourceCode) sqlc.CreateLifecycleSoftwareSourceCodeVersionParams {
	return sqlc.CreateLifecycleSoftwareSourceCodeVersionParams{SoftwareSourceCodeID: id, Name: value.Name, Description: value.Description, ProgrammingLanguage: value.ProgrammingLanguage, Keywords: value.Keywords, Repository: value.Repository, License: value.License, SoftwareRequirements: value.SoftwareRequirements, RuntimePlatform: value.RuntimePlatform, MainEntityOfPage: value.MainEntityOfPage, Version: value.Version, CreatedBy: actorID}
}

func ReplaceSourceCodeVersionParams(id, sourceCodeID uuid.UUID, revision int32, value metadata.SoftwareSourceCode) sqlc.ReplaceLifecycleSoftwareSourceCodeDraftParams {
	return sqlc.ReplaceLifecycleSoftwareSourceCodeDraftParams{
		Name: value.Name, Description: value.Description, ProgrammingLanguage: value.ProgrammingLanguage,
		Keywords: value.Keywords, Repository: value.Repository, License: value.License,
		SoftwareRequirements: value.SoftwareRequirements, RuntimePlatform: value.RuntimePlatform,
		MainEntityOfPage: value.MainEntityOfPage, Version: value.Version, ID: id,
		SoftwareSourceCodeID: sourceCodeID, Revision: revision,
	}
}

//nolint:dupl // Source code and application children require parallel typed SQLC operations.
func PersistSourceCodeChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value metadata.SoftwareSourceCode) error {
	for _, item := range value.Identifiers {
		if err := q.AddLifecycleSoftwareSourceCodeIdentifier(ctx, sqlc.AddLifecycleSoftwareSourceCodeIdentifierParams{SourceIri: item.SourceIRI, Type: string(item.Type), Value: item.Value, SoftwareSourceCodeVersionID: &versionID}); err != nil {
			return fmt.Errorf("add source code identifier: %w", err)
		}
	}
	for _, item := range value.Creators {
		if err := q.AddLifecycleSoftwareSourceCodeCreator(ctx, sqlc.AddLifecycleSoftwareSourceCodeCreatorParams{SoftwareSourceCodeVersionID: versionID, OrganizationID: item.ID}); err != nil {
			return fmt.Errorf("add source code creator: %w", err)
		}
	}
	for _, item := range value.ContactPoints {
		if err := q.AddLifecycleSoftwareSourceCodeContactPoint(ctx, sqlc.AddLifecycleSoftwareSourceCodeContactPointParams{SoftwareSourceCodeVersionID: versionID, ContactPointID: item.ID}); err != nil {
			return fmt.Errorf("add source code contact point: %w", err)
		}
	}
	for _, item := range value.Categories {
		if err := q.AddLifecycleSoftwareSourceCodeCategory(ctx, sqlc.AddLifecycleSoftwareSourceCodeCategoryParams{SoftwareSourceCodeVersionID: versionID, CategoryID: item.ID}); err != nil {
			return fmt.Errorf("add source code category: %w", err)
		}
	}
	return nil
}

func sourceCodeResult(ctx context.Context, q *sqlc.Queries, sourceCodeID, versionID uuid.UUID) (softwarelifecycle.SourceCode, error) {
	row, err := q.GetLifecycleSoftwareSourceCodeVersion(ctx, sqlc.GetLifecycleSoftwareSourceCodeVersionParams{VersionID: versionID, SoftwareSourceCodeID: sourceCodeID})
	if err != nil {
		return softwarelifecycle.SourceCode{}, fmt.Errorf("get source code version: %w", err)
	}
	ids, err := q.ListLifecycleSoftwareSourceCodeIdentifiers(ctx, &versionID)
	if err != nil {
		return softwarelifecycle.SourceCode{}, fmt.Errorf("list source code identifiers: %w", err)
	}
	creators, err := q.ListLifecycleSoftwareSourceCodeCreators(ctx, versionID)
	if err != nil {
		return softwarelifecycle.SourceCode{}, fmt.Errorf("list source code creators: %w", err)
	}
	contacts, err := q.ListLifecycleSoftwareSourceCodeContactPoints(ctx, versionID)
	if err != nil {
		return softwarelifecycle.SourceCode{}, fmt.Errorf("list source code contact points: %w", err)
	}
	categories, err := q.ListLifecycleSoftwareSourceCodeCategories(ctx, versionID)
	if err != nil {
		return softwarelifecycle.SourceCode{}, fmt.Errorf("list source code categories: %w", err)
	}
	value := metadata.SoftwareSourceCode{
		ID:                   row.SoftwareSourceCodeVersion.ID,
		SourceIRI:            row.SoftwareSourceCode.SourceIri,
		Name:                 row.SoftwareSourceCodeVersion.Name,
		Description:          row.SoftwareSourceCodeVersion.Description,
		ProgrammingLanguage:  row.SoftwareSourceCodeVersion.ProgrammingLanguage,
		Keywords:             row.SoftwareSourceCodeVersion.Keywords,
		Repository:           row.SoftwareSourceCodeVersion.Repository,
		License:              row.SoftwareSourceCodeVersion.License,
		SoftwareRequirements: row.SoftwareSourceCodeVersion.SoftwareRequirements,
		RuntimePlatform:      row.SoftwareSourceCodeVersion.RuntimePlatform,
		MainEntityOfPage:     row.SoftwareSourceCodeVersion.MainEntityOfPage,
		Version:              row.SoftwareSourceCodeVersion.Version,
		Creators:             nil,
		Identifiers:          nil,
		ContactPoints:        nil,
		Categories:           nil,
	}
	for _, item := range ids {
		value.Identifiers = append(value.Identifiers, identifier(item.Identifier))
	}
	for _, item := range creators {
		creator, err := organizationWithContactPoints(ctx, q, item)
		if err != nil {
			return softwarelifecycle.SourceCode{}, err
		}
		value.Creators = append(value.Creators, creator)
	}
	for _, item := range contacts {
		value.ContactPoints = append(value.ContactPoints, contactPoint(item.ContactPoint))
	}
	for _, item := range categories {
		value.Categories = append(value.Categories, category(item.Category.ID, item.Category.SourceIri, item.Category.Name, item.Group))
	}
	return softwarelifecycle.SourceCode{SourceCodeID: row.SoftwareSourceCodeVersion.SoftwareSourceCodeID, VersionID: row.SoftwareSourceCodeVersion.ID, Revision: row.SoftwareSourceCodeVersion.Revision, Status: metadata.DatasetStatus(row.SoftwareSourceCodeVersion.Status), CreatedBy: row.SoftwareSourceCodeVersion.CreatedBy, Group: categoryScheme(row.Group), Metadata: value}, nil
}
