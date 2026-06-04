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

func TestParseSearchParamsRejectsInvalidPaginationAndDates(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
		wantErr  string
	}{
		{
			name:     "page size below minimum",
			rawQuery: "pageSize=0",
			wantErr:  `parse page size: pageSize must be >= 1`,
		},
		{
			name:     "page size above maximum",
			rawQuery: "pageSize=101",
			wantErr:  `parse page size: pageSize must be <= 100`,
		},
		{
			name:     "page number below minimum",
			rawQuery: "pageNumber=0",
			wantErr:  `parse page number: pageNumber must be >= 1`,
		},
		{
			name:     "bad date from",
			rawQuery: "dateFrom=not-a-date",
			wantErr:  `parse dateFrom: parse dateFrom: parsing time "not-a-date" as "2006-01-02T15:04:05Z07:00": cannot parse "not-a-date" as "2006"`,
		},
		{
			name:     "inverted date range",
			rawQuery: "dateFrom=2025-01-02T00:00:00Z&dateTo=2025-01-01T00:00:00Z",
			wantErr:  `validate date range: dateTo must be >= dateFrom`,
		},
		{
			name:     "invalid versioning status",
			rawQuery: "versioningStatus=bogus",
			wantErr:  `parse versioning status filter: parse versioningStatus: invalid value 'bogus' for VersioningStatus`,
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
