package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func TestSearchMapsPrefixedDatasetAndWebservice(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/access/search":
			writeTestJSON(t, w, map[string]any{
				"total":      2,
				"totalItems": 2,
				"page":       1,
				"totalPages": 1,
				"items": []map[string]any{
					{
						"type": "dataset",
						"id":   "dataset-1",
						"href": "http://example.test/dataset-1",
						"fields": map[string]any{
							"dct:title": []any{"Dataset One"},
							"dct:publisher": []any{
								map[string]any{"id": "org-1", "type": "organization"},
							},
							"dcat:theme": []any{
								map[string]any{"id": "concept-1", "type": "concept"},
							},
						},
					},
					{
						"type": "webservice",
						"id":   "webservice-1",
						"href": "http://example.test/webservice-1",
						"fields": map[string]any{
							"schema:name":     "Webservice One",
							"schema:provider": map[string]any{"id": "org-1", "type": "organization"},
						},
					},
				},
			})
		case "/api/v2/access/organization/org-1":
			writeTestJSON(t, w, map[string]any{
				"id":    "org-1",
				"shape": "organization",
				"href":  "http://example.test/org-1",
				"properties": map[string]any{
					"schema:legalName": "Organization One",
				},
			})
		case "/api/v2/access/concept/concept-1":
			writeTestJSON(t, w, map[string]any{
				"id":    "concept-1",
				"shape": "concept",
				"href":  "http://example.test/concept-1",
				"properties": map[string]any{
					"skos:prefLabel": "Concept One",
				},
			})
		case "/api/v2/access/dataset/dataset-1":
			writeTestJSON(t, w, map[string]any{
				"id":    "dataset-1",
				"shape": "dataset",
				"href":  "http://example.test/dataset-1",
				"properties": map[string]any{
					"dcat:distribution": []any{
						map[string]any{"id": "distribution-1", "type": "distribution", "name": "Distribution One"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService(newTestMetadataClient(server.URL), 8)

	resp, err := service.Search(context.Background(), SearchParams{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("len(resp.Items) = %d", len(resp.Items))
	}

	dataset, ok := resp.Items[0].(api.DatasetItem)
	if !ok {
		t.Fatalf("resp.Items[0] type = %T", resp.Items[0])
	}
	if dataset.Name != "Dataset One" {
		t.Fatalf("dataset.Name = %q", dataset.Name)
	}
	if len(dataset.Providers) != 1 || dataset.Providers[0].Name != "Organization One" {
		t.Fatalf("dataset.Providers = %#v", dataset.Providers)
	}
	if len(dataset.Concepts) != 1 || dataset.Concepts[0].Name != "Concept One" {
		t.Fatalf("dataset.Concepts = %#v", dataset.Concepts)
	}
	if len(dataset.Distributions) != 1 || dataset.Distributions[0].ID != "distribution-1" {
		t.Fatalf("dataset.Distributions = %#v", dataset.Distributions)
	}

	webservice, ok := resp.Items[1].(api.WebServiceItem)
	if !ok {
		t.Fatalf("resp.Items[1] type = %T", resp.Items[1])
	}
	if webservice.Name != "Webservice One" {
		t.Fatalf("webservice.Name = %q", webservice.Name)
	}
	if webservice.Provider.Name != "Organization One" {
		t.Fatalf("webservice.Provider = %#v", webservice.Provider)
	}
	if len(webservice.Concepts) != 0 {
		t.Fatalf("webservice.Concepts = %#v", webservice.Concepts)
	}
}

func TestSearchMapsPrefixedSoftwareItems(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/access/search":
			writeTestJSON(t, w, map[string]any{
				"total":      2,
				"totalItems": 2,
				"page":       1,
				"totalPages": 1,
				"items": []map[string]any{
					{
						"type": "softwareapplication",
						"id":   "software-application-1",
						"href": "http://example.test/software-application-1",
						"fields": map[string]any{
							"schema:name":     "Software Application One",
							"schema:provider": []any{map[string]any{"id": "org-1", "type": "organization"}},
						},
					},
					{
						"type": "softwaresourcecode",
						"id":   "software-source-code-1",
						"href": "http://example.test/software-source-code-1",
						"fields": map[string]any{
							"schema:name":     "Software Source Code One",
							"schema:provider": []any{map[string]any{"id": "org-1", "type": "organization"}},
						},
					},
				},
			})
		case "/api/v2/access/organization/org-1":
			writeTestJSON(t, w, map[string]any{
				"id":    "org-1",
				"shape": "organization",
				"href":  "http://example.test/org-1",
				"properties": map[string]any{
					"schema:legalName": "Organization One",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService(newTestMetadataClient(server.URL), 8)

	resp, err := service.Search(context.Background(), SearchParams{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("len(resp.Items) = %d", len(resp.Items))
	}

	softwareApplication, ok := resp.Items[0].(api.SoftwareApplicationItem)
	if !ok {
		t.Fatalf("resp.Items[0] type = %T", resp.Items[0])
	}
	if softwareApplication.Name != "Software Application One" {
		t.Fatalf("softwareApplication.Name = %q", softwareApplication.Name)
	}
	if len(softwareApplication.Provider) != 1 || softwareApplication.Provider[0].Name != "Organization One" {
		t.Fatalf("softwareApplication.Provider = %#v", softwareApplication.Provider)
	}

	softwareSourceCode, ok := resp.Items[1].(api.SoftwareSourceCodeItem)
	if !ok {
		t.Fatalf("resp.Items[1] type = %T", resp.Items[1])
	}
	if softwareSourceCode.Name != "Software Source Code One" {
		t.Fatalf("softwareSourceCode.Name = %q", softwareSourceCode.Name)
	}
	if len(softwareSourceCode.Provider) != 1 || softwareSourceCode.Provider[0].Name != "Organization One" {
		t.Fatalf("softwareSourceCode.Provider = %#v", softwareSourceCode.Provider)
	}
}
