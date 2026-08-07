package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/domain"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/importer"
	"github.com/google/uuid"
)

// ImportBatch replaces owned dataset children and upserts reachable shared resources
// in one transaction.
func (d *PostgresDB) ImportBatch(ctx context.Context, batch importer.Batch) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, scheme := range batch.CategorySchemes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO category_schemes (id, source_iri, name, concept)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (source_iri) DO UPDATE SET name = excluded.name, concept = excluded.concept`,
			scheme.ID, scheme.SourceIRI, scheme.Name, scheme.Concept); err != nil {
			return fmt.Errorf("upsert category scheme %q: %w", scheme.SourceIRI, err)
		}
	}
	for _, category := range batch.Categories {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO categories (id, source_iri, scheme_id, name)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (source_iri) DO UPDATE SET scheme_id = excluded.scheme_id, name = excluded.name`,
			category.ID, category.SourceIRI, category.Scheme.ID, category.Name); err != nil {
			return fmt.Errorf("upsert category %q: %w", category.SourceIRI, err)
		}
	}
	for _, organization := range batch.Organizations {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO organizations (id, source_iri, name, street, locality, postal_code, country, logo_url, page_url, email)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (source_iri) DO UPDATE SET name = excluded.name, street = excluded.street,
			locality = excluded.locality, postal_code = excluded.postal_code, country = excluded.country,
			logo_url = excluded.logo_url, page_url = excluded.page_url, email = excluded.email`,
			organization.ID, organization.SourceIRI, organization.Name, organization.Street, organization.Locality,
			organization.PostalCode, organization.Country, organization.LogoURL, organization.PageURL, organization.Email); err != nil {
			return fmt.Errorf("upsert organization %q: %w", organization.SourceIRI, err)
		}
		for _, identifier := range organization.Identifiers {
			if err := insertIdentifier(ctx, tx, identifier); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO organization_identifiers (organization_id, identifier_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, organization.ID, identifier.ID); err != nil {
				return fmt.Errorf("link organization identifier: %w", err)
			}
		}
	}
	for _, contact := range batch.ContactPoints {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO contact_points (id, source_iri, email, role)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (source_iri) DO UPDATE SET email = excluded.email, role = excluded.role`,
			contact.ID, contact.SourceIRI, contact.Email, contact.Role); err != nil {
			return fmt.Errorf("upsert contact point %q: %w", contact.SourceIRI, err)
		}
	}

	for _, dataset := range batch.Datasets {
		if err := d.replaceDataset(ctx, tx, dataset); err != nil {
			return err
		}
	}
	for _, edge := range batch.CategoryEdges {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO category_children (parent_category_id, child_category_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING`, edge.ParentID, edge.ChildID); err != nil {
			return fmt.Errorf("upsert category edge: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit import transaction: %w", err)
	}
	return nil
}

func (d *PostgresDB) replaceDataset(ctx context.Context, tx *sql.Tx, dataset domain.Dataset) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM dataset_identifiers WHERE dataset_id = $1`, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset identifiers %q: %w", dataset.SourceIRI, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dataset_contact_points WHERE dataset_id = $1`, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset contacts %q: %w", dataset.SourceIRI, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dataset_providers WHERE dataset_id = $1`, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset providers %q: %w", dataset.SourceIRI, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dataset_categories WHERE dataset_id = $1`, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset categories %q: %w", dataset.SourceIRI, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM web_services WHERE id IN (SELECT web_service_id FROM distributions WHERE dataset_id = $1 AND web_service_id IS NOT NULL)`, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset web services %q: %w", dataset.SourceIRI, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM distributions WHERE dataset_id = $1`, dataset.ID); err != nil {
		return fmt.Errorf("clear dataset distributions %q: %w", dataset.SourceIRI, err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO datasets (id, source_iri, title, description, temporal_start, temporal_end, status,
			editor_user_id, group_id, created_at, updated_at, original_dataset_id, spatial_point, spatial_polygon)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			CASE WHEN $13::text IS NULL THEN NULL ELSE ST_GeomFromText($13::text, 4326) END,
			CASE WHEN $14::text IS NULL THEN NULL ELSE ST_GeomFromText($14::text, 4326) END)
		ON CONFLICT (source_iri) DO UPDATE SET id = excluded.id, title = excluded.title,
			description = excluded.description, temporal_start = excluded.temporal_start,
			temporal_end = excluded.temporal_end, status = excluded.status, editor_user_id = excluded.editor_user_id,
			group_id = excluded.group_id, updated_at = excluded.updated_at, original_dataset_id = excluded.original_dataset_id,
			spatial_point = excluded.spatial_point, spatial_polygon = excluded.spatial_polygon`,
		dataset.ID, dataset.SourceIRI, dataset.Title, dataset.Description, temporalStart(dataset), temporalEnd(dataset),
		dataset.Status, nullableUserID(dataset), nullableGroupID(dataset), createdAt(dataset), updatedAt(dataset), dataset.OriginalDatasetID,
		pointWKT(dataset.SpatialCoverage), polygonWKT(dataset.SpatialCoverage)); err != nil {
		return fmt.Errorf("upsert dataset %q: %w", dataset.SourceIRI, err)
	}

	for _, identifier := range dataset.PersistentIdentifiers {
		if err := insertIdentifier(ctx, tx, identifier); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO dataset_identifiers (dataset_id, identifier_id) VALUES ($1, $2)`, dataset.ID, identifier.ID); err != nil {
			return fmt.Errorf("link dataset identifier: %w", err)
		}
	}
	for _, contact := range dataset.ContactPoints {
		if _, err := tx.ExecContext(ctx, `INSERT INTO dataset_contact_points (dataset_id, contact_point_id) VALUES ($1, $2)`, dataset.ID, contact.ID); err != nil {
			return fmt.Errorf("link dataset contact point: %w", err)
		}
	}
	for _, organization := range dataset.Providers {
		if _, err := tx.ExecContext(ctx, `INSERT INTO dataset_providers (dataset_id, organization_id) VALUES ($1, $2)`, dataset.ID, organization.ID); err != nil {
			return fmt.Errorf("link dataset provider: %w", err)
		}
	}
	for _, category := range dataset.Categories {
		if _, err := tx.ExecContext(ctx, `INSERT INTO dataset_categories (dataset_id, category_id) VALUES ($1, $2)`, dataset.ID, category.ID); err != nil {
			return fmt.Errorf("link dataset category: %w", err)
		}
	}
	for _, distribution := range dataset.Distributions {
		if err := insertDistribution(ctx, tx, dataset.ID, distribution); err != nil {
			return err
		}
	}
	return nil
}

func insertIdentifier(ctx context.Context, tx *sql.Tx, identifier domain.Identifier) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO identifiers (id, source_iri, type, value) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET source_iri = excluded.source_iri, type = excluded.type, value = excluded.value`,
		identifier.ID, identifier.SourceIRI, identifier.Type, identifier.Value)
	if err != nil {
		return fmt.Errorf("upsert identifier: %w", err)
	}
	return nil
}

