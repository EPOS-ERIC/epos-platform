package storage

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenPostgres() (*sql.DB, error) {
	host := getenv("USER_GROUP_DB_HOST", "localhost")
	port := getenv("USER_GROUP_DB_PORT", "5432")
	name := getenv("USER_GROUP_DB_NAME", "user_group")
	user := getenv("USER_GROUP_DB_USER", "user_group")
	pass := getenv("USER_GROUP_DB_PASSWORD", "user_group")

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
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
