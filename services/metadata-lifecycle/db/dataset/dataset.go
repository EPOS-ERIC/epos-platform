// Package dataset provides PostgreSQL persistence for dataset lifecycle data.
package dataset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	resourcedb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/resource"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo persists each lifecycle operation in one database transaction.
type Repo struct {
	db  *pgxpool.Pool
	q   *sqlc.Queries
	log *slog.Logger
}

// New creates a dataset repository.
func New(log *slog.Logger, pool *pgxpool.Pool) *Repo {
	return &Repo{
		db:  pool,
		q:   sqlc.New(pool),
		log: log,
	}
}

func (r *Repo) CreateDraft(ctx context.Context, actorID string, groupID uuid.UUID, value domain.DatasetInput) (domain.DatasetVersion, error) {
	var result domain.DatasetVersion
	err := dbtx.Run(ctx, r.db, r.q, func(q *sqlc.Queries) error {
		if _, err := resourcedb.IdentifierIdentities(value.PersistentIdentifiers, nil, "metadata.persistent_identifiers"); err != nil {
			return err
		}
		distributions, err := distributionIdentities(value.Distributions, nil)
		if err != nil {
			return err
		}
		parameters, err := parameterIdentities(value.Distributions, nil)
		if err != nil {
			return err
		}
		layers, err := layerIdentities(value.Distributions, nil)
		if err != nil {
			return err
		}
		if err := validateDatasetCategories(ctx, q, groupID, value); err != nil {
			return err
		}

		dataset, err := q.CreateLifecycleDataset(ctx, sqlc.CreateLifecycleDatasetParams{
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create dataset: %w", err)
		}

		version, err := q.CreateLifecycleDatasetVersion(ctx, ToCreateVersionParams(dataset.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create dataset version: %w", err)
		}

		if err := persistChildren(ctx, q, version.ID, value, nil, distributions, parameters, layers); err != nil {
			return fmt.Errorf("persist dataset metadata: %w", err)
		}
		result, err = datasetResultByID(ctx, q, version.DatasetID, version.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, resourcedb.LifecycleError(err)
}

func (r *Repo) ReplaceDraft(
	ctx context.Context,
	datasetID,
	versionID uuid.UUID,
	value domain.DatasetDraftInput,
) (domain.DatasetVersion, []uuid.UUID, error) {
	var result domain.DatasetVersion
	var removed []uuid.UUID
	err := dbtx.Run(ctx, r.db, r.q, func(q *sqlc.Queries) error {
		if len(value.Metadata.CategoryIDs) > 0 {
			dataset, err := q.GetLifecycleDataset(ctx, datasetID)
			if err != nil {
				return fmt.Errorf("get dataset for category validation: %w", err)
			}
			if err := validateDatasetCategories(ctx, q, dataset.GroupID, value.Metadata); err != nil {
				return err
			}
		}

		version, err := q.ReplaceLifecycleDatasetDraft(ctx, ToReplaceDraftParams(datasetID, versionID, value.Revision, value.Metadata))
		if err != nil {
			return fmt.Errorf("replace draft: %w", err)
		}
		current, err := datasetResultByID(ctx, q, datasetID, versionID)
		if err != nil {
			return err
		}
		identities, err := resourcedb.IdentifierIdentities(value.Metadata.PersistentIdentifiers, current.Metadata.PersistentIdentifiers, "metadata.persistent_identifiers")
		if err != nil {
			return err
		}
		distributions, err := distributionIdentities(value.Metadata.Distributions, current.Metadata.Distributions)
		if err != nil {
			return err
		}
		retained := make(map[uuid.UUID]struct{}, len(distributions))
		for _, distribution := range distributions {
			retained[distribution.id] = struct{}{}
		}
		for _, distribution := range current.Metadata.Distributions {
			if _, ok := retained[distribution.ID]; !ok {
				removed = append(removed, distribution.ID)
			}
		}
		parameters, err := parameterIdentities(value.Metadata.Distributions, current.Metadata.Distributions)
		if err != nil {
			return err
		}
		layers, err := layerIdentities(value.Metadata.Distributions, current.Metadata.Distributions)
		if err != nil {
			return err
		}

		if err := q.DeleteLifecycleDatasetVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("delete draft children: %w", err)
		}
		if err := persistChildren(ctx, q, version.ID, value.Metadata, identities, distributions, parameters, layers); err != nil {
			return fmt.Errorf("persist draft metadata: %w", err)
		}
		result, err = datasetResultByID(ctx, q, version.DatasetID, version.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, removed, resourcedb.LifecycleError(err)
}

func validateDatasetCategories(ctx context.Context, q *sqlc.Queries, groupID uuid.UUID, value domain.DatasetInput) error {
	invalidIndex, err := resourcedb.CategoriesBelongToGroup(ctx, q, groupID, value.CategoryIDs)
	if err != nil {
		return fmt.Errorf("validate dataset categories: %w", err)
	}
	if invalidIndex >= 0 {
		return apperror.Field(fmt.Sprintf("metadata.categories[%d]", invalidIndex), "Category must belong to this group and appear only once.", nil)
	}
	return nil
}

// Get returns one selected version with all version-owned metadata and canonical references.
func (r *Repo) Get(ctx context.Context, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	q := r.q
	version, err := q.GetLifecycleDatasetVersion(ctx, sqlc.GetLifecycleDatasetVersionParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeNotFound)
		}
		return domain.DatasetVersion{}, fmt.Errorf("get dataset version: %w", err)
	}
	return datasetResultFromRow(ctx, q, version)
}

func (r *Repo) GetDistributionDatasetVersion(ctx context.Context, distributionID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	row, err := r.q.GetDistributionLifecycleDatasetVersion(ctx, distributionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, uuid.Nil, apperror.Default(apperror.CodeNotFound)
		}

		return uuid.Nil, uuid.Nil, fmt.Errorf("get dataset version for distribution: %w", err)
	}

	return row.DatasetID, row.VersionID, nil
}

func (r *Repo) List(ctx context.Context, datasetID uuid.UUID) ([]domain.ResourceVersionSummary, error) {
	rows, err := r.q.ListLifecycleDatasetVersionSummaries(ctx, datasetID)
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
	version, err := r.q.GetPublishedLifecycleDatasetVersion(ctx, datasetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeNotFound)
		}
		return domain.DatasetVersion{}, fmt.Errorf("get published dataset version: %w", err)
	}
	return r.Get(ctx, datasetID, version.ID)
}

func (r *Repo) GetActive(ctx context.Context, datasetID uuid.UUID) (domain.DatasetVersion, error) {
	version, err := r.q.GetActiveLifecycleDatasetVersion(ctx, datasetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeNotFound)
		}
		return domain.DatasetVersion{}, fmt.Errorf("get active dataset version: %w", err)
	}
	return r.Get(ctx, datasetID, version.ID)
}

