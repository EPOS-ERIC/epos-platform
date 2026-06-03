package handlers

import (
	"net/http/httptest"
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
			wantErr:  `Unsupported search type: concept`,
		},
		{
			name:     "mixed supported and unsupported types are rejected",
			rawQuery: "type=dataset,distribution",
			wantErr:  `Unsupported search type: distribution`,
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

func TestParseSearchParamsRejectsInvalidPaginationAndDates(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
		wantErr  string
	}{
		{
			name:     "page size below minimum",
			rawQuery: "pageSize=0",
			wantErr:  `pageSize must be between 1 and 100`,
		},
		{
			name:     "page size above maximum",
			rawQuery: "pageSize=101",
			wantErr:  `pageSize must be between 1 and 100`,
		},
		{
			name:     "page number below minimum",
			rawQuery: "pageNumber=0",
			wantErr:  `pageNumber must be >= 1`,
		},
		{
			name:     "bad date from",
			rawQuery: "dateFrom=not-a-date",
			wantErr:  `dateFrom must be RFC3339`,
		},
		{
			name:     "inverted date range",
			rawQuery: "dateFrom=2025-01-02T00:00:00Z&dateTo=2025-01-01T00:00:00Z",
			wantErr:  `dateTo must be greater than or equal to dateFrom`,
		},
		{
			name:     "invalid versioning status",
			rawQuery: "versioningStatus=bogus",
			wantErr:  `versioningStatus must be one of DRAFT, SUBMITTED, PUBLISHED, ARCHIVED`,
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
