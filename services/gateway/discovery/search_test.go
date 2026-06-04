package discovery

import (
	"context"
	"net/url"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/gatewayhref"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
)

const (
	defaultSearchPath             = "/api/v2/access/search?includeConstraints=true&page=1&resultsPerPage=10&type=dataset&type=organization&type=softwareapplication&type=softwaresourcecode&type=webservice&versioningStatus=PUBLISHED"
	datasetSearchPath             = "/api/v2/access/search?includeConstraints=true&page=1&resultsPerPage=10&type=dataset&versioningStatus=PUBLISHED"
	organizationSearchPath        = "/api/v2/access/search?includeConstraints=true&page=1&resultsPerPage=10&type=organization&versioningStatus=PUBLISHED"
	softwareApplicationSearchPath = "/api/v2/access/search?includeConstraints=true&page=1&resultsPerPage=10&type=softwareapplication&versioningStatus=PUBLISHED"
	softwareSourceCodeSearchPath  = "/api/v2/access/search?includeConstraints=true&page=1&resultsPerPage=10&type=softwaresourcecode&versioningStatus=PUBLISHED"
	webserviceSearchPath          = "/api/v2/access/search?includeConstraints=true&page=1&resultsPerPage=10&type=webservice&versioningStatus=PUBLISHED"
	emptyDatasetSearchPath        = "/api/v2/access/search?includeConstraints=true&page=1&q=__no_such_dataset__&resultsPerPage=10&type=dataset&versioningStatus=PUBLISHED"

	softwareApplicationID    = "https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0"
	softwareSourceCodeID     = "https://doi.org/10.5281/zenodo.17598403"
	datasetThemeID           = "category:earthquakeparameters/v1.0"
	datasetThemeHref         = "v1/concept/category:earthquakeparameters/v1.0"
	datasetPublisherHref     = "v1/organization/PIC:999472675/v1.0"
	datasetDistributionID    = "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events/distribution/restful/v1.0"
	datasetDistributionName  = "TEST1 - Parameters of historical earthquakes (1000-1899) - FDSN event"
	datasetDistributionHref  = "v1/distribution/https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events/distribution/restful/v1.0"
	currentAHEADWebserviceID = "https://www.epos-eu.org/epos-dcat-ap/Seismology/WebService/AHEAD/restful/fdsnws-event/v1.0"
	currentAHEADUppsalaID    = "ROR:048a87296/v1.0"
	webserviceProviderID     = "PIC:999472675/v1.0"
	webserviceProviderHref   = "v1/organization/PIC:999472675/v1.0"
	webserviceProviderName   = "INGV - Istituto Nazionale di Geofisica e Vulcanologia"
)

type searchCase struct {
	name            string
	routes          map[string]string
	params          SearchParams
	wantPageNumber  int
	wantPageSize    int
	wantTotal       int
	wantTotalPages  int
	wantItems       int
	wantFirstAssert func(t *testing.T, item api.SearchItem)
}

