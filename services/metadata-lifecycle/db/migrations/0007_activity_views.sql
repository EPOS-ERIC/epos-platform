-- +goose Up
CREATE VIEW resources_all AS
SELECT
    'dataset'::text AS resource_type,
    id,
    group_id,
    created_at,
    created_by
FROM
    datasets
UNION
ALL
SELECT
    'software-application',
    id,
    group_id,
    created_at,
    created_by
FROM
    software_applications
UNION
ALL
SELECT
    'software-source-code',
    id,
    group_id,
    created_at,
    created_by
FROM
    software_source_codes;

CREATE VIEW resource_versions_all AS
SELECT
    'dataset'::text AS resource_type,
    version.id AS version_id,
    resource.id AS resource_id,
    resource.group_id,
    version.created_at,
    version.created_by,
    version.submitted_at,
    version.submitted_by,
    version.rejected_at,
    version.rejected_by,
    version.rejection_note,
    version.published_at,
    version.published_by,
    version.archived_at,
    version.archived_by
FROM
    dataset_versions AS version
    JOIN datasets AS resource ON resource.id = version.dataset_id
UNION
ALL
SELECT
    'software-application',
    version.id,
    resource.id,
    resource.group_id,
    version.created_at,
    version.created_by,
    version.submitted_at,
    version.submitted_by,
    version.rejected_at,
    version.rejected_by,
    version.rejection_note,
    version.published_at,
    version.published_by,
    version.archived_at,
    version.archived_by
FROM
    software_application_versions AS version
    JOIN software_applications AS resource ON resource.id = version.software_application_id
UNION
ALL
SELECT
    'software-source-code',
    version.id,
    resource.id,
    resource.group_id,
    version.created_at,
    version.created_by,
    version.submitted_at,
    version.submitted_by,
    version.rejected_at,
    version.rejected_by,
    version.rejection_note,
    version.published_at,
    version.published_by,
    version.archived_at,
    version.archived_by
FROM
    software_source_code_versions AS version
    JOIN software_source_codes AS resource ON resource.id = version.software_source_code_id;

-- +goose Down
DROP VIEW IF EXISTS resource_versions_all,
resources_all;
