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

	err := Run(configForTest("127.0.0.1"), nil, nil, nil)
	if err == nil {
		t.Fatal("expected invalid listen address error")
	}
}

func configForTest(listen string) config.Config {
	cfg := config.Default()
	cfg.Server.Listen = listen
	return cfg
}

func TestScalarRouteRedirects(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil, nil, nil, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/scalar", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/scalar/" {
		t.Fatalf("unexpected redirect location: %s", location)
	}
}

func TestScalarRouteServesEmbeddedAssets(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil, nil, nil, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/scalar/api-reference.js", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "@scalar/api-reference@") {
		t.Fatal("expected vendored scalar script body")
	}
}

func TestGatewayDoesNotExposeUserGroupRoutes(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil, nil, nil, nil, nil, "")
	paths := []string{"/v1/users", "/v1/users/user-1", "/v1/groups", "/v1/groups/group-1"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
			}
		})
	}
}

func TestRequestTimeoutHandlerTimesOutLongRequests(t *testing.T) {
	t.Parallel()

	handler := requestTimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		_, _ = fmt.Fprint(w, "ok")
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
		_, _ = fmt.Fprint(w, "ok")
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
