package handlers

import (
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
			wantErr:  "unsupported search type",
		},
		{
			name:     "mixed supported and unsupported types are rejected",
			rawQuery: "type=dataset,distribution",
			wantErr:  "unsupported search type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/search?"+tt.rawQuery, nil)

			_, err := parseSearchParams(req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("unexpected error: %q", err.Error())
			}
		})
	}
}

func TestParseSearchParamsRejectsInvalidPaginationAndDates(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
		wantErr  string
	}{
		{
			name:     "page size below minimum",
			rawQuery: "pageSize=0",
			wantErr:  "page size",
		},
		{
			name:     "page size above maximum",
			rawQuery: "pageSize=101",
			wantErr:  "page size",
		},
		{
			name:     "page number below minimum",
			rawQuery: "pageNumber=0",
			wantErr:  "page number",
		},
		{
			name:     "bad date from",
			rawQuery: "dateFrom=not-a-date",
			wantErr:  "date from",
		},
		{
			name:     "inverted date range",
			rawQuery: "dateFrom=2025-01-02T00:00:00Z&dateTo=2025-01-01T00:00:00Z",
			wantErr:  "date to must be greater than or equal to date from",
		},
		{
			name:     "invalid versioning status",
			rawQuery: "versioningStatus=bogus",
			wantErr:  "versioning status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/search?"+tt.rawQuery, nil)

			_, err := parseSearchParams(req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("unexpected error: %q", err.Error())
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
