package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/usergroup"
)

func TestAuthMiddlewareAttachesAuthenticatedUser(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false)
	handler := authMiddleware(authService, provisioner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := handlers.AuthenticatedUserFromContext(r.Context())
		if !ok {
			t.Fatal("missing authenticated user in context")
		}
		if user.Identity.AuthIdentifier != "user-123" {
			t.Fatalf("unexpected auth identifier: %s", user.Identity.AuthIdentifier)
		}
		if user.User.ID != "local-1" {
			t.Fatalf("unexpected local user id: %s", user.User.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/users/123", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false)
	handler := authMiddleware(authService, provisioner)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/users/123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, false, false)
	handler := authMiddleware(authService, provisioner)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/users/123", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsProvisioningFailure(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, true)
	handler := authMiddleware(authService, provisioner)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/users/123", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func newTestAuthStack(t *testing.T, active bool, failCreate bool) (*auth.Service, *usergroup.Service) {
	t.Helper()

	usergroupSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users":
			_ = json.NewEncoder(w).Encode([]struct {
				ID             string `json:"id"`
				AuthIdentifier string `json:"authIdentifier"`
				Email          string `json:"email"`
				GivenName      string `json:"givenName"`
				FamilyName     string `json:"familyName"`
				Status         string `json:"status"`
			}{{
				ID:             "local-1",
				AuthIdentifier: "user-123",
				Email:          "user@example.org",
				GivenName:      "User",
				FamilyName:     "Example",
				Status:         "ACTIVE",
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/by-auth/user-123":
			if failCreate {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "local-1",
				"authIdentifier": "user-123",
				"email":          "user@example.org",
				"givenName":      "User",
				"familyName":     "Example",
				"status":         "ACTIVE",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/users":
			if failCreate {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "local-1",
				"authIdentifier": "user-123",
				"email":          "user@example.org",
				"givenName":      "User",
				"familyName":     "Example",
				"status":         "ACTIVE",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(usergroupSrv.Close)

	usergroupClient, err := clients.NewUsergroup(usergroupSrv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new usergroup client: %v", err)
	}
	provisioner := usergroup.NewService(usergroupClient)

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":            active,
			"eduPersonUniqueId": "user-123",
			"email":             "user@example.org",
			"firstname":         "User",
			"lastName":          "Example",
		})
	}))
	t.Cleanup(authSrv.Close)

	authService, err := auth.NewService(authSrv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	return authService, provisioner
}
