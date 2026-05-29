// Package logging provides structured logging utilities with configurable levels and output destinations.
package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

type requestContextKey struct{}

// Request carries request-scoped metadata that should be attached to context-aware log records.
type Request struct {
	RequestID string
	Method    string
	Path      string
	Query     string
}

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
	return &contextHandler{next: slog.NewJSONHandler(writer, &slog.HandlerOptions{
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
	})}
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

// WithRequest returns a context enriched with request-scoped logging metadata.
func WithRequest(ctx context.Context, request Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, request)
}

// RequestFromContext returns request-scoped logging metadata stored in ctx, if present.
func RequestFromContext(ctx context.Context) (Request, bool) {
	request, ok := ctx.Value(requestContextKey{}).(Request)
	return request, ok
}

// NewRequestID generates a random request identifier suitable for log correlation.
func NewRequestID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate request id: %w", err)
	}

	return hex.EncodeToString(buf), nil
}

type contextHandler struct {
	next slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	request, ok := RequestFromContext(ctx)
	if !ok {
		return h.next.Handle(ctx, record)
	}

	enriched := record.Clone()
	attrs := make([]slog.Attr, 0, 1)
	requestAttrs := make([]any, 0, 4)
	if request.RequestID != "" {
		requestAttrs = append(requestAttrs, slog.String("id", request.RequestID))
	}
	if request.Method != "" {
		requestAttrs = append(requestAttrs, slog.String("method", request.Method))
	}
	if request.Path != "" {
		requestAttrs = append(requestAttrs, slog.String("path", request.Path))
	}
	if request.Query != "" {
		requestAttrs = append(requestAttrs, slog.String("query", request.Query))
	}
	if len(requestAttrs) > 0 {
		attrs = append(attrs, slog.Group("request", requestAttrs...))
	}
	if len(attrs) > 0 {
		enriched.AddAttrs(attrs...)
	}

	return h.next.Handle(ctx, enriched)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{next: h.next.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{next: h.next.WithGroup(name)}
}
