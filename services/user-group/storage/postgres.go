package storage

import (
	"database/sql"

	sharedpostgres "github.com/EPOS-ERIC/epos-platform/packages/go/postgres"
)

func OpenPostgres() (*sql.DB, error) {
	return sharedpostgres.Open(sharedpostgres.Config{
		HostEnv:         "USER_GROUP_DB_HOST",
		HostDefault:     "localhost",
		PortEnv:         "USER_GROUP_DB_PORT",
		PortDefault:     "5432",
		NameEnv:         "USER_GROUP_DB_NAME",
		NameDefault:     "user_group",
		UserEnv:         "USER_GROUP_DB_USER",
		UserDefault:     "user_group",
		PasswordEnv:     "USER_GROUP_DB_PASSWORD",
		PasswordDefault: "user_group",
	})
}
