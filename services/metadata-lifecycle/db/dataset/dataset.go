// Package dataset provides PostgreSQL persistence for dataset lifecycle data.
package dataset

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	resourcedb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/resource"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo persists each lifecycle operation in one database transaction.
type Repo struct {
	db *pgxpool.Pool
}

// New creates a dataset repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{db: pool}
}

func (r *Repo) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value domain.DatasetInput) (domain.DatasetVersion, error) {
	var result domain.DatasetVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		valid, err := resourcedb.CategoriesBelongToGroup(ctx, q, groupID, value.CategoryIDs)
		if err != nil {
			return fmt.Errorf("validate dataset categories: %w", err)
		}
		if !valid {
			return domain.ErrInvalidDataset
		}

		dataset, err := q.CreateLifecycleDataset(ctx, sqlc.CreateLifecycleDatasetParams{
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create dataset: %w", err)
		}

		params, err := ToCreateVersionParams(dataset.ID, actorID, value)
		if err != nil {
			return fmt.Errorf("build dataset version: %w", err)
		}

		version, err := q.CreateLifecycleDatasetVersion(ctx, params)
		if err != nil {
			return fmt.Errorf("create dataset version: %w", err)
		}

		if err := PersistChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist dataset metadata: %w", err)
		}
		result, err = datasetResultByID(ctx, q, version.DatasetID, version.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, lifecycleError(err)
}

func (r *Repo) ReplaceDraft(
	ctx context.Context,
	datasetID,
	versionID uuid.UUID,
	value domain.DatasetDraftInput,
) (domain.DatasetVersion, error) {
	var result domain.DatasetVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if len(value.Metadata.CategoryIDs) > 0 {
			dataset, err := q.GetLifecycleDataset(ctx, datasetID)
			if err != nil {
				return fmt.Errorf("get dataset for category validation: %w", err)
			}
			valid, err := resourcedb.CategoriesBelongToGroup(ctx, q, dataset.GroupID, value.Metadata.CategoryIDs)
			if err != nil {
				return fmt.Errorf("validate dataset categories: %w", err)
			}
			if !valid {
				return domain.ErrInvalidDataset
			}
		}

		params, err := ToReplaceDraftParams(datasetID, versionID, value.Revision, value.Metadata)
		if err != nil {
			return fmt.Errorf("build draft replacement: %w", err)
		}

		version, err := q.ReplaceLifecycleDatasetDraft(ctx, params)
		if err != nil {
			return fmt.Errorf("replace draft: %w", err)
		}

		if err := q.DeleteLifecycleDatasetVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("delete draft children: %w", err)
		}
		if err := PersistChildren(ctx, q, version.ID, value.Metadata); err != nil {
			return fmt.Errorf("persist draft metadata: %w", err)
		}
		result, err = datasetResultByID(ctx, q, version.DatasetID, version.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, lifecycleError(err)
}

// Get returns one selected version with all version-owned metadata and canonical references.
func (r *Repo) Get(ctx context.Context, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	q := sqlc.New(r.db)
	version, err := q.GetLifecycleDatasetVersion(ctx, sqlc.GetLifecycleDatasetVersionParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetNotFound
		}
		return domain.DatasetVersion{}, fmt.Errorf("get dataset version: %w", err)
	}
	return datasetResultFromRow(ctx, q, version)
}

func (r *Repo) List(ctx context.Context, datasetID uuid.UUID) ([]domain.ResourceVersionSummary, error) {
	rows, err := sqlc.New(r.db).ListLifecycleDatasetVersionSummaries(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("list dataset version summaries: %w", err)
	}

	result := make([]domain.ResourceVersionSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainDatasetVersionSummary(row))
	}
	return result, nil
}

func (r *Repo) GetPublished(ctx context.Context, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	version, err := sqlc.New(r.db).GetPublishedLifecycleDatasetVersion(ctx, datasetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetNotFound
		}
		return domain.DatasetVersion{}, fmt.Errorf("get published dataset version: %w", err)
	}
	return r.Get(ctx, datasetID, version.ID)
}

func (r *Repo) GetActive(ctx context.Context, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	version, err := sqlc.New(r.db).GetActiveLifecycleDatasetVersion(ctx, datasetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetNotFound
		}
		return domain.DatasetVersion{}, fmt.Errorf("get active dataset version: %w", err)
	}
	return r.Get(ctx, datasetID, version.ID)
}

