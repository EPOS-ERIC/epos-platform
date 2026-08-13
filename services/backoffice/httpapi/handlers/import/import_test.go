package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataimport"
	"github.com/google/uuid"
)

type fakeImportService struct {
	actor uuid.UUID
	body  string
}

func (s *fakeImportService) Import(_ context.Context, actor uuid.UUID, input io.Reader) (metadataimport.Result, error) {
	s.actor = actor
	content, err := io.ReadAll(input)
	if err != nil {
		return metadataimport.Result{}, fmt.Errorf("read import: %w", err)
	}
	s.body = string(content)
	return metadataimport.Result{Datasets: 1}, nil
}

func TestTurtleImportRequiresActorAndContentType(t *testing.T) {
	handler := New(&fakeImportService{})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/turtle", strings.NewReader("metadata"))
	response := httptest.NewRecorder()
	handler.Turtle(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing actor status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/imports/turtle", strings.NewReader("metadata"))
	request.Header.Set(transport.ActorHeader, uuid.NewString())
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.Turtle(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("invalid content type status = %d", response.Code)
	}
}

func TestTurtleImportPassesRawBody(t *testing.T) {
	service := &fakeImportService{}
	handler := New(service)
	actorID := uuid.New()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/turtle", strings.NewReader("@prefix dcat: <http://www.w3.org/ns/dcat#> ."))
	request.Header.Set(transport.ActorHeader, actorID.String())
	request.Header.Set("Content-Type", "text/turtle; charset=utf-8")
	response := httptest.NewRecorder()

	handler.Turtle(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.actor != actorID || !strings.HasPrefix(service.body, "@prefix") {
		t.Fatalf("actor = %s, body = %q", service.actor, service.body)
	}
	var body resultResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Datasets != 1 || body.Warnings == nil {
		t.Fatalf("response = %#v", body)
	}
}
