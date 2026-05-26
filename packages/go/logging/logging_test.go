package logging

import (
	"context"
	"log/slog"
	"sync"
	"testing"
)

func resetLoggingState(t *testing.T) {
	t.Helper()

	if logFile != nil {
		if err := logFile.Close(); err != nil {
			t.Fatalf("close log file: %v", err)
		}
	}

	level = slog.LevelInfo
	base = nil
	once = sync.Once{}
	logFile = nil
}

func TestComponentLevelEnvKey(t *testing.T) {
	got := componentLevelEnvKey("gateway-http-client")
	if got != "LOG_LEVEL_GATEWAY_HTTP_CLIENT" {
		t.Fatalf("componentLevelEnvKey() = %q, want %q", got, "LOG_LEVEL_GATEWAY_HTTP_CLIENT")
	}
}

func TestGetUsesComponentLevelOverride(t *testing.T) {
	resetLoggingState(t)
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("LOG_LEVEL_GATEWAY_HTTP_CLIENT", "INFO")

	ctx := context.Background()
	upstreamLogger := Get("gateway-http-client")
	otherLogger := Get("server")

	if upstreamLogger.Enabled(ctx, slog.LevelDebug) {
		t.Fatal("expected gateway-http-client logger to disable debug level")
	}

	if !upstreamLogger.Enabled(ctx, slog.LevelInfo) {
		t.Fatal("expected gateway-http-client logger to enable info level")
	}

	if !otherLogger.Enabled(ctx, slog.LevelDebug) {
		t.Fatal("expected other component logger to inherit global debug level")
	}
}

func TestGetFallsBackToGlobalLevelForInvalidComponentOverride(t *testing.T) {
	resetLoggingState(t)
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("LOG_LEVEL_GATEWAY_HTTP_CLIENT", "INVALID")

	logger := Get("gateway-http-client")
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("expected invalid component override to fall back to global debug level")
	}
}
