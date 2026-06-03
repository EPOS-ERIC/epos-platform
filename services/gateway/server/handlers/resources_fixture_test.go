package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResourceHandlersUseFixtureResponses(t *testing.T) {
	t.Parallel()

	handler := newResourcesHandler(t, resourceFixtureRoutes())

	tests := []struct {
		name   string
		id     string
		call   func(*Resources, http.ResponseWriter, *http.Request)
		want   string
		target string
	}{
		{
			name:   "dataset",
			id:     "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Dataset/v1.0",
			call:   (*Resources).GetDatasetByID,
			want:   "INTERMAGNET Geomagnetic Observatory Data",
			target: "/v1/dataset/id",
		},
		{
			name:   "distribution",
			id:     "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Distribution/v1.0",
			call:   (*Resources).GetDistributionByID,
			want:   "Data cadence",
			target: "/v1/distribution/id",
		},
		{
			name:   "webservice",
			id:     "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Webservice/v1.0",
			call:   (*Resources).GetWebserviceByID,
			want:   "Web service for INTERMAGNET data",
			target: "/v1/webservice/id",
		},
		{
			name:   "softwaresourcecode",
			id:     "http://www.epos-eu.org/epos-dcat-ap/software/Dataset/001/sgg/SSC/v1.0",
			call:   (*Resources).GetSoftwareSourceCodeByID,
			want:   "Analyze earthquakes, permanent displacement and geology",
			target: "/v1/softwaresourcecode/id",
		},
		{
			name:   "organization",
			id:     "http://viaf.org/viaf/136554353//v1.0",
			call:   (*Resources).GetOrganizationByID,
			want:   "British Geological Survey",
			target: "/v1/organization/id",
		},
		{
			name:   "concept",
			id:     "category:geomagneticdata/v1.0",
			call:   (*Resources).GetConceptByID,
			want:   "Geomagnetic data",
			target: "/v1/concept/id",
		},
		{
			name:   "conceptscheme",
			id:     "category:geomagneticobservatis/v1.0",
			call:   (*Resources).GetConceptSchemeByID,
			want:   "Geomagnetic Observations",
			target: "/v1/conceptscheme/id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithURLParam(http.MethodGet, tt.target, "id", tt.id)
			rec := httptest.NewRecorder()

			tt.call(handler, rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Fatalf("body = %s", rec.Body.String())
			}
		})
	}
}

func TestResourceHandlersReturnNotImplementedForSoftwareEndpoints(t *testing.T) {
	t.Parallel()

	handler := newResourcesHandler(t, nil)

	tests := []struct {
		name   string
		call   func(*Resources, http.ResponseWriter, *http.Request)
		target string
	}{
		{name: "softwareapplication", call: (*Resources).GetSoftwareApplicationByID, target: "/v1/softwareapplication/id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithURLParam(http.MethodGet, tt.target, "id", "example")
			rec := httptest.NewRecorder()

			tt.call(handler, rec, req)

			if rec.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}
