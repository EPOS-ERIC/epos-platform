package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataread"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (d *PostgresDB) Get(ctx context.Context, resourceType metadataread.ResourceType, id uuid.UUID) (metadataread.Resource, error) {
	var resource metadataread.Resource
	var err error
	switch resourceType {
	case metadataread.ResourceTypeDataset:
		var value metadata.Dataset
		value, err = d.dataset(ctx, id)
		if err != nil {
			break
		}

		resource.Dataset = &value
	case metadataread.ResourceTypeDistribution:
		var value metadata.Distribution
		value, err = d.distribution(ctx, id)
		if err != nil {
			break
		}

		resource.Distribution = &value
	case metadataread.ResourceTypeWebService:
		var value metadata.WebService
		value, err = d.webService(ctx, id)
		if err != nil {
			break
		}

		resource.WebService = &value
	case metadataread.ResourceTypeOrganization:
		var value metadata.Organization
		value, err = d.organization(ctx, id)
		if err != nil {
			break
		}

		resource.Organization = &value
	case metadataread.ResourceTypeConcept:
		var value metadata.Category
		value, err = d.category(ctx, id)
		if err != nil {
			break
		}

		resource.Concept = &value
	case metadataread.ResourceTypeConceptScheme:
		var value metadata.CategoryScheme
		value, err = d.categoryScheme(ctx, id)
		if err != nil {
			break
		}

		resource.ConceptScheme = &value
	case metadataread.ResourceTypeSoftwareApplication:
		var value metadata.SoftwareApplication
		value, err = d.softwareApplication(ctx, id)
		if err != nil {
			break
		}

		resource.SoftwareApplication = &value
	case metadataread.ResourceTypeSoftwareSourceCode:
		var value metadata.SoftwareSourceCode
		value, err = d.softwareSourceCode(ctx, id)
		if err != nil {
			break
		}

		resource.SoftwareSourceCode = &value
	default:
		return metadataread.Resource{}, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return metadataread.Resource{}, metadataread.ErrNotFound
	}

	if err != nil {
		return metadataread.Resource{}, fmt.Errorf("read %s %s: %w", resourceType, id, err)
	}

	return resource, nil
}

func (d *PostgresDB) softwareApplication(ctx context.Context, id uuid.UUID) (metadata.SoftwareApplication, error) {
	row, err := d.Identity.GetSoftwareApplicationByID(ctx, id)
	if err != nil {
		return metadata.SoftwareApplication{}, fmt.Errorf("get software application %s: %w", id, err)
	}

	result := metadata.SoftwareApplication{
		ID:               row.ID,
		SourceIRI:        row.SourceIri,
		Name:             row.Name,
		Description:      row.Description,
		MainEntityOfPage: row.MainEntityOfPage,
		License:          row.License,
		Version:          row.Version,
		Keywords:         row.Keywords,
		DownloadURL:      row.DownloadUrl,
		Requirements:     row.Requirements,
		OperatingSystem:  row.OperatingSystem,
		Creators:         nil,
		Identifiers:      nil,
		Categories:       nil,
	}

	if result.Identifiers, err = d.softwareApplicationIdentifiers(ctx, row.ID); err != nil {
		return metadata.SoftwareApplication{}, err
	}

	if result.Creators, err = d.softwareApplicationCreators(ctx, row.ID); err != nil {
		return metadata.SoftwareApplication{}, err
	}

	if result.Categories, err = d.softwareApplicationCategories(ctx, row.ID); err != nil {
		return metadata.SoftwareApplication{}, err
	}

	return result, nil
}

