-- +goose Up

CREATE TYPE web_service_http_method AS ENUM ('GET', 'POST', 'PUT', 'DELETE');
CREATE TYPE web_service_type AS ENUM ('WMS', 'GEOJSON', 'DYNAMIC', 'WMTS', 'WFS', 'COVJSON');
CREATE TYPE web_service_visible_on AS ENUM ('GRAPH', 'TABLE', 'MAP');
CREATE TYPE web_service_parameter_location AS ENUM ('PATH', 'QUERY', 'HEADER');
CREATE TYPE parameter_schema_kind AS ENUM ('STRING', 'BOOLEAN', 'INTEGER', 'FLOAT', 'DATETIME', 'TEMPLATE');

CREATE TYPE string_parameter_enum_option AS (
    value text,
    label text
);

CREATE TYPE integer_parameter_enum_option AS (
    value bigint,
    label text
);

CREATE TYPE float_parameter_enum_option AS (
    value double precision,
    label text
);

-- Shared shape rules for any {value, label} option list, converted to jsonb.
-- Composite typing already guarantees each field's SQL type; this checks
-- list-level invariants that composite types cannot express: non-empty lists,
-- non-null and non-blank values, non-blank optional labels, and uniqueness.
-- +goose StatementBegin
CREATE FUNCTION is_valid_enum_option_list(
    input_value jsonb,
    default_value text DEFAULT NULL
) RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT
        input_value IS NULL
        OR (
            jsonb_array_length(input_value) > 0
            AND NOT EXISTS (
                SELECT 1
                FROM jsonb_array_elements(input_value) AS elem
                WHERE elem ->> 'value' IS NULL
                   OR elem ->> 'value' = ''
                   OR (elem -> 'label' IS NOT NULL AND btrim(elem ->> 'label') = '')
            )
            AND (
                SELECT count(DISTINCT elem -> 'value')
                FROM jsonb_array_elements(input_value) AS elem
            ) = jsonb_array_length(input_value)
            AND (
                default_value IS NULL
                OR EXISTS (
                    SELECT 1
                    FROM jsonb_array_elements(input_value) AS elem
                    WHERE elem ->> 'value' = default_value
                )
            )
        )
$$;
-- +goose StatementEnd

-- Shared numeric-bound rules for integer and float enum lists. numeric
-- represents both bigint and finite double precision values without loss for
-- these comparisons; float callers reject non-finite values separately.
-- +goose StatementBegin
CREATE FUNCTION enum_option_list_matches_bounds(
    input_value jsonb,
    minimum_value numeric,
    maximum_value numeric,
    default_value numeric
) RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT
        input_value IS NULL
        OR (
            NOT EXISTS (
                SELECT 1
                FROM jsonb_array_elements(input_value) AS elem
                WHERE (minimum_value IS NOT NULL AND (elem ->> 'value')::numeric < minimum_value)
                   OR (maximum_value IS NOT NULL AND (elem ->> 'value')::numeric > maximum_value)
            )
            AND (
                default_value IS NULL
                OR EXISTS (
                    SELECT 1
                    FROM jsonb_array_elements(input_value) AS elem
                    WHERE (elem ->> 'value')::numeric = default_value
                )
            )
        )
$$;
-- +goose StatementEnd

-- Float-only validation for values that jsonb/numeric cannot represent as
-- meaningful finite enum options. Integer callers do not need this check.
-- +goose StatementBegin
CREATE FUNCTION enum_option_list_has_non_finite_float(input_value float_parameter_enum_option[]) RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT EXISTS (
        SELECT 1
        FROM unnest(input_value) AS opt
        WHERE (opt).value = 'NaN'::double precision
           OR (opt).value = 'Infinity'::double precision
           OR (opt).value = '-Infinity'::double precision
    );
$$;
-- +goose StatementEnd

CREATE TABLE identifiers (
    -- Identifier identity and RDF value metadata
    id                              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    type                            nonblank_text NOT NULL,
    value                           nonblank_text NOT NULL,

    -- Exactly one immutable owner
    dataset_version_id              uuid REFERENCES dataset_versions(id) ON DELETE CASCADE,
    organization_id                 uuid REFERENCES organizations(id) ON DELETE CASCADE,
    software_application_version_id uuid REFERENCES software_application_versions(id) ON DELETE CASCADE,
    software_source_code_version_id uuid REFERENCES software_source_code_versions(id) ON DELETE CASCADE,

    -- Identifiers are owned values, never shared canonical records.
    CHECK (num_nonnulls(
        dataset_version_id,
        organization_id,
        software_application_version_id,
        software_source_code_version_id
    ) = 1)
);
CREATE UNIQUE INDEX identifiers_dataset_version_content_uq ON identifiers (dataset_version_id, type, value) WHERE dataset_version_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_organization_content_uq ON identifiers (organization_id, type, value) WHERE organization_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_software_application_version_content_uq ON identifiers (software_application_version_id, type, value) WHERE software_application_version_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_software_source_code_version_content_uq ON identifiers (software_source_code_version_id, type, value) WHERE software_source_code_version_id IS NOT NULL;

