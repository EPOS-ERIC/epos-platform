// Package discovery provides PostgreSQL persistence for resource discovery.
package discovery

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
	log *slog.Logger
}

func New(log *slog.Logger, pool *pgxpool.Pool) *Repo {
	return &Repo{
		Queries: sqlc.New(pool),
		log:     log.With("component", "discovery-repository"),
	}
}

func (r *Repo) Search(ctx context.Context, filter domain.DiscoveryFilter) ([]domain.DiscoveryCandidate, error) {
	datasetParams, err := toSearchDiscoveryDatasetsParams(filter)
	if err != nil {
		return nil, err
	}
	r.log.DebugContext(ctx, "search discovery datasets", "params", datasetParams)
	datasets, err := r.SearchDiscoveryDatasets(ctx, datasetParams)
	if err != nil {
		return nil, fmt.Errorf("search datasets: %w", err)
	}
	r.log.DebugContext(ctx, "search discovery datasets", "results", len(datasets))

	result, err := toDatasetCandidates(datasets)
	if err != nil {
		return nil, fmt.Errorf("map datasets: %w", err)
	}
	r.log.DebugContext(ctx, "search discovery datasets", "candidates", len(result))

	applications, err := r.SearchDiscoveryApplications(ctx, toSearchDiscoveryApplicationsParams(filter))
	if err != nil {
		return nil, fmt.Errorf("search software applications: %w", err)
	}
	applicationCandidates, err := toApplicationCandidates(applications)
	if err != nil {
		return nil, fmt.Errorf("map software applications: %w", err)
	}
	result = append(result, applicationCandidates...)

	sourceCodes, err := r.SearchDiscoverySourceCodes(ctx, toSearchDiscoverySourceCodesParams(filter))
	if err != nil {
		return nil, fmt.Errorf("search software source codes: %w", err)
	}
	sourceCodeCandidates, err := toSourceCodeCandidates(sourceCodes)
	if err != nil {
		return nil, fmt.Errorf("map software source codes: %w", err)
	}
	result = append(result, sourceCodeCandidates...)

	return result, nil
}

// DatasetRelations loads discovery-visible dataset relationships for all requested versions.
func (r *Repo) DatasetRelations(ctx context.Context, versionIDs []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	result := make(map[uuid.UUID]domain.DiscoveryRelations)

	providers, err := r.ListDiscoveryDatasetProviders(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list dataset providers: %w", err)
	}
	providerRows := discoveryRelationRows(providers, func(row sqlc.ListDiscoveryDatasetProvidersRow) discoveryRelationRow {
		return discoveryRelationRow{versionID: row.VersionID, id: row.ID, name: row.Name}
	})
	if err := addDiscoveryReferences(result, providerRows, func(relations *domain.DiscoveryRelations, reference domain.DiscoveryReference) {
		relations.Providers = append(relations.Providers, reference)
	}, "dataset provider"); err != nil {
		return nil, err
	}

	categories, err := r.ListDiscoveryDatasetCategories(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list dataset categories: %w", err)
	}
	categoryRows := discoveryRelationRows(categories, func(row sqlc.ListDiscoveryDatasetCategoriesRow) discoveryRelationRow {
		return discoveryRelationRow{versionID: row.VersionID, id: row.ID, name: row.Name}
	})
	if err := addDiscoveryReferences(result, categoryRows, func(relations *domain.DiscoveryRelations, reference domain.DiscoveryReference) {
		relations.Categories = append(relations.Categories, reference)
	}, "dataset category"); err != nil {
		return nil, err
	}

	distributions, err := r.ListDiscoveryDatasetDistributions(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list dataset distributions: %w", err)
	}
	distributionRows := discoveryRelationRows(distributions, func(row sqlc.ListDiscoveryDatasetDistributionsRow) discoveryRelationRow {
		return discoveryRelationRow{versionID: row.VersionID, id: row.ID, name: row.Name}
	})
	if err := addDiscoveryReferences(result, distributionRows, func(relations *domain.DiscoveryRelations, reference domain.DiscoveryReference) {
		relations.Distributions = append(relations.Distributions, reference)
	}, "dataset distribution"); err != nil {
		return nil, err
	}

	return result, nil
}

// SoftwareApplicationRelations loads discovery-visible application relationships in batches.
func (r *Repo) SoftwareApplicationRelations(ctx context.Context, versionIDs []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	return r.softwareRelations(ctx, versionIDs, r.applicationCreators, r.applicationCategories)
}

// SoftwareSourceCodeRelations loads discovery-visible source-code relationships in batches.
func (r *Repo) SoftwareSourceCodeRelations(ctx context.Context, versionIDs []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	return r.softwareRelations(ctx, versionIDs, r.sourceCodeCreators, r.sourceCodeCategories)
}

