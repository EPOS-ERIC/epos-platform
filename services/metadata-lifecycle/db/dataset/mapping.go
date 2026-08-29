package dataset

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

// ToCreateVersionParams builds SQLC parameters for a new dataset version.
func ToCreateVersionParams(datasetID uuid.UUID, actorID string, value domain.DatasetInput) (sqlc.CreateLifecycleDatasetVersionParams, error) {
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

// ToReplaceDraftParams builds SQLC parameters for replacing a dataset draft.
func ToReplaceDraftParams(datasetID, id uuid.UUID, revision int32, value domain.DatasetInput) (sqlc.ReplaceLifecycleDatasetDraftParams, error) {
	p, err := ToCreateVersionParams(uuid.Nil, "", value)
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

func toDomainDatasetVersion(version sqlc.GetLifecycleDatasetVersionRow, metadata domain.DatasetMetadata) (domain.DatasetVersion, error) {
	result, err := domain.NewDatasetVersion(
		version.DatasetID,
		version.ID,
		version.Revision,
		domain.Status(version.Status),
		domain.Actor(version.CreatedBy),
		toDomainGroup(version.Group),
		version.CreatedAt,
		version.UpdatedAt,
		metadata,
	)
	if err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("validate dataset result: %w", err)
	}
	return result, nil
}

func temporal(start, end *time.Time) *domain.Temporal {
	if start == nil && end == nil {
		return nil
	}
	return &domain.Temporal{Start: start, End: end}
}

func spatialGeometry(value *domain.Spatial) (*geom.Point, *geom.Polygon, error) {
	if value == nil {
		return nil, nil, nil
	}
	if value.Point != nil {
		point, err := parsePoint(*value.Point)
		return point, nil, err
	}
	if len(value.Polygon) == 0 {
		return nil, nil, nil
	}
	polygon, err := parsePolygon(value.Polygon)
	return nil, polygon, err
}

func parsePoint(value domain.Coordinate) (*geom.Point, error) {
	coordinate, err := parseCoordinate(value)
	if err != nil {
		return nil, err
	}

	return geom.NewPointFlat(geom.XY, coordinate).SetSRID(4326), nil
}

func parsePolygon(values []domain.Coordinate) (*geom.Polygon, error) {
	coordinates := make([]geom.Coord, len(values))
	for i, value := range values {
		coordinate, err := parseCoordinate(value)
		if err != nil {
			return nil, err
		}
		coordinates[i] = coordinate
	}

	polygon, err := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{coordinates})
	if err != nil {
		return nil, fmt.Errorf("parse polygon coordinates: %w", err)
	}

	return polygon.SetSRID(4326), nil
}

func parseCoordinate(value domain.Coordinate) (geom.Coord, error) {
	return geom.Coord{value.Longitude, value.Latitude}, nil
}

func spatialFromGeometry(point *geom.Point, polygon *geom.Polygon) (*domain.Spatial, error) {
	if point == nil && polygon == nil {
		return nil, nil
	}
	if point != nil {
		return &domain.Spatial{
			Point: &domain.Coordinate{
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
	coordinates := make([]domain.Coordinate, len(rings[0]))
	for i, coordinate := range rings[0] {
		coordinates[i] = domain.Coordinate{Longitude: coordinate.X(), Latitude: coordinate.Y()}
	}
	return &domain.Spatial{Point: nil, Polygon: coordinates}, nil
}

func toDomainWebService(value sqlc.WebService) (domain.WebService, error) {
	spatial, err := spatialFromGeometry(value.SpatialPoint, value.SpatialPolygon)
	if err != nil {
		return domain.WebService{}, fmt.Errorf("decode web service geometry: %w", err)
	}
	var documentation *domain.Documentation
	if value.DocumentationUri != nil {
		documentation = &domain.Documentation{
			Title:       value.DocumentationTitle,
			Description: value.DocumentationDescription,
			URI:         *value.DocumentationUri,
		}
	}
	return domain.WebService{
		ID:               value.ID,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateUrl,
		Parameters:       nil,
		Format:           value.Format,
		Provider:         nil,
		Documentation:    documentation,
		SpatialCoverage:  spatial,
		TemporalCoverage: temporal(value.TemporalStart, value.TemporalEnd),
		ContactPoints:    nil,
	}, nil
}

func toDomainDistribution(value sqlc.Distribution, services map[uuid.UUID]domain.WebService) domain.Distribution {
	result := domain.Distribution{
		ID:               value.ID,
		Title:            value.Title,
		Description:      value.Description,
		License:          value.License,
		Type:             domain.DistributionType(value.Type),
		Issued:           value.SourceIssuedAt,
		Modified:         value.SourceModifiedAt,
		ByteSize:         nil,
		WebService:       nil,
		DownloadableFile: nil,
	}
	if value.ByteSize != nil {
		byteSize := int64(*value.ByteSize)
		result.ByteSize = &byteSize
	}
	if value.WebServiceID != nil {
		service := services[*value.WebServiceID]
		result.WebService = &service
		return result
	}
	result.DownloadableFile = &domain.DownloadableFile{
		URL:    *value.DownloadableFileUrl,
		Format: *value.DownloadableFileFormat,
	}
	return result
}

func toDomainCategoryMetadata(value sqlc.ListLifecycleDatasetCategoriesRow) domain.CategoryMetadata {
	return domain.CategoryMetadata{
		ID:   value.ID,
		Name: value.Name,
		Scheme: domain.Group{
			ID:          uuid.Nil,
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
	}
}

func toDomainOrganizationMetadata(value sqlc.Organization) domain.OrganizationMetadata {
	return domain.OrganizationMetadata{
		ID:            value.ID,
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

func toDomainContactPoint(value sqlc.ContactPoint) domain.ContactPoint {
	return domain.ContactPoint{
		ID:    value.ID,
		Email: value.Email,
		Role:  value.Role,
	}
}

func toDomainIdentifier(value sqlc.Identifier) domain.Identifier {
	return domain.Identifier{
		ID:    value.ID,
		Type:  domain.IdentifierType(value.Type),
		Value: value.Value,
	}
}

func toDomainGroup(value sqlc.Group) domain.Group {
	return domain.Group{
		ID:          value.ID,
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		Concept:     string(value.Concept),
	}
}