func (r *Repo) Submit(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("submit dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).SubmitLifecycleDatasetVersion(ctx, sqlc.SubmitLifecycleDatasetVersionParams{
		ActorID: &actorID,
		ID:      versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetConflict
		}
		return domain.DatasetVersion{}, fmt.Errorf("submit dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Withdraw(ctx context.Context, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("withdraw dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).WithdrawLifecycleDatasetVersion(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetConflict
		}
		return domain.DatasetVersion{}, fmt.Errorf("withdraw dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Reject(ctx context.Context, actorID string, datasetID, versionID uuid.UUID, value domain.DatasetRejectionInput) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("reject dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).RejectLifecycleDatasetVersion(ctx, sqlc.RejectLifecycleDatasetVersionParams{
		ActorID:       &actorID,
		RejectionNote: &value.Note,
		ID:            versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetConflict
		}
		return domain.DatasetVersion{}, fmt.Errorf("reject dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Reopen(ctx context.Context, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("reopen dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).ReopenLifecycleDatasetVersion(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetConflict
		}
		return domain.DatasetVersion{}, fmt.Errorf("reopen dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Publish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("publish dataset version: %w", err)
	}

	var result domain.DatasetVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		if err := q.ArchivePublishedLifecycleDatasetVersion(ctx, sqlc.ArchivePublishedLifecycleDatasetVersionParams{
			ActorID:   &actorID,
			DatasetID: datasetID,
		}); err != nil {
			return fmt.Errorf("archive published dataset version: %w", err)
		}

		v, err := q.PublishLifecycleDatasetVersion(ctx, sqlc.PublishLifecycleDatasetVersionParams{
			ActorID:   &actorID,
			VersionID: versionID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrDatasetConflict
			}
			return fmt.Errorf("publish dataset version: %w", err)
		}

		result, err = datasetResultByID(ctx, q, datasetID, v.ID)
		return err
	})
	if err != nil {
		return domain.DatasetVersion{}, lifecycleError(err)
	}
	return result, nil
}

func (r *Repo) Unpublish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("unpublish dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).UnpublishLifecycleDatasetVersion(ctx, sqlc.UnpublishLifecycleDatasetVersionParams{
		ActorID: &actorID,
		ID:      versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, domain.ErrDatasetConflict
		}
		return domain.DatasetVersion{}, fmt.Errorf("unpublish dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Discard(ctx context.Context, datasetID, versionID uuid.UUID) error {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return fmt.Errorf("discard dataset: %w", err)
	}

	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		id, err := q.DiscardLifecycleDatasetVersion(ctx, versionID)
		if err != nil {
			return fmt.Errorf("discard dataset version: %w", err)
		}
		if err := q.DeleteEmptyLifecycleDataset(ctx, id); err != nil {
			return fmt.Errorf("delete empty dataset: %w", err)
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrDatasetConflict
	}
	if err != nil {
		return fmt.Errorf("discard dataset transaction: %w", err)
	}

	return nil
}

func (r *Repo) DraftFromPublication(ctx context.Context, actorID string, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	return r.clone(ctx, actorID, datasetID, datasetID, sqlc.ResourceStatusPUBLISHED, uuid.Nil, true)
}

func (r *Repo) DraftFromLatestArchive(ctx context.Context, actorID string, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	return r.clone(ctx, actorID, datasetID, datasetID, sqlc.ResourceStatusARCHIVED, uuid.Nil, true)
}

