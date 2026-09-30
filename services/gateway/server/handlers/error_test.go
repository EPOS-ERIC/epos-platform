package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestRequestTimeoutPropagatesDeadline(t *testing.T) {
	deadlineObserved := make(chan bool, 1)
	handler := RequestTimeout(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		deadlineObserved <- errors.Is(r.Context().Err(), context.DeadlineExceeded)
	}), 10*time.Millisecond)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !<-deadlineObserved {
		t.Fatal("request context ended without deadline exceeded")
	}
}

func TestNewErrorRecordsTimeoutOnTrace(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = provider.Shutdown(context.Background()) }()
	ctx, span := provider.Tracer("gateway-test").Start(context.Background(), "request")

	handler := &ogenHandler{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	response := handler.NewError(ctx, context.DeadlineExceeded)
	if response.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusGatewayTimeout)
	}
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	if ended[0].Status().Code != codes.Error {
		t.Fatalf("span status = %v, want error", ended[0].Status())
	}
	if len(ended[0].Events()) == 0 {
		t.Fatal("timeout was not recorded as a span event")
	}
}

func TestWriteErrorRecordsServerFailureOnTrace(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = provider.Shutdown(context.Background()) }()
	ctx, span := provider.Tracer("gateway-test").Start(context.Background(), "request")

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeError(log, ctx, httptest.NewRecorder(), errors.New("private failure"))
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 || ended[0].Status().Code != codes.Error || len(ended[0].Events()) == 0 {
		t.Fatalf("recorded span = %+v, want error status and event", ended)
	}
}

func TestProblemResponseUsesSharedErrorAndTrace(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	if err != nil {
		t.Fatal(err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	}))
	err = apperror.Field("metadata.title", "Title must not be blank.", errors.New("private diagnostic"))
	status, problem := problemResponse(ctx, err)
	if status != 400 || problem.Type != "validation_failed" || problem.Title != "Bad Request" || problem.Status != 400 {
		t.Fatalf("status/problem = %d/%+v", status, problem)
	}
	if !problem.Target.Set || problem.Target.Value != "metadata.title" {
		t.Fatalf("Target = %+v", problem.Target)
	}
	if !problem.Instance.Set || problem.Instance.Value != "traces/0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("Instance = %+v", problem.Instance)
	}
}

func TestProblemResponseReportsNestedEmptyParameterLabel(t *testing.T) {
	const body = `{"resource_type":"dataset","revision":1,"metadata":{"title":"title","description":"description","distributions":[{"title":"distribution","type":"WEB_SERVICE","web_service":{"name":"service","template_url":"https://example.org/query","method":"GET","type":"GEOJSON","format":"application/json","parameters":[{"kind":"DATETIME","position":0,"name":"starttime","label":"","required":true,"read_only":false,"location":"QUERY","explode":false}]}}]}}`
	var request gatewayapi.ReplaceResourceDraftRequest
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatal(err)
	}
	draft, ok := request.GetReplaceDatasetDraftRequest()
	if !ok {
		t.Fatal("decoded request is not a dataset draft")
	}
	if err := draft.Validate(); err == nil {
		t.Fatal("expected generated request validation to reject empty parameter label")
	} else {
		status, problem := problemResponse(context.Background(), err)
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
		}
		if !problem.Target.Set || problem.Target.Value != "metadata.distributions[0].web_service.parameters[0].label" {
			t.Fatalf("target = %+v", problem.Target)
		}
		if !problem.Detail.Set || problem.Detail.Value != "Field must not be blank." {
			t.Fatalf("detail = %+v", problem.Detail)
		}
	}
}

func TestWriteErrorDoesNotExposeUnknownCause(t *testing.T) {
	response := httptest.NewRecorder()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeError(log, context.Background(), response, errors.New("private SQL password"))
	if response.Code != 500 {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{`"type":"internal"`, `"title":"Internal Server Error"`, `"status":500`, "An unexpected error occurred."} {
		if !strings.Contains(body, want) {
			t.Errorf("body %q missing %q", body, want)
		}
	}
	if strings.Contains(body, "private SQL password") {
		t.Fatal("internal error leaked into problem response")
	}
}