func insertDistribution(ctx context.Context, tx *sql.Tx, datasetID uuid.UUID, distribution domain.Distribution) error {
	if distribution.WebService != nil {
		if err := insertWebService(ctx, tx, distribution.ID, *distribution.WebService); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO distributions (id, source_iri, dataset_id, title, description, license,
			downloadable_file_url, downloadable_file_format, web_service_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, distribution.ID, distribution.SourceIRI, datasetID,
		distribution.Title, distribution.Description, distribution.License, downloadableURL(distribution), downloadableFormat(distribution), webServiceID(distribution))
	if err != nil {
		return fmt.Errorf("insert distribution %q: %w", distribution.SourceIRI, err)
	}
	return nil
}

func insertWebService(ctx context.Context, tx *sql.Tx, distributionID uuid.UUID, service domain.WebService) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO web_services (id, source_iri, name, description, template_url, format,
			documentation_title, documentation_description, documentation_uri, spatial_point, spatial_polygon,
			temporal_start, temporal_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
			CASE WHEN $10::text IS NULL THEN NULL ELSE ST_GeomFromText($10::text, 4326) END,
			CASE WHEN $11::text IS NULL THEN NULL ELSE ST_GeomFromText($11::text, 4326) END,
			$12, $13)`, service.ID, service.SourceIRI, service.Name, service.Description,
		service.TemplateURL, service.Format, documentationTitle(service), documentationDescription(service), documentationURI(service),
		pointWKT(service.SpatialCoverage), polygonWKT(service.SpatialCoverage), temporalStartValue(service.TemporalCoverage), temporalEndValue(service.TemporalCoverage))
	if err != nil {
		return fmt.Errorf("insert web service %q: %w", service.SourceIRI, err)
	}
	for _, parameter := range service.Parameters {
		record, err := domain.EncodeParameter(parameter)
		if err != nil {
			return fmt.Errorf("encode parameter: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO web_service_parameters (id, source_iri, web_service_id, kind, variable, property, label,
				required, multiple_values, read_only, config)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`, record.Metadata.ID, record.Metadata.SourceIRI, service.ID,
			record.Kind, record.Metadata.Variable, record.Metadata.Property, record.Metadata.Label, record.Metadata.Required,
			record.Metadata.MultipleValues, record.Metadata.ReadOnly, record.Config); err != nil {
			return fmt.Errorf("insert web service parameter: %w", err)
		}
	}
	_ = distributionID
	return nil
}

