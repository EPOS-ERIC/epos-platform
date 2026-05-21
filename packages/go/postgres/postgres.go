package postgres

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	HostEnv      string
	HostDefault  string
	PortEnv      string
	PortDefault  string
	NameEnv      string
	NameDefault  string
	UserEnv      string
	UserDefault  string
	PasswordEnv  string
	PasswordDefault string
}

func Open(cfg Config) (*sql.DB, error) {
	host := getenv(cfg.HostEnv, cfg.HostDefault)
	port := getenv(cfg.PortEnv, cfg.PortDefault)
	name := getenv(cfg.NameEnv, cfg.NameDefault)
	user := getenv(cfg.UserEnv, cfg.UserDefault)
	pass := getenv(cfg.PasswordEnv, cfg.PasswordDefault)

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
