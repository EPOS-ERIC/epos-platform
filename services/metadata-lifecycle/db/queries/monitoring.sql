-- name: ListPublishedMonitoringDistributions :many
SELECT
    concept.id AS concept_id,
    concept.name AS concept_name,
    owning_group.id AS group_id,
    owning_group.name AS group_name,
    sqlc.embed(distribution)
FROM
    distributions AS distribution
    JOIN dataset_versions AS version ON version.id = distribution.dataset_version_id
    JOIN datasets AS dataset ON dataset.id = version.dataset_id
    JOIN groups AS owning_group ON owning_group.id = dataset.group_id
    JOIN concepts AS concept ON concept.id = owning_group.concept_id
WHERE
    version.status = 'PUBLISHED'
ORDER BY
    lower(concept.name),
    concept.id,
    owning_group.position,
    owning_group.name,
    owning_group.id,
    distribution.id;

-- name: ListMonitoringDatasetContactPoints :many
SELECT
    link.dataset_version_id,
    sqlc.embed(contact)
FROM
    dataset_version_contact_points AS link
    JOIN contact_points AS contact ON contact.id = link.contact_point_id
WHERE
    link.dataset_version_id = ANY(sqlc.arg(dataset_version_ids)::uuid [])
ORDER BY
    link.dataset_version_id,
    contact.id;

-- name: ListMonitoringWebServices :many
SELECT
    service.*
FROM
    web_services AS service
WHERE
    service.id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    service.id;