// softwareRelations combines creator and category projections shared by software resources.
func (r *Repo) softwareRelations(
	ctx context.Context,
	versionIDs []uuid.UUID,
	creators func(context.Context, []uuid.UUID) ([]discoveryRelationRow, error),
	categories func(context.Context, []uuid.UUID) ([]discoveryRelationRow, error),
) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	result := make(map[uuid.UUID]domain.DiscoveryRelations)
	if err := loadDiscoveryRelations(
		result,
		func() ([]discoveryRelationRow, error) { return creators(ctx, versionIDs) },
		func(relations *domain.DiscoveryRelations, reference domain.DiscoveryReference) {
			relations.Creators = append(relations.Creators, reference)
		},
		"software creator",
		func() ([]discoveryRelationRow, error) { return categories(ctx, versionIDs) },
		func(relations *domain.DiscoveryRelations, reference domain.DiscoveryReference) {
			relations.Categories = append(relations.Categories, reference)
		},
		"software category",
	); err != nil {
		return nil, err
	}
	return result, nil
}

// applicationCreators loads application creator rows and maps them to discovery rows.
func (r *Repo) applicationCreators(ctx context.Context, versionIDs []uuid.UUID) ([]discoveryRelationRow, error) {
	rows, err := r.ListDiscoveryApplicationCreators(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list application creators: %w", err)
	}
	return discoveryRelationRows(rows, func(row sqlc.ListDiscoveryApplicationCreatorsRow) discoveryRelationRow {
		return discoveryRelationRow{versionID: row.VersionID, id: row.ID, name: row.Name}
	}), nil
}

// applicationCategories loads application category rows and maps them to discovery rows.
func (r *Repo) applicationCategories(ctx context.Context, versionIDs []uuid.UUID) ([]discoveryRelationRow, error) {
	rows, err := r.ListDiscoveryApplicationCategories(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list application categories: %w", err)
	}
	return discoveryRelationRows(rows, func(row sqlc.ListDiscoveryApplicationCategoriesRow) discoveryRelationRow {
		return discoveryRelationRow{versionID: row.VersionID, id: row.ID, name: row.Name}
	}), nil
}

// sourceCodeCreators loads source-code creator rows and maps them to discovery rows.
func (r *Repo) sourceCodeCreators(ctx context.Context, versionIDs []uuid.UUID) ([]discoveryRelationRow, error) {
	rows, err := r.ListDiscoverySourceCodeCreators(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list source code creators: %w", err)
	}
	return discoveryRelationRows(rows, func(row sqlc.ListDiscoverySourceCodeCreatorsRow) discoveryRelationRow {
		return discoveryRelationRow{versionID: row.VersionID, id: row.ID, name: row.Name}
	}), nil
}

// sourceCodeCategories loads source-code category rows and maps them to discovery rows.
func (r *Repo) sourceCodeCategories(ctx context.Context, versionIDs []uuid.UUID) ([]discoveryRelationRow, error) {
	rows, err := r.ListDiscoverySourceCodeCategories(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list source code categories: %w", err)
	}
	return discoveryRelationRows(rows, func(row sqlc.ListDiscoverySourceCodeCategoriesRow) discoveryRelationRow {
		return discoveryRelationRow{versionID: row.VersionID, id: row.ID, name: row.Name}
	}), nil
}

type discoveryRelationRow struct {
	versionID uuid.UUID
	id        uuid.UUID
	name      string
}

// discoveryRelationRows converts typed SQL rows into the common projection shape.
func discoveryRelationRows[T any](rows []T, values func(T) discoveryRelationRow) []discoveryRelationRow {
	result := make([]discoveryRelationRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, values(row))
	}
	return result
}

// loadDiscoveryRelations loads two relationship projections into one version-indexed map.
func loadDiscoveryRelations(
	result map[uuid.UUID]domain.DiscoveryRelations,
	first func() ([]discoveryRelationRow, error),
	firstAdd func(*domain.DiscoveryRelations, domain.DiscoveryReference),
	firstKind string,
	second func() ([]discoveryRelationRow, error),
	secondAdd func(*domain.DiscoveryRelations, domain.DiscoveryReference),
	secondKind string,
) error {
	firstRows, err := first()
	if err != nil {
		return err
	}
	if err := addDiscoveryReferences(result, firstRows, firstAdd, firstKind); err != nil {
		return err
	}

	secondRows, err := second()
	if err != nil {
		return err
	}
	return addDiscoveryReferences(result, secondRows, secondAdd, secondKind)
}

// addDiscoveryReferences validates and appends mapped references to their version.
func addDiscoveryReferences(
	result map[uuid.UUID]domain.DiscoveryRelations,
	rows []discoveryRelationRow,
	add func(*domain.DiscoveryRelations, domain.DiscoveryReference),
	kind string,
) error {
	for _, row := range rows {
		reference, err := domain.NewDiscoveryReference(row.id, row.name)
		if err != nil {
			return fmt.Errorf("construct %s reference: %w", kind, err)
		}
		relations := result[row.versionID]
		add(&relations, reference)
		result[row.versionID] = relations
	}
	return nil
}