func TestSearchReturnsExpectedResults(t *testing.T) {
	t.Parallel()

	tests := []searchCase{
		{
			name:           "default search",
			routes:         defaultSearchRoutes(),
			wantPageNumber: 1,
			wantPageSize:   10,
			wantTotal:      13,
			wantTotalPages: 2,
			wantItems:      10,
			wantFirstAssert: func(t *testing.T, item api.SearchItem) {
				assertOrganizationSearchItem(t, item, "PIC:999472675/v1.0", "INGV - Istituto Nazionale di Geofisica e Vulcanologia", gatewayhref.Build(api.OrganizationType, "PIC:999472675/v1.0"))
			},
		},
		{
			name:           "dataset search",
			routes:         baseSearchRoutes(datasetSearchPath, "testdata/metadata-lifecycle/access/search/dataset-first-page.json"),
			params:         SearchParams{Types: []api.ResourceType{api.DatasetType}},
			wantPageNumber: 1,
			wantPageSize:   3,
			wantTotal:      3,
			wantTotalPages: 1,
			wantItems:      3,
			wantFirstAssert: func(t *testing.T, item api.SearchItem) {
				assertDatasetSearchItem(t, item)
			},
		},
		{
			name:           "organization search",
			routes:         baseSearchRoutes(organizationSearchPath, "testdata/metadata-lifecycle/access/search/organization-first-page.json"),
			params:         SearchParams{Types: []api.ResourceType{api.OrganizationType}},
			wantPageNumber: 1,
			wantPageSize:   6,
			wantTotal:      6,
			wantTotalPages: 1,
			wantItems:      6,
			wantFirstAssert: func(t *testing.T, item api.SearchItem) {
				assertOrganizationSearchItem(t, item, "PIC:999472675/v1.0", "INGV - Istituto Nazionale di Geofisica e Vulcanologia", gatewayhref.Build(api.OrganizationType, "PIC:999472675/v1.0"))
			},
		},
		{
			name:           "software application search",
			routes:         softwareApplicationSearchRoutes(),
			params:         SearchParams{Types: []api.ResourceType{api.SoftwareApplicationType}},
			wantPageNumber: 1,
			wantPageSize:   1,
			wantTotal:      1,
			wantTotalPages: 1,
			wantItems:      1,
			wantFirstAssert: func(t *testing.T, item api.SearchItem) {
				assertSoftwareApplicationSearchItem(t, item)
			},
		},
		{
			name:           "software source code search",
			routes:         softwareSourceCodeSearchRoutes(),
			params:         SearchParams{Types: []api.ResourceType{api.SoftwareSourceCodeType}},
			wantPageNumber: 1,
			wantPageSize:   1,
			wantTotal:      1,
			wantTotalPages: 1,
			wantItems:      1,
			wantFirstAssert: func(t *testing.T, item api.SearchItem) {
				assertSoftwareSourceCodeSearchItem(t, item)
			},
		},
		{
			name:           "webservice search",
			routes:         baseSearchRoutes(webserviceSearchPath, "testdata/metadata-lifecycle/access/search/webservice-first-page.json"),
			params:         SearchParams{Types: []api.ResourceType{api.WebserviceType}},
			wantPageNumber: 1,
			wantPageSize:   2,
			wantTotal:      2,
			wantTotalPages: 1,
			wantItems:      2,
			wantFirstAssert: func(t *testing.T, item api.SearchItem) {
				assertWebserviceSearchItem(t, item)
			},
		},
		{
			name:           "empty dataset search",
			routes:         baseSearchRoutes(emptyDatasetSearchPath, "testdata/metadata-lifecycle/access/search/dataset-empty.json"),
			params:         SearchParams{Query: "__no_such_dataset__", Types: []api.ResourceType{api.DatasetType}},
			wantPageNumber: 1,
			wantPageSize:   0,
			wantTotal:      0,
			wantTotalPages: 0,
			wantItems:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := testfixtures.NewMetadataServer(t, tt.routes)
			defer server.Close()

			service := NewService(testfixtures.NewMetadataClient(server.URL), 8)

			resp, err := service.Search(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			assertSearchResponse(t, resp, tt.wantPageNumber, tt.wantPageSize, tt.wantTotal, tt.wantTotalPages, tt.wantItems)
			if tt.wantFirstAssert != nil {
				tt.wantFirstAssert(t, resp.Items[0])
			}
		})
	}
}

func TestSearchRejectsUnsupportedTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		types   []api.ResourceType
		wantErr string
	}{
		{
			name:    "concept type is rejected",
			types:   []api.ResourceType{api.ResourceType("concept")},
			wantErr: `build metadata search request: validate search types: unsupported search type "concept"; allowed values: [dataset organization softwareapplication softwaresourcecode webservice]`,
		},
		{
			name:    "mixed supported and unsupported types are rejected",
			types:   []api.ResourceType{api.DatasetType, api.ResourceType("distribution")},
			wantErr: `build metadata search request: validate search types: unsupported search type "distribution"; allowed values: [dataset organization softwareapplication softwaresourcecode webservice]`,
		},
	}

	service := NewService(testfixtures.NewMetadataClient("http://localhost:8080"), 8)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.Search(context.Background(), SearchParams{Types: tt.types})
			if err == nil {
				t.Fatal("Search() error = nil")
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("Search() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func assertSearchResponse(t *testing.T, resp api.SearchResponse, wantPageNumber, wantPageSize, wantTotal, wantTotalPages, wantItems int) {
	t.Helper()

	if resp.PageNumber != wantPageNumber {
		t.Fatalf("PageNumber = %d, want %d", resp.PageNumber, wantPageNumber)
	}
	if resp.PageSize != wantPageSize {
		t.Fatalf("PageSize = %d, want %d", resp.PageSize, wantPageSize)
	}
	if resp.Total != wantTotal {
		t.Fatalf("Total = %d, want %d", resp.Total, wantTotal)
	}
	if resp.TotalPages != wantTotalPages {
		t.Fatalf("TotalPages = %d, want %d", resp.TotalPages, wantTotalPages)
	}
	if len(resp.Items) != wantItems {
		t.Fatalf("len(Items) = %d, want %d", len(resp.Items), wantItems)
	}
}

func assertSearchBaseItem(t *testing.T, got api.BaseSearchItem, wantType api.ResourceType, wantID, wantName, wantHref string) {
	t.Helper()

	if got.Type != wantType {
		t.Fatalf("Type = %q, want %q", got.Type, wantType)
	}
	if got.ID != wantID {
		t.Fatalf("ID = %q, want %q", got.ID, wantID)
	}
	if got.Name != wantName {
		t.Fatalf("Name = %q, want %q", got.Name, wantName)
	}
	if got.Href != wantHref {
		t.Fatalf("Href = %q, want %q", got.Href, wantHref)
	}
}

func assertResourceRef(t *testing.T, got api.ResourceRef, wantID, wantName, wantHref string) {
	t.Helper()

	if got.ID != wantID {
		t.Fatalf("ID = %q, want %q", got.ID, wantID)
	}
	if got.Name != wantName {
		t.Fatalf("Name = %q, want %q", got.Name, wantName)
	}
	if got.Href != wantHref {
		t.Fatalf("Href = %q, want %q", got.Href, wantHref)
	}
}

func assertDatasetSearchItem(t *testing.T, item api.SearchItem) {
	t.Helper()

	dataset, ok := item.(api.DatasetItem)
	if !ok {
		t.Fatalf("item = %T, want api.DatasetItem", item)
	}

	assertSearchBaseItem(t, dataset.BaseSearchItem, api.DatasetType, "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events/v1.0", "TEST1 - Historical earthquakes 1000-1899", gatewayhref.Build(api.DatasetType, "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events/v1.0"))
	if len(dataset.Providers) != 1 {
		t.Fatalf("Providers = %#v", dataset.Providers)
	}
	assertResourceRef(t, dataset.Providers[0], "PIC:999472675/v1.0", "INGV - Istituto Nazionale di Geofisica e Vulcanologia", datasetPublisherHref)
	if len(dataset.Concepts) != 1 {
		t.Fatalf("Concepts = %#v", dataset.Concepts)
	}
	assertResourceRef(t, dataset.Concepts[0], datasetThemeID, "", datasetThemeHref)
	if len(dataset.Distributions) != 1 {
		t.Fatalf("Distributions = %#v", dataset.Distributions)
	}
	if dataset.Distributions[0].ID != datasetDistributionID || dataset.Distributions[0].Name != datasetDistributionName || dataset.Distributions[0].Href != datasetDistributionHref {
		t.Fatalf("Distributions[0] = %#v", dataset.Distributions[0])
	}
}

func assertOrganizationSearchItem(t *testing.T, item api.SearchItem, wantID, wantName, wantHref string) {
	t.Helper()

	organization, ok := item.(api.OrganizationItem)
	if !ok {
		t.Fatalf("item = %T, want api.OrganizationItem", item)
	}

	assertSearchBaseItem(t, organization.BaseSearchItem, api.OrganizationType, wantID, wantName, wantHref)
}

func assertSoftwareApplicationSearchItem(t *testing.T, item api.SearchItem) {
	t.Helper()

	softwareApplication, ok := item.(api.SoftwareApplicationItem)
	if !ok {
		t.Fatalf("item = %T, want api.SoftwareApplicationItem", item)
	}

	assertSearchBaseItem(t, softwareApplication.BaseSearchItem, api.SoftwareApplicationType, softwareApplicationID, "Catalog to XLS converter", gatewayhref.Build(api.SoftwareApplicationType, softwareApplicationID))
	if len(softwareApplication.Provider) != 1 {
		t.Fatalf("Provider = %#v", softwareApplication.Provider)
	}
	assertResourceRef(t, softwareApplication.Provider[0], currentAHEADUppsalaID, "Uppsala University", gatewayhref.Build(api.OrganizationType, currentAHEADUppsalaID))
}

func assertSoftwareSourceCodeSearchItem(t *testing.T, item api.SearchItem) {
	t.Helper()

	softwareSourceCode, ok := item.(api.SoftwareSourceCodeItem)
	if !ok {
		t.Fatalf("item = %T, want api.SoftwareSourceCodeItem", item)
	}

	assertSearchBaseItem(t, softwareSourceCode.BaseSearchItem, api.SoftwareSourceCodeType, softwareSourceCodeID, "Discovery, automation/scripting and integration of EPOS data and services. Basic workflows in JupyterLab.", gatewayhref.Build(api.SoftwareSourceCodeType, softwareSourceCodeID))
	if len(softwareSourceCode.Provider) != 1 {
		t.Fatalf("Provider = %#v", softwareSourceCode.Provider)
	}
	assertResourceRef(t, softwareSourceCode.Provider[0], currentAHEADUppsalaID, "Uppsala University", gatewayhref.Build(api.OrganizationType, currentAHEADUppsalaID))
}

func assertWebserviceSearchItem(t *testing.T, item api.SearchItem) {
	t.Helper()

	webservice, ok := item.(api.WebServiceItem)
	if !ok {
		t.Fatalf("item = %T, want api.WebServiceItem", item)
	}

	assertSearchBaseItem(t, webservice.BaseSearchItem, api.WebserviceType, currentAHEADWebserviceID, "TEST1- Historical earthquakes 1000-1899 (FDSN-event)", gatewayhref.Build(api.WebserviceType, currentAHEADWebserviceID))
	assertResourceRef(t, webservice.Provider, webserviceProviderID, webserviceProviderName, webserviceProviderHref)
	if len(webservice.Concepts) != 0 {
		t.Fatalf("Concepts = %#v", webservice.Concepts)
	}
}

func defaultSearchRoutes() map[string]string {
	routes := baseSearchRoutes(defaultSearchPath, "testdata/metadata-lifecycle/access/search/default-supported-first-page.json")
	return routes
}

func baseSearchRoutes(searchPath, searchFixture string) map[string]string {
	routes := fdsnEventFixturePaths()
	routes[searchPath] = searchFixture
	return routes
}

func accessPath(shape, id string) string {
	return "/api/v2/access/" + shape + "/" + url.PathEscape(id)
}

func softwareApplicationSearchRoutes() map[string]string {
	routes := fdsnEventFixturePaths()
	routes[softwareApplicationSearchPath] = "testdata/metadata-lifecycle/access/search/softwareapplication-first-page.json"
	routes[accessPath("softwareapplication", softwareApplicationID)] = "testdata/metadata-lifecycle/access/softwareapplication/catalog-to-xls.json"
	routes[accessPath("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-4c77d796/v1.0")] = "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-doi.json"
	routes[accessPath("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-b54da560/v1.0")] = "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-ddss-id.json"
	routes[accessPath("organization", currentAHEADUppsalaID)] = "testdata/metadata-lifecycle/access/organization/uppsala.json"
	routes[accessPath("concept", "category:software_appl_conc_multi/v1.0")] = "testdata/metadata-lifecycle/access/concept/software-appl-conc-multi.json"
	return routes
}

func fdsnEventFixturePaths() map[string]string {
	return map[string]string{
		accessPath("dataset", "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events/v1.0"):                                               "testdata/metadata-lifecycle/access/dataset/intermagnet.json",
		accessPath("dataset", "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/bibliography/v1.0"):                                         "testdata/metadata-lifecycle/access/dataset/bibliography.json",
		accessPath("dataset", "http://www.epos-eu.org/epos-dcat-ap/Seismology/SeismologicalProducts/ESMFlatFile/Dataset/001/v1.0"):                       "testdata/metadata-lifecycle/access/dataset/esmflatfile.json",
		accessPath("distribution", datasetDistributionID):                                                                                                "testdata/metadata-lifecycle/access/distribution/intermagnet.json",
		accessPath("distribution", "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/bibliography/distribution/restful/v1.0"):               "testdata/metadata-lifecycle/access/distribution/bibliography.json",
		accessPath("distribution", "http://www.epos-eu.org/epos-dcat-ap/Seismology/SeismologicalProducts/ESMFlatFile/Dataset/001/Distribution/001/v1.0"): "testdata/metadata-lifecycle/access/distribution/esmflatfile.json",
		accessPath("webservice", currentAHEADWebserviceID):                                                                                               "testdata/metadata-lifecycle/access/webservice/intermagnet.json",
		accessPath("webservice", "https://www.epos-eu.org/epos-dcat-ap/Seismology/WebService/AHEAD/restful/bibliography/v1.0"):                           "testdata/metadata-lifecycle/access/webservice/bibliography.json",
		accessPath("organization", "PIC:999472675/v1.0"):                                                                                                 "testdata/metadata-lifecycle/access/organization/pic.json",
		accessPath("organization", "MOID:https://gnss-metadata.eu/MOID/datasupplier.641dd8c7e06540da270881d8/v1.0"):                                      "testdata/metadata-lifecycle/access/organization/moid.json",
		accessPath("organization", "https://doi.org/10.7914/SN/IT"):                                                                                      "testdata/metadata-lifecycle/access/organization/doi-it.json",
		accessPath("operation", "https://www.epos-eu.org/epos-dcat-ap/Seismology/WebService/AHEAD/restful/fdsnws-event/Operation/v1.0"):                  "testdata/metadata-lifecycle/access/operation/intermagnet.json",
		accessPath("iritemplate", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplate-383ba416/v1.0"):                                  "testdata/metadata-lifecycle/access/iritemplate/intermagnet.json",
		accessPath("periodoftime", "https://epos-eu.org/anthropogenic_hazards/software/bnode-periodoftime-854f13c1/v1.0"):                                "testdata/metadata-lifecycle/access/periodoftime/intermagnet.json",
		accessPath("location", "https://epos-eu.org/anthropogenic_hazards/software/bnode-location-2df7160e/v1.0"):                                        "testdata/metadata-lifecycle/access/location/webservice.json",
		accessPath("location", "https://epos-eu.org/anthropogenic_hazards/software/bnode-location-f75195ba/v1.0"):                                        "testdata/metadata-lifecycle/access/location/intermagnet.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-bec6757f/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/nodata.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-6e1d03bf/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/limit.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-cc040606/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/includeallmagnitudes.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-ae32613b/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/includeallorigins.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-e296da48/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/orderby.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-c6edf57b/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/maxmagnitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-01f77125/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/minmagnitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-79219970/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/maxradiuskm.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-acd0818a/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/longitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-2fa4d342/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/latitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-99212efd/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/maxlongitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-59522638/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/minlongitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-c2ea443d/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/maxlatitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-c19cb3e0/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/minlatitude.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-b0c41f63/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/eventid.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-2f9a6fda/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/endtime.json",
		accessPath("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-9f1f2b7f/v1.0"):                    "testdata/metadata-lifecycle/access/iritemplatemapping/starttime.json",
		accessPath("apidocumentation", "https://www.epos-eu.org/epos-dcat-ap/Seismology/WebService/AHEAD/APIDocumentation/fdsnws-event/v1.0"):            "testdata/metadata-lifecycle/access/apidocumentation/intermagnet.json",
	}
}

func softwareSourceCodeSearchRoutes() map[string]string {
	routes := fdsnEventFixturePaths()
	routes[softwareSourceCodeSearchPath] = "testdata/metadata-lifecycle/access/search/softwaresourcecode-first-page.json"
	routes[accessPath("softwaresourcecode", softwareSourceCodeID)] = "testdata/metadata-lifecycle/access/softwaresourcecode/zenodo.json"
	routes[accessPath("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-0ebd586f/v1.0")] = "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-0ebd586f.json"
	routes[accessPath("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-4c77d796/v1.0")] = "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-4c77d796.json"
	routes[accessPath("organization", currentAHEADUppsalaID)] = "testdata/metadata-lifecycle/access/organization/uppsala.json"
	routes[accessPath("concept", "category:software_jup_conc_multi/v1.0")] = "testdata/metadata-lifecycle/access/concept/software-jup-conc-multi.json"
	return routes
}