CREATE TABLE dataset_version_providers (
    dataset_version_id uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    organization_id    uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (dataset_version_id, organization_id)
);
CREATE TABLE software_application_version_creators (
    software_application_version_id uuid NOT NULL REFERENCES software_application_versions(id) ON DELETE CASCADE,
    organization_id                 uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (software_application_version_id, organization_id)
);
CREATE TABLE software_source_code_version_creators (
    software_source_code_version_id uuid NOT NULL REFERENCES software_source_code_versions(id) ON DELETE CASCADE,
    organization_id                 uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (software_source_code_version_id, organization_id)
);
CREATE TABLE software_application_version_contact_points (
    software_application_version_id uuid NOT NULL REFERENCES software_application_versions(id) ON DELETE CASCADE,
    contact_point_id               uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (software_application_version_id, contact_point_id)
);
CREATE TABLE software_source_code_version_contact_points (
    software_source_code_version_id uuid NOT NULL REFERENCES software_source_code_versions(id) ON DELETE CASCADE,
    contact_point_id                uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (software_source_code_version_id, contact_point_id)
);
CREATE TABLE dataset_version_contact_points (
    dataset_version_id uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    contact_point_id   uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (dataset_version_id, contact_point_id)
);

CREATE TABLE web_services (
    -- Version-owned web service identity
    id                        uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_version_id        uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,

    -- Web service metadata
    name                      nonblank_text NOT NULL,
    description               nonblank_text_opt,
    template_url              nonblank_text NOT NULL,
    method                    web_service_http_method NOT NULL,
    type                      web_service_type NOT NULL,
    visible_on                web_service_visible_on[],
    format                    nonblank_text NOT NULL,
    provider_organization_id  uuid REFERENCES organizations(id) ON DELETE SET NULL,
    documentation_title       nonblank_text_opt,
    documentation_description nonblank_text_opt,
    documentation_uri         nonblank_text_opt,
    spatial_coverage          geometry(Geometry, 4326),
    temporal_start            timestamptz,
    temporal_end              timestamptz,

    CHECK (temporal_start IS NULL OR temporal_end IS NULL OR temporal_start <= temporal_end),
    CHECK ((documentation_title IS NULL AND documentation_description IS NULL AND documentation_uri IS NULL) OR documentation_uri IS NOT NULL),
    CHECK (visible_on IS NULL OR cardinality(visible_on) > 0)
);
CREATE TABLE web_service_bodies (
    id             uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    web_service_id uuid NOT NULL UNIQUE REFERENCES web_services(id) ON DELETE CASCADE,
    content_type   nonblank_text NOT NULL,
    template       nonblank_text NOT NULL
);
CREATE TABLE web_service_contact_points (
    web_service_id   uuid NOT NULL REFERENCES web_services(id) ON DELETE CASCADE,
    contact_point_id uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (web_service_id, contact_point_id)
);
CREATE TABLE parameters (
    id                    uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    position              integer NOT NULL,
    name                  nonblank_text NOT NULL,
    label                 nonblank_text NOT NULL,
    description           nonblank_text_opt,
    required              boolean NOT NULL DEFAULT false,
    read_only             boolean NOT NULL DEFAULT false,
    web_service_id        uuid REFERENCES web_services(id) ON DELETE CASCADE,
    web_service_body_id   uuid REFERENCES web_service_bodies(id) ON DELETE CASCADE,
    template_parameter_id uuid REFERENCES parameters(id) ON DELETE CASCADE,

    schema_kind           parameter_schema_kind NOT NULL,
    property              nonblank_text_opt,
    location              web_service_parameter_location,
    explode               boolean NOT NULL DEFAULT false,

    CHECK (num_nonnulls(web_service_id, web_service_body_id, template_parameter_id) = 1),
    CHECK (
        (web_service_id IS NOT NULL AND location IS NOT NULL)
        OR (web_service_id IS NULL AND location IS NULL AND property IS NULL AND NOT explode)
    ),
    CHECK (NOT explode OR (web_service_id IS NOT NULL AND location = 'QUERY')),
    CHECK (schema_kind <> 'TEMPLATE' OR web_service_id IS NOT NULL),
    CHECK (template_parameter_id IS NULL OR schema_kind <> 'TEMPLATE'),
    UNIQUE (id, schema_kind)
);

