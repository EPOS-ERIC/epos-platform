package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/usergroup"
)

func TestRunReturnsErrorForInvalidListenAddress(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Server: config.Server{
			Listen: "127.0.0.1",
		},
	}

	err := Run(cfg, nil, nil, nil, nil)
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

	body := rec.Body.String()

	var users []struct {
		ID             string `json:"id"`
		AuthIdentifier string `json:"auth_identifier"`
		Email          string `json:"email"`
		GivenName      string `json:"given_name"`
		FamilyName     string `json:"family_name"`
		IsSuperAdmin   bool   `json:"is_super_admin"`
		Status         string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatalf("decode users: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("unexpected users response: %+v", users)
	}

	user := users[0]
	if user.ID != "local-1" || user.AuthIdentifier != "user-123" || user.Email != "user@example.org" || user.GivenName != "User" || user.FamilyName != "Example" || user.Status != "ACTIVE" {
		t.Fatalf("unexpected users response: %+v", user)
	}

	if !strings.Contains(body, `"is_super_admin":`) {
		t.Fatal("expected is_super_admin field in response")
	}
}

func TestUsersRouteRejectsMissingToken(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
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

func TestUsersRouteAllowsSelfPatch(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/self", strings.NewReader(`{"given_name":"Ada"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var user struct {
		ID        string `json:"id"`
		GivenName string `json:"given_name"`
		Email     string `json:"email"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode user: %v", err)
	}

	if user.ID != "local-1" || user.GivenName != "Ada" {
		t.Fatalf("unexpected user response: %+v", user)
	}
}

func TestUsersRouteRejectsSelfPatchAdminFields(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/self", strings.NewReader(`{"status":"ACTIVE"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
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

	req := httptest.NewRequest(http.MethodGet, "/v1/users/self", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var user struct {
		ID          string `json:"id"`
		Memberships []struct {
			Group struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Href string `json:"href"`
			} `json:"group"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"memberships"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode user: %v", err)
	}

	if user.ID != "local-1" || len(user.Memberships) != 2 {
		t.Fatalf("unexpected user response: %+v", user)
	}

	if user.Memberships[0].Group.ID != "group-1" || user.Memberships[0].Role != "VIEWER" || user.Memberships[0].Status != "ACTIVE" {
		t.Fatalf("unexpected first membership: %+v", user.Memberships[0])
	}
}

func TestUsersRouteAllowsSuperAdminLookup(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/users/local-2", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var user struct {
		ID          string `json:"id"`
		Memberships []struct {
			Group struct {
				ID string `json:"id"`
			} `json:"group"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"memberships"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode user: %v", err)
	}

	if user.ID != "local-2" || len(user.Memberships) != 1 || user.Memberships[0].Group.ID != "group-1" || user.Memberships[0].Role != "ADMIN" {
		t.Fatalf("unexpected user response: %+v", user)
	}
}

func TestUsersRouteReturnsNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/users/local-404", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
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

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/local-2", strings.NewReader(`{"given_name":"Ada"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGroupsRouteCreatesGroup(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups", strings.NewReader(`{"slug":"research","name":"Research","description":"Research group","is_system":false}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var group struct {
		ID          string `json:"id"`
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IsSystem    bool   `json:"is_system"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&group); err != nil {
		t.Fatalf("decode group: %v", err)
	}

	if group.ID != "group-1" || group.Slug != "research" || group.Name != "Research" || group.Description != "Research group" || group.IsSystem {
		t.Fatalf("unexpected group response: %+v", group)
	}
}

func TestGroupsRouteListsGroupsWithMembership(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var groups []struct {
		ID         string `json:"id"`
		Membership *struct {
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"membership"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&groups); err != nil {
		t.Fatalf("decode groups: %v", err)
	}

	if len(groups) != 3 || groups[0].Membership == nil || groups[1].Membership == nil {
		t.Fatalf("unexpected groups response: %+v", groups)
	}
	if groups[0].Membership.Role != "VIEWER" || groups[1].Membership.Status != "PENDING" || groups[2].Membership != nil {
		t.Fatalf("unexpected memberships: %+v %+v", groups[0].Membership, groups[1].Membership)
	}
}

func TestGroupsRouteListsGroupsWithoutMembership(t *testing.T) {
	t.Parallel()

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
				IsSuperAdmin   bool   `json:"isSuperAdmin"`
				Status         string `json:"status"`
			}{})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/by-auth/local-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "local-1",
				"authIdentifier": "local-1",
				"email":          "user@example.org",
				"givenName":      "User",
				"familyName":     "Example",
				"isSuperAdmin":   false,
				"status":         "ACTIVE",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/local-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "local-1",
				"authIdentifier": "local-1",
				"email":          "user@example.org",
				"givenName":      "User",
				"familyName":     "Example",
				"isSuperAdmin":   false,
				"status":         "ACTIVE",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups":
			_ = json.NewEncoder(w).Encode([]struct {
				ID          string `json:"id"`
				Slug        string `json:"slug"`
				Name        string `json:"name"`
				Description string `json:"description"`
				IsSystem    bool   `json:"isSystem"`
			}{
				{ID: "group-1", Slug: "research", Name: "Research", Description: "Research group", IsSystem: false},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			_ = json.NewEncoder(w).Encode([]struct {
				UserId string `json:"userId"`
				Role   string `json:"role"`
				Status string `json:"status"`
			}{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(usergroupSrv.Close)

	client, err := clients.NewUsergroup(usergroupSrv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":            true,
			"eduPersonUniqueId": "local-1",
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

	router := NewRouter(nil, nil, nil, authService, usergroup.NewService(client), "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var groups []struct {
		ID         string    `json:"id"`
		Membership *struct{} `json:"membership"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&groups); err != nil {
		t.Fatalf("decode groups: %v", err)
	}

	if len(groups) != 1 || groups[0].Membership != nil {
		t.Fatalf("unexpected groups response: %+v", groups)
	}
}

func TestGroupsRouteGetsGroupWithMembership(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/group-1", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var group struct {
		ID         string `json:"id"`
		Membership *struct {
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"membership"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&group); err != nil {
		t.Fatalf("decode group: %v", err)
	}

	if group.ID != "group-1" || group.Membership == nil || group.Membership.Role != "VIEWER" {
		t.Fatalf("unexpected group response: %+v", group)
	}
}

func TestGroupsRouteGetsGroupWithoutMembership(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/group-3", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var group struct {
		ID         string    `json:"id"`
		Membership *struct{} `json:"membership"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&group); err != nil {
		t.Fatalf("decode group: %v", err)
	}

	if group.ID != "group-3" || group.Membership != nil {
		t.Fatalf("unexpected group response: %+v", group)
	}
}

func TestGroupsRouteGetsGroupNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/group-404", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGroupsRouteGetsGroupBadRequest(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/bad-group-id", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGroupsRouteRejectsMissingTokenOnDetail(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/group-1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGroupsRouteRejectsMissingToken(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups", strings.NewReader(`{"slug":"research","name":"Research"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGroupsRouteRejectsNonAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups", strings.NewReader(`{"slug":"research","name":"Research"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGroupsRouteRejectsBadPayload(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups", strings.NewReader(`{"slug":""}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGroupsRouteRejectsCreateConflict(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups", strings.NewReader(`{"slug":"duplicate","name":"Duplicate"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestGroupsRoutePatchesGroupBySuperAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/group-1", strings.NewReader(`{"name":"Research Team","description":"Updated","is_system":false}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var group struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&group); err != nil {
		t.Fatalf("decode group: %v", err)
	}

	if group.ID != "group-1" || group.Name != "Research Team" || group.Description != "Updated" {
		t.Fatalf("unexpected group response: %+v", group)
	}
}

func TestGroupsRoutePatchesGroupByActiveAdmin(t *testing.T) {
	t.Parallel()

	usergroupSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/by-auth/user-456":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "local-2",
				"authIdentifier": "user-456",
				"email":          "admin@example.org",
				"givenName":      "Admin",
				"familyName":     "Example",
				"isSuperAdmin":   false,
				"status":         "ACTIVE",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/users/local-2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "local-2",
				"authIdentifier": "user-456",
				"email":          "admin@example.org",
				"givenName":      "Admin",
				"familyName":     "Example",
				"isSuperAdmin":   false,
				"status":         "ACTIVE",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/groups/group-1/memberships":
			_ = json.NewEncoder(w).Encode([]struct {
				UserId string `json:"userId"`
				Role   string `json:"role"`
				Status string `json:"status"`
			}{
				{UserId: "local-2", Role: "ADMIN", Status: "ACTIVE"},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/groups/group-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "group-1",
				"slug":        "research",
				"name":        "Research Admin",
				"description": "Updated",
				"isSystem":    false,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(usergroupSrv.Close)

	client, err := clients.NewUsergroup(usergroupSrv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new usergroup client: %v", err)
	}
	usergroupSvc := usergroup.NewService(client)

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":            true,
			"eduPersonUniqueId": "user-456",
			"email":             "admin@example.org",
			"firstname":         "Admin",
			"lastName":          "Example",
		})
	}))
	t.Cleanup(authSrv.Close)

	authSvc, err := auth.NewService(authSrv.URL, config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	router := NewRouter(nil, nil, nil, authSvc, usergroupSvc, "")
	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/group-1", strings.NewReader(`{"name":"Research Admin","description":"Updated"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGroupsRouteRejectsPatchForNonAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/group-2", strings.NewReader(`{"name":"Nope"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGroupsRouteRejectsPatchNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/group-404", strings.NewReader(`{"name":"Missing"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGroupsRouteRejectsPatchBadRequest(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/bad-group-id", strings.NewReader(`{"name":"Invalid"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGroupsRouteRejectsPatchConflict(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/group-1", strings.NewReader(`{"slug":"duplicate"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestGroupsRouteDeletesGroup(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-1", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestGroupsRouteRejectsDeleteMissingToken(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGroupsRouteRejectsDeleteNonAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-1", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGroupsRouteRejectsDeleteNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-404", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGroupsRouteRejectsDeleteBadRequest(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/bad-group-id", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGroupsRouteListsMemberships(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/group-1/memberships", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var memberships []struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"user"`
		Role   string `json:"role"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&memberships); err != nil {
		t.Fatalf("decode memberships: %v", err)
	}

	if len(memberships) != 2 {
		t.Fatalf("unexpected memberships len: %d", len(memberships))
	}
	if memberships[0].User.ID != "local-1" || memberships[0].User.Name != "Alice Rossi" || memberships[0].User.Href != "v1/users/local-1" || memberships[0].Role != "VIEWER" || memberships[0].Status != "ACTIVE" {
		t.Fatalf("unexpected membership[0]: %+v", memberships[0])
	}
	if memberships[1].User.ID != "local-2" || memberships[1].User.Name != "Bob Meyer" || memberships[1].User.Href != "v1/users/local-2" || memberships[1].Role != "ADMIN" || memberships[1].Status != "ACTIVE" {
		t.Fatalf("unexpected membership[1]: %+v", memberships[1])
	}
}

func TestGroupsRouteRejectsMembershipListForbidden(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/group-1/memberships", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGroupsRouteRejectsMembershipListNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/groups/group-404/memberships", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

type groupMembershipResponse struct {
	UserID  string `json:"user_id"`
	GroupID string `json:"group_id"`
	Role    string `json:"role"`
	Status  string `json:"status"`
}

func TestGroupsRouteMembershipMutations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		authStack  [3]bool
		method     string
		path       string
		body       string
		wantStatus int
		wantRole   string
		wantState  string
	}{
		{
			name:       "request membership",
			authStack:  [3]bool{true, false, false},
			method:     http.MethodPost,
			path:       "/v1/groups/group-1/memberships",
			body:       `{"role":"VIEWER"}`,
			wantStatus: http.StatusCreated,
			wantRole:   "VIEWER",
			wantState:  "PENDING",
		},
		{
			name:       "request membership as super admin",
			authStack:  [3]bool{true, false, true},
			method:     http.MethodPost,
			path:       "/v1/groups/group-1/memberships",
			body:       `{"role":"VIEWER"}`,
			wantStatus: http.StatusCreated,
			wantRole:   "VIEWER",
			wantState:  "ACTIVE",
		},
		{
			name:       "patch membership role only",
			authStack:  [3]bool{true, false, true},
			method:     http.MethodPatch,
			path:       "/v1/groups/group-1/memberships/local-1",
			body:       `{"role":"REVIEWER"}`,
			wantStatus: http.StatusOK,
			wantRole:   "REVIEWER",
			wantState:  "ACTIVE",
		},
		{
			name:       "patch membership status only",
			authStack:  [3]bool{true, false, true},
			method:     http.MethodPatch,
			path:       "/v1/groups/group-1/memberships/local-1",
			body:       `{"status":"REVOKED"}`,
			wantStatus: http.StatusOK,
			wantRole:   "VIEWER",
			wantState:  "REVOKED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authService, provisioner := newTestAuthStack(t, tc.authStack[0], tc.authStack[1], tc.authStack[2])
			router := NewRouter(nil, nil, nil, authService, provisioner, "")

			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer token-123")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}

			var membership groupMembershipResponse
			if err := json.NewDecoder(rec.Body).Decode(&membership); err != nil {
				t.Fatalf("decode membership: %v", err)
			}

			if membership.UserID != "local-1" || membership.GroupID != "group-1" || membership.Role != tc.wantRole || membership.Status != tc.wantState {
				t.Fatalf("unexpected membership response: %+v", membership)
			}
		})
	}
}

func TestGroupsRouteRejectsMembershipPatchForbidden(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/group-1/memberships/local-1", strings.NewReader(`{"status":"REVOKED"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGroupsRouteRejectsMembershipPatchNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPatch, "/v1/groups/group-404/memberships/local-404", strings.NewReader(`{"status":"REVOKED"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGroupsRouteDeletesMembershipSelf(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-1/memberships/local-1", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestGroupsRouteDeletesMembershipAsAdmin(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-1/memberships/local-2", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestGroupsRouteRejectsMembershipDeleteForbidden(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-1/memberships/local-2", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGroupsRouteRejectsMembershipDeleteNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, true)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/groups/group-404/memberships/local-404", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGroupsRouteRejectsMembershipMissingToken(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups/group-1/memberships", strings.NewReader(`{"role":"VIEWER"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGroupsRouteRejectsMembershipBadRole(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups/group-1/memberships", strings.NewReader(`{"role":"NOPE"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGroupsRouteRejectsMembershipNotFound(t *testing.T) {
	t.Parallel()

	authService, provisioner := newTestAuthStack(t, true, false, false)
	router := NewRouter(nil, nil, nil, authService, provisioner, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/groups/group-404/memberships", strings.NewReader(`{"role":"VIEWER"}`))
	req.Header.Set("Authorization", "Bearer token-123")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
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