func (d *PostgresDB) softwareSourceCode(ctx context.Context, id uuid.UUID) (metadata.SoftwareSourceCode, error) {
	row, err := d.Identity.GetSoftwareSourceCodeByID(ctx, id)
	if err != nil {
		return metadata.SoftwareSourceCode{}, fmt.Errorf("get software source code %s: %w", id, err)
	}

	result := metadata.SoftwareSourceCode{
		ID:                   row.ID,
		SourceIRI:            row.SourceIri,
		Name:                 row.Name,
		Description:          row.Description,
		ProgrammingLanguage:  row.ProgrammingLanguage,
		Keywords:             row.Keywords,
		Repository:           row.Repository,
		License:              row.License,
		SoftwareRequirements: row.SoftwareRequirements,
		RuntimePlatform:      row.RuntimePlatform,
		MainEntityOfPage:     row.MainEntityOfPage,
		Version:              row.Version,
		Creators:             nil,
		Identifiers:          nil,
		Categories:           nil,
	}

	if result.Identifiers, err = d.softwareSourceCodeIdentifiers(ctx, row.ID); err != nil {
		return metadata.SoftwareSourceCode{}, err
	}

	if result.Creators, err = d.softwareSourceCodeCreators(ctx, row.ID); err != nil {
		return metadata.SoftwareSourceCode{}, err
	}

	if result.Categories, err = d.softwareSourceCodeCategories(ctx, row.ID); err != nil {
		return metadata.SoftwareSourceCode{}, err
	}

	return result, nil
}

func (d *PostgresDB) dataset(ctx context.Context, id uuid.UUID) (metadata.Dataset, error) {
	row, err := d.Identity.GetDatasetByID(ctx, id)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("get dataset %s: %w", id, err)
	}

	result := metadata.Dataset{
		ID:                    row.ID,
		SourceIRI:             row.SourceIri,
		Title:                 row.Title,
		Description:           row.Description,
		Keywords:              row.Keywords,
		AccrualPeriodicity:    row.AccrualPeriodicity,
		QualityAnnotation:     row.QualityAnnotation,
		AccessRight:           row.AccessRight,
		Distributions:         nil,
		SpatialCoverage:       nil,
		TemporalCoverage:      temporal(row.TemporalStart, row.TemporalEnd),
		PersistentIdentifiers: nil,
		ContactPoints:         nil,
		Providers:             nil,
		Categories:            nil,
		SourceCreated:         row.SourceCreatedAt,
		SourceUpdated:         row.SourceUpdatedAt,
		Status:                metadata.DatasetStatus(row.Status),
		Editor:                nil,
		Group:                 nil,
		Created:               row.CreatedAt,
		Updated:               row.UpdatedAt,
		OriginalDatasetID:     row.OriginalDatasetID,
	}

	if result.SpatialCoverage, err = spatial(row.SpatialPoint, row.SpatialPolygon); err != nil {
		return metadata.Dataset{}, fmt.Errorf("read dataset spatial coverage %s: %w", id, err)
	}

	if result.PersistentIdentifiers, err = d.datasetIdentifiers(ctx, row.ID); err != nil {
		return metadata.Dataset{}, err
	}

	if result.ContactPoints, err = d.datasetContacts(ctx, row.ID); err != nil {
		return metadata.Dataset{}, err
	}

	if result.Providers, err = d.datasetProviders(ctx, row.ID); err != nil {
		return metadata.Dataset{}, err
	}

	if result.Categories, err = d.datasetCategories(ctx, row.ID); err != nil {
		return metadata.Dataset{}, err
	}

	rows, err := d.Identity.ListDatasetDistributions(ctx, row.ID)
	if err != nil {
		return metadata.Dataset{}, fmt.Errorf("list dataset distributions %s: %w", id, err)
	}

	result.Distributions = make([]metadata.Distribution, 0, len(rows))
	for _, row := range rows {
		distribution, err := d.mapDistribution(ctx, row)
		if err != nil {
			return metadata.Dataset{}, fmt.Errorf("read dataset distribution %s: %w", row.ID, err)
		}

		result.Distributions = append(result.Distributions, distribution)
	}

	return result, nil
}

