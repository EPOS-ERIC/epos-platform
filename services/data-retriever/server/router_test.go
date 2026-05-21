package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
)

type fakeRepo struct{}

func (f fakeRepo) Create(_ context.Context, _ domain.ValidatedRequest, _ retriever.Result) (repository.RetrievalRecord, error) {
	return repository.RetrievalRecord{ID: 1, SourceID: "svc", Method: "GET", CreatedAt: time.Now().UTC()}, nil
}
func (f fakeRepo) GetByID(_ context.Context, _ int64) (repository.RetrievalRecord, error) {
	return repository.RetrievalRecord{ID: 1}, nil
}
func (f fakeRepo) Search(_ context.Context, _ string, _ string, _ string, _ int, _ int) ([]repository.SearchResult, error) {
	return []repository.SearchResult{}, nil
}

type fakeClient struct{}

func (f fakeClient) Execute(_ context.Context, _ domain.ValidatedRequest) (retriever.Result, error) {
	return retriever.Result{StatusCode: 200, ContentType: "application/json", Payload: []byte("{}"), PayloadHash: "x", ResponseDate: time.Now().UTC()}, nil
}

func TestOpenAPIEndpoint(t *testing.T) {
	router := NewRouter(fakeRepo{}, fakeClient{})
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestRetrieveRejectsDeleteBody(t *testing.T) {
	router := NewRouter(fakeRepo{}, fakeClient{})
	body := `{"source_id":"svc","method":"DELETE","url_template":"https://example.org/{id}","path_params":{"id":"1"},"body":{"content_type":"text/plain","raw":"x"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/retrieve", strings.NewReader(body))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] == "" {
		t.Fatalf("expected error message")
	}
}
