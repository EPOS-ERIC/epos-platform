package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
)

type structuredError struct{}

func (structuredError) Error() string { return "invalid organization" }

func (structuredError) LogAttrs() []slog.Attr {
	return []slog.Attr{slog.String("error_target", "organizations[0].street")}
}

func TestWrappedErrorAttrs(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(newMultiHandler(slog.NewJSONHandler(&out, nil)))
	err := fmt.Errorf("sync: %w", structuredError{})

	log.Error("failed", "error", err)
	log.With("error", err).Error("failed")

	decoder := json.NewDecoder(&out)
	for range 2 {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		if entry["error"] != err.Error() || entry["error_target"] != "organizations[0].street" {
			t.Fatalf("unexpected log entry: %v", entry)
		}
	}
}

func TestConfiguredLevel(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  slog.Level
	}{
		{name: "default", want: slog.LevelInfo},
		{name: "debug", value: "DEBUG", want: slog.LevelDebug},
		{name: "info", value: "INFO", want: slog.LevelInfo},
		{name: "warn", value: "WARN", want: slog.LevelWarn},
		{name: "error", value: "ERROR", want: slog.LevelError},
		{name: "invalid", value: "INVALID", want: slog.LevelInfo},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", test.value)
			if got := configuredLevel(); got != test.want {
				t.Fatalf("configuredLevel() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewUsesConfiguredLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "DEBUG")

	logger := New("test")
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("expected debug logging to be enabled")
	}

	t.Setenv("LOG_LEVEL", "INFO")
	logger = New("test")
	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("expected debug logging to be disabled")
	}
}