func (d *PostgresDB) distribution(ctx context.Context, id uuid.UUID) (metadata.Distribution, error) {
	row, err := d.Identity.GetDistributionByID(ctx, id)
	if err != nil {
		return metadata.Distribution{}, fmt.Errorf("get distribution %s: %w", id, err)
	}

	return d.mapDistribution(ctx, sqlc.ListDatasetDistributionsRow(row))
}

func (d *PostgresDB) mapDistribution(ctx context.Context, row sqlc.ListDatasetDistributionsRow) (metadata.Distribution, error) {
	result := metadata.Distribution{
		ID:               row.ID,
		SourceIRI:        row.SourceIri,
		Title:            row.Title,
		Description:      row.Description,
		License:          row.License,
		Type:             metadata.DistributionType(row.Type),
		Issued:           row.SourceIssuedAt,
		Modified:         row.SourceModifiedAt,
		ByteSize:         nil,
		WebService:       nil,
		DownloadableFile: nil,
	}

	if row.ByteSize != nil {
		value := int64(*row.ByteSize)
		result.ByteSize = &value
	}

	if row.WebServiceID != nil {
		webService, err := d.webService(ctx, *row.WebServiceID)
		if err != nil {
			return metadata.Distribution{}, fmt.Errorf("read web service %s: %w", *row.WebServiceID, err)
		}

		result.WebService = &webService
	}

	if row.DownloadableFileUrl != nil || row.DownloadableFileFormat != nil {
		result.DownloadableFile = &metadata.DownloadableFile{
			URL:    stringOrEmpty(row.DownloadableFileUrl),
			Format: stringOrEmpty(row.DownloadableFileFormat),
		}
	}

	return result, nil
}

func (d *PostgresDB) webService(ctx context.Context, id uuid.UUID) (metadata.WebService, error) {
	row, err := d.Identity.GetWebServiceByID(ctx, id)
	if err != nil {
		return metadata.WebService{}, fmt.Errorf("get web service %s: %w", id, err)
	}

	result := metadata.WebService{
		ID:               row.ID,
		SourceIRI:        row.SourceIri,
		Name:             row.Name,
		Description:      row.Description,
		TemplateURL:      row.TemplateUrl,
		Parameters:       nil,
		Format:           row.Format,
		Provider:         nil,
		Documentation:    nil,
		SpatialCoverage:  nil,
		TemporalCoverage: temporal(row.TemporalStart, row.TemporalEnd),
		ContactPoints:    nil,
	}

	if result.SpatialCoverage, err = spatial(row.SpatialPoint, row.SpatialPolygon); err != nil {
		return metadata.WebService{}, err
	}

	if row.DocumentationTitle != nil || row.DocumentationDescription != nil || row.DocumentationUri != nil {
		result.Documentation = &metadata.Documentation{
			SourceIRI:   "",
			Title:       row.DocumentationTitle,
			Description: row.DocumentationDescription,
			URI:         stringOrEmpty(row.DocumentationUri),
		}
	}

	if row.ProviderOrganizationID != nil {
		provider, err := d.Identity.GetOrganizationByID(ctx, *row.ProviderOrganizationID)
		if err != nil {
			return metadata.WebService{}, fmt.Errorf("get web service provider %s: %w", id, err)
		}
		providerValue := mapOrganization(provider)
		result.Provider = &providerValue
	}

	var contacts []sqlc.ContactPoint
	if contacts, err = d.Identity.ListWebServiceContactPoints(ctx, row.ID); err != nil {
		return metadata.WebService{}, fmt.Errorf("list web service contact points %s: %w", id, err)
	}

	result.ContactPoints = mapContacts(contacts)

	parameters, err := d.Identity.ListWebServiceParameters(ctx, row.ID)
	if err != nil {
		return metadata.WebService{}, fmt.Errorf("list web service parameters %s: %w", id, err)
	}

	result.Parameters = make([]metadata.Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		decoded, err := metadata.DecodeParameter(metadata.ParameterRecord{
			Kind: metadata.ParameterKind(parameter.Kind),
			Metadata: metadata.ParameterMetadata{
				ID:             parameter.ID,
				SourceIRI:      parameter.SourceIri,
				Kind:           metadata.ParameterKind(parameter.Kind),
				Variable:       parameter.Variable,
				Property:       parameter.Property,
				Label:          parameter.Label,
				Required:       parameter.Required,
				MultipleValues: parameter.MultipleValues,
				ReadOnly:       parameter.ReadOnly,
			},
			Config: parameter.Config,
		})
		if err != nil {
			return metadata.WebService{}, fmt.Errorf("decode parameter %s: %w", parameter.SourceIri, err)
		}

		result.Parameters = append(result.Parameters, decoded)
	}

	return result, nil
}

