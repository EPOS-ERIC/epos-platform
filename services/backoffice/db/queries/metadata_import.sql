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

-- name: UpsertSoftwareApplication :exec
INSERT INTO software_applications (id, source_iri, name, description, main_entity_of_page, license, version, keywords, download_url, requirements, operating_system)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (source_iri) DO UPDATE SET id = excluded.id, name = excluded.name, description = excluded.description, main_entity_of_page = excluded.main_entity_of_page, license = excluded.license, version = excluded.version, keywords = excluded.keywords, download_url = excluded.download_url, requirements = excluded.requirements, operating_system = excluded.operating_system;

-- name: UpsertSoftwareSourceCode :exec
INSERT INTO software_source_codes (id, source_iri, name, description, programming_language, keywords, repository, license, software_requirements, runtime_platform, main_entity_of_page, version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (source_iri) DO UPDATE SET id = excluded.id, name = excluded.name, description = excluded.description, programming_language = excluded.programming_language, keywords = excluded.keywords, repository = excluded.repository, license = excluded.license, software_requirements = excluded.software_requirements, runtime_platform = excluded.runtime_platform, main_entity_of_page = excluded.main_entity_of_page, version = excluded.version;

-- name: DeleteSoftwareApplicationIdentifiers :exec
DELETE FROM software_application_identifiers WHERE software_application_id = $1;
-- name: DeleteSoftwareSourceCodeIdentifiers :exec
DELETE FROM software_source_code_identifiers WHERE software_source_code_id = $1;
-- name: DeleteSoftwareApplicationCreators :exec
DELETE FROM software_application_creators WHERE software_application_id = $1;
-- name: DeleteSoftwareSourceCodeCreators :exec
DELETE FROM software_source_code_creators WHERE software_source_code_id = $1;
-- name: DeleteSoftwareApplicationCategories :exec
DELETE FROM software_application_categories WHERE software_application_id = $1;
-- name: DeleteSoftwareSourceCodeCategories :exec
DELETE FROM software_source_code_categories WHERE software_source_code_id = $1;
-- name: LinkSoftwareApplicationIdentifier :exec
INSERT INTO software_application_identifiers (software_application_id, identifier_id) VALUES ($1, $2);
-- name: LinkSoftwareSourceCodeIdentifier :exec
INSERT INTO software_source_code_identifiers (software_source_code_id, identifier_id) VALUES ($1, $2);
-- name: LinkSoftwareApplicationCreator :exec
INSERT INTO software_application_creators (software_application_id, organization_id) VALUES ($1, $2);
-- name: LinkSoftwareSourceCodeCreator :exec
INSERT INTO software_source_code_creators (software_source_code_id, organization_id) VALUES ($1, $2);
-- name: LinkSoftwareApplicationCategory :exec
INSERT INTO software_application_categories (software_application_id, category_id) VALUES ($1, $2);
-- name: LinkSoftwareSourceCodeCategory :exec
INSERT INTO software_source_code_categories (software_source_code_id, category_id) VALUES ($1, $2);

-- name: UpsertIdentifier :exec
INSERT INTO identifiers (id, source_iri, type, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET source_iri = excluded.source_iri, type = excluded.type, value = excluded.value;

-- name: InsertDistribution :exec
INSERT INTO distributions (id, source_iri, dataset_id, title, description, license, type, source_issued_at, source_modified_at, byte_size,
	downloadable_file_url, downloadable_file_format, web_service_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(source_issued_at), sqlc.narg(source_modified_at), sqlc.narg(byte_size), $8, $9, $10);

-- name: InsertWebService :exec
INSERT INTO web_services (id, source_iri, name, description, template_url, format, provider_organization_id,
	documentation_title, documentation_description, documentation_uri, spatial_point, spatial_polygon,
	temporal_start, temporal_end)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
	CASE WHEN sqlc.narg(spatial_point)::text IS NULL THEN NULL ELSE ST_GeomFromText(sqlc.narg(spatial_point)::text, 4326) END,
	CASE WHEN sqlc.narg(spatial_polygon)::text IS NULL THEN NULL ELSE ST_GeomFromText(sqlc.narg(spatial_polygon)::text, 4326) END,
	sqlc.narg(temporal_start), sqlc.narg(temporal_end));

-- name: LinkWebServiceContactPoint :exec
INSERT INTO web_service_contact_points (web_service_id, contact_point_id)
VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: InsertWebServiceParameter :exec
INSERT INTO web_service_parameters (id, source_iri, web_service_id, kind, variable, property, label,
	required, multiple_values, read_only, config)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetDatasetByID :one
SELECT id, source_iri, title, description, keywords, accrual_periodicity, quality_annotation, access_right,
    temporal_start, temporal_end, status, editor_user_id, group_id, created_at, updated_at, source_created_at,
    source_updated_at, original_dataset_id, COALESCE(ST_AsText(spatial_point), '') AS spatial_point,
    COALESCE(ST_AsText(spatial_polygon), '') AS spatial_polygon
FROM datasets
WHERE id = $1;

-- name: GetSoftwareApplicationByID :one
SELECT id, source_iri, name, description, main_entity_of_page, license, version, keywords, download_url, requirements, operating_system FROM software_applications WHERE id = $1;

-- name: GetSoftwareSourceCodeByID :one
SELECT id, source_iri, name, description, programming_language, keywords, repository, license, software_requirements, runtime_platform, main_entity_of_page, version FROM software_source_codes WHERE id = $1;

-- name: GetDistributionByID :one
SELECT id, source_iri, dataset_id, title, description, license, type, source_issued_at, source_modified_at,
    byte_size, downloadable_file_url, downloadable_file_format, web_service_id
FROM distributions
WHERE id = $1;

-- name: GetWebServiceByID :one
SELECT id, source_iri, name, description, template_url, format, documentation_title, documentation_description,
    documentation_uri, provider_organization_id, COALESCE(ST_AsText(spatial_point), '') AS spatial_point,
    COALESCE(ST_AsText(spatial_polygon), '') AS spatial_polygon,
    temporal_start, temporal_end
FROM web_services
WHERE id = $1;

-- name: GetOrganizationByID :one
SELECT id, source_iri, name, street, locality, postal_code, country, logo_url, page_url, email, lei_code
FROM organizations
WHERE id = $1;

-- name: GetCategoryByID :one
SELECT c.id, c.source_iri, c.name, s.id AS scheme_id, s.source_iri AS scheme_source_iri, s.name AS scheme_name,
    s.label AS scheme_label, s.logo AS scheme_logo, s.page AS scheme_page, s.color AS scheme_color,
    s.position AS scheme_position, s.concept AS scheme_concept
FROM categories c
JOIN category_schemes s ON s.id = c.scheme_id
WHERE c.id = $1;

-- name: GetCategorySchemeByID :one
SELECT id, source_iri, name, label, logo, page, color, position, concept
FROM category_schemes
WHERE id = $1;

-- name: ListDatasetIdentifiers :many
SELECT i.id, i.source_iri, i.type, i.value
FROM identifiers i
JOIN dataset_identifiers di ON di.identifier_id = i.id
WHERE di.dataset_id = $1
ORDER BY i.source_iri;

-- name: ListOrganizationIdentifiers :many
SELECT i.id, i.source_iri, i.type, i.value
FROM identifiers i
JOIN organization_identifiers oi ON oi.identifier_id = i.id
WHERE oi.organization_id = $1
ORDER BY i.source_iri;

-- name: ListSoftwareApplicationIdentifiers :many
SELECT i.id, i.source_iri, i.type, i.value FROM identifiers i
JOIN software_application_identifiers sai ON sai.identifier_id = i.id
WHERE sai.software_application_id = $1 ORDER BY i.source_iri;

-- name: ListSoftwareSourceCodeIdentifiers :many
SELECT i.id, i.source_iri, i.type, i.value FROM identifiers i
JOIN software_source_code_identifiers ssci ON ssci.identifier_id = i.id
WHERE ssci.software_source_code_id = $1 ORDER BY i.source_iri;

-- name: ListDatasetProviders :many
SELECT o.id, o.source_iri, o.name, o.street, o.locality, o.postal_code, o.country, o.logo_url, o.page_url, o.email, o.lei_code
FROM organizations o
JOIN dataset_providers dp ON dp.organization_id = o.id
WHERE dp.dataset_id = $1
ORDER BY o.source_iri;

-- name: ListDatasetCategories :many
SELECT c.id, c.source_iri, c.name, s.id AS scheme_id, s.source_iri AS scheme_source_iri, s.name AS scheme_name,
    s.label AS scheme_label, s.logo AS scheme_logo, s.page AS scheme_page, s.color AS scheme_color,
    s.position AS scheme_position, s.concept AS scheme_concept
FROM categories c
JOIN dataset_categories dc ON dc.category_id = c.id
JOIN category_schemes s ON s.id = c.scheme_id
WHERE dc.dataset_id = $1
ORDER BY c.source_iri;

-- name: ListSoftwareApplicationCategories :many
SELECT c.id, c.source_iri, c.name, s.id AS scheme_id, s.source_iri AS scheme_source_iri, s.name AS scheme_name,
    s.label AS scheme_label, s.logo AS scheme_logo, s.page AS scheme_page, s.color AS scheme_color,
    s.position AS scheme_position, s.concept AS scheme_concept
FROM categories c JOIN software_application_categories sac ON sac.category_id = c.id
JOIN category_schemes s ON s.id = c.scheme_id WHERE sac.software_application_id = $1 ORDER BY c.source_iri;

-- name: ListSoftwareSourceCodeCategories :many
SELECT c.id, c.source_iri, c.name, s.id AS scheme_id, s.source_iri AS scheme_source_iri, s.name AS scheme_name,
    s.label AS scheme_label, s.logo AS scheme_logo, s.page AS scheme_page, s.color AS scheme_color,
    s.position AS scheme_position, s.concept AS scheme_concept
FROM categories c JOIN software_source_code_categories sscc ON sscc.category_id = c.id
JOIN category_schemes s ON s.id = c.scheme_id WHERE sscc.software_source_code_id = $1 ORDER BY c.source_iri;

-- name: ListDatasetContactPoints :many
SELECT cp.id, cp.source_iri, cp.email, cp.role
FROM contact_points cp
JOIN dataset_contact_points dcp ON dcp.contact_point_id = cp.id
WHERE dcp.dataset_id = $1
ORDER BY cp.source_iri;

-- name: ListDatasetDistributions :many
SELECT id, source_iri, dataset_id, title, description, license, type, source_issued_at, source_modified_at,
    byte_size, downloadable_file_url, downloadable_file_format, web_service_id
FROM distributions
WHERE dataset_id = $1
ORDER BY source_iri;

-- name: ListSoftwareApplicationCreators :many
SELECT o.id, o.source_iri, o.name, o.street, o.locality, o.postal_code, o.country, o.logo_url, o.page_url, o.email, o.lei_code FROM organizations o
JOIN software_application_creators sac ON sac.organization_id = o.id
WHERE sac.software_application_id = $1 ORDER BY o.source_iri;

-- name: ListSoftwareSourceCodeCreators :many
SELECT o.id, o.source_iri, o.name, o.street, o.locality, o.postal_code, o.country, o.logo_url, o.page_url, o.email, o.lei_code FROM organizations o
JOIN software_source_code_creators sscc ON sscc.organization_id = o.id
WHERE sscc.software_source_code_id = $1 ORDER BY o.source_iri;

-- name: ListWebServiceContactPoints :many
SELECT cp.id, cp.source_iri, cp.email, cp.role
FROM contact_points cp
JOIN web_service_contact_points wscp ON wscp.contact_point_id = cp.id
WHERE wscp.web_service_id = $1
ORDER BY cp.source_iri;

-- name: ListWebServiceParameters :many
SELECT id, source_iri, web_service_id, kind, variable, property, label, required, multiple_values, read_only, config
FROM web_service_parameters
WHERE web_service_id = $1
ORDER BY source_iri;

-- name: ListCategoryChildren :many
SELECT c.id, c.source_iri, c.name
FROM categories c
JOIN category_children cc ON cc.child_category_id = c.id
WHERE cc.parent_category_id = $1
ORDER BY c.source_iri;

-- name: ListCategorySchemeTopConcepts :many
SELECT c.id, c.source_iri, c.name
FROM categories c
WHERE c.scheme_id = $1
  AND NOT EXISTS (
      SELECT 1
      FROM category_children cc
      WHERE cc.child_category_id = c.id
  )
ORDER BY c.source_iri;
