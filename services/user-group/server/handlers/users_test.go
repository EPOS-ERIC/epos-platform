package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

func TestUsersCRUD(t *testing.T) {
	repo := repository.NewUserMemoryRepository()
	h := NewUsersHandler(repo)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/users", h.Create)
	mux.HandleFunc("GET /api/v2/users", h.List)
	mux.HandleFunc("GET /api/v2/users/by-auth/{authIdentifier}", h.GetByAuthIdentifier)
	mux.HandleFunc("GET /api/v2/users/{userId}", h.GetByID)
	mux.HandleFunc("PUT /api/v2/users/{userId}", h.Replace)
	mux.HandleFunc("PATCH /api/v2/users/{userId}", h.Patch)
	mux.HandleFunc("DELETE /api/v2/users/{userId}", h.Delete)

	createReq := `{"authIdentifier":"abc@aaai.epos-eu.org","email":"a@b.com","givenName":"A","familyName":"B"}`
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/v2/users", bytes.NewBufferString(createReq)))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRec.Code)
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode created user: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected non-empty user id")
	}

	getByAuthRec := httptest.NewRecorder()
	mux.ServeHTTP(getByAuthRec, httptest.NewRequest(http.MethodGet, "/api/v2/users/by-auth/abc@aaai.epos-eu.org", nil))
	if getByAuthRec.Code != http.StatusOK {
		t.Fatalf("expected by-auth status %d, got %d", http.StatusOK, getByAuthRec.Code)
	}

	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/v2/users", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d", http.StatusOK, listRec.Code)
	}

	replaceReq := `{"authIdentifier":"abc2@aaai.epos-eu.org","email":"x@y.com","givenName":"X","familyName":"Y","status":"INACTIVE"}`
	replaceRec := httptest.NewRecorder()
	mux.ServeHTTP(replaceRec, httptest.NewRequest(http.MethodPut, "/api/v2/users/"+id, bytes.NewBufferString(replaceReq)))
	if replaceRec.Code != http.StatusOK {
		t.Fatalf("expected replace status %d, got %d", http.StatusOK, replaceRec.Code)
	}

	patchReq := `{"status":"ACTIVE"}`
	patchRec := httptest.NewRecorder()
	mux.ServeHTTP(patchRec, httptest.NewRequest(http.MethodPatch, "/api/v2/users/"+id, bytes.NewBufferString(patchReq)))
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected patch status %d, got %d", http.StatusOK, patchRec.Code)
	}

	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, httptest.NewRequest(http.MethodDelete, "/api/v2/users/"+id, nil))
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected delete status %d, got %d", http.StatusNoContent, deleteRec.Code)
	}
}
