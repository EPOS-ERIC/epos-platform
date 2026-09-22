package storage

import (
	"database/sql"

	sharedpostgres "github.com/EPOS-ERIC/epos-platform/packages/go/postgres"
)

func OpenPostgres() (*sql.DB, error) {
	return sharedpostgres.Open(sharedpostgres.Config{
		HostEnv:         "SHARING_DB_HOST",
		HostDefault:     "localhost",
		PortEnv:         "SHARING_DB_PORT",
		PortDefault:     "5432",
		NameEnv:         "SHARING_DB_NAME",
		NameDefault:     "sharing",
		UserEnv:         "SHARING_DB_USER",
		UserDefault:     "sharing",
		PasswordEnv:     "SHARING_DB_PASSWORD",
		PasswordDefault: "sharing",
	})
}
