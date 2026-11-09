-- +goose Up
CREATE TABLE docker (
    name TEXT NOT NULL PRIMARY KEY,
    config_yaml TEXT NOT NULL
);

CREATE TABLE ingested_files (
    environment_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (environment_name, file_path)
);

CREATE TABLE latest_release_cache (
    id INTEGER PRIMARY KEY DEFAULT 1,
    tag_name TEXT NOT NULL,
    fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE image_update_cache (
    image_ref TEXT NOT NULL PRIMARY KEY,
    remote_digest TEXT NOT NULL,
    remote_created_at DATETIME,
    fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE image_update_cache;
DROP TABLE latest_release_cache;
DROP TABLE ingested_files;
DROP TABLE docker;
