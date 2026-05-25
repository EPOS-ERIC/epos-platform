package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

func TestMembershipAndFriendshipEndpoints(t *testing.T) {
	usersRepo := repository.NewUserMemoryRepository()
	groupsRepo := repository.NewGroupMemoryRepository()
	membershipsRepo := repository.NewMembershipMemoryRepository()
	friendshipsRepo := repository.NewFriendshipMemoryRepository()

	u1, _ := usersRepo.Create("a1", "", "", "", "ACTIVE")
	u2, _ := usersRepo.Create("a2", "", "", "", "ACTIVE")
	g, _ := groupsRepo.Create("g1", "Group 1", "", false)

	membershipsHandler := NewMembershipsHandler(usersRepo, groupsRepo, membershipsRepo)
	friendshipsHandler := NewFriendshipsHandler(usersRepo, friendshipsRepo)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v2/groups/{groupId}/memberships/{userId}", membershipsHandler.Upsert)
	mux.HandleFunc("GET /api/v2/groups/{groupId}/memberships", membershipsHandler.ListByGroup)
	mux.HandleFunc("DELETE /api/v2/groups/{groupId}/memberships/{userId}", membershipsHandler.Delete)
	mux.HandleFunc("POST /api/v2/users/{userId}/friends/{friendUserId}", friendshipsHandler.Create)
	mux.HandleFunc("DELETE /api/v2/users/{userId}/friends/{friendUserId}", friendshipsHandler.Delete)

	putBody := `{"role":"EDITOR","status":"ACTIVE"}`
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, httptest.NewRequest(http.MethodPut, "/api/v2/groups/"+g.ID+"/memberships/"+u1.ID, bytes.NewBufferString(putBody)))
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected membership upsert status %d, got %d", http.StatusOK, putRec.Code)
	}

	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/v2/groups/"+g.ID+"/memberships", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected membership list status %d, got %d", http.StatusOK, listRec.Code)
	}
	var memberships []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &memberships); err != nil {
		t.Fatalf("failed to decode memberships: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected one membership, got %d", len(memberships))
	}

	friendRec := httptest.NewRecorder()
	mux.ServeHTTP(friendRec, httptest.NewRequest(http.MethodPost, "/api/v2/users/"+u1.ID+"/friends/"+u2.ID, nil))
	if friendRec.Code != http.StatusCreated {
		t.Fatalf("expected friendship create status %d, got %d", http.StatusCreated, friendRec.Code)
	}

	unfriendRec := httptest.NewRecorder()
	mux.ServeHTTP(unfriendRec, httptest.NewRequest(http.MethodDelete, "/api/v2/users/"+u1.ID+"/friends/"+u2.ID, nil))
	if unfriendRec.Code != http.StatusNoContent {
		t.Fatalf("expected friendship delete status %d, got %d", http.StatusNoContent, unfriendRec.Code)
	}
}
