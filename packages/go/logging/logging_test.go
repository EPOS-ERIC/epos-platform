package logging

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestContextHandlerAddsRequest(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(newHandler(&output, slog.LevelInfo)).With("component", "test")

	ctx := WithRequest(context.Background(), Request{
		Method: "GET",
		Path:   "/v1/search",
		Query:  "q=tsunami",
	})

	logger.InfoContext(ctx, "request log")

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	request, ok := payload["request"].(map[string]any)
	if !ok {
		t.Fatalf("request field = %#v", payload["request"])
	}
	if request["method"] != "GET" {
		t.Fatalf("request.method = %#v", request["method"])
	}
	if request["path"] != "/v1/search" {
		t.Fatalf("request.path = %#v", request["path"])
	}
	if request["query"] != "q=tsunami" {
		t.Fatalf("request.query = %#v", request["query"])
	}
}

func TestContextHandlerAddsActor(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(newHandler(&output, slog.LevelInfo)).With("component", "test")

	ctx := WithActor(context.Background(), Actor{
		ID:             "local-1",
		AuthIdentifier: "user-123",
		IsSuperAdmin:   true,
	})

	logger.WarnContext(ctx, "access denied")

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	actor, ok := payload["actor"].(map[string]any)
	if !ok {
		t.Fatalf("actor field = %#v", payload["actor"])
	}
	if actor["id"] != "local-1" {
		t.Fatalf("actor.id = %#v", actor["id"])
	}
	if actor["auth_identifier"] != "user-123" {
		t.Fatalf("actor.auth_identifier = %#v", actor["auth_identifier"])
	}
	if actor["is_super_admin"] != true {
		t.Fatalf("actor.is_super_admin = %#v", actor["is_super_admin"])
	}
}

func TestContextHandlerAddsRequestAndActor(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(newHandler(&output, slog.LevelInfo)).With("component", "test")

	ctx := WithRequest(context.Background(), Request{
		Method: "GET",
		Path:   "/v1/users",
	})
	ctx = WithActor(ctx, Actor{ID: "local-1"})

	logger.InfoContext(ctx, "request log")

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	if _, ok := payload["request"].(map[string]any); !ok {
		t.Fatalf("request field = %#v", payload["request"])
	}
	if _, ok := payload["actor"].(map[string]any); !ok {
		t.Fatalf("actor field = %#v", payload["actor"])
	}
}

func TestContextHandlerSkipsMissingRequest(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(newHandler(&output, slog.LevelInfo)).With("component", "test")

	logger.InfoContext(context.Background(), "request log")

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	if _, ok := payload["request"]; ok {
		t.Fatalf("unexpected request field = %#v", payload["request"])
	}
}
