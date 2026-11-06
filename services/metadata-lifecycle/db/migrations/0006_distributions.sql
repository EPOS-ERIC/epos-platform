-- +goose Up

CREATE TABLE distributions (
    -- Version-owned distribution identity
    id                       uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id              uuid NOT NULL DEFAULT uuid_generate_v4(),
    dataset_version_id       uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,

    -- Distribution metadata
    title                    nonblank_text NOT NULL,
    description              nonblank_text NOT NULL,
    license                  nonblank_text_opt,
    byte_size                integer,

    -- Delivery method. Constraint below requires one complete representation.
    type                     text NOT NULL,
    downloadable_file_url    nonblank_text_opt,
    downloadable_file_format nonblank_text_opt,
    web_service_id           uuid UNIQUE REFERENCES web_services(id) ON DELETE CASCADE,

    UNIQUE (dataset_version_id, resource_id),
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

-- Web services and their distributions are created separately within a transaction.
-- Check ownership at commit so neither can be left without its counterpart.
-- +goose StatementBegin
CREATE FUNCTION validate_web_service_distribution() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    service_id uuid;
BEGIN
    IF TG_TABLE_NAME = 'web_services' THEN
        service_id := COALESCE(NEW.id, OLD.id);
    ELSE
        service_id := OLD.web_service_id;
    END IF;

    IF service_id IS NOT NULL
        AND EXISTS (SELECT 1 FROM web_services WHERE id = service_id)
        AND NOT EXISTS (SELECT 1 FROM distributions WHERE web_service_id = service_id) THEN
        RAISE EXCEPTION 'web service % must belong to a distribution', service_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER web_service_distribution_owner
AFTER INSERT OR UPDATE ON web_services
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_web_service_distribution();

CREATE CONSTRAINT TRIGGER distribution_web_service_owner
AFTER UPDATE OR DELETE ON distributions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_web_service_distribution();

-- +goose Down

DROP TABLE IF EXISTS distributions;
DROP TRIGGER IF EXISTS web_service_distribution_owner ON web_services;
DROP FUNCTION IF EXISTS validate_web_service_distribution();