func (r *Repo) CopyPublished(ctx context.Context, actorID string, sourceDatasetID, targetGroupID uuid.UUID) (domain.DatasetVersion, error) {
	var result domain.DatasetVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		source, err := q.GetPublishedLifecycleDatasetVersion(ctx, sourceDatasetID)
		if err != nil {
			return fmt.Errorf("get published source version: %w", err)
		}
		logical, err := q.CreateLifecycleDataset(ctx, sqlc.CreateLifecycleDatasetParams{
			GroupID:   targetGroupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create copied dataset: %w", err)
		}
		version, err := q.CloneLifecycleDatasetVersion(ctx, sqlc.CloneLifecycleDatasetVersionParams{
			SourceVersionID: source.ID,
			SourceStatus:    sqlc.ResourceStatusPUBLISHED,
			TargetDatasetID: logical.ID,
			ActorID:         actorID,
			TargetGroupID:   logical.GroupID,
			CopyCategories:  false,
		})
		if err != nil {
			return fmt.Errorf("clone published dataset version: %w", err)
		}
		result, err = datasetResultByID(ctx, q, version.DatasetID, version.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, lifecycleError(err)
}

func (r *Repo) clone(
	ctx context.Context,
	actorID string,
	sourceDatasetID,
	targetDatasetID uuid.UUID,
	status sqlc.ResourceStatus,
	targetGroupID uuid.UUID,
	categories bool,
) (domain.DatasetVersion, error) {
	var result domain.DatasetVersion
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		var source sqlc.DatasetVersion
		var err error
		if status == sqlc.ResourceStatusPUBLISHED {
			source, err = q.GetPublishedLifecycleDatasetVersion(ctx, sourceDatasetID)
		} else {
			source, err = q.GetLatestArchivedLifecycleDatasetVersion(ctx, sourceDatasetID)
		}
		if err != nil {
			return fmt.Errorf("get source dataset version: %w", err)
		}
		if targetGroupID == uuid.Nil {
			logical, err := q.GetLifecycleDataset(ctx, targetDatasetID)
			if err != nil {
				return fmt.Errorf("get target dataset: %w", err)
			}
			targetGroupID = logical.GroupID
		}
		v, err := q.CloneLifecycleDatasetVersion(ctx, sqlc.CloneLifecycleDatasetVersionParams{
			SourceVersionID: source.ID,
			SourceStatus:    status,
			TargetDatasetID: targetDatasetID,
			ActorID:         actorID,
			TargetGroupID:   targetGroupID,
			CopyCategories:  categories,
		})
		if err != nil {
			return fmt.Errorf("clone dataset version: %w", err)
		}
		result, err = datasetResultByID(ctx, q, v.DatasetID, v.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, lifecycleError(err)
}

func (r *Repo) ensureVersion(ctx context.Context, datasetID, versionID uuid.UUID) error {
	_, err := sqlc.New(r.db).EnsureLifecycleDatasetVersion(ctx, sqlc.EnsureLifecycleDatasetVersionParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrDatasetNotFound
	}
	if err != nil {
		return fmt.Errorf("ensure dataset version: %w", err)
	}
	return nil
}

func datasetResultByID(ctx context.Context, q *sqlc.Queries, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	row, err := q.GetLifecycleDatasetVersion(ctx, sqlc.GetLifecycleDatasetVersionParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("get dataset result: %w", err)
	}
	return datasetResultFromRow(ctx, q, row)
}

func datasetResultFromRow(ctx context.Context, q *sqlc.Queries, version sqlc.GetLifecycleDatasetVersionRow) (domain.DatasetVersion, error) {
	metadataValue, err := readDatasetMetadata(ctx, q, version)
	if err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("read dataset metadata: %w", err)
	}
	return toDomainDatasetVersion(version, metadataValue)
}

func PersistChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value domain.DatasetInput) error {
	for _, identifier := range value.PersistentIdentifiers {
		if err := q.AddLifecycleDatasetIdentifier(ctx, sqlc.AddLifecycleDatasetIdentifierParams{
			Type:             string(identifier.Type),
			Value:            identifier.Value,
			DatasetVersionID: &versionID,
		}); err != nil {
			return fmt.Errorf("add identifier: %w", err)
		}
	}
	for _, providerID := range value.ProviderIDs {
		if err := q.AddLifecycleDatasetProvider(ctx, sqlc.AddLifecycleDatasetProviderParams{
			DatasetVersionID: versionID,
			OrganizationID:   providerID,
		}); err != nil {
			return fmt.Errorf("add provider: %w", err)
		}
	}
	for _, contactID := range value.ContactPointIDs {
		if err := q.AddLifecycleDatasetContactPoint(ctx, sqlc.AddLifecycleDatasetContactPointParams{
			DatasetVersionID: versionID,
			ContactPointID:   contactID,
		}); err != nil {
			return fmt.Errorf("add contact point: %w", err)
		}
	}
	for _, categoryID := range value.CategoryIDs {
		if err := q.AddLifecycleDatasetCategory(ctx, sqlc.AddLifecycleDatasetCategoryParams{
			DatasetVersionID: versionID,
			CategoryID:       categoryID,
		}); err != nil {
			return fmt.Errorf("add category: %w", err)
		}
	}

	services := make([]*uuid.UUID, len(value.Distributions))
	for i, distribution := range value.Distributions {
		if distribution.WebService == nil {
			continue
		}
		service := distribution.WebService
		var start, end *time.Time
		if service.TemporalCoverage != nil {
			start, end = service.TemporalCoverage.Start, service.TemporalCoverage.End
		}
		providerID := service.ProviderID
		var title, description, uri *string
		if service.Documentation != nil {
			title, description, uri = service.Documentation.Title, service.Documentation.Description, &service.Documentation.URI
		}
		created, err := q.AddLifecycleWebService(ctx, sqlc.AddLifecycleWebServiceParams{
			DatasetVersionID:         versionID,
			Name:                     service.Name,
			Description:              service.Description,
			TemplateUrl:              service.TemplateURL,
			Method:                   sqlc.WebServiceHttpMethod(service.Method),
			Type:                     sqlc.WebServiceType(service.Type),
			VisibleOn:                visibleOnValues(service.VisibleOn),
			Format:                   service.Format,
			ProviderOrganizationID:   providerID,
			DocumentationTitle:       title,
			DocumentationDescription: description,
			DocumentationUri:         uri,
			SpatialCoverage:          service.SpatialCoverage,
			TemporalStart:            start,
			TemporalEnd:              end,
		})
		if err != nil {
			return fmt.Errorf("add web service: %w", err)
		}
		services[i] = &created.ID
		for _, contactID := range service.ContactPointIDs {
			if err := q.AddLifecycleWebServiceContactPoint(ctx, sqlc.AddLifecycleWebServiceContactPointParams{
				WebServiceID:   created.ID,
				ContactPointID: contactID,
			}); err != nil {
				return fmt.Errorf("add web service contact point: %w", err)
			}
		}
		var bodyID *uuid.UUID
		if service.Body != nil {
			body, err := q.AddLifecycleWebServiceBody(ctx, sqlc.AddLifecycleWebServiceBodyParams{
				WebServiceID: created.ID,
				ContentType:  service.Body.ContentType,
				Template:     service.Body.Template,
			})
			if err != nil {
				return fmt.Errorf("add web service body: %w", err)
			}
			bodyID = &body.ID
			for _, parameter := range service.Body.Parameters {
				if _, err := persistParameter(ctx, q, parameter, nil, bodyID, nil, nil, nil, false); err != nil {
					return fmt.Errorf("add web service body parameter: %w", err)
				}
			}
		}
		for _, parameter := range service.Parameters {
			location := parameter.Location
			if _, err := persistParameter(ctx, q, parameter.Parameter, &created.ID, nil, nil, parameter.Property, &location, parameter.Explode); err != nil {
				return fmt.Errorf("add web service parameter: %w", err)
			}
		}
	}
	for i, distribution := range value.Distributions {
		params := sqlc.AddLifecycleDistributionParams{
			DatasetVersionID:       versionID,
			Title:                  distribution.Title,
			Description:            distribution.Description,
			License:                distribution.License,
			SourceIssuedAt:         distribution.Issued,
			SourceModifiedAt:       distribution.Modified,
			ByteSize:               nil,
			Type:                   string(distribution.Type),
			DownloadableFileUrl:    nil,
			DownloadableFileFormat: nil,
			WebServiceID:           nil,
		}
		if distribution.ByteSize != nil {
			byteSize := int32(*distribution.ByteSize)
			params.ByteSize = &byteSize
		}
		if distribution.DownloadableFile != nil {
			params.DownloadableFileUrl = &distribution.DownloadableFile.URL
			params.DownloadableFileFormat = &distribution.DownloadableFile.Format
		}
		if distribution.WebService != nil {
			if services[i] == nil {
				return fmt.Errorf("distribution web service is missing")
			}
			params.WebServiceID = services[i]
		}
		if err := q.AddLifecycleDistribution(ctx, params); err != nil {
			return fmt.Errorf("add distribution: %w", err)
		}
	}
	return nil
}

func visibleOnValues(values []domain.WebServiceVisibleOn) []sqlc.WebServiceVisibleOn {
	if values == nil {
		return nil
	}
	result := make([]sqlc.WebServiceVisibleOn, len(values))
	for i, value := range values {
		result[i] = sqlc.WebServiceVisibleOn(value)
	}
	return result
}

func lifecycleError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrDatasetConflict
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23514":
			return fmt.Errorf("database validation: %w", domain.ErrInvalidDataset)
		case "23001", "23505", "23P01", "P0001":
			return fmt.Errorf("database invariant %q: %w", pgErr.ConstraintName, domain.ErrDatasetConflict)
		default:
			if strings.HasPrefix(pgErr.Code, "23") {
				return fmt.Errorf("database integrity error %q: %w", pgErr.Code, domain.ErrDatasetInternal)
			}
		}
	}
	return err
}

