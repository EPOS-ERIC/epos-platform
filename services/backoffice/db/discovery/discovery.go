// Package discovery provides PostgreSQL persistence for resource discovery.
package discovery

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/discovery"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	db *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{db: pool} }

func (r *Repo) Search(ctx context.Context, filter discovery.Filter) ([]discovery.Candidate, error) {
	q := sqlc.New(r.db)
	statuses := make([]string, 0, len(filter.VersioningStatuses))
	for _, value := range filter.VersioningStatuses {
		statuses = append(statuses, string(value))
	}

	types := make([]string, 0, len(filter.Types))
	for _, value := range filter.Types {
		types = append(types, string(value))
	}

	query := optionalString(filter.Query)
	spatial := optionalString(filter.SpatialFilter)
	var result []discovery.Candidate

	datasets, err := q.SearchDiscoveryDatasets(ctx, sqlc.SearchDiscoveryDatasetsParams{
		VersioningStatuses: statuses,
		Query:              query,
		SpatialFilter:      spatial,
		DateFrom:           filter.DateFrom,
		DateTo:             filter.DateTo,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		ResourceTypes:      types,
	})
	if err != nil {
		return nil, fmt.Errorf("search datasets: %w", err)
	}

	for _, row := range datasets {
		result = append(result, discovery.Candidate{
			Type:       discovery.Dataset,
			ResourceID: row.ResourceID,
			VersionID:  row.VersionID,
			GroupID:    row.GroupID,
			Status:     discovery.Status(row.Status),
			SourceIRI:  row.SourceIri,
			Name:       row.Name,
		})
	}

	applications, err := q.SearchDiscoveryApplications(ctx, sqlc.SearchDiscoveryApplicationsParams{
		VersioningStatuses: statuses,
		Query:              query,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		ResourceTypes:      types,
	})
	if err != nil {
		return nil, fmt.Errorf("search software applications: %w", err)
	}
	for _, row := range applications {
		result = append(result, discovery.Candidate{
			Type:       discovery.SoftwareApplication,
			ResourceID: row.ResourceID,
			VersionID:  row.VersionID,
			GroupID:    row.GroupID,
			Status:     discovery.Status(row.Status),
			SourceIRI:  row.SourceIri,
			Name:       row.Name,
		})
	}

	sourceCodes, err := q.SearchDiscoverySourceCodes(ctx, sqlc.SearchDiscoverySourceCodesParams{
		VersioningStatuses: statuses,
		Query:              query,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		ResourceTypes:      types,
	})
	if err != nil {
		return nil, fmt.Errorf("search software source codes: %w", err)
	}
	for _, row := range sourceCodes {
		result = append(result, discovery.Candidate{
			Type:       discovery.SoftwareSourceCode,
			ResourceID: row.ResourceID,
			VersionID:  row.VersionID,
			GroupID:    row.GroupID,
			Status:     discovery.Status(row.Status),
			SourceIRI:  row.SourceIri,
			Name:       row.Name,
		})
	}
	return result, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
