package discovery

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
)

type searchFixtureCase struct {
	name         string
	routes       map[string]string
	resourceType api.ResourceType
	wantName     string
	wantHref     string
}

func TestSearchMapsFixtures(t *testing.T) {
	t.Parallel()

	tests := []searchFixtureCase{
		{
			name: "software application",
			routes: map[string]string{
				"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwareapplication-first-page.json",
				"/api/v2/access/softwareapplication/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2FCatalogToXls%2Fv1.0":           "testdata/metadata-lifecycle/access/softwareapplication/catalog-to-xls.json",
				"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-4c77d796%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-doi.json",
				"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-b54da560%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-ddss-id.json",
				"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                                                              "testdata/metadata-lifecycle/access/organization/uppsala.json",
				"/api/v2/access/concept/category%3Asoftware_appl_conc_multi%2Fv1.0":                                                               "testdata/metadata-lifecycle/access/concept/software-appl-conc-multi.json",
			},
			resourceType: api.SoftwareApplicationType,
			wantName:     "Catalog to XLS converter",
			wantHref:     "v1/softwareapplication/https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0",
		},
		{
			name: "software source code",
			routes: map[string]string{
				"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwaresourcecode-first-page.json",
				"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fzenodo.17598403":                                             "testdata/metadata-lifecycle/access/softwaresourcecode/zenodo.json",
				"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-0ebd586f%2Fv1.0": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-0ebd586f.json",
				"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-4c77d796%2Fv1.0": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-4c77d796.json",
				"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                                                              "testdata/metadata-lifecycle/access/organization/uppsala.json",
				"/api/v2/access/concept/category%3Asoftware_jup_conc_multi%2Fv1.0":                                                                "testdata/metadata-lifecycle/access/concept/software-jup-conc-multi.json",
			},
			resourceType: api.SoftwareSourceCodeType,
			wantName:     "Discovery, automation/scripting and integration of EPOS data and services. Basic workflows in JupyterLab.",
			wantHref:     "v1/softwaresourcecode/https://doi.org/10.5281/zenodo.17598403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := testfixtures.NewMetadataServer(t, tt.routes)
			defer server.Close()

			service := NewService(testfixtures.NewMetadataClient(server.URL), 8)

			resp, err := service.Search(context.Background(), SearchParams{Types: []api.ResourceType{tt.resourceType}})
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if len(resp.Items) != 1 {
				t.Fatalf("len(resp.Items) = %d", len(resp.Items))
			}

			item := resp.Items[0]

			var name string
			var href string
			var provider []api.ResourceRef

			switch v := item.(type) {
			case api.SoftwareApplicationItem:
				name = v.Name
				href = v.Href
				provider = v.Provider
			case api.SoftwareSourceCodeItem:
				name = v.Name
				href = v.Href
				provider = v.Provider
			default:
				t.Fatalf("unexpected item type %T", item)
			}

			if name != tt.wantName {
				t.Fatalf("item.Name = '%s'", name)
			}
			if len(provider) != 1 || provider[0].Name != "Uppsala University" {
				t.Fatalf("item.Provider = %#v", provider)
			}
			if href != tt.wantHref {
				t.Fatalf("item.Href = '%s'", href)
			}
		})
	}
}

func TestSearchSkipsMalformedSoftwareSourceCodeFixture(t *testing.T) {
	t.Parallel()

	server := testfixtures.NewMetadataServer(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwaresourcecode-first-page.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fzenodo.17598403":                                             "testdata/metadata-lifecycle/access/softwaresourcecode/zenodo.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-0ebd586f%2Fv1.0": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-0ebd586f.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-4c77d796%2Fv1.0": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-4c77d796.json",
		"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                                                              "testdata/metadata-lifecycle/access/organization/uppsala.json",
		"/api/v2/access/concept/category%3Asoftware_jup_conc_multi%2Fv1.0":                                                                "testdata/metadata-lifecycle/access/concept/software-jup-conc-multi.json",
	})
	defer server.Close()

	service := NewService(testfixtures.NewMetadataClient(server.URL), 8)

	resp, err := service.Search(context.Background(), SearchParams{Types: []api.ResourceType{api.SoftwareSourceCodeType}})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("len(resp.Items) = %d", len(resp.Items))
	}
}
