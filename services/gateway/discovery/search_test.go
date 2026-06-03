package discovery

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
)

func TestSearchMapsSoftwareApplicationFixture(t *testing.T) {
	t.Parallel()

	server := testfixtures.NewMetadataServer(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwareapplication-first-page.json",
		"/api/v2/access/softwareapplication/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2FCatalogToXls%2Fv1.0":           "testdata/metadata-lifecycle/access/softwareapplication/catalog-to-xls.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-4c77d796%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-doi.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-b54da560%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-ddss-id.json",
		"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                                                              "testdata/metadata-lifecycle/access/organization/uppsala.json",
		"/api/v2/access/concept/category%3Asoftware_appl_conc_multi%2Fv1.0":                                                               "testdata/metadata-lifecycle/access/concept/software-appl-conc-multi.json",
	})
	defer server.Close()

	service := NewService(testfixtures.NewMetadataClient(server.URL), 8)

	resp, err := service.Search(context.Background(), SearchParams{Types: []api.ResourceType{api.SoftwareApplicationType}})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("len(resp.Items) = %d", len(resp.Items))
	}

	softwareApplication, ok := resp.Items[0].(api.SoftwareApplicationItem)
	if !ok {
		t.Fatalf("resp.Items[0] type = %T", resp.Items[0])
	}
	if softwareApplication.Name != "Catalog to XLS converter" {
		t.Fatalf("softwareApplication.Name = %q", softwareApplication.Name)
	}
	if len(softwareApplication.Provider) != 1 || softwareApplication.Provider[0].Name != "Uppsala University" {
		t.Fatalf("softwareApplication.Provider = %#v", softwareApplication.Provider)
	}
	if softwareApplication.Href != "v1/softwareapplication/https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0" {
		t.Fatalf("softwareApplication.Href = %q", softwareApplication.Href)
	}
}

func TestSearchMapsSoftwareSourceCodeFixture(t *testing.T) {
	t.Parallel()

	server := testfixtures.NewMetadataServer(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwaresourcecode-first-page.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fzenodo.17598403":              "testdata/metadata-lifecycle/access/softwaresourcecode/zenodo.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fbnode-propertyvalue-0ebd586f": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-0ebd586f.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fbnode-propertyvalue-4c77d796": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-4c77d796.json",
		"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                               "testdata/metadata-lifecycle/access/organization/uppsala.json",
		"/api/v2/access/concept/category%3Asoftware_jup_conc_multi%2Fv1.0":                                 "testdata/metadata-lifecycle/access/concept/software-jup-conc-multi.json",
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

	softwareSourceCode, ok := resp.Items[0].(api.SoftwareSourceCodeItem)
	if !ok {
		t.Fatalf("resp.Items[0] type = %T", resp.Items[0])
	}
	if softwareSourceCode.Name != "Discovery, automation/scripting and integration of EPOS data and services. Basic workflows in JupyterLab." {
		t.Fatalf("softwareSourceCode.Name = %q", softwareSourceCode.Name)
	}
	if len(softwareSourceCode.Provider) != 1 || softwareSourceCode.Provider[0].Name != "Uppsala University" {
		t.Fatalf("softwareSourceCode.Provider = %#v", softwareSourceCode.Provider)
	}
	if softwareSourceCode.Href != "v1/softwaresourcecode/https://doi.org/10.5281/zenodo.17598403" {
		t.Fatalf("softwareSourceCode.Href = %q", softwareSourceCode.Href)
	}
}

func TestSearchSkipsMalformedSoftwareSourceCodeFixture(t *testing.T) {
	t.Parallel()

	server := testfixtures.NewMetadataServer(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwaresourcecode-first-page.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fzenodo.17598403":              "testdata/metadata-lifecycle/access/softwaresourcecode/zenodo.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fbnode-propertyvalue-0ebd586f": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-0ebd586f.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fbnode-propertyvalue-4c77d796": "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-4c77d796.json",
		"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                               "testdata/metadata-lifecycle/access/organization/uppsala.json",
		"/api/v2/access/concept/category%3Asoftware_jup_conc_multi%2Fv1.0":                                 "testdata/metadata-lifecycle/access/concept/software-jup-conc-multi.json",
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
