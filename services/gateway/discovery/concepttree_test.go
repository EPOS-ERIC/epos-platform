package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestGetConceptTree(t *testing.T) {
	t.Parallel()

	var (
		mu          sync.Mutex
		conceptHits = map[string]int{}
		searchHits  int
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/access/search/conceptscheme":
			mu.Lock()
			searchHits++
			mu.Unlock()

			page := r.URL.Query().Get("page")
			if page == "2" {
				writeTestJSON(t, w, map[string]any{
					"total":      2,
					"totalItems": 2,
					"page":       2,
					"totalPages": 2,
					"items": []map[string]any{
						{"type": "conceptscheme", "id": "scheme-a", "href": "http://example.test/scheme-a"},
					},
				})
				return
			}

			writeTestJSON(t, w, map[string]any{
				"total":      2,
				"totalItems": 2,
				"page":       1,
				"totalPages": 2,
				"items": []map[string]any{
					{"type": "conceptscheme", "id": "scheme-b", "href": "http://example.test/scheme-b"},
				},
			})
		case "/api/v2/access/conceptscheme/scheme-a":
			writeTestJSON(t, w, map[string]any{
				"id":    "scheme-a",
				"shape": "conceptscheme",
				"href":  "http://example.test/scheme-a",
				"properties": map[string]any{
					"skos:prefLabel":         "Scheme A",
					"dct:title":              "Scheme A Title",
					"dct:description":        "Scheme A Description",
					"schema:orderItemNumber": "2",
					"skos:hasTopConcept": []map[string]any{
						{"id": "concept-a", "type": "concept"},
					},
				},
			})
		case "/api/v2/access/conceptscheme/scheme-b":
			writeTestJSON(t, w, map[string]any{
				"id":    "scheme-b",
				"shape": "conceptscheme",
				"href":  "http://example.test/scheme-b",
				"properties": map[string]any{
					"skos:prefLabel": "Scheme B",
					"dct:title":      "Scheme B Title",
					"skos:hasTopConcept": []map[string]any{
						{"id": "concept-b", "type": "concept"},
					},
				},
			})
		case "/api/v2/access/concept/concept-a":
			mu.Lock()
			conceptHits["concept-a"]++
			mu.Unlock()

			writeTestJSON(t, w, map[string]any{
				"id":    "concept-a",
				"shape": "concept",
				"href":  "http://example.test/concept-a",
				"properties": map[string]any{
					"skos:prefLabel":  "Concept A",
					"skos:definition": "Concept A Description",
					"skos:narrower": []map[string]any{
						{"id": "concept-a-child", "type": "concept"},
					},
				},
			})
		case "/api/v2/access/concept/concept-a-child":
			mu.Lock()
			conceptHits["concept-a-child"]++
			mu.Unlock()

			writeTestJSON(t, w, map[string]any{
				"id":    "concept-a-child",
				"shape": "concept",
				"href":  "http://example.test/concept-a-child",
				"properties": map[string]any{
					"skos:prefLabel":  "Concept A Child",
					"skos:definition": "Concept A Child Description",
				},
			})
		case "/api/v2/access/concept/concept-b":
			mu.Lock()
			conceptHits["concept-b"]++
			mu.Unlock()

			writeTestJSON(t, w, map[string]any{
				"id":    "concept-b",
				"shape": "concept",
				"href":  "http://example.test/concept-b",
				"properties": map[string]any{
					"skos:prefLabel": "Concept B",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService(newTestMetadataClient(server.URL), 8)

	tree, err := service.GetConceptTree(context.Background())
	if err != nil {
		t.Fatalf("GetConceptTree() error = %v", err)
	}

	if len(tree.Schemes) != 2 {
		t.Fatalf("len(tree.Schemes) = %d", len(tree.Schemes))
	}
	if tree.Schemes[0].ID != "scheme-a" {
		t.Fatalf("tree.Schemes[0].ID = %q", tree.Schemes[0].ID)
	}
	if tree.Schemes[0].Name != "Scheme A Title" {
		t.Fatalf("tree.Schemes[0].Name = %q", tree.Schemes[0].Name)
	}
	if len(tree.Schemes[0].TopConcepts) != 1 || tree.Schemes[0].TopConcepts[0].Href != "v1/resources/concept/concept-a" {
		t.Fatalf("tree.Schemes[0].TopConcepts = %#v", tree.Schemes[0].TopConcepts)
	}
	if len(tree.Schemes[0].Children) != 1 || tree.Schemes[0].Children[0].ID != "concept-a" {
		t.Fatalf("tree.Schemes[0].Children = %#v", tree.Schemes[0].Children)
	}
	if len(tree.Schemes[0].Children[0].Children) != 1 || tree.Schemes[0].Children[0].Children[0].ID != "concept-a-child" {
		t.Fatalf("tree.Schemes[0].Children[0].Children = %#v", tree.Schemes[0].Children[0].Children)
	}
	if tree.Schemes[1].ID != "scheme-b" {
		t.Fatalf("tree.Schemes[1].ID = %q", tree.Schemes[1].ID)
	}

	mu.Lock()
	defer mu.Unlock()
	if searchHits != 2 {
		t.Fatalf("searchHits = %d", searchHits)
	}
	if conceptHits["concept-a"] != 1 || conceptHits["concept-a-child"] != 1 || conceptHits["concept-b"] != 1 {
		t.Fatalf("conceptHits = %#v", conceptHits)
	}
}

func TestGetConceptTreeCycle(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/access/search/conceptscheme":
			writeTestJSON(t, w, map[string]any{
				"total":      1,
				"totalItems": 1,
				"page":       1,
				"totalPages": 1,
				"items": []map[string]any{
					{"type": "conceptscheme", "id": "scheme-a", "href": "http://example.test/scheme-a"},
				},
			})
		case "/api/v2/access/conceptscheme/scheme-a":
			writeTestJSON(t, w, map[string]any{
				"id":    "scheme-a",
				"shape": "conceptscheme",
				"href":  "http://example.test/scheme-a",
				"properties": map[string]any{
					"skos:prefLabel": "Scheme A",
					"dct:title":      "Scheme A Title",
					"skos:hasTopConcept": []map[string]any{
						{"id": "concept-a", "type": "concept"},
					},
				},
			})
		case "/api/v2/access/concept/concept-a":
			writeTestJSON(t, w, map[string]any{
				"id":    "concept-a",
				"shape": "concept",
				"href":  "http://example.test/concept-a",
				"properties": map[string]any{
					"skos:prefLabel": "Concept A",
					"skos:narrower": []map[string]any{
						{"id": "concept-b", "type": "concept"},
					},
				},
			})
		case "/api/v2/access/concept/concept-b":
			writeTestJSON(t, w, map[string]any{
				"id":    "concept-b",
				"shape": "concept",
				"href":  "http://example.test/concept-b",
				"properties": map[string]any{
					"skos:prefLabel": "Concept B",
					"skos:narrower": []map[string]any{
						{"id": "concept-a", "type": "concept"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService(newTestMetadataClient(server.URL), 8)

	_, err := service.GetConceptTree(context.Background())
	if err == nil {
		t.Fatal("GetConceptTree() error = nil")
	}
	if !errors.Is(err, ErrUpstreamRequestFailed) {
		t.Fatalf("errors.Is(err, ErrUpstreamRequestFailed) = false, err = %v", err)
	}
}

func TestGetConceptTreeSkipsSchemeWhenPrefLabelMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/access/search/conceptscheme":
			writeTestJSON(t, w, map[string]any{
				"total":      1,
				"totalItems": 1,
				"page":       1,
				"totalPages": 1,
				"items": []map[string]any{
					{"type": "conceptscheme", "id": "scheme-a", "href": "http://example.test/scheme-a"},
				},
			})
		case "/api/v2/access/conceptscheme/scheme-a":
			writeTestJSON(t, w, map[string]any{
				"id":    "scheme-a",
				"shape": "conceptscheme",
				"href":  "http://example.test/scheme-a",
				"properties": map[string]any{
					"dct:title": "Scheme A Title",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService(newTestMetadataClient(server.URL), 8)

	tree, err := service.GetConceptTree(context.Background())
	if err != nil {
		t.Fatalf("GetConceptTree() error = %v", err)
	}

	if len(tree.Schemes) != 0 {
		t.Fatalf("len(tree.Schemes) = %d", len(tree.Schemes))
	}
}

func newTestMetadataClient(baseURL string) *metadataclient.APIClient {
	config := metadataclient.NewConfiguration()
	config.Servers = metadataclient.ServerConfigurations{{URL: baseURL}}
	return metadataclient.NewAPIClient(config)
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("json encode: %v", err)
	}
}
