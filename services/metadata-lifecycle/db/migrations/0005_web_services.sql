-- +goose Up

CREATE TYPE web_service_http_method AS ENUM ('GET', 'POST', 'PUT', 'DELETE');
CREATE TYPE web_service_type AS ENUM ('WMS', 'HTTP', 'WMTS', 'WFS');
CREATE TYPE web_service_visible_on AS ENUM ('GRAPH', 'TABLE', 'MAP');
CREATE TYPE web_service_parameter_location AS ENUM ('PATH', 'QUERY', 'HEADER');
CREATE TYPE parameter_schema_kind AS ENUM ('STRING', 'BOOLEAN', 'INTEGER', 'FLOAT', 'DATETIME', 'TEMPLATE');
CREATE TYPE web_service_parameter_role AS ENUM ('FORMAT', 'START_DATE', 'END_DATE', 'MIN_LON', 'MAX_LON', 'MIN_LAT', 'MAX_LAT');

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

CREATE TABLE web_services (
    -- Version-owned web service identity
    id                        uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_version_id        uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,

    -- Web service metadata
    name                      nonblank_text NOT NULL,
    description               nonblank_text_opt,
    type                      web_service_type NOT NULL,
    visible_on                web_service_visible_on[],
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

CREATE TABLE generic_web_services (
    web_service_id uuid PRIMARY KEY REFERENCES web_services(id) ON DELETE CASCADE,
    template_url   nonblank_text NOT NULL,
    method         web_service_http_method NOT NULL,
    format         nonblank_text NOT NULL
);

CREATE TABLE wms_services (
    web_service_id        uuid PRIMARY KEY REFERENCES web_services(id) ON DELETE CASCADE,
    capabilities_url      nonblank_text NOT NULL,
    default_layer         nonblank_text NOT NULL,
    transparency_editable boolean NOT NULL,
    default_transparency  boolean NOT NULL
);

-- +goose StatementBegin
CREATE FUNCTION is_valid_wms_styles(styles text[]) RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT cardinality(styles) > 0
        AND array_position(styles, NULL) IS NULL
        AND (SELECT count(DISTINCT style) FROM unnest(styles) AS style) = cardinality(styles);
$$;
-- +goose StatementEnd

CREATE TABLE wms_layers (
    web_service_id    uuid NOT NULL REFERENCES wms_services(web_service_id) ON DELETE CASCADE,
    name              nonblank_text NOT NULL,
    position          integer NOT NULL,
    default_style     text NOT NULL,
    styles            text[] NOT NULL,
    PRIMARY KEY (web_service_id, name),
    UNIQUE (web_service_id, position),
    CHECK (is_valid_wms_styles(styles)),
    CHECK (default_style = ANY(styles))
);

-- Added after table creation because these foreign keys are circular.
ALTER TABLE wms_services
    ADD CONSTRAINT wms_services_default_layer_fk
    FOREIGN KEY (web_service_id, default_layer)
    REFERENCES wms_layers(web_service_id, name)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE web_service_bodies (
    id             uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    web_service_id uuid NOT NULL UNIQUE REFERENCES generic_web_services(web_service_id) ON DELETE CASCADE,
    content_type   nonblank_text NOT NULL,
    template       nonblank_text NOT NULL,
    body_required  boolean NOT NULL DEFAULT TRUE
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
    web_service_id        uuid REFERENCES generic_web_services(web_service_id) ON DELETE CASCADE,
    wms_get_map_service_id          uuid REFERENCES wms_services(web_service_id) ON DELETE CASCADE,
    wms_get_feature_info_service_id uuid REFERENCES wms_services(web_service_id) ON DELETE CASCADE,
    web_service_body_id   uuid REFERENCES web_service_bodies(id) ON DELETE CASCADE,
    template_parameter_id uuid REFERENCES parameters(id) ON DELETE CASCADE,

    schema_kind           parameter_schema_kind NOT NULL,

    role                  web_service_parameter_role,
    property              nonblank_text_opt,
    location              web_service_parameter_location,
    explode               boolean NOT NULL DEFAULT false,

    CHECK (num_nonnulls(web_service_id, wms_get_map_service_id, wms_get_feature_info_service_id, web_service_body_id, template_parameter_id) = 1),
    CHECK (
        (num_nonnulls(web_service_id, wms_get_map_service_id, wms_get_feature_info_service_id) = 1 AND location IS NOT NULL)
        OR (num_nonnulls(web_service_id, wms_get_map_service_id, wms_get_feature_info_service_id) = 0 AND location IS NULL AND property IS NULL AND NOT explode)
    ),
    CHECK (role IS NULL OR (num_nonnulls(web_service_id, wms_get_map_service_id, wms_get_feature_info_service_id) = 1 AND schema_kind <> 'TEMPLATE')),
    CHECK (NOT explode OR (num_nonnulls(web_service_id, wms_get_map_service_id, wms_get_feature_info_service_id) = 1 AND location = 'QUERY')),
    CHECK (location <> 'PATH' OR required),
    CHECK (schema_kind <> 'TEMPLATE' OR num_nonnulls(web_service_id, wms_get_map_service_id, wms_get_feature_info_service_id) = 1),
    CHECK (template_parameter_id IS NULL OR schema_kind <> 'TEMPLATE'),
    UNIQUE (web_service_id, role),
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
CREATE UNIQUE INDEX parameters_wms_get_map_name_uq ON parameters (wms_get_map_service_id, name) WHERE wms_get_map_service_id IS NOT NULL;
CREATE UNIQUE INDEX parameters_wms_get_feature_info_name_uq ON parameters (wms_get_feature_info_service_id, name) WHERE wms_get_feature_info_service_id IS NOT NULL;
CREATE UNIQUE INDEX parameters_wms_get_map_role_uq ON parameters (wms_get_map_service_id, role) WHERE wms_get_map_service_id IS NOT NULL AND role IS NOT NULL;
CREATE UNIQUE INDEX parameters_wms_get_feature_info_role_uq ON parameters (wms_get_feature_info_service_id, role) WHERE wms_get_feature_info_service_id IS NOT NULL AND role IS NOT NULL;
CREATE UNIQUE INDEX parameters_web_service_body_name_uq ON parameters (web_service_body_id, name) WHERE web_service_body_id IS NOT NULL;
CREATE UNIQUE INDEX parameters_template_name_uq ON parameters (template_parameter_id, name) WHERE template_parameter_id IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION validate_web_service_configuration() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    v_web_service_id uuid;
    v_type web_service_type;
    v_has_generic boolean;
    v_has_wms boolean;
BEGIN
    IF TG_TABLE_NAME = 'web_services' THEN
        v_web_service_id := COALESCE(NEW.id, OLD.id);
    ELSE
        v_web_service_id := COALESCE(NEW.web_service_id, OLD.web_service_id);
    END IF;

    SELECT type INTO v_type FROM web_services WHERE id = v_web_service_id;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    SELECT EXISTS (SELECT 1 FROM generic_web_services WHERE web_service_id = v_web_service_id) INTO v_has_generic;
    SELECT EXISTS (SELECT 1 FROM wms_services WHERE web_service_id = v_web_service_id) INTO v_has_wms;

    IF (v_type = 'WMS' AND (v_has_generic OR NOT v_has_wms))
        OR (v_type <> 'WMS' AND (NOT v_has_generic OR v_has_wms)) THEN
        RAISE EXCEPTION 'web service % must have exactly one configuration matching type %', v_web_service_id, v_type
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER web_services_configuration_integrity
AFTER INSERT OR UPDATE OR DELETE ON web_services
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_web_service_configuration();
CREATE CONSTRAINT TRIGGER generic_web_services_configuration_integrity
AFTER INSERT OR UPDATE OR DELETE ON generic_web_services
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_web_service_configuration();
CREATE CONSTRAINT TRIGGER wms_services_configuration_integrity
AFTER INSERT OR UPDATE OR DELETE ON wms_services
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_web_service_configuration();

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

-- +goose Down

DROP TABLE IF EXISTS parameter_templates, datetime_parameter_schemas, float_parameter_schemas, integer_parameter_schemas, boolean_parameter_schemas, string_parameter_schemas, parameters, web_service_bodies, web_service_contact_points, generic_web_services, wms_layers, wms_services, web_services;
DROP FUNCTION IF EXISTS is_valid_wms_styles(text[]);
DROP FUNCTION IF EXISTS validate_web_service_configuration();
DROP FUNCTION IF EXISTS validate_parameter_schema();
DROP FUNCTION IF EXISTS enum_option_list_has_non_finite_float(float_parameter_enum_option[]);
DROP FUNCTION IF EXISTS enum_option_list_matches_bounds(jsonb, numeric, numeric, numeric);
DROP FUNCTION IF EXISTS is_valid_enum_option_list(jsonb, text);
DROP TYPE IF EXISTS float_parameter_enum_option, integer_parameter_enum_option, string_parameter_enum_option, parameter_schema_kind, web_service_parameter_role, web_service_parameter_location, web_service_visible_on, web_service_type, web_service_http_method;
