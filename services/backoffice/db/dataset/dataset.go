// Package dataset provides PostgreSQL persistence for dataset lifecycle data.
package dataset

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/dbtx"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twpayne/go-geom"
)

// Repo persists each lifecycle operation in one database transaction.
type Repo struct {
	db *pgxpool.Pool
}

// New creates a dataset repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{db: pool}
}

func (r *Repo) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value metadata.Dataset) (datasetlifecycle.Dataset, error) {
	var result datasetlifecycle.Dataset
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		dataset, err := q.CreateLifecycleDataset(ctx, sqlc.CreateLifecycleDatasetParams{
			SourceIri: value.SourceIRI,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create dataset: %w", err)
		}

		params, err := VersionParams(dataset.ID, actorID, value)
		if err != nil {
			return fmt.Errorf("build dataset version: %w", err)
		}

		version, err := q.CreateLifecycleDatasetVersion(ctx, params)
		if err != nil {
			return fmt.Errorf("create dataset version: %w", err)
		}

		if err := PersistDatasetChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist dataset metadata: %w", err)
		}
		result, err = datasetResultByID(ctx, q, version.DatasetID, version.ID)
		if err != nil {
			return err
		}
		result.Metadata.SourceIRI = dataset.SourceIri
		return nil
	})
	return result, lifecycleError(err)
}

