// Package discovery provides PostgreSQL persistence for resource discovery.
package discovery

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{Queries: sqlc.New(pool)} }

func (r *Repo) Search(ctx context.Context, filter domain.DiscoveryFilter) ([]domain.DiscoveryCandidate, error) {
	datasets, err := r.SearchDiscoveryDatasets(ctx, toSearchDiscoveryDatasetsParams(filter))
	if err != nil {
		return nil, fmt.Errorf("search datasets: %w", err)
	}

	result, err := toDatasetCandidates(datasets)
	if err != nil {
		return nil, fmt.Errorf("map datasets: %w", err)
	}

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
