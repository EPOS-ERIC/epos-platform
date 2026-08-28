package discovery

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toSearchDiscoveryDatasetsParams(filter domain.DiscoveryFilter) sqlc.SearchDiscoveryDatasetsParams {
	return sqlc.SearchDiscoveryDatasetsParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		SpatialFilter:      filter.SpatialFilter,
		DateFrom:           filter.DateFrom,
		DateTo:             filter.DateTo,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		ResourceTypes:      resourceTypeStrings(filter.Types),
	}
}

func toSearchDiscoveryApplicationsParams(filter domain.DiscoveryFilter) sqlc.SearchDiscoveryApplicationsParams {
	return sqlc.SearchDiscoveryApplicationsParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		ResourceTypes:      resourceTypeStrings(filter.Types),
	}
}

func toSearchDiscoverySourceCodesParams(filter domain.DiscoveryFilter) sqlc.SearchDiscoverySourceCodesParams {
	return sqlc.SearchDiscoverySourceCodesParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		ResourceTypes:      resourceTypeStrings(filter.Types),
	}
}

func toDatasetCandidates(rows []sqlc.SearchDiscoveryDatasetsRow) ([]domain.DiscoveryCandidate, error) {
	result := make([]domain.DiscoveryCandidate, 0, len(rows))
	for _, row := range rows {
		candidate, err := toDiscoveryCandidate(
			domain.Dataset,
			row.ResourceID,
			row.VersionID,
			row.GroupID,
			string(row.Status),
			row.Name,
		)
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
		candidate, err := toDiscoveryCandidate(
			domain.SoftwareApplication,
			row.ResourceID,
			row.VersionID,
			row.GroupID,
			string(row.Status),
			row.Name,
		)
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
		candidate, err := toDiscoveryCandidate(
			domain.SoftwareSourceCode,
			row.ResourceID,
			row.VersionID,
			row.GroupID,
			string(row.Status),
			row.Name,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	return result, nil
}

func toDiscoveryCandidate(resourceType domain.ResourceType, resourceID, versionID, groupID uuid.UUID, status, name string) (domain.DiscoveryCandidate, error) {
	candidate, err := domain.NewDiscoveryCandidate(domain.DiscoveryCandidate{
		Type:       resourceType,
		ResourceID: resourceID,
		VersionID:  versionID,
		GroupID:    groupID,
		Status:     domain.Status(status),
		Name:       name,
	})
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
