package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/turtle"
	"github.com/google/uuid"
)

// ImportBatch replaces owned dataset children and upserts reachable shared resources
// in one transaction.
func (d *PostgresDB) ImportBatch(ctx context.Context, batch turtle.Batch) error {
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)

	for _, scheme := range batch.CategorySchemes {
		if err := queries.UpsertCategoryScheme(ctx, sqlc.UpsertCategorySchemeParams{
			ID:        scheme.ID,
			SourceIri: scheme.SourceIRI,
			Name:      scheme.Name,
			Concept:   sqlc.TopLevelConcept(scheme.Concept),
		}); err != nil {
			return fmt.Errorf("upsert category scheme %q: %w", scheme.SourceIRI, err)
		}
	}
	for _, category := range batch.Categories {
		if err := queries.UpsertCategory(ctx, sqlc.UpsertCategoryParams{
			ID:        category.ID,
			SourceIri: category.SourceIRI,
			SchemeID:  category.Scheme.ID,
			Name:      category.Name,
		}); err != nil {
			return fmt.Errorf("upsert category %q: %w", category.SourceIRI, err)
		}
	}
	for _, organization := range batch.Organizations {
		if err := queries.UpsertOrganization(ctx, sqlc.UpsertOrganizationParams{
			ID:         organization.ID,
			SourceIri:  organization.SourceIRI,
			Name:       organization.Name,
			Street:     nullableStringPtr(organization.Street),
			Locality:   nullableStringPtr(organization.Locality),
			PostalCode: nullableStringPtr(organization.PostalCode),
			Country:    nullableStringPtr(organization.Country),
			LogoUrl:    nullableStringPtr(organization.LogoURL),
			PageUrl:    nullableStringPtr(organization.PageURL),
			Email:      nullableStringPtr(organization.Email),
		}); err != nil {
			return fmt.Errorf("upsert organization %q: %w", organization.SourceIRI, err)
		}
		for _, identifier := range organization.Identifiers {
			if err := insertIdentifier(ctx, queries, identifier); err != nil {
				return err
			}
			if err := queries.LinkOrganizationIdentifier(ctx, sqlc.LinkOrganizationIdentifierParams{
				OrganizationID: organization.ID,
				IdentifierID:   identifier.ID,
			}); err != nil {
				return fmt.Errorf("link organization identifier: %w", err)
			}
		}
	}
	for _, contact := range batch.ContactPoints {
		if err := queries.UpsertContactPoint(ctx, sqlc.UpsertContactPointParams{
			ID:        contact.ID,
			SourceIri: contact.SourceIRI,
			Email:     contact.Email,
			Role:      string(contact.Role),
		}); err != nil {
			return fmt.Errorf("upsert contact point %q: %w", contact.SourceIRI, err)
		}
	}

	for _, dataset := range batch.Datasets {
		if err := replaceDataset(ctx, queries, dataset); err != nil {
			return err
		}
	}
	for _, edge := range batch.CategoryEdges {
		if err := queries.LinkCategoryChild(ctx, sqlc.LinkCategoryChildParams{
			ParentCategoryID: edge.ParentID,
			ChildCategoryID:  edge.ChildID,
		}); err != nil {
			return fmt.Errorf("upsert category edge: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import transaction: %w", err)
	}
	return nil
}

