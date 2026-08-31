package logging

import (
	"context"
	"log/slog"
	"testing"
)

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
