package usergroup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
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
				CreatedAt:      stringPtr("2026-06-16T12:00:00Z"),
				UpdatedAt:      stringPtr("2026-06-16T12:30:00Z"),
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
	users, err := service.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("unexpected users count: %d", len(users))
	}

	if users[0].ID != "local-1" || users[0].Status != "ACTIVE" {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if users[0].CreatedAt != "2026-06-16T12:00:00Z" {
		t.Fatalf("unexpected first user created at: %s", users[0].CreatedAt)
	}
	if users[1].ID != "local-2" || users[1].Status != "INACTIVE" {
		t.Fatalf("unexpected second user: %+v", users[1])
	}
}

func stringPtr(value string) *string {
	return &value
}
