package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMetadataReadGetRejectsNonUUID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/resources/dataset/source-iri", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("resourceType", "dataset")
	routeContext.URLParams.Add("id", "source-iri")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	rec := httptest.NewRecorder()

	NewMetadataRead(nil).Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
