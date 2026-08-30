package discovery

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
)

func toSearchDiscoveryDatasetsParams(filter domain.DiscoveryFilter) (sqlc.SearchDiscoveryDatasetsParams, error) {
	spatialFilter, err := spatialFilterWKT(filter.SpatialFilter)
	if err != nil {
		return sqlc.SearchDiscoveryDatasetsParams{}, fmt.Errorf("encode spatial filter: %w", err)
	}
	return sqlc.SearchDiscoveryDatasetsParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		SpatialFilter:      spatialFilter,
		DateFrom:           filter.DateFrom,
		DateTo:             filter.DateTo,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		EditorID:           filter.EditorID,
		ResourceTypes:      resourceTypeStrings(filter.Types),
		Concepts:           topLevelConceptStrings(filter.Concepts),
		GroupID:            filter.GroupID,
	}, nil
}

func spatialFilterWKT(value *domain.SpatialFilter) (*string, error) {
	if value == nil {
		return nil, nil
	}

	var geometry geom.T
	if value.Point != nil {
		geometry = geom.NewPointFlat(geom.XY, geom.Coord{value.Point.Longitude, value.Point.Latitude}).SetSRID(4326)
	} else {
		coordinates := make([]geom.Coord, len(value.Polygon))
		for i, coordinate := range value.Polygon {
			coordinates[i] = geom.Coord{coordinate.Longitude, coordinate.Latitude}
		}
		polygon, err := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{coordinates})
		if err != nil {
			return nil, fmt.Errorf("build polygon: %w", err)
		}
		geometry = polygon.SetSRID(4326)
	}
	wktValue, err := wkt.Marshal(geometry)
	if err != nil {
		return nil, fmt.Errorf("marshal WKT: %w", err)
	}
	return &wktValue, nil
}

func toSearchDiscoveryApplicationsParams(filter domain.DiscoveryFilter) sqlc.SearchDiscoveryApplicationsParams {
	return sqlc.SearchDiscoveryApplicationsParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		EditorID:           filter.EditorID,
		ResourceTypes:      resourceTypeStrings(filter.Types),
		Concepts:           topLevelConceptStrings(filter.Concepts),
		GroupID:            filter.GroupID,
	}
}

func toSearchDiscoverySourceCodesParams(filter domain.DiscoveryFilter) sqlc.SearchDiscoverySourceCodesParams {
	return sqlc.SearchDiscoverySourceCodesParams{
		VersioningStatuses: statusStrings(filter.VersioningStatuses),
		Query:              filter.Query,
		ProviderIds:        filter.Providers,
		Keywords:           filter.Keywords,
		EditorID:           filter.EditorID,
		ResourceTypes:      resourceTypeStrings(filter.Types),
		Concepts:           topLevelConceptStrings(filter.Concepts),
		GroupID:            filter.GroupID,
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

func topLevelConceptStrings(values []domain.TopLevelConcept) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}
