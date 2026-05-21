package storage

import (
	"database/sql"

	sharedpostgres "github.com/EPOS-ERIC/epos-platform/packages/go/postgres"
)

func OpenPostgres() (*sql.DB, error) {
	return sharedpostgres.Open(sharedpostgres.Config{
		HostEnv:         "DATA_RETRIEVER_DB_HOST",
		HostDefault:     "localhost",
		PortEnv:         "DATA_RETRIEVER_DB_PORT",
		PortDefault:     "5432",
		NameEnv:         "DATA_RETRIEVER_DB_NAME",
		NameDefault:     "data_retriever",
		UserEnv:         "DATA_RETRIEVER_DB_USER",
		UserDefault:     "data_retriever",
		PasswordEnv:     "DATA_RETRIEVER_DB_PASSWORD",
		PasswordDefault: "data_retriever",
	})
}
