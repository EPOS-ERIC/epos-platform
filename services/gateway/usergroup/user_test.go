package usergroup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func TestEnsureUserCreatesMissingUser(t *testing.T) {
	t.Parallel()

	created := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/users/by-auth/"):
			w.Header().Set("Content-Type", "application/json")
			if created {
				_ = json.NewEncoder(w).Encode(struct {
					ID             string `json:"id"`
					AuthIdentifier string `json:"authIdentifier"`
					Email          string `json:"email"`
					GivenName      string `json:"givenName"`
					FamilyName     string `json:"familyName"`
					Status         string `json:"status"`
				}{
					ID:             "local-1",
					AuthIdentifier: "user-123",
					Email:          "user@example.org",
					GivenName:      "User",
					FamilyName:     "Example",
					Status:         "ACTIVE",
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/users":
			w.Header().Set("Content-Type", "application/json")
			var payload struct {
				AuthIdentifier string `json:"authIdentifier"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			if payload.AuthIdentifier != "user-123" {
				t.Fatalf("unexpected authIdentifier: %v", payload.AuthIdentifier)
			}
			created = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(struct {
				ID             string `json:"id"`
				AuthIdentifier string `json:"authIdentifier"`
				Email          string `json:"email"`
				GivenName      string `json:"givenName"`
				FamilyName     string `json:"familyName"`
				Status         string `json:"status"`
			}{
				ID:             "local-1",
				AuthIdentifier: "user-123",
				Email:          "user@example.org",
				GivenName:      "User",
				FamilyName:     "Example",
				Status:         "ACTIVE",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	user, err := service.EnsureUser(context.Background(), Identity{
		AuthIdentifier: "user-123",
		Email:          "user@example.org",
		GivenName:      "User",
		FamilyName:     "Example",
	})
	if err != nil {
		t.Fatalf("ensure user: %v", err)
	}

	if user.ID != "local-1" {
		t.Fatalf("unexpected user id: %s", user.ID)
	}
	if user.AuthIdentifier != "user-123" {
		t.Fatalf("unexpected auth identifier: %s", user.AuthIdentifier)
	}
	if !created {
		t.Fatal("expected create path")
	}
}

func TestListUsers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/users" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]struct {
			ID             string  `json:"id"`
			AuthIdentifier string  `json:"authIdentifier"`
			Email          string  `json:"email"`
			GivenName      string  `json:"givenName"`
			FamilyName     string  `json:"familyName"`
			Status         string  `json:"status"`
			CreatedAt      *string `json:"createdAt,omitempty"`
			UpdatedAt      *string `json:"updatedAt,omitempty"`
		}{
			{
				ID:             "local-1",
				AuthIdentifier: "user-123",
				Email:          "user@example.org",
				GivenName:      "User",
				FamilyName:     "Example",
				Status:         "ACTIVE",
				CreatedAt:      new("2026-06-16T12:00:00Z"),
				UpdatedAt:      new("2026-06-16T12:30:00Z"),
			},
			{
				ID:             "local-2",
				AuthIdentifier: "user-456",
				Email:          "other@example.org",
				GivenName:      "Other",
				FamilyName:     "Example",
				Status:         "INACTIVE",
			},
		})
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	users, err := service.ListUsers(context.Background(), api.User{IsSuperAdmin: true})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("unexpected users count: %d", len(users))
	}

	if users[0].ID != "local-1" || users[0].Status != api.ActiveStatus {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if users[0].CreatedAt != "2026-06-16T12:00:00Z" {
		t.Fatalf("unexpected first user created at: %s", users[0].CreatedAt)
	}
	if users[1].ID != "local-2" || users[1].Status != api.InactiveStatus {
		t.Fatalf("unexpected second user: %+v", users[1])
	}
}

func TestPatchUser(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/users/local-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

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

		if payload.Email != "ada@example.org" || payload.GivenName != "Ada" || payload.FamilyName != "Lovelace" || !payload.IsSuperAdmin || payload.Status != "ACTIVE" {
			t.Fatalf("unexpected payload: %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
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
			Email:          "ada@example.org",
			GivenName:      "Ada",
			FamilyName:     "Lovelace",
			IsSuperAdmin:   true,
			Status:         "ACTIVE",
		})
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	user, err := service.PatchUser(context.Background(), api.User{IsSuperAdmin: true}, "local-1", api.PatchUserRequest{
		Email:        new("ada@example.org"),
		GivenName:    new("Ada"),
		FamilyName:   new("Lovelace"),
		IsSuperAdmin: new(true),
		Status:       new(api.ActiveStatus),
	})
	if err != nil {
		t.Fatalf("patch user: %v", err)
	}

	if user.Email != "ada@example.org" || user.GivenName != "Ada" || user.FamilyName != "Lovelace" || !user.IsSuperAdmin {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestPatchUserAllowsSelfProfileUpdate(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/users/local-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var payload struct {
			Email      string `json:"email"`
			GivenName  string `json:"givenName"`
			FamilyName string `json:"familyName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode patch payload: %v", err)
		}

		if payload.GivenName != "Ada" || payload.Email != "" || payload.FamilyName != "" {
			t.Fatalf("unexpected payload: %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			ID             string `json:"id"`
			AuthIdentifier string `json:"authIdentifier"`
			Email          string `json:"email"`
			GivenName      string `json:"givenName"`
			FamilyName     string `json:"familyName"`
			Status         string `json:"status"`
		}{
			ID:             "local-1",
			AuthIdentifier: "user-123",
			Email:          "user@example.org",
			GivenName:      "Ada",
			FamilyName:     "Example",
			Status:         "ACTIVE",
		})
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	user, err := service.PatchUser(context.Background(), api.User{ID: "local-1"}, "local-1", api.PatchUserRequest{
		GivenName: new("Ada"),
	})
	if err != nil {
		t.Fatalf("patch self user: %v", err)
	}

	if user.GivenName != "Ada" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestPatchUserRejectsSelfAdminFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.PatchUser(context.Background(), api.User{ID: "local-1"}, "local-1", api.PatchUserRequest{Status: new(api.ActiveStatus)}); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("patch self admin fields = %v, want ErrBadRequest", err)
	}
}

func TestAuthorizationInMethods(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/local-1":
			_ = json.NewEncoder(w).Encode(struct {
				ID             string `json:"id"`
				AuthIdentifier string `json:"authIdentifier"`
				Email          string `json:"email"`
				GivenName      string `json:"givenName"`
				FamilyName     string `json:"familyName"`
				Status         string `json:"status"`
			}{
				ID:             "local-1",
				AuthIdentifier: "user-123",
				Email:          "user@example.org",
				GivenName:      "User",
				FamilyName:     "Example",
				Status:         "ACTIVE",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups":
			_ = json.NewEncoder(w).Encode([]struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				{ID: "group-1", Name: "Research"},
				{ID: "group-2", Name: "Data"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-1", Role: "VIEWER", Status: "ACTIVE"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-2/memberships":
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-1", Role: "EDITOR", Status: "PENDING"}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/users/local-1":
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
				Email:          "user@example.org",
				GivenName:      "User",
				FamilyName:     "Example",
				IsSuperAdmin:   true,
				Status:         "ACTIVE",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)

	if _, err := service.ListUsers(context.Background(), api.User{IsSuperAdmin: true}); err != nil {
		t.Fatalf("admin list users: %v", err)
	}

	if _, err := service.GetUser(context.Background(), api.User{ID: "local-1"}, "local-1"); err != nil {
		t.Fatalf("self get user: %v", err)
	}

	if _, err := service.ListUsers(context.Background(), api.User{ID: "local-1"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("list users forbidden = %v, want ErrForbidden", err)
	}

	if _, err := service.GetUser(context.Background(), api.User{ID: "local-1"}, "local-2"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("get user forbidden = %v, want ErrForbidden", err)
	}

	if _, err := service.PatchUser(context.Background(), api.User{ID: "local-1"}, "local-2", api.PatchUserRequest{}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("patch user forbidden = %v, want ErrForbidden", err)
	}
}
