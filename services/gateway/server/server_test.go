package server

import (
	"encoding/json"
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

	err := Run(cfg, nil, nil, nil)
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

	NewRouter(nil, nil, nil, nil, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
	}

	if location := rec.Header().Get("Location"); location != "/scalar/" {
		t.Fatalf("expected redirect to /scalar/, got %q", location)
	}
}

func TestScalarRouteServesEmbeddedAssets(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil, nil, nil, nil, nil, "")

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

func TestUsersRouteListsUsers(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var users []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatalf("decode users: %v", err)
	}

	if len(users) != 1 || users[0].ID != "local-1" {
		t.Fatalf("unexpected users response: %+v", users)
	}
}

func TestUsersRoutePatchesUser(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/local-1", strings.NewReader(`{"given_name":"Ada","family_name":"Lovelace","email":"ada@example.org","is_super_admin":true,"status":"ACTIVE"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var user struct {
		ID           string `json:"id"`
		Email        string `json:"email"`
		IsSuperAdmin bool   `json:"is_super_admin"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode user: %v", err)
	}

	if user.ID != "local-1" || user.Email != "ada@example.org" || !user.IsSuperAdmin {
		t.Fatalf("unexpected user response: %+v", user)
	}
}

func TestUsersRouteRejectsPatchAuthIdentifier(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/local-1", strings.NewReader(`{"auth_identifier":"nope"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUsersRouteRejectsListForNonAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUsersRouteAllowsSelfLookup(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/users/local-1", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUsersRouteRejectsOtherUserLookupForNonAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/users/local-2", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUsersRouteRejectsPatchForNonAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/local-1", strings.NewReader(`{"given_name":"Ada"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
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
