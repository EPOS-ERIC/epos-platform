package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func TestParseSearchParamsRejectsUnsupportedSearchTypes(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
		wantErr  string
	}{
		{
			name:     "concept type is rejected",
			rawQuery: "type=concept",
			wantErr:  `parse type filter: unsupported search type "concept"; allowed values: [dataset organization softwareapplication softwaresourcecode webservice]`,
		},
		{
			name:     "mixed supported and unsupported types are rejected",
			rawQuery: "type=dataset,distribution",
			wantErr:  `parse type filter: unsupported search type "distribution"; allowed values: [dataset organization softwareapplication softwaresourcecode webservice]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/search?"+tt.rawQuery, nil)

			_, err := parseSearchParams(req)
			if err == nil {
				t.Fatal("expected error")
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("unexpected error: got %q want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseSearchParamsAcceptsSupportedSearchTypes(t *testing.T) {
	req := httptest.NewRequest("GET", "/search?type=dataset,webservice", nil)

	params, err := parseSearchParams(req)
	if err != nil {
		t.Fatalf("parse search params: %v", err)
	}

	want := []api.ResourceType{api.DatasetType, api.WebserviceType}
	if len(params.Types) != len(want) {
		t.Fatalf("unexpected types length: got %d want %d", len(params.Types), len(want))
	}

	for i := range want {
		if params.Types[i] != want[i] {
			t.Fatalf("unexpected type at index %d: got %q want %q", i, params.Types[i], want[i])
		}
	}
}

func TestGetConceptTreeUsesFixtureResponses(t *testing.T) {
	t.Parallel()

	handler := newDiscoveryHandler(t, conceptTreeFixtureRoutes())
	req := httptest.NewRequest(http.MethodGet, "/v1/concept", nil)
	rec := httptest.NewRecorder()

	handler.GetConceptTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Anthropogenic Hazards") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Dataset") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