CREATE TABLE string_parameter_schemas (
    parameter_id    uuid PRIMARY KEY,
    schema_kind    parameter_schema_kind NOT NULL DEFAULT 'STRING',
    pattern        nonblank_text_opt,
    default_value  nonblank_text_opt,
    enum           string_parameter_enum_option[],
    multiple_values boolean NOT NULL DEFAULT false,

    FOREIGN KEY (parameter_id, schema_kind) REFERENCES parameters(id, schema_kind) ON DELETE CASCADE,
    CHECK (schema_kind = 'STRING'),
    CHECK (is_valid_enum_option_list(to_jsonb(enum), default_value)),
    CHECK (NOT multiple_values OR enum IS NOT NULL)
);

CREATE TABLE boolean_parameter_schemas (
    parameter_id   uuid PRIMARY KEY,
    schema_kind    parameter_schema_kind NOT NULL DEFAULT 'BOOLEAN',
    default_value  boolean,

    FOREIGN KEY (parameter_id, schema_kind) REFERENCES parameters(id, schema_kind) ON DELETE CASCADE,
    CHECK (schema_kind = 'BOOLEAN')
);

CREATE TABLE integer_parameter_schemas (
    parameter_id    uuid PRIMARY KEY,
    schema_kind     parameter_schema_kind NOT NULL DEFAULT 'INTEGER',
    minimum         bigint,
    maximum         bigint,
    default_value   bigint,
    enum            integer_parameter_enum_option[],
    multiple_values boolean NOT NULL DEFAULT false,

    FOREIGN KEY (parameter_id, schema_kind) REFERENCES parameters(id, schema_kind) ON DELETE CASCADE,
    CHECK (schema_kind = 'INTEGER'),
    CHECK (is_valid_enum_option_list(to_jsonb(enum))),
    CHECK (enum_option_list_matches_bounds(to_jsonb(enum), minimum, maximum, default_value)),
    CHECK (NOT multiple_values OR enum IS NOT NULL),
    CHECK (minimum IS NULL OR maximum IS NULL OR minimum <= maximum),
    CHECK (default_value IS NULL OR minimum IS NULL OR default_value >= minimum),
    CHECK (default_value IS NULL OR maximum IS NULL OR default_value <= maximum)
);

CREATE TABLE float_parameter_schemas (
    parameter_id    uuid PRIMARY KEY,
    schema_kind     parameter_schema_kind NOT NULL DEFAULT 'FLOAT',
    minimum         double precision,
    maximum         double precision,
    default_value   double precision,
    enum            float_parameter_enum_option[],
    multiple_values boolean NOT NULL DEFAULT false,

    FOREIGN KEY (parameter_id, schema_kind) REFERENCES parameters(id, schema_kind) ON DELETE CASCADE,
    CHECK (schema_kind = 'FLOAT'),
    CHECK (is_valid_enum_option_list(to_jsonb(enum))),
    CHECK (enum IS NULL OR NOT enum_option_list_has_non_finite_float(enum)),
    CHECK (enum_option_list_matches_bounds(to_jsonb(enum), minimum::numeric, maximum::numeric, default_value::numeric)),
    CHECK (NOT multiple_values OR enum IS NOT NULL),
    CHECK (minimum IS NULL OR maximum IS NULL OR minimum <= maximum),
    CHECK (default_value IS NULL OR minimum IS NULL OR default_value >= minimum),
    CHECK (default_value IS NULL OR maximum IS NULL OR default_value <= maximum)
);

CREATE TABLE datetime_parameter_schemas (
    parameter_id   uuid PRIMARY KEY,
    schema_kind    parameter_schema_kind NOT NULL DEFAULT 'DATETIME',
    minimum        timestamptz,
    maximum        timestamptz,
    default_value  timestamptz,
    layout         nonblank_text_opt,

    FOREIGN KEY (parameter_id, schema_kind) REFERENCES parameters(id, schema_kind) ON DELETE CASCADE,
    CHECK (schema_kind = 'DATETIME'),
    CHECK (minimum IS NULL OR maximum IS NULL OR minimum <= maximum)
);

