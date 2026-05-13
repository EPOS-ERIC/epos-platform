package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func TestParseSearchParamsDefaults(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/v1/search", nil)

	params, err := parseSearchParams(req)
	if err != nil {
		t.Fatalf("parseSearchParams() error = %v", err)
	}

	if params.PageSize != defaultPageSize {
		t.Fatalf("PageSize = %d, want %d", params.PageSize, defaultPageSize)
	}

	if params.PageNumber != defaultPageNumber {
		t.Fatalf("PageNumber = %d, want %d", params.PageNumber, defaultPageNumber)
	}

	if len(params.VersioningStatus) != 1 || params.VersioningStatus[0] != api.PublishedStatus {
		t.Fatalf("VersioningStatus = %v, want [%s]", params.VersioningStatus, api.PublishedStatus)
	}
}

func TestParseSearchParamsParsesCSVAndDates(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/v1/search?q=test&pageSize=25&pageNumber=2&type=dataset,distribution&dateFrom=2020-01-01T00:00:00Z&dateTo=2020-01-02T00:00:00Z&serviceProvider=INGV,UNINA&versioningStatus=PUBLISHED,DRAFT&keywords=seismology,stations", nil)

	params, err := parseSearchParams(req)
	if err != nil {
		t.Fatalf("parseSearchParams() error = %v", err)
	}

	if params.Query != "test" {
		t.Fatalf("Query = %q, want test", params.Query)
	}

	if len(params.Types) != 2 || params.Types[0] != api.DatasetType || params.Types[1] != api.DistributionType {
		t.Fatalf("Types = %v", params.Types)
	}

	if len(params.ServiceProviders) != 2 || params.ServiceProviders[0] != "INGV" || params.ServiceProviders[1] != "UNINA" {
		t.Fatalf("ServiceProviders = %v", params.ServiceProviders)
	}

	if len(params.Keywords) != 2 || params.Keywords[0] != "seismology" || params.Keywords[1] != "stations" {
		t.Fatalf("Keywords = %v", params.Keywords)
	}

	if got, want := params.DateFrom.Format(time.RFC3339), "2020-01-01T00:00:00Z"; got != want {
		t.Fatalf("DateFrom = %s, want %s", got, want)
	}

	if got, want := params.DateTo.Format(time.RFC3339), "2020-01-02T00:00:00Z"; got != want {
		t.Fatalf("DateTo = %s, want %s", got, want)
	}
}

func TestParseSearchParamsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{name: "page size below minimum", url: "/v1/search?pageSize=0"},
		{name: "invalid type", url: "/v1/search?type=bad"},
		{name: "invalid date", url: "/v1/search?dateFrom=bad"},
		{name: "date range reversed", url: "/v1/search?dateFrom=2020-01-02T00:00:00Z&dateTo=2020-01-01T00:00:00Z"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)

			if _, err := parseSearchParams(req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
