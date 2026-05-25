package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

func TestGroupsCRUD(t *testing.T) {
	repo := repository.NewGroupMemoryRepository()
	h := NewGroupsHandler(repo)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/groups", h.Create)
	mux.HandleFunc("GET /api/v2/groups", h.List)
	mux.HandleFunc("GET /api/v2/groups/{groupId}", h.GetByID)
	mux.HandleFunc("PUT /api/v2/groups/{groupId}", h.Replace)
	mux.HandleFunc("PATCH /api/v2/groups/{groupId}", h.Patch)
	mux.HandleFunc("DELETE /api/v2/groups/{groupId}", h.Delete)

	createReq := `{"slug":"seismology","name":"Seismology","description":"TCS"}`
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/v2/groups", bytes.NewBufferString(createReq)))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRec.Code)
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode created group: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected non-empty group id")
	}

	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/v2/groups", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d", http.StatusOK, listRec.Code)
	}

	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/api/v2/groups/"+id, nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d", http.StatusOK, getRec.Code)
	}

	replaceReq := `{"slug":"seismology-main","name":"Seismology Main","description":"updated","isSystem":false}`
	replaceRec := httptest.NewRecorder()
	mux.ServeHTTP(replaceRec, httptest.NewRequest(http.MethodPut, "/api/v2/groups/"+id, bytes.NewBufferString(replaceReq)))
	if replaceRec.Code != http.StatusOK {
		t.Fatalf("expected replace status %d, got %d", http.StatusOK, replaceRec.Code)
	}

	patchReq := `{"name":"Seismology Patched"}`
	patchRec := httptest.NewRecorder()
	mux.ServeHTTP(patchRec, httptest.NewRequest(http.MethodPatch, "/api/v2/groups/"+id, bytes.NewBufferString(patchReq)))
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected patch status %d, got %d", http.StatusOK, patchRec.Code)
	}

	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, httptest.NewRequest(http.MethodDelete, "/api/v2/groups/"+id, nil))
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected delete status %d, got %d", http.StatusNoContent, deleteRec.Code)
	}

	notFoundRec := httptest.NewRecorder()
	mux.ServeHTTP(notFoundRec, httptest.NewRequest(http.MethodGet, "/api/v2/groups/"+id, nil))
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("expected not found status %d, got %d", http.StatusNotFound, notFoundRec.Code)
	}
}