func replaceDataset(ctx context.Context, queries *sqlc.Queries, dataset metadata.Dataset) error {
	if err := queries.DeleteDatasetIdentifiers(ctx, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset identifiers %q: %w", dataset.SourceIRI, err)
	}
	if err := queries.DeleteDatasetContactPoints(ctx, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset contacts %q: %w", dataset.SourceIRI, err)
	}
	if err := queries.DeleteDatasetProviders(ctx, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset providers %q: %w", dataset.SourceIRI, err)
	}
	if err := queries.DeleteDatasetCategories(ctx, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset categories %q: %w", dataset.SourceIRI, err)
	}
	if err := queries.DeleteDatasetWebServices(ctx, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset web services %q: %w", dataset.SourceIRI, err)
	}
	if err := queries.DeleteDatasetDistributions(ctx, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset distributions %q: %w", dataset.SourceIRI, err)
	}

	if err := queries.UpsertDataset(ctx, sqlc.UpsertDatasetParams{
		ID:                dataset.ID,
		SourceIri:         dataset.SourceIRI,
		Title:             dataset.Title,
		Description:       dataset.Description,
		TemporalStart:     temporalStart(dataset),
		TemporalEnd:       temporalEnd(dataset),
		Status:            sqlc.DatasetStatus(dataset.Status),
		EditorUserID:      nullableUserID(dataset),
		GroupID:           nullableGroupID(dataset),
		SourceCreatedAt:   dataset.SourceCreated,
		SourceUpdatedAt:   dataset.SourceUpdated,
		OriginalDatasetID: dataset.OriginalDatasetID,
		SpatialPoint:      pointWKT(dataset.SpatialCoverage),
		SpatialPolygon:    polygonWKT(dataset.SpatialCoverage),
	}); err != nil {
		return fmt.Errorf("upsert dataset %q: %w", dataset.SourceIRI, err)
	}

	for _, identifier := range dataset.PersistentIdentifiers {
		if err := insertIdentifier(ctx, queries, identifier); err != nil {
			return err
		}
		if err := queries.LinkDatasetIdentifier(ctx, sqlc.LinkDatasetIdentifierParams{DatasetID: dataset.ID, IdentifierID: identifier.ID}); err != nil {
			return fmt.Errorf("link dataset identifier: %w", err)
		}
	}
	for _, contact := range dataset.ContactPoints {
		if err := queries.LinkDatasetContactPoint(ctx, sqlc.LinkDatasetContactPointParams{DatasetID: dataset.ID, ContactPointID: contact.ID}); err != nil {
			return fmt.Errorf("link dataset contact point: %w", err)
		}
	}
	for _, organization := range dataset.Providers {
		if err := queries.LinkDatasetProvider(ctx, sqlc.LinkDatasetProviderParams{DatasetID: dataset.ID, OrganizationID: organization.ID}); err != nil {
			return fmt.Errorf("link dataset provider: %w", err)
		}
	}
	for _, category := range dataset.Categories {
		if err := queries.LinkDatasetCategory(ctx, sqlc.LinkDatasetCategoryParams{DatasetID: dataset.ID, CategoryID: category.ID}); err != nil {
			return fmt.Errorf("link dataset category: %w", err)
		}
	}
	for _, distribution := range dataset.Distributions {
		if err := insertDistribution(ctx, queries, dataset.ID, distribution); err != nil {
			return err
		}
	}
	return nil
}

func insertIdentifier(ctx context.Context, queries *sqlc.Queries, identifier metadata.Identifier) error {
	if err := queries.UpsertIdentifier(ctx, sqlc.UpsertIdentifierParams{
		ID:        identifier.ID,
		SourceIri: identifier.SourceIRI,
		Type:      string(identifier.Type),
		Value:     identifier.Value,
	}); err != nil {
		return fmt.Errorf("upsert identifier: %w", err)
	}
	return nil
}

func insertDistribution(ctx context.Context, queries *sqlc.Queries, datasetID uuid.UUID, distribution metadata.Distribution) error {
	if distribution.WebService != nil {
		if err := insertWebService(ctx, queries, *distribution.WebService); err != nil {
			return err
		}
	}
	if err := queries.InsertDistribution(ctx, sqlc.InsertDistributionParams{
		ID:                     distribution.ID,
		SourceIri:              distribution.SourceIRI,
		DatasetID:              datasetID,
		Title:                  distribution.Title,
		Description:            distribution.Description,
		License:                nullableStringPtr(distribution.License),
		DownloadableFileUrl:    downloadableURL(distribution),
		DownloadableFileFormat: downloadableFormat(distribution),
		WebServiceID:           webServiceID(distribution),
	}); err != nil {
		return fmt.Errorf("insert distribution %q: %w", distribution.SourceIRI, err)
	}
	return nil
}

