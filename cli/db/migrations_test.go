package db

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestInitialMigration(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := runMigrations(db); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`INSERT INTO docker (name, config_yaml) VALUES ('test', '{}')`,
		`INSERT INTO ingested_files (environment_name, file_path) VALUES ('test', 'file')`,
		`INSERT INTO latest_release_cache (tag_name) VALUES ('v1')`,
		`INSERT INTO image_update_cache (image_ref, remote_digest) VALUES ('image', 'digest')`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if err := goose.Down(db, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := runMigrations(db); err != nil {
		t.Fatal(err)
	}
}