func (d *PostgresDB) organization(ctx context.Context, id uuid.UUID) (metadata.Organization, error) {
	row, err := d.Identity.GetOrganizationByID(ctx, id)
	if err != nil {
		return metadata.Organization{}, fmt.Errorf("get organization %s: %w", id, err)
	}

	result := mapOrganization(row)

	identifiers, err := d.Identity.ListOrganizationIdentifiers(ctx, row.ID)
	if err != nil {
		return metadata.Organization{}, fmt.Errorf("list organization identifiers %s: %w", id, err)
	}

	result.Identifiers = mapIdentifiers(identifiers)

	return result, nil
}

func (d *PostgresDB) category(ctx context.Context, id uuid.UUID) (metadata.Category, error) {
	row, err := d.Identity.GetCategoryByID(ctx, id)
	if err != nil {
		return metadata.Category{}, fmt.Errorf("get category %s: %w", id, err)
	}

	result := mapCategory(row)

	children, err := d.Identity.ListCategoryChildren(ctx, row.ID)
	if err != nil {
		return metadata.Category{}, fmt.Errorf("list category children %s: %w", id, err)
	}

	result.Children = make([]metadata.Category, 0, len(children))
	for _, child := range children {
		result.Children = append(result.Children, metadata.Category{
			ID:        child.ID,
			SourceIRI: child.SourceIri,
			Name:      child.Name,
			Scheme: metadata.CategoryScheme{
				ID:          uuid.Nil,
				SourceIRI:   "",
				Name:        "",
				Label:       nil,
				Logo:        nil,
				Page:        nil,
				Color:       nil,
				Position:    0,
				Concept:     "",
				TopConcepts: nil,
			},
			Children: nil,
		})
	}

	return result, nil
}

func (d *PostgresDB) categoryScheme(ctx context.Context, id uuid.UUID) (metadata.CategoryScheme, error) {
	row, err := d.Identity.GetCategorySchemeByID(ctx, id)
	if err != nil {
		return metadata.CategoryScheme{}, fmt.Errorf("get category scheme %s: %w", id, err)
	}

	result := metadata.CategoryScheme{
		ID:          row.ID,
		SourceIRI:   row.SourceIri,
		Name:        row.Name,
		Label:       row.Label,
		Logo:        row.Logo,
		Page:        row.Page,
		Color:       row.Color,
		Position:    row.Position,
		Concept:     metadata.TopLevelConcept(row.Concept),
		TopConcepts: nil,
	}

	concepts, err := d.Identity.ListCategorySchemeTopConcepts(ctx, row.ID)
	if err != nil {
		return metadata.CategoryScheme{}, fmt.Errorf("list category scheme top concepts %s: %w", id, err)
	}

	result.TopConcepts = make([]metadata.Category, 0, len(concepts))
	for _, concept := range concepts {
		result.TopConcepts = append(result.TopConcepts, metadata.Category{
			ID:        concept.ID,
			SourceIRI: concept.SourceIri,
			Name:      concept.Name,
			Scheme: metadata.CategoryScheme{
				ID:          uuid.Nil,
				SourceIRI:   "",
				Name:        "",
				Label:       nil,
				Logo:        nil,
				Page:        nil,
				Color:       nil,
				Position:    0,
				Concept:     "",
				TopConcepts: nil,
			},
			Children: nil,
		})
	}

	return result, nil
}