func insertWebService(ctx context.Context, queries *sqlc.Queries, service metadata.WebService) error {
	if err := queries.InsertWebService(ctx, sqlc.InsertWebServiceParams{
		ID:                       service.ID,
		SourceIri:                service.SourceIRI,
		Name:                     service.Name,
		Description:              service.Description,
		TemplateUrl:              service.TemplateURL,
		Format:                   service.Format,
		DocumentationTitle:       documentationTitle(service),
		DocumentationDescription: documentationDescription(service),
		DocumentationUri:         documentationURI(service),
		SpatialPoint:             pointWKT(service.SpatialCoverage),
		SpatialPolygon:           polygonWKT(service.SpatialCoverage),
		TemporalStart:            temporalStartValue(service.TemporalCoverage),
		TemporalEnd:              temporalEndValue(service.TemporalCoverage),
	}); err != nil {
		return fmt.Errorf("insert web service %q: %w", service.SourceIRI, err)
	}
	for _, provider := range service.Providers {
		if err := queries.LinkWebServiceProvider(ctx, sqlc.LinkWebServiceProviderParams{
			WebServiceID:   service.ID,
			OrganizationID: provider.ID,
		}); err != nil {
			return fmt.Errorf("link web service provider: %w", err)
		}
	}
	for _, contact := range service.ContactPoints {
		if err := queries.LinkWebServiceContactPoint(ctx, sqlc.LinkWebServiceContactPointParams{
			WebServiceID:   service.ID,
			ContactPointID: contact.ID,
		}); err != nil {
			return fmt.Errorf("link web service contact point: %w", err)
		}
	}
	for _, parameter := range service.Parameters {
		record, err := metadata.EncodeParameter(parameter)
		if err != nil {
			return fmt.Errorf("encode parameter: %w", err)
		}
		if err := queries.InsertWebServiceParameter(ctx, sqlc.InsertWebServiceParameterParams{
			ID:             record.Metadata.ID,
			SourceIri:      record.Metadata.SourceIRI,
			WebServiceID:   service.ID,
			Kind:           sqlc.ParameterKind(record.Kind),
			Variable:       record.Metadata.Variable,
			Property:       record.Metadata.Property,
			Label:          record.Metadata.Label,
			Required:       record.Metadata.Required,
			MultipleValues: record.Metadata.MultipleValues,
			ReadOnly:       record.Metadata.ReadOnly,
			Config:         record.Config,
		}); err != nil {
			return fmt.Errorf("insert web service parameter: %w", err)
		}
	}
	return nil
}

func temporalStart(dataset metadata.Dataset) *time.Time {
	if dataset.TemporalCoverage == nil || dataset.TemporalCoverage.Start == nil {
		return nil
	}
	return dataset.TemporalCoverage.Start
}

func temporalEnd(dataset metadata.Dataset) *time.Time {
	if dataset.TemporalCoverage == nil || dataset.TemporalCoverage.End == nil {
		return nil
	}
	return dataset.TemporalCoverage.End
}

func nullableUserID(dataset metadata.Dataset) *uuid.UUID {
	if dataset.Editor == nil {
		return nil
	}
	return &dataset.Editor.ID
}

func nullableGroupID(dataset metadata.Dataset) *uuid.UUID {
	if dataset.Group == nil {
		return nil
	}
	return &dataset.Group.ID
}

func downloadableURL(distribution metadata.Distribution) *string {
	if distribution.DownloadableFile == nil {
		return nil
	}
	return nullableString(distribution.DownloadableFile.URL)
}

func downloadableFormat(distribution metadata.Distribution) *string {
	if distribution.DownloadableFile == nil {
		return nil
	}
	return nullableString(distribution.DownloadableFile.Format)
}

func webServiceID(distribution metadata.Distribution) *uuid.UUID {
	if distribution.WebService == nil {
		return nil
	}
	return &distribution.WebService.ID
}

func documentationTitle(service metadata.WebService) *string {
	if service.Documentation == nil || service.Documentation.Title == nil {
		return nil
	}
	return nullableString(*service.Documentation.Title)
}

func documentationDescription(service metadata.WebService) *string {
	if service.Documentation == nil || service.Documentation.Description == nil {
		return nil
	}
	return nullableString(*service.Documentation.Description)
}

func documentationURI(service metadata.WebService) *string {
	if service.Documentation == nil || strings.TrimSpace(service.Documentation.URI) == "" {
		return nil
	}
	return nullableString(service.Documentation.URI)
}

func pointWKT(spatial *metadata.Spatial) *string {
	if spatial == nil || spatial.Point == nil {
		return nil
	}
	return nullableString(fmt.Sprintf("POINT(%g %g)", spatial.Point.Longitude, spatial.Point.Latitude))
}

func polygonWKT(spatial *metadata.Spatial) *string {
	if spatial == nil || len(spatial.Polygon) == 0 {
		return nil
	}
	coordinates := make([]string, len(spatial.Polygon))
	for i, coordinate := range spatial.Polygon {
		coordinates[i] = fmt.Sprintf("%g %g", coordinate.Longitude, coordinate.Latitude)
	}
	return nullableString("POLYGON((" + strings.Join(coordinates, ",") + "))")
}

func temporalStartValue(temporal *metadata.Temporal) *time.Time {
	if temporal == nil || temporal.Start == nil {
		return nil
	}
	return temporal.Start
}

func temporalEndValue(temporal *metadata.Temporal) *time.Time {
	if temporal == nil || temporal.End == nil {
		return nil
	}
	return temporal.End
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableStringPtr(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}