func (r *Repo) Submit(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("submit dataset version: %w", err)
	}

	v, err := r.q.SubmitLifecycleDatasetVersion(ctx, sqlc.SubmitLifecycleDatasetVersionParams{
		ActorID: &actorID,
		ID:      versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeConflict)
		}
		return domain.DatasetVersion{}, fmt.Errorf("submit dataset version: %w", err)
	}
	return datasetResultByID(ctx, r.q, v.DatasetID, v.ID)
}

func (r *Repo) Withdraw(ctx context.Context, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("withdraw dataset version: %w", err)
	}

	v, err := r.q.WithdrawLifecycleDatasetVersion(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeConflict)
		}
		return domain.DatasetVersion{}, fmt.Errorf("withdraw dataset version: %w", err)
	}
	return datasetResultByID(ctx, r.q, v.DatasetID, v.ID)
}

func (r *Repo) Reject(ctx context.Context, actorID string, datasetID, versionID uuid.UUID, value domain.RejectionInput) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("reject dataset version: %w", err)
	}

	v, err := r.q.RejectLifecycleDatasetVersion(ctx, sqlc.RejectLifecycleDatasetVersionParams{
		ActorID:       &actorID,
		RejectionNote: &value.Note,
		ID:            versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeConflict)
		}
		return domain.DatasetVersion{}, fmt.Errorf("reject dataset version: %w", err)
	}
	return datasetResultByID(ctx, r.q, v.DatasetID, v.ID)
}

func (r *Repo) Reopen(ctx context.Context, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("reopen dataset version: %w", err)
	}

	v, err := r.q.ReopenLifecycleDatasetVersion(ctx, versionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeConflict)
		}
		return domain.DatasetVersion{}, fmt.Errorf("reopen dataset version: %w", err)
	}
	return datasetResultByID(ctx, r.q, v.DatasetID, v.ID)
}