func (d *PostgresDB) datasetIdentifiers(ctx context.Context, id uuid.UUID) ([]metadata.Identifier, error) {
	rows, err := d.Identity.ListDatasetIdentifiers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list dataset identifiers: %w", err)
	}

	return mapIdentifiers(rows), nil
}

func (d *PostgresDB) softwareApplicationIdentifiers(ctx context.Context, id uuid.UUID) ([]metadata.Identifier, error) {
	rows, err := d.Identity.ListSoftwareApplicationIdentifiers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list software application identifiers: %w", err)
	}

	return mapIdentifiers(rows), nil
}

func (d *PostgresDB) softwareSourceCodeIdentifiers(ctx context.Context, id uuid.UUID) ([]metadata.Identifier, error) {
	rows, err := d.Identity.ListSoftwareSourceCodeIdentifiers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list software source code identifiers: %w", err)
	}

	return mapIdentifiers(rows), nil
}

func (d *PostgresDB) softwareApplicationCreators(ctx context.Context, id uuid.UUID) ([]metadata.Organization, error) {
	rows, err := d.Identity.ListSoftwareApplicationCreators(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list software application creators: %w", err)
	}

	return mapOrganizations(rows), nil
}

func (d *PostgresDB) softwareSourceCodeCreators(ctx context.Context, id uuid.UUID) ([]metadata.Organization, error) {
	rows, err := d.Identity.ListSoftwareSourceCodeCreators(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list software source code creators: %w", err)
	}

	return mapOrganizations(rows), nil
}

func (d *PostgresDB) softwareApplicationCategories(ctx context.Context, id uuid.UUID) ([]metadata.Category, error) {
	rows, err := d.Identity.ListSoftwareApplicationCategories(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list software application categories: %w", err)
	}

	result := make([]metadata.Category, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapCategory(sqlc.GetCategoryByIDRow(row)))
	}

	return result, nil
}

func (d *PostgresDB) softwareSourceCodeCategories(ctx context.Context, id uuid.UUID) ([]metadata.Category, error) {
	rows, err := d.Identity.ListSoftwareSourceCodeCategories(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list software source code categories: %w", err)
	}

	result := make([]metadata.Category, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapCategory(sqlc.GetCategoryByIDRow(row)))
	}

	return result, nil
}

func (d *PostgresDB) datasetContacts(ctx context.Context, id uuid.UUID) ([]metadata.ContactPoint, error) {
	rows, err := d.Identity.ListDatasetContactPoints(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list dataset contact points: %w", err)
	}

	return mapContacts(rows), nil
}

func (d *PostgresDB) datasetProviders(ctx context.Context, id uuid.UUID) ([]metadata.Organization, error) {
	rows, err := d.Identity.ListDatasetProviders(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list dataset providers: %w", err)
	}

	return mapOrganizations(rows), nil
}

func (d *PostgresDB) datasetCategories(ctx context.Context, id uuid.UUID) ([]metadata.Category, error) {
	rows, err := d.Identity.ListDatasetCategories(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list dataset categories: %w", err)
	}

	result := make([]metadata.Category, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapCategory(sqlc.GetCategoryByIDRow(row)))
	}

	return result, nil
}

func mapOrganization(row sqlc.Organization) metadata.Organization {
	return metadata.Organization{
		ID:          row.ID,
		SourceIRI:   row.SourceIri,
		Name:        row.Name,
		Street:      row.Street,
		Locality:    row.Locality,
		PostalCode:  row.PostalCode,
		Country:     row.Country,
		LogoURL:     row.LogoUrl,
		PageURL:     row.PageUrl,
		Email:       row.Email,
		LEICode:     row.LeiCode,
		Identifiers: nil,
	}
}

