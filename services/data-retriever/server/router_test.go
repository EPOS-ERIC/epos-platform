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

type fakeClientBinary struct{}

func (f fakeClientBinary) Execute(_ context.Context, _ domain.ValidatedRequest) (retriever.Result, error) {
	return retriever.Result{StatusCode: 200, ContentType: "application/octet-stream", Payload: []byte{0x1, 0x2}, PayloadHash: "x", IsBinary: true, ResponseDate: time.Now().UTC()}, nil
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

func TestRetrieveReturnsRawPayload(t *testing.T) {
	router := NewRouter(fakeRepo{}, fakeClient{})
	body := `{"source_id":"svc","method":"GET","url_template":"https://example.org/{id}","path_params":{"id":"1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/retrieve", strings.NewReader(body))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "{}" {
		t.Fatalf("expected raw payload body, got %q", rr.Body.String())
	}
	if rr.Header().Get("X-Retrieval-Id") == "" {
		t.Fatal("expected X-Retrieval-Id header")
	}
}

func TestRetrieveRedirectsForBinaryPayload(t *testing.T) {
	router := NewRouter(fakeRepo{}, fakeClientBinary{})
	body := `{"source_id":"svc","method":"GET","url_template":"https://example.org/files/{id}","path_params":{"id":"1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/retrieve", strings.NewReader(body))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "https://example.org/files/1" {
		t.Fatalf("unexpected redirect location: %s", rr.Header().Get("Location"))
	}
}
