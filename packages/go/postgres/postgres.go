// Package postgres owns PostgreSQL connection setup and migrations.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	defaultMaxAttempts = 20
	defaultRetryDelay  = 2 * time.Second
)

// Config contains resolved connection settings and an optional pgx connection
// initializer.
type Config struct {
	Host         string
	Port         string
	Database     string
	User         string
	Password     string
	AfterConnect func(context.Context, *pgx.Conn) error
}

// Open connects to PostgreSQL, applies embedded Goose migrations, and returns
// an instrumented pgx pool. It retries each initial connection ping up to 20
// times with 2 seconds between attempts; migration failures are not retried.
func Open(ctx context.Context, log *slog.Logger, cfg Config, migrations fs.FS) (*pgxpool.Pool, error) {
	if log == nil {
		log = slog.Default()
	}

	dsn, err := connectionString(cfg)
	if err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres pool config: %w", err)
	}
	poolConfig.AfterConnect = cfg.AfterConnect
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithSpanNameFunc(queryOperationName))

	migrationDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}
	if err := retryPing(ctx, log, "migration database", func() error {
		return migrationDB.PingContext(ctx)
	}); err != nil {
		return nil, closeMigrationDB(migrationDB, err)
	}
	if err := applyMigrations(ctx, log, migrationDB, migrations); err != nil {
		return nil, closeMigrationDB(migrationDB, err)
	}
	if err := migrationDB.Close(); err != nil {
		return nil, fmt.Errorf("close migration database: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("record postgres pool stats: %w", err)
	}
	if err := retryPing(ctx, log, "postgres pool", func() error {
		return pool.Ping(ctx)
	}); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func connectionString(cfg Config) (string, error) {
	if cfg.Host == "" || cfg.Port == "" || cfg.Database == "" || cfg.User == "" {
		return "", fmt.Errorf("postgres host, port, database, and user are required")
	}
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		Path:     "/" + cfg.Database,
		RawQuery: url.Values{"sslmode": {"disable"}}.Encode(),
	}).String(), nil
}

func retryPing(ctx context.Context, log *slog.Logger, target string, ping func() error) error {
	for attempt := 1; attempt <= defaultMaxAttempts; attempt++ {
		if err := ping(); err == nil {
			log.InfoContext(ctx, "postgres connected", "target", target, "attempt", attempt, "max_attempts", defaultMaxAttempts)
			return nil
		} else {
			if ctx.Err() != nil {
				return fmt.Errorf("connect to %s: %w", target, errors.Join(err, ctx.Err()))
			}
			if attempt == defaultMaxAttempts {
				return fmt.Errorf("connect to %s after %d attempts: %w", target, defaultMaxAttempts, err)
			}
			log.WarnContext(ctx, "postgres connection attempt failed", "target", target, "attempt", attempt, "max_attempts", defaultMaxAttempts, "retry_delay", defaultRetryDelay, "error", err)
		}

		timer := time.NewTimer(defaultRetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("connect to %s: %w", target, ctx.Err())
		case <-timer.C:
		}
	}
	return fmt.Errorf("connect to %s: retry loop ended unexpectedly", target)
}

func applyMigrations(ctx context.Context, log *slog.Logger, db *sql.DB, migrations fs.FS) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	hasPending, err := provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("check pending migrations: %w", err)
	}
	if !hasPending {
		log.DebugContext(ctx, "database migrations are current")
		return nil
	}

	log.InfoContext(ctx, "applying database migrations")
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.InfoContext(ctx, "database migrations applied")
	return nil
}

func closeMigrationDB(db *sql.DB, cause error) error {
	if err := db.Close(); err != nil {
		cause = errors.Join(cause, fmt.Errorf("close migration database: %w", err))
	}
	return cause
}

func queryOperationName(statement string) string {
	for {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			return "UNKNOWN"
		}

		line, rest, hasNextLine := strings.Cut(statement, "\n")
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "--" && fields[1] == "name:" {
			return fields[2]
		}
		if !strings.HasPrefix(line, "--") || !hasNextLine {
			break
		}
		statement = rest
	}

	for word := range strings.FieldsSeq(statement) {
		return strings.ToUpper(word)
	}
	return "UNKNOWN"
}
