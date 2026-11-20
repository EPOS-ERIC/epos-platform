// Package logging configures structured logging for a service.
package logging

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// New creates a JSON logger that writes to stdout and exports records through
// OpenTelemetry. The service name identifies the OTEL logger instrumentation.
func New(serviceName string) *slog.Logger {
	level := configuredLevel()
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	otelHandler := otelslog.NewHandler(serviceName)

	return slog.New(newMultiHandler(
		newLevelHandler(jsonHandler, level),
		newLevelHandler(otelHandler, level),
	))
}

func configuredLevel() slog.Level {
	value := os.Getenv("LOG_LEVEL")
	if value == "" {
		return slog.LevelInfo
	}

	switch strings.ToUpper(value) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range m.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var extra []slog.Attr
	record.Attrs(func(attr slog.Attr) bool {
		extra = appendErrorAttrs(extra, attr)
		return true
	})
	if len(extra) > 0 {
		record = record.Clone()
		record.AddAttrs(extra...)
	}

	for _, handler := range m.handlers {
		if handler.Enabled(ctx, record.Level) {
			if err := handler.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var extra []slog.Attr
	for _, attr := range attrs {
		extra = appendErrorAttrs(extra, attr)
	}
	if len(extra) > 0 {
		attrs = append(extra, attrs...)
	}

	handlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

// appendErrorAttrs keeps the original error for each sink and adds its structured diagnostics.
func appendErrorAttrs(attrs []slog.Attr, attr slog.Attr) []slog.Attr {
	if err, ok := attr.Value.Any().(error); ok {
		if structured, ok := errors.AsType[interface {
			error
			LogAttrs() []slog.Attr
		}](err); ok {
			return append(attrs, structured.LogAttrs()...)
		}
	}
	return attrs
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

type levelHandler struct {
	next  slog.Handler
	level slog.Level
}

func newLevelHandler(next slog.Handler, level slog.Level) slog.Handler {
	return &levelHandler{next: next, level: level}
}

func (h *levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.next.Enabled(ctx, level)
}

func (h *levelHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level < h.level {
		return nil
	}
	return h.next.Handle(ctx, record)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{next: h.next.WithAttrs(attrs), level: h.level}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{next: h.next.WithGroup(name), level: h.level}
}
