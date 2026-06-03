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
			name:   "softwareapplication",
			id:     "https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0",
			call:   (*Resources).GetSoftwareApplicationByID,
			want:   "Catalog to XLS converter",
			target: "/v1/softwareapplication/id",
		},
		{
			name:   "softwaresourcecode",
			id:     "https://doi.org/10.5281/zenodo.17598403",
			call:   (*Resources).GetSoftwareSourceCodeByID,
			want:   "Discovery, automation/scripting and integration of EPOS data and services. Basic workflows in JupyterLab.",
			target: "/v1/softwaresourcecode/id",
		},
		{
			name:   "organization",
			id:     "ROR:048a87296/v1.0",
			call:   (*Resources).GetOrganizationByID,
			want:   "Uppsala University",
			target: "/v1/organization/id",
		},
		{
			name:   "concept",
			id:     "category:software_jup_conc_multi/v1.0",
			call:   (*Resources).GetConceptByID,
			want:   "Multidisciplinary software",
			target: "/v1/concept/id",
		},
		{
			name:   "conceptscheme",
			id:     "category:software_jup/v1.0",
			call:   (*Resources).GetConceptSchemeByID,
			want:   "Jupyter Notebooks",
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
