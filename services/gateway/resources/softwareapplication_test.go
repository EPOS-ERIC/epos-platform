package resources

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapSoftwareApplication(t *testing.T) {
	t.Parallel()

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/softwareapplication/catalog-to-xls.json")
	fetch := newFixtureAccessDetailFetcher(t, map[string]string{
		fixtureKey("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-4c77d796/v1.0"): "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-doi.json",
		fixtureKey("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-b54da560/v1.0"): "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-ddss-id.json",
	})

	softwareApplication, err := mapSoftwareApplication(context.Background(), detail, fetch, logging.Get("test"))
	if err != nil {
		t.Fatalf("mapSoftwareApplication() error = %v", err)
	}

	if softwareApplication.ID != "https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0" {
		t.Fatalf("softwareApplication.ID = %q", softwareApplication.ID)
	}
	if softwareApplication.Name != "Catalog to XLS converter" {
		t.Fatalf("softwareApplication.Name = %q", softwareApplication.Name)
	}
	if softwareApplication.Description == "" {
		t.Fatalf("softwareApplication.Description = %q", softwareApplication.Description)
	}
	if softwareApplication.DownloadURL != "https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0/download" {
		t.Fatalf("softwareApplication.DownloadURL = %q", softwareApplication.DownloadURL)
	}
	if softwareApplication.OperatingSystem != "Windows 7" {
		t.Fatalf("softwareApplication.OperatingSystem = %q", softwareApplication.OperatingSystem)
	}
	if softwareApplication.MainEntityOfPage != "https://episodesplatform.eu/#app:CatalogToXls" {
		t.Fatalf("softwareApplication.MainEntityOfPage = %q", softwareApplication.MainEntityOfPage)
	}
	if softwareApplication.Requirements != "Windows 7" {
		t.Fatalf("softwareApplication.Requirements = %q", softwareApplication.Requirements)
	}
	if softwareApplication.License != "https://creativecommons.org/licenses/by/4.0/" {
		t.Fatalf("softwareApplication.License = %q", softwareApplication.License)
	}
	if softwareApplication.Version != "1.0" {
		t.Fatalf("softwareApplication.Version = %q", softwareApplication.Version)
	}
	if len(softwareApplication.ContactPoints) != 1 || softwareApplication.ContactPoints[0].Href != "http://localhost:8080/api/v2/access/contactpoint/https%3A%2F%2Fror.org%2F048a87296%2FlegalContact%2Fv1.0" {
		t.Fatalf("softwareApplication.ContactPoints = %#v", softwareApplication.ContactPoints)
	}
	if len(softwareApplication.Concepts) != 1 || softwareApplication.Concepts[0].Href != "v1/concept/category:software_appl_conc_multi/v1.0" {
		t.Fatalf("softwareApplication.Concepts = %#v", softwareApplication.Concepts)
	}
	if len(softwareApplication.Creator) != 0 {
		t.Fatalf("softwareApplication.Creator = %#v", softwareApplication.Creator)
	}

	wantKeywords := map[string]bool{
		"format conversion": true,
		"data download":     true,
		"data export":       true,
	}
	if len(softwareApplication.Keywords) != len(wantKeywords) {
		t.Fatalf("softwareApplication.Keywords = %#v", softwareApplication.Keywords)
	}
	for _, keyword := range softwareApplication.Keywords {
		if !wantKeywords[keyword] {
			t.Fatalf("unexpected keyword %q in %#v", keyword, softwareApplication.Keywords)
		}
	}

	if len(softwareApplication.Identifiers) != 2 {
		t.Fatalf("softwareApplication.Identifiers = %#v", softwareApplication.Identifiers)
	}
	if softwareApplication.Identifiers[0] != (api.Identifier{Type: "DOI", Value: "10.5281/zenodo.17598403"}) {
		t.Fatalf("softwareApplication.Identifiers[0] = %#v", softwareApplication.Identifiers[0])
	}
	if softwareApplication.Identifiers[1] != (api.Identifier{Type: "DDSS-ID", Value: "WP14-DDSS-A01"}) {
		t.Fatalf("softwareApplication.Identifiers[1] = %#v", softwareApplication.Identifiers[1])
	}
}

func TestMapSoftwareApplicationMissingName(t *testing.T) {
	t.Parallel()

	_, err := mapSoftwareApplication(context.Background(), &metadataclient.AccessDetailResponse{Properties: map[string]any{}}, nil, logging.Get("test"))
	if err == nil {
		t.Fatal("mapSoftwareApplication() error = nil")
	}
}
