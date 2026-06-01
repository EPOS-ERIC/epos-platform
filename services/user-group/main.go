package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/server"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/storage"
)

func main() {
	log := logging.Get("user-group")
	defer func() {
		if err := logging.Close(); err != nil {
			log.Error("failed to close logging", "error", err)
		}
	}()

	otelShutdown, err := observability.Start(context.Background(), observability.Config{})
	if err != nil {
		log.Error("failed to setup observability", "error", fmt.Errorf("setup observability: %w", err))
		os.Exit(1)
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			log.Error("otel shutdown failed", "error", err)
		}
	}()

	addr := os.Getenv("USER_GROUP_LISTEN_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	db, err := openPostgresWithRetry(log)
	if err != nil {
		log.Error("failed to open postgres after retries", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close postgres", "error", err)
		}
	}()

	migrationsDir := os.Getenv("USER_GROUP_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	if err := storage.ApplyMigrations(db, migrationsDir); err != nil {
		log.Error("failed to apply migrations", "migrations_dir", migrationsDir, "error", err)
		os.Exit(1)
	}

	handler := server.NewRouter(
		repository.NewUserPostgresRepository(db),
		repository.NewGroupPostgresRepository(db),
		repository.NewMembershipPostgresRepository(db),
		repository.NewFriendshipPostgresRepository(db),
		db,
	)
	printBanner(addr)
	log.Info("starting http server", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}

func printBanner(addr string) {
	logoHollow := `⣠⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⣤⣶⣶⣶⣶⣶⣤⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⣤⣤⣶⣶⣦⣤⣤⣀⡀⠀⠀⠀
⣿⣿⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⣿⣿⠛⠛⠛⠛⠛⠛⠛⠛⠿⠿⣿⣿⣷⣄⠀⠀⠀⣀⣴⣿⣿⡿⠿⠛⠛⠛⠛⠛⠻⢿⣿⣿⣦⣄⠀⠀⠀⣠⣾⣿⣿⠿⠟⠛⠛⠛⠿⠿⣿⣿⣷⣤⡀
⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⢻⣿⣷⣀⣾⣿⡿⠛⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⢿⣿⣷⣄⣾⣿⡟⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⣻⣿⣷
⣿⣿⠀⠀⠀⠀⢠⣤⣤⣤⣤⣤⣤⣤⣤⣿⣿⠀⠀⠀⠀⢀⣴⣶⣷⣶⣄⠀⠀⠀⢻⣿⣿⣿⠏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣿⣿⣿⡟⠀⠀⠀⠀⣠⣤⣤⣤⣄⡀⣠⣾⣿⡿⠁
⣿⣿⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⢼⣿⣟⢙⣿⣿⠀⠀⠀⠘⣿⣿⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⣿⣿⡇⠀⠀⠀⢸⣿⣿⣿⣿⣿⣿⣿⡿⠋⠀⠀
⣿⣿⠀⠀⠀⠀⠈⠉⠉⠉⠉⠉⠉⢹⣿⣿⣿⠀⠀⠀⠀⠘⢿⣿⣿⣿⠟⠀⠀⠀⢸⣿⡏⠀⠀⠀⠀⠀⠀⠀⣠⣶⣶⣶⣤⠀⠀⠀⠀⠀⠀⠀⢸⣿⣇⠀⠀⠀⠀⠈⠉⠙⠛⠛⠿⢿⣿⣷⣄⠀
⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣰⣿⣿⡇⠀⠀⠀⠀⠀⠀⠰⣿⣿⣋⣿⣿⡇⠀⠀⠀⠀⠀⠀⢸⣿⣿⣦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣆
⣿⣿⠀⠀⠀⠀⢰⣶⣶⣶⣶⣶⣶⣾⣿⣿⣿⠀⠀⠀⠀⣀⣀⣀⣀⣠⣤⣴⣿⣿⣿⣿⣇⠀⠀⠀⠀⠀⠀⠀⠻⣿⣿⣿⠟⠀⠀⠀⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣷⣶⣶⣦⣤⡀⠀⠀⠀⢸⣿⣿
⣿⣿⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⣿⣿⡿⠿⠿⠿⠟⠛⠁⠸⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⡀⠀⠀⠀⠀⠀⠀⢀⣿⣿⣿⣿⢿⣿⣿⣿⣿⣿⣿⡇⠀⠀⠀⢠⣿⣿
⣿⣿⠀⠀⠀⠀⠈⠉⠉⠉⠉⠉⠉⠉⠉⣿⣿⠀⠀⠀⠀⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠹⣿⣿⣄⠀⠀⠀⠀⠀⢸⣿⣿⣿⣇⠀⠀⠀⠀⠀⢠⣾⣿⣿⠟⠁⠀⠀⠉⠉⠉⠉⠁⠀⠀⠀⠀⣼⣿⡟
⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⠀⠀⠀⠀⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣷⣄⠀⠀⢀⣿⣿⠟⣿⣿⡄⠀⠀⣠⣴⣿⣿⣿⣷⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⣾⣿⡟⠁
⣿⣿⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣿⣿⣶⣶⣶⣶⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣿⣶⣾⣿⡿⠀⢻⣿⣷⣶⣿⣿⡿⠋⠁⠉⠻⢿⣿⣿⣷⣶⣶⣶⣶⣾⣿⣿⣿⠟⠋⠀⠀
⠈⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⠛⠛⠋⠀⠀⠀⠙⠛⠛⠋⠁⠀⠀⠀⠀⠀⠀⠀⠉⠉⠛⠛⠛⠛⠋⠉⠁⠀⠀⠀⠀⠀`

	println()
	println(logoHollow)
	println()
	println("EPOS User Group Service")
	println("PostgreSQL | HTTP API")
	println("Copyright (c) 2026 EPOS ERIC")
	println("https://www.epos-eu.org/")
	println("Database    " + postgresTarget())
	println("Server      http://localhost" + addr)
	println("------------------------------------------------------------")
}

func openPostgresWithRetry(log *slog.Logger) (*sql.DB, error) {
	maxAttempts := getenvInt("USER_GROUP_DB_CONNECT_MAX_ATTEMPTS", 20)
	retryDelay := getenvDuration("USER_GROUP_DB_CONNECT_RETRY_DELAY", 2*time.Second)

	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err := storage.OpenPostgres()
		if err == nil {
			log.Info("postgres connected", "target", postgresTarget(), "attempt", attempt, "max_attempts", maxAttempts)
			return db, nil
		}

		lastErr = err
		log.Warn("postgres connection attempt failed", "attempt", attempt, "max_attempts", maxAttempts, "retry_delay", retryDelay.String(), "error", err)
		if attempt < maxAttempts {
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("postgres unreachable at %s: %w", postgresTarget(), lastErr)
}

func postgresTarget() string {
	host := getenv("USER_GROUP_DB_HOST", "localhost")
	port := getenv("USER_GROUP_DB_PORT", "5432")
	name := getenv("USER_GROUP_DB_NAME", "user_group")
	user := getenv("USER_GROUP_DB_USER", "user_group")
	return fmt.Sprintf("%s:%s/%s (user=%s)", host, port, name, user)
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