func mapOrganizations(rows []sqlc.Organization) []metadata.Organization {
	result := make([]metadata.Organization, 0, len(rows))

	for _, row := range rows {
		result = append(result, mapOrganization(row))
	}

	return result
}

func mapIdentifiers(rows []sqlc.Identifier) []metadata.Identifier {
	result := make([]metadata.Identifier, 0, len(rows))

	for _, row := range rows {
		result = append(result, metadata.Identifier{
			ID:        row.ID,
			SourceIRI: row.SourceIri,
			Type:      metadata.IdentifierType(row.Type),
			Value:     row.Value,
		})
	}

	return result
}

func mapContacts(rows []sqlc.ContactPoint) []metadata.ContactPoint {
	result := make([]metadata.ContactPoint, 0, len(rows))

	for _, row := range rows {
		result = append(result, metadata.ContactPoint{
			ID:        row.ID,
			SourceIRI: row.SourceIri,
			Email:     row.Email,
			Role:      metadata.ContactPointRole(row.Role),
		})
	}

	return result
}

func mapCategory(row sqlc.GetCategoryByIDRow) metadata.Category {
	return metadata.Category{
		ID:        row.ID,
		SourceIRI: row.SourceIri,
		Name:      row.Name,
		Scheme: metadata.CategoryScheme{
			ID:          row.SchemeID,
			SourceIRI:   row.SchemeSourceIri,
			Name:        row.SchemeName,
			Label:       row.SchemeLabel,
			Logo:        row.SchemeLogo,
			Page:        row.SchemePage,
			Color:       row.SchemeColor,
			Position:    row.SchemePosition,
			Concept:     metadata.TopLevelConcept(row.SchemeConcept),
			TopConcepts: nil,
		},
		Children: nil,
	}
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func temporal(start, end *time.Time) *metadata.Temporal {
	if start == nil && end == nil {
		return nil
	}

	return &metadata.Temporal{Start: start, End: end}
}

// sqlc exposes ST_AsText expressions as interface{}; constrain values at this boundary.
func spatial(pointValue, polygonValue any) (*metadata.Spatial, error) {
	pointText := geometryText(pointValue)
	polygonText := geometryText(polygonValue)

	if pointText == "" && polygonText == "" {
		return nil, nil
	}

	result := &metadata.Spatial{
		Point:   nil,
		Polygon: nil,
	}

	if pointText != "" {
		if !strings.HasPrefix(pointText, "POINT(") || !strings.HasSuffix(pointText, ")") {
			return nil, fmt.Errorf("invalid point geometry: %s", pointText)
		}

		point, err := parseCoordinate(pointText[len("POINT(") : len(pointText)-1])
		if err != nil {
			return nil, fmt.Errorf("parse point geometry: %w", err)
		}

		result.Point = &point
	}

	if polygonText != "" {
		if !strings.HasPrefix(polygonText, "POLYGON((") || !strings.HasSuffix(polygonText, "))") {
			return nil, fmt.Errorf("invalid polygon geometry: %s", polygonText)
		}

		for value := range strings.SplitSeq(polygonText[len("POLYGON(("):len(polygonText)-2], ",") {
			coordinate, err := parseCoordinate(value)
			if err != nil {
				return nil, fmt.Errorf("parse polygon geometry: %w", err)
			}

			result.Polygon = append(result.Polygon, coordinate)
		}
	}

	return result, nil
}

func geometryText(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return ""
	}
}

func parseCoordinate(value string) (metadata.Coordinate, error) {
	parts := strings.Fields(value)

	if len(parts) != 2 {
		return metadata.Coordinate{}, fmt.Errorf("invalid coordinate: %s", value)
	}

	longitude, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return metadata.Coordinate{}, fmt.Errorf("parse longitude: %w", err)
	}

	latitude, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return metadata.Coordinate{}, fmt.Errorf("parse latitude: %w", err)
	}

	return metadata.Coordinate{
		Latitude:  latitude,
		Longitude: longitude,
	}, nil
}