func (r *Repo) Publish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("publish dataset version: %w", err)
	}

	var result domain.DatasetVersion
	err := dbtx.Run(ctx, r.db, r.q, func(q *sqlc.Queries) error {
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
				return apperror.Default(apperror.CodeConflict)
			}
			return fmt.Errorf("publish dataset version: %w", err)
		}

		result, err = datasetResultByID(ctx, q, datasetID, v.ID)
		return err
	})
	if err != nil {
		return domain.DatasetVersion{}, resourcedb.LifecycleError(err)
	}
	return result, nil
}

func (r *Repo) Unpublish(ctx context.Context, actorID string, datasetID, versionID uuid.UUID) (domain.DatasetVersion, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("unpublish dataset version: %w", err)
	}

	v, err := r.q.UnpublishLifecycleDatasetVersion(ctx, sqlc.UnpublishLifecycleDatasetVersionParams{
		ActorID: &actorID,
		ID:      versionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DatasetVersion{}, apperror.Default(apperror.CodeConflict)
		}
		return domain.DatasetVersion{}, fmt.Errorf("unpublish dataset version: %w", err)
	}
	return datasetResultByID(ctx, r.q, v.DatasetID, v.ID)
}

func (r *Repo) Discard(ctx context.Context, datasetID, versionID uuid.UUID) ([]uuid.UUID, error) {
	if err := r.ensureVersion(ctx, datasetID, versionID); err != nil {
		return nil, fmt.Errorf("discard dataset: %w", err)
	}

	var removed []uuid.UUID
	err := dbtx.Run(ctx, r.db, r.q, func(q *sqlc.Queries) error {
		distributions, err := q.ListLifecycleDistributions(ctx, versionID)
		if err != nil {
			return fmt.Errorf("list discarded distributions: %w", err)
		}
		for _, distribution := range distributions {
			removed = append(removed, distribution.ID)
		}
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
		return nil, apperror.Default(apperror.CodeConflict)
	}
	if err != nil {
		return nil, fmt.Errorf("discard dataset transaction: %w", err)
	}

	return removed, nil
}

func (r *Repo) DraftFromPublication(ctx context.Context, actorID string, datasetID uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error) {
	return r.clone(ctx, actorID, datasetID, sqlc.ResourceStatusPUBLISHED)
}

func (r *Repo) DraftFromLatestArchive(ctx context.Context, actorID string, datasetID uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error) {
	return r.clone(ctx, actorID, datasetID, sqlc.ResourceStatusARCHIVED)
}

func (r *Repo) CopyPublished(ctx context.Context, actorID string, sourceDatasetID, targetGroupID uuid.UUID) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error) {
	var result domain.DatasetVersion
	var pairs map[uuid.UUID]uuid.UUID
	err := dbtx.Run(ctx, r.db, r.q, func(q *sqlc.Queries) error {
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
		})
		if err != nil {
			return fmt.Errorf("clone published dataset version: %w", err)
		}
		pairs, err = clonedDistributions(ctx, q, source.ID, version.ID)
		if err != nil {
			return err
		}
		result, err = datasetResultByID(ctx, q, version.DatasetID, version.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, pairs, resourcedb.LifecycleError(err)
}

func (r *Repo) clone(
	ctx context.Context,
	actorID string,
	datasetID uuid.UUID,
	status sqlc.ResourceStatus,
) (domain.DatasetVersion, map[uuid.UUID]uuid.UUID, error) {
	var result domain.DatasetVersion
	var pairs map[uuid.UUID]uuid.UUID
	err := dbtx.Run(ctx, r.db, r.q, func(q *sqlc.Queries) error {
		var source sqlc.DatasetVersion
		var err error
		if status == sqlc.ResourceStatusPUBLISHED {
			source, err = q.GetPublishedLifecycleDatasetVersion(ctx, datasetID)
		} else {
			source, err = q.GetLatestArchivedLifecycleDatasetVersion(ctx, datasetID)
		}
		if err != nil {
			return fmt.Errorf("get source dataset version: %w", err)
		}
		v, err := q.CloneLifecycleDatasetVersion(ctx, sqlc.CloneLifecycleDatasetVersionParams{
			SourceVersionID: source.ID,
			SourceStatus:    status,
			TargetDatasetID: datasetID,
			ActorID:         actorID,
		})
		if err != nil {
			return fmt.Errorf("clone dataset version: %w", err)
		}
		pairs, err = clonedDistributions(ctx, q, source.ID, v.ID)
		if err != nil {
			return err
		}
		result, err = datasetResultByID(ctx, q, v.DatasetID, v.ID)
		if err != nil {
			return err
		}
		return nil
	})
	return result, pairs, resourcedb.LifecycleError(err)
}

func clonedDistributions(ctx context.Context, q *sqlc.Queries, sourceID, targetID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := q.ListClonedLifecycleDistributions(ctx, sqlc.ListClonedLifecycleDistributionsParams{SourceVersionID: sourceID, TargetVersionID: targetID})
	if err != nil {
		return nil, fmt.Errorf("list cloned distributions: %w", err)
	}

	pairs := make(map[uuid.UUID]uuid.UUID, len(rows))
	for _, row := range rows {
		pairs[row.SourceID] = row.TargetID
	}

	return pairs, nil
}

func (r *Repo) ensureVersion(ctx context.Context, datasetID, versionID uuid.UUID) error {
	_, err := r.q.EnsureLifecycleDatasetVersion(ctx, sqlc.EnsureLifecycleDatasetVersionParams{
		DatasetID: datasetID,
		VersionID: versionID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Default(apperror.CodeNotFound)
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
	return persistChildren(ctx, q, versionID, value, nil, nil, nil, nil)
}

func persistChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value domain.DatasetInput, identities map[uuid.UUID]domain.Identifier, distributions []distributionIdentity, parameters map[uuid.UUID]domain.Parameter, layers map[uuid.UUID]uuid.UUID) error {
	if distributions == nil {
		distributions = make([]distributionIdentity, len(value.Distributions))
		for i := range distributions {
			distributions[i] = distributionIdentity{id: uuid.New(), resourceID: uuid.New(), serviceID: uuid.New(), serviceResourceID: uuid.New()}
		}
	}
	for _, identifier := range value.PersistentIdentifiers {
		id, resourceID := resourcedb.IdentifierIDs(identifier, identities)
		if err := q.AddLifecycleDatasetIdentifier(ctx, sqlc.AddLifecycleDatasetIdentifierParams{
			ID:               id,
			ResourceID:       resourceID,
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

	for i, distribution := range value.Distributions {
		if distribution.WebService == nil {
			continue
		}
		service := distribution.WebService
		common := service.CommonInput()
		serviceID := distributions[i].serviceID
		err := q.AddLifecycleWebService(ctx, sqlc.AddLifecycleWebServiceParams{
			ID:                     serviceID,
			ResourceID:             distributions[i].serviceResourceID,
			DatasetVersionID:       versionID,
			Type:                   sqlc.WebServiceType(service.ServiceType()),
			VisibleOn:              visibleOnValues(common.VisibleOn),
			ProviderOrganizationID: common.ProviderID,
			DocumentationUri:       common.DocumentationURI,
			Keywords:               common.Keywords,
		})
		if err != nil {
			return fmt.Errorf("add web service: %w", err)
		}
		for _, contactID := range common.ContactPointIDs {
			if err := q.AddLifecycleWebServiceContactPoint(ctx, sqlc.AddLifecycleWebServiceContactPointParams{
				WebServiceID:   serviceID,
				ContactPointID: contactID,
			}); err != nil {
				return fmt.Errorf("add web service contact point: %w", err)
			}
		}
		switch configuration := service.(type) {
		case *domain.HTTPWebServiceInput, *domain.DynamicWebServiceInput:
			generic := genericWebServiceInput(service)
			if err := q.AddLifecycleGenericWebService(ctx, sqlc.AddLifecycleGenericWebServiceParams{
				WebServiceID: serviceID,
				TemplateUrl:  generic.TemplateURL,
				Method:       sqlc.WebServiceHttpMethod(generic.Method),
				Format:       generic.Format,
			}); err != nil {
				return fmt.Errorf("add generic web service configuration: %w", err)
			}
			var bodyID *uuid.UUID
			if generic.Body != nil {
				body, err := q.AddLifecycleWebServiceBody(ctx, sqlc.AddLifecycleWebServiceBodyParams{
					WebServiceID: serviceID,
					ContentType:  generic.Body.ContentType,
					Template:     generic.Body.Template,
					BodyRequired: *generic.Body.Required,
				})
				if err != nil {
					return fmt.Errorf("add web service body: %w", err)
				}
				bodyID = &body.ID
				for _, parameter := range generic.Body.Parameters {
					if _, err := persistParameter(ctx, q, parameter, parameterOwner{webServiceBodyID: bodyID}, nil, false, parameters); err != nil {
						return fmt.Errorf("add web service body parameter: %w", err)
					}
				}
			}
			for _, parameter := range generic.Parameters {
				location := parameter.Location
				if _, err := persistParameter(ctx, q, parameter.ParameterInput, parameterOwner{webServiceID: &serviceID}, &location, parameter.Explode, parameters); err != nil {
					return fmt.Errorf("add web service parameter: %w", err)
				}
			}
		case *domain.WFSWebServiceInput:
			if err := q.AddLifecycleWFSService(ctx, sqlc.AddLifecycleWFSServiceParams{
				WebServiceID:          serviceID,
				GetFeatureTemplateUrl: configuration.GetFeatureTemplateURL,
				Version:               configuration.Version,
				FeatureTypes:          configuration.FeatureTypes,
				DefaultFeatureType:    configuration.DefaultFeatureType,
				OutputFormats:         configuration.OutputFormats,
				DefaultOutputFormat:   configuration.DefaultOutputFormat,
				ResultLimit:           configuration.ResultLimit,
				ResultLimitEditable:   configuration.ResultLimitEditable,
			}); err != nil {
				return fmt.Errorf("add WFS configuration: %w", err)
			}
			for _, parameter := range configuration.Parameters {
				location := parameter.Location
				if _, err := persistParameter(ctx, q, parameter.ParameterInput, parameterOwner{wfsServiceID: &serviceID}, &location, parameter.Explode, parameters); err != nil {
					return fmt.Errorf("add WFS parameter: %w", err)
				}
			}
		case *domain.WMTSWebServiceInput:
			if err := q.AddLifecycleWMTSService(ctx, sqlc.AddLifecycleWMTSServiceParams{
				WebServiceID: serviceID, CapabilitiesUrl: configuration.CapabilitiesURL, DefaultLayer: configuration.DefaultLayer,
			}); err != nil {
				return fmt.Errorf("add WMTS configuration: %w", err)
			}
			for position, layer := range configuration.Layers {
				id, resourceID := layerIDs(layer.ResourceID, layers)
				styles := layer.Styles
				if styles == nil {
					styles = []string{}
				}
				if err := q.AddLifecycleWMTSLayer(ctx, sqlc.AddLifecycleWMTSLayerParams{
					ID:           id,
					ResourceID:   resourceID,
					WebServiceID: serviceID,
					Name:         layer.Name,
					Position:     int32(position),
					DefaultStyle: layer.DefaultStyle,
					Styles:       styles,
				}); err != nil {
					return fmt.Errorf("add WMTS layer %q: %w", layer.Name, err)
				}
			}
			for _, parameter := range configuration.GetTileParameters {
				location := parameter.Location
				if _, err := persistParameter(ctx, q, parameter.ParameterInput, parameterOwner{wmtsGetTileServiceID: &serviceID}, &location, parameter.Explode, parameters); err != nil {
					return fmt.Errorf("add WMTS GetTile parameter: %w", err)
				}
			}
			for _, parameter := range configuration.GetFeatureInfoParameters {
				location := parameter.Location
				if _, err := persistParameter(ctx, q, parameter.ParameterInput, parameterOwner{wmtsGetFeatureInfoServiceID: &serviceID}, &location, parameter.Explode, parameters); err != nil {
					return fmt.Errorf("add WMTS GetFeatureInfo parameter: %w", err)
				}
			}
		case *domain.WMSWebServiceInput:
			if err := q.AddLifecycleWMSService(ctx, sqlc.AddLifecycleWMSServiceParams{
				WebServiceID:         serviceID,
				CapabilitiesUrl:      configuration.CapabilitiesURL,
				DefaultLayer:         configuration.DefaultLayer,
				TransparencyEditable: configuration.TransparencyEditable,
				DefaultTransparency:  configuration.DefaultTransparency,
			}); err != nil {
				return fmt.Errorf("add WMS configuration: %w", err)
			}
			for layerPosition, layer := range configuration.Layers {
				id, resourceID := layerIDs(layer.ResourceID, layers)
				if err := q.AddLifecycleWMSLayer(ctx, sqlc.AddLifecycleWMSLayerParams{
					ID:           id,
					ResourceID:   resourceID,
					WebServiceID: serviceID,
					Name:         layer.Name,
					Position:     int32(layerPosition),
					DefaultStyle: layer.DefaultStyle,
					Styles:       layer.Styles,
				}); err != nil {
					return fmt.Errorf("add WMS layer %q: %w", layer.Name, err)
				}
			}
			for _, parameter := range configuration.GetMapParameters {
				location := parameter.Location
				if _, err := persistParameter(ctx, q, parameter.ParameterInput, parameterOwner{wmsGetMapServiceID: &serviceID}, &location, parameter.Explode, parameters); err != nil {
					return fmt.Errorf("add WMS GetMap parameter: %w", err)
				}
			}
			for _, parameter := range configuration.GetFeatureInfoParameters {
				location := parameter.Location
				if _, err := persistParameter(ctx, q, parameter.ParameterInput, parameterOwner{wmsGetFeatureInfoServiceID: &serviceID}, &location, parameter.Explode, parameters); err != nil {
					return fmt.Errorf("add WMS GetFeatureInfo parameter: %w", err)
				}
			}
		default:
			return fmt.Errorf("unsupported web service configuration %T", service)
		}
	}
	for i, distribution := range value.Distributions {
		params := sqlc.AddLifecycleDistributionParams{
			ID:                     distributions[i].id,
			ResourceID:             distributions[i].resourceID,
			DatasetVersionID:       versionID,
			Title:                  distribution.Title,
			Description:            distribution.Description,
			License:                distribution.License,
			ByteSize:               distribution.ByteSize,
			Type:                   sqlc.DistributionType(distribution.Type),
			DownloadableFileUrl:    nil,
			DownloadableFileFormat: nil,
			WebServiceID:           nil,
		}
		if distribution.DownloadableFile != nil {
			params.DownloadableFileUrl = &distribution.DownloadableFile.URL
			params.DownloadableFileFormat = &distribution.DownloadableFile.Format
		}
		if distribution.WebService != nil {
			params.WebServiceID = &distributions[i].serviceID
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

func readDatasetMetadata(ctx context.Context, q *sqlc.Queries, version sqlc.GetLifecycleDatasetVersionRow) (domain.DatasetMetadata, error) {
	identifiers, err := q.ListLifecycleDatasetIdentifiers(ctx, &version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list identifiers: %w", err)
	}

	providers, err := q.ListLifecycleDatasetProviders(ctx, version.ID)
	if err != nil {
		return domain.DatasetMetadata{}, fmt.Errorf("list providers: %w", err)
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
		ID:                         version.ID,
		Title:                      version.Title,
		Description:                version.Description,
		Keywords:                   version.Keywords,
		AccrualPeriodicity:         (*domain.AccrualPeriodicity)(version.AccrualPeriodicity),
		QualityAnnotation:          version.QualityAnnotation,
		ConformsTo:                 version.ConformsTo,
		Provenance:                 version.Provenance,
		Type:                       (*domain.DatasetType)(version.Type),
		Version:                    version.Version,
		SpatialResolutionInMeters:  version.SpatialResolutionInMeters,
		TemporalResolution:         version.TemporalResolution,
		AccessRight:                (*domain.AccessRight)(version.AccessRight),
		Distributions:              nil,
		SpatialCoverage:            version.SpatialCoverage,
		TemporalCoverage:           temporal(version.TemporalStart, version.TemporalEnd),
		GeologicalTemporalCoverage: geologicalTemporal(version.GeologicalTemporalStartMya, version.GeologicalTemporalEndMya),
		PersistentIdentifiers:      nil,
		ContactPoints:              nil,
		Providers:                  nil,
		Categories:                 nil,
		SourceCreated:              version.SourceCreatedAt,
		SourceUpdated:              version.SourceUpdatedAt,
	}
	for _, item := range identifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, toDomainIdentifier(item.Identifier))
	}

	for _, item := range providers {
		provider := toDomainOrganizationMetadata(item)
		result.Providers = append(result.Providers, provider)
	}

	for _, item := range contacts {
		result.ContactPoints = append(result.ContactPoints, toDomainContactPoint(item))
	}

	for _, item := range categories {
		result.Categories = append(result.Categories, domain.CategoryMetadata{
			ID:   item.ID,
			Name: item.Name,
		})
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
	services := make([]sqlc.WebService, len(rows))
	for i, row := range rows {
		services[i] = row.WebService
	}
	result, failures, err := readWebServiceRows(ctx, q, services)
	if err != nil {
		return nil, err
	}
	for id, err := range failures {
		return nil, fmt.Errorf("load web service %s: %w", id, err)
	}
	for id, service := range result {
		if err := service.Normalize(); err != nil {
			return nil, fmt.Errorf("parse web service %s: %w", id, err)
		}
	}
	return result, nil
}