func readDatasetMetadata(ctx context.Context, q *sqlc.Queries, version sqlc.GetLifecycleDatasetVersionRow) (domain.DatasetMetadata, error) {
	identifiers, err := q.ListLifecycleDatasetIdentifiers(ctx, &version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list identifiers: %w", err)
	}

	providers, err := q.ListLifecycleDatasetProviders(ctx, version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list providers: %w", err)
	}

	providerIDs := make([]uuid.UUID, len(providers))
	for i, provider := range providers {
		providerIDs[i] = provider.ID
	}

	providerContacts, err := q.ListLifecycleOrganizationContactPoints(ctx, providerIDs)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list provider contact points: %w", err)
	}

	contacts, err := q.ListLifecycleDatasetContactPoints(ctx, version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list contact points: %w", err)
	}

	categories, err := q.ListLifecycleDatasetCategories(ctx, version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list categories: %w", err)
	}

	services, err := readWebServices(ctx, q, version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, err
	}

	distributions, err := q.ListLifecycleDistributions(ctx, version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list distributions: %w", err)
	}
	result := domain.DatasetMetadata{
		ID:                    version.ID,
		Title:                 version.Title,
		Description:           version.Description,
		Keywords:              resourcedb.KeywordsFromString(version.Keywords),
		AccrualPeriodicity:    version.AccrualPeriodicity,
		QualityAnnotation:     version.QualityAnnotation,
		AccessRight:           version.AccessRight,
		Distributions:         nil,
		SpatialCoverage:       version.SpatialCoverage,
		TemporalCoverage:      temporal(version.TemporalStart, version.TemporalEnd),
		PersistentIdentifiers: nil,
		ContactPoints:         nil,
		Providers:             nil,
		Categories:            nil,
		SourceCreated:         version.SourceCreatedAt,
		SourceUpdated:         version.SourceUpdatedAt,
	}

	for _, item := range identifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, toDomainIdentifier(item.Identifier))
	}

	contactsByProviderID := make(map[uuid.UUID][]domain.ContactPoint, len(providers))
	for _, item := range providerContacts {
		contactsByProviderID[item.OrganizationID] = append(contactsByProviderID[item.OrganizationID], toDomainContactPoint(item.ContactPoint))
	}

	for _, item := range providers {
		provider := toDomainOrganizationMetadata(item)
		provider.ContactPoints = contactsByProviderID[item.ID]
		result.Providers = append(result.Providers, provider)
	}

	for _, item := range contacts {
		result.ContactPoints = append(result.ContactPoints, toDomainContactPoint(item))
	}

	for _, item := range categories {
		result.Categories = append(result.Categories, toDomainCategoryMetadata(item))
	}

	for _, item := range distributions {
		result.Distributions = append(result.Distributions, toDomainDistribution(item, services))
	}

	return result, nil
}

