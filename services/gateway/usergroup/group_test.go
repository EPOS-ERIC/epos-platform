package usergroup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type groupMembershipPayload struct {
	UserID  string `json:"userId"`
	GroupID string `json:"groupId,omitempty"`
	Role    string `json:"role"`
	Status  string `json:"status"`
}

func TestListGroups(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
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
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-1", Role: "VIEWER", Status: "ACTIVE"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-2/memberships":
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{})
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
	groups, err := service.ListGroups(context.Background(), api.User{ID: "local-1"})
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}

	if len(groups) != 2 || groups[0].Membership == nil || groups[1].Membership != nil {
		t.Fatalf("unexpected groups: %+v", groups)
	}
}

func TestGetGroup(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "group-1",
				"slug":        "research",
				"name":        "Research",
				"description": "Research group",
				"isSystem":    false,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-1", Role: "VIEWER", Status: "ACTIVE"}})
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
	group, err := service.GetGroup(context.Background(), api.User{ID: "local-1"}, "group-1")
	if err != nil {
		t.Fatalf("get group: %v", err)
	}

	if group.ID != "group-1" || group.Membership == nil || group.Membership.Role != "VIEWER" {
		t.Fatalf("unexpected group: %+v", group)
	}
}

func TestGetGroupReturnsBadRequest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/groups/bad-group-id" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.GetGroup(context.Background(), api.User{ID: "local-1"}, "bad-group-id"); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("get group bad request = %v, want ErrBadRequest", err)
	}
}

func TestDeleteGroup(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/groups/group-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if err := service.DeleteGroup(context.Background(), api.User{IsSuperAdmin: true}, "group-1"); err != nil {
		t.Fatalf("delete group: %v", err)
	}
}

func TestDeleteGroupRejectsNonAdmin(t *testing.T) {
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
	if err := service.DeleteGroup(context.Background(), api.User{}, "group-1"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("delete group forbidden = %v, want ErrForbidden", err)
	}
}

func TestDeleteGroupReturnsBadRequest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/groups/bad-group-id" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if err := service.DeleteGroup(context.Background(), api.User{IsSuperAdmin: true}, "bad-group-id"); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("delete group bad request = %v, want ErrBadRequest", err)
	}
}