CREATE TABLE parameter_templates (
    parameter_id uuid PRIMARY KEY,
    schema_kind  parameter_schema_kind NOT NULL DEFAULT 'TEMPLATE',
    template     nonblank_text NOT NULL,

    FOREIGN KEY (parameter_id, schema_kind) REFERENCES parameters(id, schema_kind) ON DELETE CASCADE,
    CHECK (schema_kind = 'TEMPLATE')
);

CREATE UNIQUE INDEX parameters_web_service_name_uq ON parameters (web_service_id, name) WHERE web_service_id IS NOT NULL;
CREATE UNIQUE INDEX parameters_web_service_body_name_uq ON parameters (web_service_body_id, name) WHERE web_service_body_id IS NOT NULL;
CREATE UNIQUE INDEX parameters_template_name_uq ON parameters (template_parameter_id, name) WHERE template_parameter_id IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION validate_parameter_schema() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    v_parameter_id uuid;
    v_parameter_kind parameter_schema_kind;
    v_schema_exists boolean;
    v_template_parent_id uuid;
    v_parent_kind parameter_schema_kind;
BEGIN
    IF TG_TABLE_NAME = 'parameters' THEN
        v_parameter_id := NEW.id;
    ELSE
        v_parameter_id := COALESCE(NEW.parameter_id, OLD.parameter_id);
    END IF;

    IF TG_OP = 'DELETE' AND TG_TABLE_NAME <> 'parameters' THEN
        IF NOT EXISTS (SELECT 1 FROM parameters WHERE id = v_parameter_id) THEN
            RETURN OLD;
        END IF;
    END IF;

    SELECT p.schema_kind, p.template_parameter_id
    INTO v_parameter_kind, v_template_parent_id
    FROM parameters AS p
    WHERE p.id = v_parameter_id;

    IF v_parameter_kind IS NULL THEN
        IF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
        RETURN NEW;
    END IF;

    v_schema_exists := CASE v_parameter_kind
        WHEN 'STRING' THEN EXISTS (SELECT 1 FROM string_parameter_schemas WHERE parameter_id = v_parameter_id)
        WHEN 'BOOLEAN' THEN EXISTS (SELECT 1 FROM boolean_parameter_schemas WHERE parameter_id = v_parameter_id)
        WHEN 'INTEGER' THEN EXISTS (SELECT 1 FROM integer_parameter_schemas WHERE parameter_id = v_parameter_id)
        WHEN 'FLOAT' THEN EXISTS (SELECT 1 FROM float_parameter_schemas WHERE parameter_id = v_parameter_id)
        WHEN 'DATETIME' THEN EXISTS (SELECT 1 FROM datetime_parameter_schemas WHERE parameter_id = v_parameter_id)
        WHEN 'TEMPLATE' THEN EXISTS (SELECT 1 FROM parameter_templates WHERE parameter_id = v_parameter_id)
    END;
    IF NOT v_schema_exists THEN
        RAISE EXCEPTION 'parameter % is missing schema row for kind %', v_parameter_id, v_parameter_kind;
    END IF;

    SELECT schema_kind
    INTO v_parent_kind
    FROM parameters AS parent
    WHERE parent.id = v_template_parent_id;
    IF v_template_parent_id IS NOT NULL AND v_parent_kind IS DISTINCT FROM 'TEMPLATE' THEN
        RAISE EXCEPTION 'parameter % has non-template parent', v_parameter_id;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM parameters
        WHERE id = v_parameter_id
          AND explode
          AND (
              (v_parameter_kind = 'STRING' AND NOT EXISTS (SELECT 1 FROM string_parameter_schemas WHERE parameter_id = v_parameter_id AND multiple_values))
              OR (v_parameter_kind = 'INTEGER' AND NOT EXISTS (SELECT 1 FROM integer_parameter_schemas WHERE parameter_id = v_parameter_id AND multiple_values))
              OR (v_parameter_kind = 'FLOAT' AND NOT EXISTS (SELECT 1 FROM float_parameter_schemas WHERE parameter_id = v_parameter_id AND multiple_values))
              OR v_parameter_kind IN ('BOOLEAN', 'DATETIME', 'TEMPLATE')
          )
    ) THEN
        RAISE EXCEPTION 'parameter % explode requires multiple values', v_parameter_id;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER parameters_schema_integrity
AFTER INSERT OR UPDATE ON parameters
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_parameter_schema();

