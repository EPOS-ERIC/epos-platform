-- +goose Up

ALTER TABLE configurations DROP COLUMN IF EXISTS example;
ALTER TABLE configurations DROP COLUMN IF EXISTS sharing_link_url;

-- +goose Down

ALTER TABLE configurations ADD COLUMN IF NOT EXISTS example TEXT;
ALTER TABLE configurations ADD COLUMN IF NOT EXISTS sharing_link_url TEXT;
