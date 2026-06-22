package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/usergroup"
)

func TestAuthMiddlewareAttachesAuthenticatedUser(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	handler := authMiddleware(authService, provisioner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := handlers.AuthenticatedUserFromContext(r.Context())
		if !ok {
			t.Fatal("missing authenticated user in context")
		}
		if user.ID != "local-1" {
			t.Fatalf("unexpected local user id: %s", user.ID)
		}
		actor, ok := logging.ActorFromContext(r.Context())
		if !ok {
			t.Fatal("missing actor in logging context")
		}
		if actor.ID != "local-1" {
			t.Fatalf("unexpected actor id: %s", actor.ID)
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

	authService, provisioner := newTestAuthStack(t, true, false, false)
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

	authService, provisioner := newTestAuthStack(t, false, false, false)
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

	authService, provisioner := newTestAuthStack(t, true, true, false)
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

func newTestAuthStack(t *testing.T, active, failCreate, superAdmin bool) (*auth.Service, *usergroup.Service) {
	t.Helper()

	usergroupSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && (r.URL.Path == "/api/v2/users/local-1" || r.URL.Path == "/api/v2/users/local-2") {
			if r.URL.Path == "/api/v2/users/local-1" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":         "local-1",
					"email":      "alice@example.org",
					"givenName":  "Alice",
					"familyName": "Rossi",
					"status":     "ACTIVE",
				})
				return
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         "local-2",
				"email":      "bob@example.org",
				"givenName":  "Bob",
				"familyName": "Meyer",
				"status":     "ACTIVE",
			})
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users":
			_ = json.NewEncoder(w).Encode([]struct {
				ID             string `json:"id"`
				AuthIdentifier string `json:"authIdentifier"`
				Email          string `json:"email"`
				GivenName      string `json:"givenName"`
				FamilyName     string `json:"familyName"`
				IsSuperAdmin   bool   `json:"isSuperAdmin"`
				Status         string `json:"status"`
			}{{
				ID:             "local-1",
				AuthIdentifier: "user-123",
				Email:          "user@example.org",
				GivenName:      "User",
				FamilyName:     "Example",
				IsSuperAdmin:   superAdmin,
				Status:         "ACTIVE",
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups":
			_ = json.NewEncoder(w).Encode([]struct {
				ID          string `json:"id"`
				Slug        string `json:"slug"`
				Name        string `json:"name"`
				Description string `json:"description"`
				IsSystem    bool   `json:"isSystem"`
			}{
				{ID: "group-1", Slug: "research", Name: "Research", Description: "Research group", IsSystem: false},
				{ID: "group-2", Slug: "data", Name: "Data", Description: "Data group", IsSystem: false},
				{ID: "group-3", Slug: "public", Name: "Public", Description: "Public group", IsSystem: false},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			_ = json.NewEncoder(w).Encode([]struct {
				UserID  string `json:"userId"`
				GroupID string `json:"groupId"`
				Role    string `json:"role"`
				Status  string `json:"status"`
			}{
				{UserID: "local-1", GroupID: "group-1", Role: "VIEWER", Status: "ACTIVE"},
				{UserID: "local-2", GroupID: "group-1", Role: "ADMIN", Status: "ACTIVE"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-2/memberships":
			_ = json.NewEncoder(w).Encode([]struct {
				UserID  string `json:"userId"`
				GroupID string `json:"groupId"`
				Role    string `json:"role"`
				Status  string `json:"status"`
			}{
				{UserID: "local-1", GroupID: "group-2", Role: "EDITOR", Status: "PENDING"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "group-1",
				"slug":        "research",
				"name":        "Research",
				"description": "Research group",
				"isSystem":    false,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "group-2",
				"slug":        "data",
				"name":        "Data",
				"description": "Data group",
				"isSystem":    false,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-3":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "group-3",
				"slug":        "public",
				"name":        "Public",
				"description": "Public group",
				"isSystem":    false,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-3/memberships":
			_ = json.NewEncoder(w).Encode([]struct {
				UserID  string `json:"userId"`
				GroupID string `json:"groupId"`
				Role    string `json:"role"`
				Status  string `json:"status"`
			}{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/groups":
			var payload struct {
				Slug        string `json:"slug"`
				Name        string `json:"name"`
				Description string `json:"description"`
				IsSystem    bool   `json:"isSystem"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create group payload: %v", err)
			}
			if payload.Slug == "duplicate" {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "group-1",
				"slug":        payload.Slug,
				"name":        payload.Name,
				"description": payload.Description,
				"isSystem":    payload.IsSystem,
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/groups/group-1":
			var payload struct {
				Slug        string `json:"slug"`
				Name        string `json:"name"`
				Description string `json:"description"`
				IsSystem    bool   `json:"isSystem"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode patch group payload: %v", err)
			}
			if payload.Slug == "duplicate" {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "group-1",
				"slug":        payload.Slug,
				"name":        payload.Name,
				"description": payload.Description,
				"isSystem":    payload.IsSystem,
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/groups/group-404":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/bad-group-id":
			w.WriteHeader(http.StatusBadRequest)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/groups/bad-group-id":
			w.WriteHeader(http.StatusBadRequest)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/groups/group-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/groups/group-404":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/groups/bad-group-id":
			w.WriteHeader(http.StatusBadRequest)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/groups/group-1/memberships/local-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/groups/group-1/memberships/local-2":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/groups/group-404/memberships/local-404":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/groups/group-1/memberships/local-1":
			var payload struct {
				Role   string `json:"role"`
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode membership request payload: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"userId":    "local-1",
				"groupId":   "group-1",
				"role":      payload.Role,
				"status":    payload.Status,
				"createdAt": "2026-06-17T12:48:45.574Z",
				"updatedAt": "2026-06-17T12:48:45.574Z",
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/users/local-1":
			var payload struct {
				Email        string `json:"email"`
				GivenName    string `json:"givenName"`
				FamilyName   string `json:"familyName"`
				IsSuperAdmin bool   `json:"isSuperAdmin"`
				Status       string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode patch payload: %v", err)
			}

			_ = json.NewEncoder(w).Encode(struct {
				ID             string `json:"id"`
				AuthIdentifier string `json:"authIdentifier"`
				Email          string `json:"email"`
				GivenName      string `json:"givenName"`
				FamilyName     string `json:"familyName"`
				IsSuperAdmin   bool   `json:"isSuperAdmin"`
				Status         string `json:"status"`
			}{
				ID:             "local-1",
				AuthIdentifier: "user-123",
				Email:          payload.Email,
				GivenName:      payload.GivenName,
				FamilyName:     payload.FamilyName,
				IsSuperAdmin:   payload.IsSuperAdmin,
				Status: func() string {
					if payload.Status != "" {
						return payload.Status
					}
					return "ACTIVE"
				}(),
			})
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
				"isSuperAdmin":   superAdmin,
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
				"isSuperAdmin":   superAdmin,
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