CREATE CONSTRAINT TRIGGER string_parameter_schema_integrity
AFTER INSERT OR UPDATE OR DELETE ON string_parameter_schemas
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_parameter_schema();
CREATE CONSTRAINT TRIGGER boolean_parameter_schema_integrity
AFTER INSERT OR UPDATE OR DELETE ON boolean_parameter_schemas
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_parameter_schema();
CREATE CONSTRAINT TRIGGER integer_parameter_schema_integrity
AFTER INSERT OR UPDATE OR DELETE ON integer_parameter_schemas
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_parameter_schema();
CREATE CONSTRAINT TRIGGER float_parameter_schema_integrity
AFTER INSERT OR UPDATE OR DELETE ON float_parameter_schemas
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_parameter_schema();
CREATE CONSTRAINT TRIGGER datetime_parameter_schema_integrity
AFTER INSERT OR UPDATE OR DELETE ON datetime_parameter_schemas
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_parameter_schema();
CREATE CONSTRAINT TRIGGER parameter_template_schema_integrity
AFTER INSERT OR UPDATE OR DELETE ON parameter_templates
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_parameter_schema();
CREATE TABLE distributions (
    -- Version-owned distribution identity
    id                       uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_version_id       uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,

    -- Distribution metadata
    title                    nonblank_text NOT NULL,
    description              nonblank_text NOT NULL,
    license                  nonblank_text_opt,
    source_issued_at         timestamptz,
    source_modified_at       timestamptz,
    byte_size                integer,

    -- Delivery method. Constraint below requires one complete representation.
    type                     text NOT NULL,
    downloadable_file_url    nonblank_text_opt,
    downloadable_file_format nonblank_text_opt,
    web_service_id           uuid REFERENCES web_services(id) ON DELETE CASCADE,

    CHECK (
        (type = 'WEB_SERVICE'
            AND downloadable_file_url IS NULL
            AND downloadable_file_format IS NULL
            AND web_service_id IS NOT NULL)
        OR
        (type = 'DOWNLOADABLE_FILE'
            AND downloadable_file_url IS NOT NULL
            AND downloadable_file_format IS NOT NULL
            AND web_service_id IS NULL)
    )
);

-- group_id forms a composite foreign key with category_id. Migration 0006 adds
-- a trigger that verifies it matches owning logical resource's immutable group.
CREATE TABLE dataset_version_categories (
    dataset_version_id uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    group_id           uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    category_id        uuid NOT NULL,
    PRIMARY KEY (dataset_version_id, category_id),
    FOREIGN KEY (category_id, group_id) REFERENCES categories(id, group_id) ON DELETE CASCADE
);
CREATE TABLE software_application_version_categories (
    software_application_version_id uuid NOT NULL REFERENCES software_application_versions(id) ON DELETE CASCADE,
    group_id                        uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    category_id                     uuid NOT NULL,
    PRIMARY KEY (software_application_version_id, category_id),
    FOREIGN KEY (category_id, group_id) REFERENCES categories(id, group_id) ON DELETE CASCADE
);
CREATE TABLE software_source_code_version_categories (
    software_source_code_version_id uuid NOT NULL REFERENCES software_source_code_versions(id) ON DELETE CASCADE,
    group_id                        uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    category_id                     uuid NOT NULL,
    PRIMARY KEY (software_source_code_version_id, category_id),
    FOREIGN KEY (category_id, group_id) REFERENCES categories(id, group_id) ON DELETE CASCADE
);

-- +goose Down

DROP TABLE IF EXISTS software_source_code_version_categories, software_application_version_categories, software_source_code_version_contact_points, software_application_version_contact_points, dataset_version_categories, parameter_templates, datetime_parameter_schemas, float_parameter_schemas, integer_parameter_schemas, boolean_parameter_schemas, string_parameter_schemas, parameters, web_service_bodies, web_service_contact_points, distributions, web_services, dataset_version_contact_points, software_source_code_version_creators, software_application_version_creators, dataset_version_providers, identifiers;
DROP FUNCTION IF EXISTS validate_parameter_schema();
DROP FUNCTION IF EXISTS enum_option_list_has_non_finite_float(float_parameter_enum_option[]);
DROP FUNCTION IF EXISTS enum_option_list_matches_bounds(jsonb, numeric, numeric, numeric);
DROP FUNCTION IF EXISTS is_valid_enum_option_list(jsonb, text);
DROP TYPE IF EXISTS float_parameter_enum_option, integer_parameter_enum_option, string_parameter_enum_option, parameter_schema_kind, web_service_parameter_location, web_service_visible_on, web_service_type, web_service_http_method;