func temporalStart(dataset domain.Dataset) any {
	if dataset.TemporalCoverage == nil || dataset.TemporalCoverage.Start == nil {
		return nil
	}
	return *dataset.TemporalCoverage.Start
}
func createdAt(dataset domain.Dataset) time.Time {
	if dataset.Created.IsZero() {
		return time.Now().UTC()
	}
	return dataset.Created
}
func updatedAt(dataset domain.Dataset) time.Time {
	if dataset.Updated.IsZero() {
		return time.Now().UTC()
	}
	return dataset.Updated
}
func temporalEnd(dataset domain.Dataset) any {
	if dataset.TemporalCoverage == nil || dataset.TemporalCoverage.End == nil {
		return nil
	}
	return *dataset.TemporalCoverage.End
}
func nullableUserID(dataset domain.Dataset) any {
	if dataset.Editor == nil {
		return nil
	}
	return dataset.Editor.ID
}
func nullableGroupID(dataset domain.Dataset) any {
	if dataset.Group == nil {
		return nil
	}
	return dataset.Group.ID
}
func downloadableURL(distribution domain.Distribution) any {
	if distribution.DownloadableFile == nil {
		return nil
	}
	return distribution.DownloadableFile.URL
}
func downloadableFormat(distribution domain.Distribution) any {
	if distribution.DownloadableFile == nil {
		return nil
	}
	return distribution.DownloadableFile.Format
}
func webServiceID(distribution domain.Distribution) any {
	if distribution.WebService == nil {
		return nil
	}
	return distribution.WebService.ID
}
func documentationTitle(service domain.WebService) any {
	if service.Documentation == nil || service.Documentation.Title == nil {
		return nil
	}
	return *service.Documentation.Title
}
func documentationDescription(service domain.WebService) any {
	if service.Documentation == nil || service.Documentation.Description == nil {
		return nil
	}
	return *service.Documentation.Description
}
func documentationURI(service domain.WebService) any {
	if service.Documentation == nil || strings.TrimSpace(service.Documentation.URI) == "" {
		return nil
	}
	return service.Documentation.URI
}

func pointWKT(spatial *domain.Spatial) any {
	if spatial == nil || spatial.Point == nil {
		return nil
	}
	return fmt.Sprintf("POINT(%g %g)", spatial.Point.Longitude, spatial.Point.Latitude)
}

func polygonWKT(spatial *domain.Spatial) any {
	if spatial == nil || len(spatial.Polygon) == 0 {
		return nil
	}
	coordinates := make([]string, len(spatial.Polygon))
	for i, coordinate := range spatial.Polygon {
		coordinates[i] = fmt.Sprintf("%g %g", coordinate.Longitude, coordinate.Latitude)
	}
	return "POLYGON((" + strings.Join(coordinates, ",") + "))"
}

func temporalStartValue(temporal *domain.Temporal) any {
	if temporal == nil || temporal.Start == nil {
		return nil
	}
	return *temporal.Start
}

func temporalEndValue(temporal *domain.Temporal) any {
	if temporal == nil || temporal.End == nil {
		return nil
	}
	return *temporal.End
}
