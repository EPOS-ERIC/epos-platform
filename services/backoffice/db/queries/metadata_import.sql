-- name: UpsertCategoryScheme :exec
INSERT INTO category_schemes (id, source_iri, name, concept, label, logo, page, color, position)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (source_iri) DO UPDATE SET name = excluded.name, concept = excluded.concept,
label = excluded.label, logo = excluded.logo, page = excluded.page, color = excluded.color, position = excluded.position;

-- name: UpsertCategory :exec
INSERT INTO categories (id, source_iri, scheme_id, name)
VALUES ($1, $2, $3, $4)
ON CONFLICT (source_iri) DO UPDATE SET scheme_id = excluded.scheme_id, name = excluded.name;

-- name: UpsertOrganization :exec
INSERT INTO organizations (id, source_iri, name, street, locality, postal_code, country, logo_url, page_url, email, lei_code)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (source_iri) DO UPDATE SET name = excluded.name, street = excluded.street,
locality = excluded.locality, postal_code = excluded.postal_code, country = excluded.country,
logo_url = excluded.logo_url, page_url = excluded.page_url, email = excluded.email, lei_code = excluded.lei_code;

-- name: LinkOrganizationIdentifier :exec
INSERT INTO organization_identifiers (organization_id, identifier_id)
VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UpsertContactPoint :exec
INSERT INTO contact_points (id, source_iri, email, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (source_iri) DO UPDATE SET email = excluded.email, role = excluded.role;

-- name: LinkCategoryChild :exec
INSERT INTO category_children (parent_category_id, child_category_id)
VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteDatasetIdentifiers :exec
DELETE FROM dataset_identifiers WHERE dataset_id = $1;

-- name: DeleteDatasetContactPoints :exec
DELETE FROM dataset_contact_points WHERE dataset_id = $1;

-- name: DeleteDatasetProviders :exec
DELETE FROM dataset_providers WHERE dataset_id = $1;

-- name: DeleteDatasetCategories :exec
DELETE FROM dataset_categories WHERE dataset_id = $1;

-- name: DeleteDatasetWebServices :exec
DELETE FROM web_services WHERE id IN (SELECT web_service_id FROM distributions WHERE dataset_id = $1 AND web_service_id IS NOT NULL);

-- name: DeleteDatasetDistributions :exec
DELETE FROM distributions WHERE dataset_id = $1;

-- name: UpsertDataset :exec
INSERT INTO datasets (id, source_iri, title, description, keywords, accrual_periodicity, quality_annotation, access_right,
    temporal_start, temporal_end, status, editor_user_id, group_id, source_created_at, source_updated_at, original_dataset_id, spatial_point, spatial_polygon)
VALUES ($1, $2, $3, $4, $5, sqlc.narg(accrual_periodicity), sqlc.narg(quality_annotation), sqlc.narg(access_right), $6, $7, $8, $9, $10, sqlc.narg(source_created_at), sqlc.narg(source_updated_at), $11,
	CASE WHEN sqlc.narg(spatial_point)::text IS NULL THEN NULL ELSE ST_GeomFromText(sqlc.narg(spatial_point)::text, 4326) END,
	CASE WHEN sqlc.narg(spatial_polygon)::text IS NULL THEN NULL ELSE ST_GeomFromText(sqlc.narg(spatial_polygon)::text, 4326) END)
ON CONFLICT (source_iri) DO UPDATE SET id = excluded.id, title = excluded.title,
description = excluded.description, keywords = excluded.keywords, accrual_periodicity = excluded.accrual_periodicity,
quality_annotation = excluded.quality_annotation, access_right = excluded.access_right, temporal_start = excluded.temporal_start,
temporal_end = excluded.temporal_end, status = excluded.status, editor_user_id = excluded.editor_user_id,
group_id = excluded.group_id, source_created_at = excluded.source_created_at,
source_updated_at = excluded.source_updated_at, original_dataset_id = excluded.original_dataset_id,
spatial_point = excluded.spatial_point, spatial_polygon = excluded.spatial_polygon;

-- name: LinkDatasetIdentifier :exec
INSERT INTO dataset_identifiers (dataset_id, identifier_id) VALUES ($1, $2);

-- name: LinkDatasetContactPoint :exec
INSERT INTO dataset_contact_points (dataset_id, contact_point_id) VALUES ($1, $2);

-- name: LinkDatasetProvider :exec
INSERT INTO dataset_providers (dataset_id, organization_id) VALUES ($1, $2);

-- name: LinkDatasetCategory :exec
INSERT INTO dataset_categories (dataset_id, category_id) VALUES ($1, $2);

-- name: UpsertIdentifier :exec
INSERT INTO identifiers (id, source_iri, type, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET source_iri = excluded.source_iri, type = excluded.type, value = excluded.value;

-- name: InsertDistribution :exec
INSERT INTO distributions (id, source_iri, dataset_id, title, description, license, type, source_issued_at, source_modified_at, byte_size,
	downloadable_file_url, downloadable_file_format, web_service_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(source_issued_at), sqlc.narg(source_modified_at), sqlc.narg(byte_size), $8, $9, $10);

-- name: InsertWebService :exec
INSERT INTO web_services (id, source_iri, name, description, template_url, format,
	documentation_title, documentation_description, documentation_uri, spatial_point, spatial_polygon,
	temporal_start, temporal_end)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
	CASE WHEN sqlc.narg(spatial_point)::text IS NULL THEN NULL ELSE ST_GeomFromText(sqlc.narg(spatial_point)::text, 4326) END,
	CASE WHEN sqlc.narg(spatial_polygon)::text IS NULL THEN NULL ELSE ST_GeomFromText(sqlc.narg(spatial_polygon)::text, 4326) END,
	sqlc.narg(temporal_start), sqlc.narg(temporal_end));

-- name: LinkWebServiceProvider :exec
INSERT INTO web_service_providers (web_service_id, organization_id)
VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: LinkWebServiceContactPoint :exec
INSERT INTO web_service_contact_points (web_service_id, contact_point_id)
VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: InsertWebServiceParameter :exec
INSERT INTO web_service_parameters (id, source_iri, web_service_id, kind, variable, property, label,
	required, multiple_values, read_only, config)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