func TestRequestMembership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		user        api.User
		wantPayload string
		wantStatus  api.MembershipStatus
	}{
		{name: "member", user: api.User{ID: "local-1"}, wantPayload: "PENDING", wantStatus: api.MembershipPendingStatus},
		{name: "super admin", user: api.User{ID: "local-1", IsSuperAdmin: true}, wantPayload: "ACTIVE", wantStatus: api.MembershipActiveStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != "/api/v2/groups/group-1/memberships/local-1" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}

				var payload struct {
					Role   string `json:"role"`
					Status string `json:"status"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode membership payload: %v", err)
				}
				if payload.Role != "VIEWER" || payload.Status != tt.wantPayload {
					t.Fatalf("unexpected payload: %+v", payload)
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"userId":    "local-1",
					"groupId":   "group-1",
					"role":      payload.Role,
					"status":    payload.Status,
					"createdAt": "2026-06-17T12:48:45.574Z",
					"updatedAt": "2026-06-17T12:48:45.574Z",
				})
			}))
			t.Cleanup(srv.Close)

			client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
			if err != nil {
				t.Fatalf("new client: %v", err)
			}

			service := NewService(client)
			membership, err := service.RequestMembership(context.Background(), tt.user, "group-1", api.CreateMembershipRequest{Role: api.MembershipViewerRole})
			if err != nil {
				t.Fatalf("request membership: %v", err)
			}

			if membership.UserID != "local-1" || membership.GroupID != "group-1" || membership.Role != api.MembershipViewerRole || membership.Status != tt.wantStatus {
				t.Fatalf("unexpected membership: %+v", membership)
			}
		})
	}
}

func TestRequestMembershipRejectsBadRole(t *testing.T) {
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
	if _, err := service.RequestMembership(context.Background(), api.User{ID: "local-1"}, "group-1", api.CreateMembershipRequest{Role: api.MembershipRole("NOPE")}); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("request membership bad role = %v, want ErrBadRequest", err)
	}
}

func TestRequestMembershipReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/groups/group-404/memberships/local-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.RequestMembership(context.Background(), api.User{ID: "local-1"}, "group-404", api.CreateMembershipRequest{Role: api.MembershipViewerRole}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("request membership not found = %v, want ErrNotFound", err)
	}
}

func TestPatchMembership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		members    []groupMembershipPayload
		request    api.PatchMembershipRequest
		wantRole   api.MembershipRole
		wantStatus api.MembershipStatus
	}{
		{
			name:       "updates role only",
			members:    []groupMembershipPayload{{UserID: "local-1", GroupID: "group-1", Role: "VIEWER", Status: "ACTIVE"}},
			request:    api.PatchMembershipRequest{Role: new(api.MembershipReviewerRole)},
			wantRole:   api.MembershipReviewerRole,
			wantStatus: api.MembershipActiveStatus,
		},
		{
			name:       "updates status only",
			members:    []groupMembershipPayload{{UserID: "local-1", GroupID: "group-1", Role: "EDITOR", Status: "PENDING"}},
			request:    api.PatchMembershipRequest{Status: new(api.MembershipRevokedStatus)},
			wantRole:   api.MembershipEditorRole,
			wantStatus: api.MembershipRevokedStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(tt.members)
				case r.Method == http.MethodPut && r.URL.Path == "/api/v2/groups/group-1/memberships/local-1":
					var payload struct {
						Role   string `json:"role"`
						Status string `json:"status"`
					}
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Fatalf("decode patch payload: %v", err)
					}
					if payload.Role != string(tt.wantRole) || payload.Status != string(tt.wantStatus) {
						t.Fatalf("unexpected payload: %+v", payload)
					}

					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{
						"userId":    "local-1",
						"groupId":   "group-1",
						"role":      payload.Role,
						"status":    payload.Status,
						"createdAt": "2026-06-17T12:48:45.574Z",
						"updatedAt": "2026-06-17T13:05:45.574Z",
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
			membership, err := service.PatchMembership(context.Background(), api.User{ID: "local-1", IsSuperAdmin: true}, "group-1", "local-1", tt.request)
			if err != nil {
				t.Fatalf("patch membership: %v", err)
			}

			if membership.Role != tt.wantRole || membership.Status != tt.wantStatus {
				t.Fatalf("unexpected membership: %+v", membership)
			}
		})
	}
}

func TestPatchMembershipRejectsMissingMembership(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-2", GroupID: "group-1", Role: "ADMIN", Status: "ACTIVE"}})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.PatchMembership(context.Background(), api.User{ID: "local-1", IsSuperAdmin: true}, "group-1", "local-1", api.PatchMembershipRequest{Status: new(api.MembershipRevokedStatus)}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("patch membership not found = %v, want ErrNotFound", err)
	}
}

func TestPatchMembershipRejectsNonAdmin(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-2", GroupID: "group-1", Role: "ADMIN", Status: "ACTIVE"}})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.PatchMembership(context.Background(), api.User{ID: "local-1"}, "group-1", "local-1", api.PatchMembershipRequest{Status: new(api.MembershipRevokedStatus)}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("patch membership forbidden = %v, want ErrForbidden", err)
	}
}

func TestDeleteMembershipAllowsSelfLeave(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/groups/group-1/memberships/local-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if err := service.DeleteMembership(context.Background(), api.User{ID: "local-1"}, "group-1", "local-1"); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
}

func TestDeleteMembershipAllowsAdminDelete(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-2", GroupID: "group-1", Role: "ADMIN", Status: "ACTIVE"}})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/groups/group-1/memberships/local-2":
			w.WriteHeader(http.StatusNoContent)
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
	if err := service.DeleteMembership(context.Background(), api.User{ID: "local-1", IsSuperAdmin: true}, "group-1", "local-2"); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
}

func TestDeleteMembershipRejectsNonAdminOtherUser(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-2", GroupID: "group-1", Role: "ADMIN", Status: "ACTIVE"}})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if err := service.DeleteMembership(context.Background(), api.User{ID: "local-1"}, "group-1", "local-2"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("delete membership forbidden = %v, want ErrForbidden", err)
	}
}

func TestDeleteMembershipReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/groups/group-1/memberships/local-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if err := service.DeleteMembership(context.Background(), api.User{ID: "local-1"}, "group-1", "local-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("delete membership not found = %v, want ErrNotFound", err)
	}
}

func TestListGroupMemberships(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/local-1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         "local-1",
				"givenName":  "Alice",
				"familyName": "Rossi",
				"email":      "alice@example.org",
			})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/local-2" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         "local-2",
				"givenName":  "Bob",
				"familyName": "Meyer",
				"email":      "bob@example.org",
			})
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/groups/group-1/memberships" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]struct {
			UserID  string `json:"userId"`
			GroupID string `json:"groupId"`
			Role    string `json:"role"`
			Status  string `json:"status"`
		}{
			{UserID: "local-1", GroupID: "group-1", Role: "VIEWER", Status: "ACTIVE"},
			{UserID: "local-2", GroupID: "group-1", Role: "ADMIN", Status: "ACTIVE"},
		})
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	memberships, err := service.ListGroupMemberships(context.Background(), api.User{ID: "local-1", IsSuperAdmin: true}, "group-1")
	if err != nil {
		t.Fatalf("list memberships: %v", err)
	}

	if len(memberships) != 2 {
		t.Fatalf("unexpected memberships len: %d", len(memberships))
	}
	if memberships[0].User.ID != "local-1" || memberships[0].User.Name != "Alice Rossi" || memberships[0].User.Href != "v1/users/local-1" || memberships[0].Role != api.MembershipViewerRole || memberships[0].Status != api.MembershipActiveStatus {
		t.Fatalf("unexpected membership[0]: %+v", memberships[0])
	}
	if memberships[1].User.ID != "local-2" || memberships[1].User.Name != "Bob Meyer" || memberships[1].User.Href != "v1/users/local-2" || memberships[1].Role != api.MembershipAdminRole || memberships[1].Status != api.MembershipActiveStatus {
		t.Fatalf("unexpected membership[1]: %+v", memberships[1])
	}
}

func TestListGroupMembershipsRejectsNonAdmin(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/local-1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         "local-1",
				"givenName":  "Alice",
				"familyName": "Rossi",
			})
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/groups/group-1/memberships" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-1", GroupID: "group-1", Role: "VIEWER", Status: "ACTIVE"}})
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.ListGroupMemberships(context.Background(), api.User{ID: "local-1"}, "group-1"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("list memberships forbidden = %v, want ErrForbidden", err)
	}
}

func TestListGroupMembershipsRejectsBadRequest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/groups/group-1/memberships" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.ListGroupMemberships(context.Background(), api.User{ID: "local-1", IsSuperAdmin: true}, "group-1"); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("list memberships bad request = %v, want ErrBadRequest", err)
	}
}

func TestCreateGroup(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/groups" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var payload struct {
			Slug        string `json:"slug"`
			Name        string `json:"name"`
			Description string `json:"description"`
			IsSystem    bool   `json:"isSystem"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode create payload: %v", err)
		}
		if payload.Slug != "research" || payload.Name != "Research" || payload.Description != "Research group" || payload.IsSystem {
			t.Fatalf("unexpected payload: %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "group-1",
			"slug":        payload.Slug,
			"name":        payload.Name,
			"description": payload.Description,
			"isSystem":    payload.IsSystem,
		})
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	group, err := service.CreateGroup(context.Background(), api.User{IsSuperAdmin: true}, api.CreateGroupRequest{
		Slug:        "research",
		Name:        "Research",
		Description: new("Research group"),
	})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	if group.ID != "group-1" || group.Slug != "research" || group.Name != "Research" {
		t.Fatalf("unexpected group: %+v", group)
	}
}

func TestCreateGroupRejectsNonAdmin(t *testing.T) {
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
	if _, err := service.CreateGroup(context.Background(), api.User{}, api.CreateGroupRequest{Slug: "research", Name: "Research"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("create group forbidden = %v, want ErrForbidden", err)
	}
}

func TestCreateGroupReturnsConflict(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/groups" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusConflict)
	}))
	t.Cleanup(srv.Close)

	client, err := clients.NewUsergroup(srv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	service := NewService(client)
	if _, err := service.CreateGroup(context.Background(), api.User{IsSuperAdmin: true}, api.CreateGroupRequest{Slug: "duplicate", Name: "Duplicate"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("create group conflict = %v, want ErrConflict", err)
	}
}

func TestPatchGroupReturnsConflict(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]groupMembershipPayload{{UserID: "local-1", GroupID: "group-1", Role: "ADMIN", Status: "ACTIVE"}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/groups/group-1":
			w.WriteHeader(http.StatusConflict)
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
	if _, err := service.PatchGroup(context.Background(), api.User{ID: "local-1"}, "group-1", api.PatchGroupRequest{Slug: new("duplicate")}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("patch group conflict = %v, want ErrConflict", err)
	}
}
