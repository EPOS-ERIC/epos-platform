package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/repository"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/server"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/storage"
)

func main() {
	log := logging.New("sharing").With("service", "sharing")

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

	addr := getenv("SHARING_GRPC_LISTEN_ADDR", ":8200")

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

	if err := storage.ApplyMigrations(log, db); err != nil {
		log.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("starting grpc server", "addr", addr)
	if err := server.Run(
		ctx,
		addr,
		log,
		repository.NewConfigurationPostgresRepository(db),
	); err != nil {
		log.Error("grpc server stopped", "error", err)
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
	maxAttempts := getenvInt("SHARING_DB_CONNECT_MAX_ATTEMPTS", 20)
	retryDelay := getenvDuration("SHARING_DB_CONNECT_RETRY_DELAY", 2*time.Second)

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
	host := getenv("SHARING_DB_HOST", "localhost")
	port := getenv("SHARING_DB_PORT", "5432")
	name := getenv("SHARING_DB_NAME", "sharing")
	user := getenv("SHARING_DB_USER", "sharing")
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
