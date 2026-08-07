package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

var resourceID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

func TestResourcesGetDataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/resources/dataset/"+resourceID.String() {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","source_iri":"https://example.org/dataset","title":"Dataset"}`))
	}))
	defer server.Close()

	client, err := NewResources(server.URL, testUpstreamConfig())
	if err != nil {
		t.Fatalf("NewResources() error = %v", err)
	}

	dataset, err := client.GetDataset(context.Background(), resourceID)
	if err != nil {
		t.Fatalf("GetDataset() error = %v", err)
	}
	if dataset.ID != resourceID || dataset.SourceIRI != "https://example.org/dataset" || dataset.Title != "Dataset" {
		t.Fatalf("dataset = %#v", dataset)
	}
}

func TestResourcesGetDatasetMapsDomainErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		target error
	}{
		{name: "bad request", status: http.StatusBadRequest, target: domain.ErrBadRequest},
		{name: "not found", status: http.StatusNotFound, target: domain.ErrNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
			}))
			defer server.Close()

			client, err := NewResources(server.URL, testUpstreamConfig())
			if err != nil {
				t.Fatalf("NewResources() error = %v", err)
			}
			_, err = client.GetDataset(context.Background(), resourceID)
			if !errors.Is(err, test.target) {
				t.Fatalf("GetDataset() error = %v, want %v", err, test.target)
			}
		})
	}
}

func TestResourcesGetSoftware(t *testing.T) {
	tests := []struct {
		name, resourceType, payload string
	}{
		{name: "application", resourceType: "softwareapplication", payload: `{"id":"550e8400-e29b-41d4-a716-446655440000","source_iri":"https://example.org/software","name":"Software","keywords":"one,two","creators":[{"id":"550e8400-e29b-41d4-a716-446655440001","source_iri":"organization-1","name":"Organization"}]}`},
		{name: "source code", resourceType: "softwaresourcecode", payload: `{"id":"550e8400-e29b-41d4-a716-446655440000","source_iri":"https://example.org/source","name":"Source","repository":"https://example.org/repository","identifiers":[{"type":"DOI","value":"10.1/example"}]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v2/resources/"+test.resourceType+"/"+resourceID.String() {
					t.Fatalf("path = %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(test.payload))
			}))
			defer server.Close()

			client, err := NewResources(server.URL, testUpstreamConfig())
			if err != nil {
				t.Fatalf("NewResources() error = %v", err)
			}

			if test.resourceType == "softwareapplication" {
				software, err := client.GetSoftwareApplication(context.Background(), resourceID)
				if err != nil || software.ID != resourceID || software.SourceIRI != "https://example.org/software" || software.Name != "Software" || len(software.Creators) != 1 {
					t.Fatalf("GetSoftwareApplication() = %#v, %v", software, err)
				}
				return
			}

			software, err := client.GetSoftwareSourceCode(context.Background(), resourceID)
			if err != nil || software.ID != resourceID || software.SourceIRI != "https://example.org/source" || software.Name != "Source" || len(software.Identifiers) != 1 {
				t.Fatalf("GetSoftwareSourceCode() = %#v, %v", software, err)
			}
		})
	}
}

func testUpstreamConfig() config.Upstream {
	return config.Upstream{Timeout: time.Second, RealTimeout: time.Second, CacheTTL: 0, MaxEntries: 1}
}
