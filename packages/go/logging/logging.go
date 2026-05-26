// Package logging provides structured logging utilities with configurable levels and output destinations.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	level   = slog.LevelInfo
	base    *slog.Logger
	once    sync.Once
	logFile *os.File
)

var nonAlphaNum = regexp.MustCompile(`[^A-Z0-9]+`)

func parseLevel(value string) (slog.Level, bool) {
	switch strings.ToUpper(value) {
	case "DEBUG":
		return slog.LevelDebug, true
	case "INFO":
		return slog.LevelInfo, true
	case "WARN":
		return slog.LevelWarn, true
	case "ERROR":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}

func componentLevelEnvKey(component string) string {
	normalized := strings.ToUpper(component)
	normalized = nonAlphaNum.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	return "LOG_LEVEL_" + normalized
}

func configuredGlobalLevel() slog.Level {
	v, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		return slog.LevelInfo
	}

	parsed, valid := parseLevel(v)
	if !valid {
		slog.Warn("unknown LOG_LEVEL, falling back to INFO", "value", strings.ToUpper(v))
		return slog.LevelInfo
	}

	fmt.Printf("setting up logging with level %s\n", strings.ToUpper(v))
	return parsed
}

func configuredComponentLevel(component string) (slog.Level, bool) {
	v, ok := os.LookupEnv(componentLevelEnvKey(component))
	if !ok {
		return slog.LevelInfo, false
	}

	parsed, valid := parseLevel(v)
	if !valid {
		return slog.LevelInfo, false
	}

	return parsed, true
}

func newHandler(writer io.Writer, level slog.Level) slog.Handler {
	return slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
		// Format time to second precision only
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.Attr{
						Key:   slog.TimeKey,
						Value: slog.StringValue(t.Format("2006-01-02T15:04:05Z07:00")),
					}
				}
			}
			return a
		},
	})
}

func initBase() {
	level = configuredGlobalLevel()

	var writer io.Writer = os.Stdout

	// Check for LOG_FILE environment variable
	if logPath, ok := os.LookupEnv("LOG_FILE"); ok {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304
		if err != nil {
			slog.Error("failed to open log file, falling back to stdout", "path", logPath, "error", err)
		} else {
			logFile = file
			writer = file
			fmt.Printf("logging to file: %s\n", logPath)
		}
	}

	handler := newHandler(writer, level)
	base = slog.New(handler)
	slog.SetDefault(base.With("component", "default"))
}

// Get returns a logger with the given component name.
//
// Use a package‑level variable to hold the logger:
//
//	var log = logging.Get("database")
func Get(component string) *slog.Logger {
	once.Do(initBase)

	configuredLevel, hasComponentLevel := configuredComponentLevel(component)
	if !hasComponentLevel {
		return base.With("component", component)
	}

	handler := newHandler(logWriter(), configuredLevel)
	return slog.New(handler).With("component", component)
}

func logWriter() io.Writer {
	if logFile != nil {
		return logFile
	}

	return os.Stdout
}

// Close closes the log file if one was opened.
// Call this function when your application shuts down to ensure
// the log file is properly closed.
func Close() error {
	if logFile != nil {
		return logFile.Close()
	}
	return nil
}