func (r *Repo) ReplaceDraft(
	ctx context.Context,
	datasetID,
	versionID uuid.UUID,
	revision int32,
	value metadata.Dataset,
) (datasetlifecycle.Dataset, error) {
	var result datasetlifecycle.Dataset
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		params, err := ReplaceParams(datasetID, versionID, revision, value)
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
		if err := PersistDatasetChildren(ctx, q, version.ID, value); err != nil {
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
func (r *Repo) Get(ctx context.Context, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, error) {
	q := sqlc.New(r.db)
	version, err := q.GetLifecycleDatasetVersion(ctx, sqlc.GetLifecycleDatasetVersionParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrNotFound
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("get dataset version: %w", err)
	}
	return datasetResultFromRow(ctx, q, version)
}

func (r *Repo) GetPublished(ctx context.Context, datasetID uuid.UUID) (datasetlifecycle.Dataset, error) {
	version, err := sqlc.New(r.db).GetPublishedLifecycleDatasetVersion(ctx, datasetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrNotFound
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("get published dataset version: %w", err)
	}
	return r.Get(ctx, datasetID, version.ID)
}

func (r *Repo) GetActive(ctx context.Context, datasetID uuid.UUID) (datasetlifecycle.Dataset, error) {
	version, err := sqlc.New(r.db).GetActiveLifecycleDatasetVersion(ctx, datasetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrNotFound
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("get active dataset version: %w", err)
	}
	return r.Get(ctx, datasetID, version.ID)
}

func (r *Repo) Submit(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("submit dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).SubmitLifecycleDatasetVersion(ctx, sqlc.SubmitLifecycleDatasetVersionParams{
		ActorID: &actorID,
		ID:      versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrConflict
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("submit dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Withdraw(ctx context.Context, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("withdraw dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).WithdrawLifecycleDatasetVersion(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrConflict
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("withdraw dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Reject(ctx context.Context, actorID string, datasetID, versionID uuid.UUID, note string) (datasetlifecycle.Dataset, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("reject dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).RejectLifecycleDatasetVersion(ctx, sqlc.RejectLifecycleDatasetVersionParams{
		ActorID:       &actorID,
		RejectionNote: &note,
		ID:            versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrConflict
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("reject dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Reopen(ctx context.Context, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("reopen dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).ReopenLifecycleDatasetVersion(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrConflict
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("reopen dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Publish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("publish dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).PublishLifecycleDatasetVersion(ctx, sqlc.PublishLifecycleDatasetVersionParams{
		ActorID:   &actorID,
		VersionID: versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrConflict
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("publish dataset version: %w", err)
	}
	return datasetResultByID(ctx, sqlc.New(r.db), v.DatasetID, v.ID)
}

func (r *Repo) Unpublish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("unpublish dataset version: %w", err)
	}

	v, err := sqlc.New(r.db).UnpublishLifecycleDatasetVersion(ctx, sqlc.UnpublishLifecycleDatasetVersionParams{
		ActorID: &actorID,
		ID:      versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return datasetlifecycle.Dataset{}, datasetlifecycle.ErrConflict
		}
		return datasetlifecycle.Dataset{}, fmt.Errorf("unpublish dataset version: %w", err)
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
		return datasetlifecycle.ErrConflict
	}
	return fmt.Errorf("discard dataset transaction: %w", err)
}

func (r *Repo) DraftFromPublication(ctx context.Context, actorID string, datasetID uuid.UUID) (datasetlifecycle.Dataset, error) {
	return r.clone(ctx, actorID, datasetID, datasetID, sqlc.ResourceStatusPUBLISHED, uuid.Nil, true)
}

func (r *Repo) DraftFromLatestArchive(ctx context.Context, actorID string, datasetID uuid.UUID) (datasetlifecycle.Dataset, error) {
	return r.clone(ctx, actorID, datasetID, datasetID, sqlc.ResourceStatusARCHIVED, uuid.Nil, true)
}

func (r *Repo) CopyPublished(ctx context.Context, actorID string, sourceDatasetID, targetGroupID uuid.UUID, sourceIRI string) (datasetlifecycle.Dataset, error) {
	var result datasetlifecycle.Dataset
	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		source, err := q.GetPublishedLifecycleDatasetVersion(ctx, sourceDatasetID)
		if err != nil {
			return fmt.Errorf("get published source version: %w", err)
		}
		logical, err := q.CreateLifecycleDataset(ctx, sqlc.CreateLifecycleDatasetParams{
			SourceIri: sourceIRI,
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
		result.Metadata.SourceIRI = logical.SourceIri
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
) (datasetlifecycle.Dataset, error) {
	var result datasetlifecycle.Dataset
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
		return datasetlifecycle.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("ensure dataset version: %w", err)
	}
	return nil
}

func VersionParams(datasetID uuid.UUID, actorID string, value metadata.Dataset) (sqlc.CreateLifecycleDatasetVersionParams, error) {
	var start, end *time.Time
	if value.TemporalCoverage != nil {
		start, end = value.TemporalCoverage.Start, value.TemporalCoverage.End
	}
	point, polygon, err := spatialGeometry(value.SpatialCoverage)
	if err != nil {
		return sqlc.CreateLifecycleDatasetVersionParams{}, fmt.Errorf("encode dataset geometry: %w", err)
	}
	return sqlc.CreateLifecycleDatasetVersionParams{
		DatasetID:          datasetID,
		Title:              value.Title,
		Description:        value.Description,
		Keywords:           value.Keywords,
		AccrualPeriodicity: value.AccrualPeriodicity,
		QualityAnnotation:  value.QualityAnnotation,
		AccessRight:        value.AccessRight,
		SourceCreatedAt:    value.SourceCreated,
		SourceUpdatedAt:    value.SourceUpdated,
		SpatialPoint:       point,
		SpatialPolygon:     polygon,
		TemporalStart:      start,
		TemporalEnd:        end,
		CreatedBy:          actorID,
	}, nil
}

func ReplaceParams(datasetID, id uuid.UUID, revision int32, value metadata.Dataset) (sqlc.ReplaceLifecycleDatasetDraftParams, error) {
	p, err := VersionParams(uuid.Nil, "", value)
	if err != nil {
		return sqlc.ReplaceLifecycleDatasetDraftParams{}, err
	}
	return sqlc.ReplaceLifecycleDatasetDraftParams{
		Title:              p.Title,
		Description:        p.Description,
		Keywords:           p.Keywords,
		AccrualPeriodicity: p.AccrualPeriodicity,
		QualityAnnotation:  p.QualityAnnotation,
		AccessRight:        p.AccessRight,
		SourceCreatedAt:    p.SourceCreatedAt,
		SourceUpdatedAt:    p.SourceUpdatedAt,
		SpatialPoint:       p.SpatialPoint,
		SpatialPolygon:     p.SpatialPolygon,
		TemporalStart:      p.TemporalStart,
		TemporalEnd:        p.TemporalEnd,
		DatasetID:          datasetID,
		ID:                 id,
		Revision:           revision,
	}, nil
}

func datasetResultByID(ctx context.Context, q *sqlc.Queries, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, error) {
	row, err := q.GetLifecycleDatasetVersion(ctx, sqlc.GetLifecycleDatasetVersionParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("get dataset result: %w", err)
	}
	return datasetResultFromRow(ctx, q, row)
}

func datasetResultFromRow(ctx context.Context, q *sqlc.Queries, version sqlc.GetLifecycleDatasetVersionRow) (datasetlifecycle.Dataset, error) {
	metadataValue, err := readDatasetMetadata(ctx, q, version)
	if err != nil {
		return datasetlifecycle.Dataset{}, fmt.Errorf("read dataset metadata: %w", err)
	}
	return datasetlifecycle.Dataset{
		DatasetID: version.DatasetID,
		VersionID: version.ID,
		Revision:  version.Revision,
		Status:    metadata.DatasetStatus(version.Status),
		CreatedBy: version.CreatedBy,
		Group:     categoryScheme(version.Group),
		Metadata:  metadataValue,
	}, nil
}

func temporal(start, end *time.Time) *metadata.Temporal {
	if start == nil && end == nil {
		return nil
	}
	return &metadata.Temporal{Start: start, End: end}
}

func PersistDatasetChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value metadata.Dataset) error {
	for _, identifier := range value.PersistentIdentifiers {
		if err := q.AddLifecycleDatasetIdentifier(ctx, sqlc.AddLifecycleDatasetIdentifierParams{
			SourceIri:        identifier.SourceIRI,
			Type:             string(identifier.Type),
			Value:            identifier.Value,
			DatasetVersionID: &versionID,
		}); err != nil {
			return fmt.Errorf("add identifier: %w", err)
		}
	}
	for _, provider := range value.Providers {
		if err := q.AddLifecycleDatasetProvider(ctx, sqlc.AddLifecycleDatasetProviderParams{
			DatasetVersionID: versionID,
			OrganizationID:   provider.ID,
		}); err != nil {
			return fmt.Errorf("add provider: %w", err)
		}
	}
	for _, contact := range value.ContactPoints {
		if err := q.AddLifecycleDatasetContactPoint(ctx, sqlc.AddLifecycleDatasetContactPointParams{
			DatasetVersionID: versionID,
			ContactPointID:   contact.ID,
		}); err != nil {
			return fmt.Errorf("add contact point: %w", err)
		}
	}
	for _, category := range value.Categories {
		if err := q.AddLifecycleDatasetCategory(ctx, sqlc.AddLifecycleDatasetCategoryParams{
			DatasetVersionID: versionID,
			CategoryID:       category.ID,
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
		point, polygon, err := spatialGeometry(service.SpatialCoverage)
		if err != nil {
			return fmt.Errorf("encode web service geometry: %w", err)
		}
		var start, end *time.Time
		if service.TemporalCoverage != nil {
			start, end = service.TemporalCoverage.Start, service.TemporalCoverage.End
		}
		var providerID *uuid.UUID
		if service.Provider != nil {
			providerID = &service.Provider.ID
		}
		var title, description, uri *string
		if service.Documentation != nil {
			title, description, uri = service.Documentation.Title, service.Documentation.Description, &service.Documentation.URI
		}
		created, err := q.AddLifecycleWebService(ctx, sqlc.AddLifecycleWebServiceParams{
			DatasetVersionID:         versionID,
			Name:                     service.Name,
			Description:              service.Description,
			TemplateUrl:              service.TemplateURL,
			Format:                   service.Format,
			ProviderOrganizationID:   providerID,
			DocumentationTitle:       title,
			DocumentationDescription: description,
			DocumentationUri:         uri,
			SpatialPoint:             point,
			SpatialPolygon:           polygon,
			TemporalStart:            start,
			TemporalEnd:              end,
		})
		if err != nil {
			return fmt.Errorf("add web service: %w", err)
		}
		services[i] = &created.ID
		for _, contact := range service.ContactPoints {
			if err := q.AddLifecycleWebServiceContactPoint(ctx, sqlc.AddLifecycleWebServiceContactPointParams{
				WebServiceID:   created.ID,
				ContactPointID: contact.ID,
			}); err != nil {
				return fmt.Errorf("add web service contact point: %w", err)
			}
		}
		for _, parameter := range service.Parameters {
			record, err := encodeParameter(parameter)
			if err != nil {
				return fmt.Errorf("encode web service parameter: %w", err)
			}
			m := record.metadata
			if err := q.AddLifecycleWebServiceParameter(ctx, sqlc.AddLifecycleWebServiceParameterParams{
				WebServiceID:   created.ID,
				Kind:           sqlc.ParameterKind(record.kind),
				Variable:       m.Variable,
				Property:       m.Property,
				Label:          m.Label,
				Required:       m.Required,
				MultipleValues: m.MultipleValues,
				ReadOnly:       m.ReadOnly,
				Config:         record.config,
			}); err != nil {
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
			if *distribution.ByteSize > int64(^uint32(0)>>1) || *distribution.ByteSize < -int64(^uint32(0)>>1)-1 {
				return fmt.Errorf("distribution byte size out of range")
			}
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

func lifecycleError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return datasetlifecycle.ErrConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (strings.HasPrefix(pgErr.Code, "23") || pgErr.Code == "P0001") {
		return fmt.Errorf("database invariant %q: %w", pgErr.ConstraintName, datasetlifecycle.ErrConflict)
	}
	return err
}

func spatialGeometry(value *metadata.Spatial) (*geom.Point, *geom.Polygon, error) {
	if value == nil {
		return nil, nil, nil
	}
	if value.Point != nil && len(value.Polygon) != 0 {
		return nil, nil, fmt.Errorf("spatial coverage cannot contain both point and polygon")
	}
	if value.Point != nil {
		point := geom.NewPointFlat(geom.XY, []float64{value.Point.Longitude, value.Point.Latitude})
		point.SetSRID(4326)
		return point, nil, nil
	}
	if len(value.Polygon) == 0 {
		return nil, nil, nil
	}
	coordinates := make([]geom.Coord, len(value.Polygon))
	for i, coordinate := range value.Polygon {
		coordinates[i] = geom.Coord{coordinate.Longitude, coordinate.Latitude}
	}

	polygon := geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{coordinates})
	polygon.SetSRID(4326)

	return nil, polygon, nil
}

func readDatasetMetadata(ctx context.Context, q *sqlc.Queries, version sqlc.GetLifecycleDatasetVersionRow) (metadata.Dataset, error) {
	spatial, err := spatialFromGeometry(version.SpatialPoint, version.SpatialPolygon)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("decode dataset geometry: %w", err)
	}

	identifiers, err := q.ListLifecycleDatasetIdentifiers(ctx, &version.ID)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("list identifiers: %w", err)
	}

	providers, err := q.ListLifecycleDatasetProviders(ctx, version.ID)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("list providers: %w", err)
	}

	providerIDs := make([]uuid.UUID, len(providers))
	for i, provider := range providers {
		providerIDs[i] = provider.ID
	}

	providerContacts, err := q.ListLifecycleOrganizationContactPoints(ctx, providerIDs)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("list provider contact points: %w", err)
	}

	contacts, err := q.ListLifecycleDatasetContactPoints(ctx, version.ID)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("list contact points: %w", err)
	}

	categories, err := q.ListLifecycleDatasetCategories(ctx, version.ID)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("list categories: %w", err)
	}

	services, err := readWebServices(ctx, q, version.ID)
	if err != nil {
		return metadata.Dataset{}, err
	}

	distributions, err := q.ListLifecycleDistributions(ctx, version.ID)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("list distributions: %w", err)
	}

	result := metadata.Dataset{
		ID:                    version.ID,
		SourceIRI:             version.SourceIri,
		Title:                 version.Title,
		Description:           version.Description,
		Keywords:              version.Keywords,
		AccrualPeriodicity:    version.AccrualPeriodicity,
		QualityAnnotation:     version.QualityAnnotation,
		AccessRight:           version.AccessRight,
		Distributions:         nil,
		SpatialCoverage:       spatial,
		TemporalCoverage:      temporal(version.TemporalStart, version.TemporalEnd),
		PersistentIdentifiers: nil,
		ContactPoints:         nil,
		Providers:             nil,
		Categories:            nil,
		SourceCreated:         version.SourceCreatedAt,
		SourceUpdated:         version.SourceUpdatedAt,
		Status:                metadata.DatasetStatus(version.Status),
		Created:               version.CreatedAt,
		Updated:               version.UpdatedAt,
	}

	for _, item := range identifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, identifier(item.Identifier))
	}

	contactsByProviderID := make(map[uuid.UUID][]metadata.ContactPoint, len(providers))
	for _, item := range providerContacts {
		contactsByProviderID[item.OrganizationID] = append(contactsByProviderID[item.OrganizationID], contactPoint(item.ContactPoint))
	}

	for _, item := range providers {
		provider := organization(item)
		provider.ContactPoints = contactsByProviderID[item.ID]
		result.Providers = append(result.Providers, provider)
	}

	for _, item := range contacts {
		result.ContactPoints = append(result.ContactPoints, contactPoint(item))
	}

	for _, item := range categories {
		result.Categories = append(result.Categories, metadata.Category{
			ID:        item.ID,
			SourceIRI: item.SourceIri,
			Name:      item.Name,
			Scheme: metadata.CategoryScheme{
				ID:          uuid.Nil,
				SourceIRI:   "",
				Name:        "",
				Description: nil,
				Label:       nil,
				Logo:        nil,
				Page:        nil,
				Color:       nil,
				Position:    0,
				Concept:     "",
			},
			Children: nil,
		})
	}

	for _, item := range distributions {
		distribution := metadata.Distribution{
			ID:               item.ID,
			Title:            item.Title,
			Description:      item.Description,
			License:          item.License,
			Type:             metadata.DistributionType(item.Type),
			Issued:           item.SourceIssuedAt,
			Modified:         item.SourceModifiedAt,
			ByteSize:         nil,
			WebService:       nil,
			DownloadableFile: nil,
		}
		if item.ByteSize != nil {
			byteSize := int64(*item.ByteSize)
			distribution.ByteSize = &byteSize
		}
		if item.WebServiceID != nil {
			service := services[*item.WebServiceID]
			distribution.WebService = &service
		} else {
			distribution.DownloadableFile = &metadata.DownloadableFile{
				URL:    *item.DownloadableFileUrl,
				Format: *item.DownloadableFileFormat,
			}
		}
		result.Distributions = append(result.Distributions, distribution)
	}

	return result, nil
}

func readWebServices(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID) (map[uuid.UUID]metadata.WebService, error) {
	rows, err := q.ListLifecycleWebServices(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("list web services: %w", err)
	}
	ids := make([]uuid.UUID, len(rows))
	result := make(map[uuid.UUID]metadata.WebService, len(rows))
	providerIDs := make([]uuid.UUID, 0, len(rows))
	for i, row := range rows {
		spatial, err := spatialFromGeometry(row.WebService.SpatialPoint, row.WebService.SpatialPolygon)
		if err != nil {
			return nil, fmt.Errorf("decode web service geometry: %w", err)
		}
		var documentation *metadata.Documentation
		if row.WebService.DocumentationUri != nil {
			documentation = &metadata.Documentation{
				Title:       row.WebService.DocumentationTitle,
				Description: row.WebService.DocumentationDescription,
				URI:         *row.WebService.DocumentationUri,
			}
		}
		result[row.WebService.ID] = metadata.WebService{
			ID:               row.WebService.ID,
			Name:             row.WebService.Name,
			Description:      row.WebService.Description,
			TemplateURL:      row.WebService.TemplateUrl,
			Parameters:       nil,
			Format:           row.WebService.Format,
			Provider:         nil,
			Documentation:    documentation,
			SpatialCoverage:  spatial,
			TemporalCoverage: temporal(row.WebService.TemporalStart, row.WebService.TemporalEnd),
			ContactPoints:    nil,
		}
		ids[i] = row.WebService.ID
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
	providerByID := make(map[uuid.UUID]metadata.Organization, len(organizations))
	for _, item := range organizations {
		providerByID[item.ID] = organization(item)
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
		service.ContactPoints = append(service.ContactPoints, contactPoint(item.ContactPoint))
		result[item.WebServiceID] = service
	}
	parameters, err := q.ListLifecycleWebServiceParameters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list web service parameters: %w", err)
	}
	for _, item := range parameters {
		parameter, err := decodeParameter(parameterRecord{
			kind: metadata.ParameterKind(item.Kind),
			metadata: metadata.ParameterMetadata{
				ID:             item.ID,
				Kind:           metadata.ParameterKind(item.Kind),
				Variable:       item.Variable,
				Property:       item.Property,
				Label:          item.Label,
				Required:       item.Required,
				MultipleValues: item.MultipleValues,
				ReadOnly:       item.ReadOnly,
			},
			config: item.Config,
		})
		if err != nil {
			return nil, fmt.Errorf("decode web service parameter: %w", err)
		}
		service := result[item.WebServiceID]
		service.Parameters = append(service.Parameters, parameter)
		result[item.WebServiceID] = service
	}
	return result, nil
}

func spatialFromGeometry(point *geom.Point, polygon *geom.Polygon) (*metadata.Spatial, error) {
	if point == nil && polygon == nil {
		return nil, nil
	}
	if point != nil {
		return &metadata.Spatial{
			Point: &metadata.Coordinate{
				Longitude: point.X(),
				Latitude:  point.Y(),
			},
			Polygon: nil,
		}, nil
	}
	rings := polygon.Coords()
	if len(rings) == 0 {
		return nil, fmt.Errorf("invalid polygon geometry")
	}
	coordinates := make([]metadata.Coordinate, len(rings[0]))
	for i, coordinate := range rings[0] {
		coordinates[i] = metadata.Coordinate{Longitude: coordinate.X(), Latitude: coordinate.Y()}
	}
	return &metadata.Spatial{Point: nil, Polygon: coordinates}, nil
}

func organization(value sqlc.Organization) metadata.Organization {
	return metadata.Organization{
		ID:            value.ID,
		SourceIRI:     value.SourceIri,
		Name:          value.Name,
		Street:        value.Street,
		Locality:      value.Locality,
		PostalCode:    value.PostalCode,
		Country:       value.Country,
		LogoURL:       value.LogoUrl,
		PageURL:       value.PageUrl,
		Email:         value.Email,
		LEICode:       value.LeiCode,
		ContactPoints: nil,
		Identifiers:   nil,
	}
}

func contactPoint(value sqlc.ContactPoint) metadata.ContactPoint {
	return metadata.ContactPoint{
		ID:        value.ID,
		SourceIRI: value.SourceIri,
		Email:     value.Email,
		Role:      metadata.ContactPointRole(value.Role),
	}
}

func identifier(value sqlc.Identifier) metadata.Identifier {
	return metadata.Identifier{
		ID:        value.ID,
		SourceIRI: value.SourceIri,
		Type:      metadata.IdentifierType(value.Type),
		Value:     value.Value,
	}
}

func categoryScheme(value sqlc.Group) metadata.CategoryScheme {
	return metadata.CategoryScheme{
		ID:          value.ID,
		SourceIRI:   value.SourceIri,
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		Concept:     metadata.TopLevelConcept(value.Concept),
	}
}
