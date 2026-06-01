package discovery

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
)

func TestSearchMapsDatasetFixture(t *testing.T) {
	t.Parallel()

	server := testfixtures.NewMetadataServer(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/dataset-first-page.json",
		"/api/v2/access/organization/http%3A%2F%2Fviaf.org%2Fviaf%2F136554353%2F%2Fv1.0":                                                           "testdata/metadata-lifecycle/access/organization/bgs.json",
		"/api/v2/access/concept/category%3Ageomagneticdata%2Fv1.0":                                                                                 "testdata/metadata-lifecycle/access/concept/geomagneticdata.json",
		"/api/v2/access/dataset/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2FDataset%2Fv1.0": "testdata/metadata-lifecycle/access/dataset/intermagnet.json",
	})
	defer server.Close()

	service := NewService(testfixtures.NewMetadataClient(server.URL), 8)

	resp, err := service.Search(context.Background(), SearchParams{Types: []api.ResourceType{api.DatasetType}})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("len(resp.Items) = %d", len(resp.Items))
	}

	dataset, ok := resp.Items[0].(api.DatasetItem)
	if !ok {
		t.Fatalf("resp.Items[0] type = %T", resp.Items[0])
	}
	if dataset.Name != "INTERMAGNET Geomagnetic Observatory Data" {
		t.Fatalf("dataset.Name = %q", dataset.Name)
	}
	if len(dataset.Providers) != 1 || dataset.Providers[0].Name != "BGS - British Geological Survey, UK Research and Innovation" {
		t.Fatalf("dataset.Providers = %#v", dataset.Providers)
	}
	if len(dataset.Concepts) != 1 || dataset.Concepts[0].Name != "Geomagnetic data" {
		t.Fatalf("dataset.Concepts = %#v", dataset.Concepts)
	}
	if len(dataset.Distributions) != 1 || dataset.Distributions[0].ID != "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Distribution/v1.0" {
		t.Fatalf("dataset.Distributions = %#v", dataset.Distributions)
	}
}

func TestSearchMapsWebserviceFixture(t *testing.T) {
	t.Parallel()

	server := testfixtures.NewMetadataServer(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/webservice-first-page.json",
		"/api/v2/access/organization/http%3A%2F%2Fviaf.org%2Fviaf%2F136554353%2F%2Fv1.0": "testdata/metadata-lifecycle/access/organization/bgs.json",
	})
	defer server.Close()

	service := NewService(testfixtures.NewMetadataClient(server.URL), 8)

	resp, err := service.Search(context.Background(), SearchParams{Types: []api.ResourceType{api.WebserviceType}})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("len(resp.Items) = %d", len(resp.Items))
	}

	webservice, ok := resp.Items[0].(api.WebServiceItem)
	if !ok {
		t.Fatalf("resp.Items[0] type = %T", resp.Items[0])
	}
	if webservice.Name != "Web service for INTERMAGNET data" {
		t.Fatalf("webservice.Name = %q", webservice.Name)
	}
	if webservice.Provider.Name != "BGS - British Geological Survey, UK Research and Innovation" {
		t.Fatalf("webservice.Provider = %#v", webservice.Provider)
	}
	if len(webservice.Concepts) != 0 {
		t.Fatalf("webservice.Concepts = %#v", webservice.Concepts)
	}
}

func TestSearchSkipsMalformedSoftwareSourceCodeFixture(t *testing.T) {
	t.Parallel()

	server := testfixtures.NewMetadataServer(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwaresourcecode-first-page.json",
	})
	defer server.Close()

	service := NewService(testfixtures.NewMetadataClient(server.URL), 8)

	resp, err := service.Search(context.Background(), SearchParams{Types: []api.ResourceType{api.SoftwareSourceCodeType}})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("len(resp.Items) = %d", len(resp.Items))
	}
}
