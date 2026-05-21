package storage

import (
	"database/sql"

	sharedmigrations "github.com/EPOS-ERIC/epos-platform/packages/go/migrations"
)

func ApplyMigrations(db *sql.DB, dir string) error {
	return sharedmigrations.ApplyDir(db, dir)
}
