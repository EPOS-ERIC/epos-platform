-- +goose Up
CREATE TABLE plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    description TEXT NOT NULL DEFAULT '',
    repository TEXT NOT NULL CHECK (length(btrim(repository)) > 0),
    branch TEXT NOT NULL CHECK (length(btrim(branch)) > 0),
    runtime TEXT NOT NULL CHECK (runtime IN ('python', 'java', 'binary')),
    arguments TEXT[] NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    installed_commit TEXT CHECK (length(btrim(installed_commit)) > 0),
    created_by TEXT NOT NULL CHECK (length(btrim(created_by)) > 0),
    updated_by TEXT NOT NULL CHECK (length(btrim(updated_by)) > 0)
);

CREATE TABLE conversion_relations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    distribution_id UUID NOT NULL,
    plugin_id UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    input_format TEXT NOT NULL CHECK (length(btrim(input_format)) > 0),
    output_format TEXT NOT NULL CHECK (length(btrim(output_format)) > 0),
    created_by TEXT NOT NULL CHECK (length(btrim(created_by)) > 0),
    updated_by TEXT NOT NULL CHECK (length(btrim(updated_by)) > 0),
    UNIQUE (distribution_id, plugin_id, input_format, output_format)
);

-- +goose Down
DROP TABLE conversion_relations;
DROP TABLE plugins;
