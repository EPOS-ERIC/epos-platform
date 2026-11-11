package discovery

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toSearchDiscoveryDatasetsParams(filter domain.DiscoveryFilter) (sqlc.SearchDiscoveryDatasetsParams, error) {
	return sqlc.SearchDiscoveryDatasetsParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		SpatialFilter:      filter.SpatialFilter,
		DateFrom:           filter.DateFrom,
		DateTo:             filter.DateTo,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		EditorID:           filter.EditorID,
		ResourceTypes:      resourceTypeStrings(filter.Types),
		Concepts:           filter.Concepts,
		GroupID:            filter.GroupID,
	}, nil
}

func toSearchDiscoveryApplicationsParams(filter domain.DiscoveryFilter) sqlc.SearchDiscoveryApplicationsParams {
	return sqlc.SearchDiscoveryApplicationsParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		SpatialFilter:      filter.SpatialFilter,
		DateFrom:           filter.DateFrom,
		DateTo:             filter.DateTo,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		EditorID:           filter.EditorID,
		ResourceTypes:      resourceTypeStrings(filter.Types),
		Concepts:           filter.Concepts,
		GroupID:            filter.GroupID,
	}
}

func toSearchDiscoverySourceCodesParams(filter domain.DiscoveryFilter) sqlc.SearchDiscoverySourceCodesParams {
	return sqlc.SearchDiscoverySourceCodesParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		SpatialFilter:      filter.SpatialFilter,
		DateFrom:           filter.DateFrom,
		DateTo:             filter.DateTo,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		EditorID:           filter.EditorID,
		ResourceTypes:      resourceTypeStrings(filter.Types),
		Concepts:           filter.Concepts,
		GroupID:            filter.GroupID,
	}
}

func toDatasetCandidates(rows []sqlc.SearchDiscoveryDatasetsRow) ([]domain.DiscoveryCandidate, error) {
	result := make([]domain.DiscoveryCandidate, 0, len(rows))
	for _, row := range rows {
		candidate, err := toDiscoveryCandidate(domain.DiscoveryCandidate{
			Type:       domain.Dataset,
			ResourceID: row.ResourceID,
			VersionID:  row.VersionID,
			GroupID:    row.GroupID,
			GroupName:  row.GroupName,
			Status:     domain.Status(row.Status),
			Name:       row.Name,
			UpdatedAt:  row.UpdatedAt,
			Concept:    domain.DiscoveryReference{ID: row.ConceptID, Name: row.ConceptName},
			CreatedBy:  &domain.DiscoveryCreatedBy{ID: row.CreatedByID, Name: row.CreatedByName},
		})
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	return result, nil
}

func toApplicationCandidates(rows []sqlc.SearchDiscoveryApplicationsRow) ([]domain.DiscoveryCandidate, error) {
	result := make([]domain.DiscoveryCandidate, 0, len(rows))
	for _, row := range rows {
		candidate, err := toDiscoveryCandidate(domain.DiscoveryCandidate{
			Type:       domain.SoftwareApplication,
			ResourceID: row.ResourceID,
			VersionID:  row.VersionID,
			GroupID:    row.GroupID,
			GroupName:  row.GroupName,
			Status:     domain.Status(row.Status),
			Name:       row.Name,
			UpdatedAt:  row.UpdatedAt,
			Concept:    domain.DiscoveryReference{ID: row.ConceptID, Name: row.ConceptName},
			CreatedBy:  &domain.DiscoveryCreatedBy{ID: row.CreatedByID, Name: row.CreatedByName},
		})
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	return result, nil
}

func toSourceCodeCandidates(rows []sqlc.SearchDiscoverySourceCodesRow) ([]domain.DiscoveryCandidate, error) {
	result := make([]domain.DiscoveryCandidate, 0, len(rows))
	for _, row := range rows {
		candidate, err := toDiscoveryCandidate(domain.DiscoveryCandidate{
			Type:       domain.SoftwareSourceCode,
			ResourceID: row.ResourceID,
			VersionID:  row.VersionID,
			GroupID:    row.GroupID,
			GroupName:  row.GroupName,
			Status:     domain.Status(row.Status),
			Name:       row.Name,
			UpdatedAt:  row.UpdatedAt,
			Concept:    domain.DiscoveryReference{ID: row.ConceptID, Name: row.ConceptName},
			CreatedBy:  &domain.DiscoveryCreatedBy{ID: row.CreatedByID, Name: row.CreatedByName},
		})
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	return result, nil
}

func toDiscoveryCandidate(value domain.DiscoveryCandidate) (domain.DiscoveryCandidate, error) {
	candidate, err := domain.NewDiscoveryCandidate(value)
	if err != nil {
		return domain.DiscoveryCandidate{}, fmt.Errorf("bad discovery candidate: %w", err)
	}
	return candidate, nil
}

func statusStrings(values []domain.Status) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func resourceTypeStrings(values []domain.ResourceType) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}
