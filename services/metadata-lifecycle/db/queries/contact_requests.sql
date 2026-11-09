-- name: GetContactRequestResource :one
SELECT
    version.title::text AS title,
    ARRAY(
        SELECT
            contact.email::text
        FROM
            dataset_version_contact_points link
            JOIN contact_points contact ON contact.id = link.contact_point_id
        WHERE
            link.dataset_version_id = version.id
        ORDER BY
            contact.id
    )::text [] AS emails
FROM
    dataset_versions version
WHERE
    sqlc.arg(resource_type)::text = 'dataset'
    AND version.dataset_id = sqlc.arg(resource_id)::uuid
    AND version.status = 'PUBLISHED'
UNION
ALL
SELECT
    version.name::text AS title,
    ARRAY(
        SELECT
            contact.email::text
        FROM
            software_application_version_contact_points link
            JOIN contact_points contact ON contact.id = link.contact_point_id
        WHERE
            link.software_application_version_id = version.id
        ORDER BY
            contact.id
    )::text [] AS emails
FROM
    software_application_versions version
WHERE
    sqlc.arg(resource_type)::text = 'software-application'
    AND version.software_application_id = sqlc.arg(resource_id)::uuid
    AND version.status = 'PUBLISHED'
UNION
ALL
SELECT
    version.name::text AS title,
    ARRAY(
        SELECT
            contact.email::text
        FROM
            software_source_code_version_contact_points link
            JOIN contact_points contact ON contact.id = link.contact_point_id
        WHERE
            link.software_source_code_version_id = version.id
        ORDER BY
            contact.id
    )::text [] AS emails
FROM
    software_source_code_versions version
WHERE
    sqlc.arg(resource_type)::text = 'software-source-code'
    AND version.software_source_code_id = sqlc.arg(resource_id)::uuid
    AND version.status = 'PUBLISHED';

-- name: GetContactRequestDistribution :one
SELECT
    version.title::text AS dataset_title,
    distribution.title::text AS distribution_title,
    ARRAY(
        SELECT
            contact.email::text
        FROM
            web_service_contact_points link
            JOIN contact_points contact ON contact.id = link.contact_point_id
        WHERE
            link.web_service_id = service.id
        ORDER BY
            contact.id
    )::text [] AS emails
FROM
    distributions distribution
    JOIN dataset_versions version ON version.id = distribution.dataset_version_id
    JOIN web_services service ON service.id = distribution.web_service_id
WHERE
    distribution.id = sqlc.arg(distribution_id)::uuid
    AND distribution.type = 'WEB_SERVICE'
    AND version.status = 'PUBLISHED';
