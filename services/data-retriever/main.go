package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/server"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/storage"
)

func main() {
	log := logging.New("data-retriever").With("service", "data-retriever")

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

	addr := getenv("DATA_RETRIEVER_LISTEN_ADDR", ":8100")

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

	migrationsDir := getenv("DATA_RETRIEVER_MIGRATIONS_DIR", "migrations")
	if err := storage.ApplyMigrations(db, migrationsDir); err != nil {
		log.Error("failed to apply migrations", "migrations_dir", migrationsDir, "error", err)
		os.Exit(1)
	}

	timeout, err := time.ParseDuration(getenv("DATA_RETRIEVER_HTTP_TIMEOUT", "30s"))
	if err != nil {
		log.Error("failed to parse timeout", "error", err)
		os.Exit(1)
	}

	maxBodyBytes := int64(1 << 20)
	handler := server.NewRouter(
		log,
		repository.NewRetrievalPostgresRepository(db),
		retriever.NewClient(timeout, maxBodyBytes),
		db,
	)
	log.Info("starting http server", "addr", addr, "timeout", timeout.String())

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func openPostgresWithRetry(log *slog.Logger) (*sql.DB, error) {
	maxAttempts := getenvInt("DATA_RETRIEVER_DB_CONNECT_MAX_ATTEMPTS", 20)
	retryDelay := getenvDuration("DATA_RETRIEVER_DB_CONNECT_RETRY_DELAY", 2*time.Second)

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
	host := getenv("DATA_RETRIEVER_DB_HOST", "localhost")
	port := getenv("DATA_RETRIEVER_DB_PORT", "5432")
	name := getenv("DATA_RETRIEVER_DB_NAME", "data_retriever")
	user := getenv("DATA_RETRIEVER_DB_USER", "data_retriever")
	return fmt.Sprintf("%s:%s/%s (user=%s)", host, port, name, user)
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
