package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
)

func TestRunReturnsErrorForInvalidListenAddress(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Server: config.Server{
			Listen: "127.0.0.1",
		},
	}

	err := Run(cfg, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "listen and serve") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScalarRoute(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/scalar", nil)
	rec := httptest.NewRecorder()

	NewRouter(nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
	}

	if location := rec.Header().Get("Location"); location != "/scalar/" {
		t.Fatalf("expected redirect to /scalar/, got %q", location)
	}
}

func TestScalarRouteServesEmbeddedAssets(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil, nil)

	htmlReq := httptest.NewRequest(http.MethodGet, "/scalar/", nil)
	htmlRec := httptest.NewRecorder()
	router.ServeHTTP(htmlRec, htmlReq)

	if htmlRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, htmlRec.Code)
	}

	contentType := htmlRec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected html content type, got %q", contentType)
	}

	body := htmlRec.Body.String()
	if !strings.Contains(body, "Scalar API Reference") {
		t.Fatal("expected scalar ui body")
	}

	if !strings.Contains(body, "src=\"./api-reference.js\"") {
		t.Fatal("expected scalar ui to use local script")
	}

	scriptReq := httptest.NewRequest(http.MethodGet, "/scalar/api-reference.js", nil)
	scriptRec := httptest.NewRecorder()
	router.ServeHTTP(scriptRec, scriptReq)

	if scriptRec.Code != http.StatusOK {
		t.Fatalf("expected script status %d, got %d", http.StatusOK, scriptRec.Code)
	}

	if !strings.Contains(scriptRec.Body.String(), "@scalar/api-reference@") {
		t.Fatal("expected vendored scalar script body")
	}
}

func TestRequestTimeoutHandlerTimesOutLongRequests(t *testing.T) {
	t.Parallel()

	handler := requestTimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		fmt.Fprint(w, "ok")
	}), 10*time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	if body := rec.Body.String(); body != "request timeout\n" {
		t.Fatalf("unexpected body: got %q", body)
	}
}

func TestRequestTimeoutHandlerAllowsFastRequests(t *testing.T) {
	t.Parallel()

	handler := requestTimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}), 10*time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("unexpected body: got %q", body)
	}
}
