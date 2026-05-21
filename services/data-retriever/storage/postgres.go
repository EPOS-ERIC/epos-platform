package storage

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenPostgres() (*sql.DB, error) {
	host := getenv("DATA_RETRIEVER_DB_HOST", "localhost")
	port := getenv("DATA_RETRIEVER_DB_PORT", "5432")
	name := getenv("DATA_RETRIEVER_DB_NAME", "data_retriever")
	user := getenv("DATA_RETRIEVER_DB_USER", "data_retriever")
	pass := getenv("DATA_RETRIEVER_DB_PASSWORD", "data_retriever")

	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", host, port, name, user, pass)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