func readWebServices(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID) (map[uuid.UUID]domain.WebService, error) {
	rows, err := q.ListLifecycleWebServices(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("list web services: %w", err)
	}
	ids := make([]uuid.UUID, len(rows))
	result := make(map[uuid.UUID]domain.WebService, len(rows))
	providerIDs := make([]uuid.UUID, 0, len(rows))
	for i, row := range rows {
		service, err := toDomainWebService(row.WebService)
		if err != nil {
			return nil, err
		}
		result[service.ID] = service
		ids[i] = service.ID
		if row.WebService.ProviderOrganizationID != nil {
			providerIDs = append(providerIDs, *row.WebService.ProviderOrganizationID)
		}
	}
	if len(rows) == 0 {
		return result, nil
	}
	organizations, err := q.ListLifecycleOrganizations(ctx, providerIDs)
	if err != nil {
		return nil, fmt.Errorf("list web service providers: %w", err)
	}
	providerByID := make(map[uuid.UUID]domain.OrganizationMetadata, len(organizations))
	for _, item := range organizations {
		providerByID[item.ID] = toDomainOrganizationMetadata(item)
	}
	for _, row := range rows {
		if row.WebService.ProviderOrganizationID != nil {
			service := result[row.WebService.ID]
			provider := providerByID[*row.WebService.ProviderOrganizationID]
			service.Provider = &provider
			result[row.WebService.ID] = service
		}
	}
	contacts, err := q.ListLifecycleWebServiceContactPoints(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list web service contact points: %w", err)
	}
	for _, item := range contacts {
		service := result[item.WebServiceID]
		service.ContactPoints = append(service.ContactPoints, toDomainContactPoint(item.ContactPoint))
		result[item.WebServiceID] = service
	}
	bodies, err := q.ListLifecycleWebServiceBodies(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list web service bodies: %w", err)
	}
	bodyByID := make(map[uuid.UUID]sqlc.WebServiceBody, len(bodies))
	for _, body := range bodies {
		if _, exists := result[body.WebServiceID]; !exists {
			return nil, fmt.Errorf("body %s references missing web service %s", body.ID, body.WebServiceID)
		}
		bodyByID[body.ID] = body
		service := result[body.WebServiceID]
		service.Body = &domain.RequestBody{ContentType: body.ContentType, Template: body.Template}
		result[body.WebServiceID] = service
	}
	parameters, err := q.ListLifecycleWebServiceParameters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list web service parameters: %w", err)
	}
	templateIDs := make([]uuid.UUID, 0, len(parameters))
	for _, item := range parameters {
		if item.SchemaKind == sqlc.ParameterSchemaKindTEMPLATE {
			templateIDs = append(templateIDs, item.ID)
		}
	}
	templates, err := q.ListLifecycleParameterTemplates(ctx, templateIDs)
	if err != nil {
		return nil, fmt.Errorf("list parameter templates: %w", err)
	}
	templateByID := make(map[uuid.UUID]*sqlc.ParameterTemplate, len(templates))
	for i := range templates {
		templateByID[templates[i].ParameterID] = &templates[i]
	}
	children, err := q.ListLifecycleTemplateParameters(ctx, templateIDs)
	if err != nil {
		return nil, fmt.Errorf("list template parameters: %w", err)
	}
	childrenByParentID := make(map[uuid.UUID][]sqlc.Parameter, len(children))
	parameterIDs := make([]uuid.UUID, 0, len(parameters)+len(children))
	for _, parameter := range parameters {
		parameterIDs = append(parameterIDs, parameter.ID)
	}
	for _, child := range children {
		if child.TemplateParameterID == nil {
			return nil, fmt.Errorf("template parameter %s has no parent", child.ID)
		}
		childrenByParentID[*child.TemplateParameterID] = append(childrenByParentID[*child.TemplateParameterID], child)
		parameterIDs = append(parameterIDs, child.ID)
	}
	schemas, err := readParameterSchemas(ctx, q, parameterIDs)
	if err != nil {
		return nil, err
	}
	bodyParameters := make(map[uuid.UUID][]domain.Parameter, len(bodies))
	for _, item := range parameters {
		parameter, err := domainParameter(item, templateByID[item.ID], childrenByParentID[item.ID], schemas)
		if err != nil {
			return nil, fmt.Errorf("decode web service parameter: %w", err)
		}
		if item.WebServiceID != nil {
			service, exists := result[*item.WebServiceID]
			if !exists {
				return nil, fmt.Errorf("parameter %s references missing web service %s", item.ID, *item.WebServiceID)
			}
			var location domain.ParameterLocation
			if item.Location != nil {
				location = domain.ParameterLocation(*item.Location)
			}
			service.Parameters = append(service.Parameters, domain.WebServiceParameter{
				ID:        item.ID,
				Parameter: parameter,
				Property:  item.Property,
				Location:  location,
				Explode:   item.Explode,
			})
			result[*item.WebServiceID] = service
			continue
		}
		if item.WebServiceBodyID == nil {
			return nil, fmt.Errorf("parameter %s has no owner", item.ID)
		}
		if _, exists := bodyByID[*item.WebServiceBodyID]; !exists {
			return nil, fmt.Errorf("parameter %s references missing web service body %s", item.ID, *item.WebServiceBodyID)
		}
		bodyParameters[*item.WebServiceBodyID] = append(bodyParameters[*item.WebServiceBodyID], parameter)
	}
	for _, body := range bodies {
		service := result[body.WebServiceID]
		bodyValue := *service.Body
		bodyValue.Parameters = bodyParameters[body.ID]
		service.Body = &bodyValue
		result[body.WebServiceID] = service
	}
	for id, service := range result {
		parsed, err := domain.NewWebService(service)
		if err != nil {
			return nil, fmt.Errorf("parse web service %s: %w", id, err)
		}
		result[id] = parsed
	}
	return result, nil
}
