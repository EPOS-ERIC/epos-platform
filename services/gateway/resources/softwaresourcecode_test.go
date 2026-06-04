package resources

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/gatewayhref"
)

func TestMapSoftwareSourceCode(t *testing.T) {
	t.Parallel()

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/softwaresourcecode/zenodo.json")
	fetch := newFixtureAccessDetailFetcher(t, map[string]string{
		fixtureKey("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-0ebd586f/v1.0"): "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-0ebd586f.json",
		fixtureKey("propertyvalue", "https://epos-eu.org/anthropogenic_hazards/software/bnode-propertyvalue-4c77d796/v1.0"): "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-4c77d796.json",
	})

	softwareSourceCode, err := mapSoftwareSourceCode(context.Background(), detail, fetch, logging.Get("test"))
	if err != nil {
		t.Fatalf("mapSoftwareSourceCode() error = %v", err)
	}

	if softwareSourceCode.ID != "https://doi.org/10.5281/zenodo.17598403" {
		t.Fatalf("softwareSourceCode.ID = %q", softwareSourceCode.ID)
	}
	if softwareSourceCode.Name != "Discovery, automation/scripting and integration of EPOS data and services. Basic workflows in JupyterLab." {
		t.Fatalf("softwareSourceCode.Name = %q", softwareSourceCode.Name)
	}
	if softwareSourceCode.Description == "" {
		t.Fatalf("softwareSourceCode.Description = %q", softwareSourceCode.Description)
	}
	if softwareSourceCode.ProgrammingLanguage != "Python" {
		t.Fatalf("softwareSourceCode.ProgrammingLanguage = %q", softwareSourceCode.ProgrammingLanguage)
	}
	if softwareSourceCode.Repository != "https://raw.githubusercontent.com/daoane/epos-multidisciplinary-service-integration/refs/heads/main/Basic_workflows_service_integration.ipynb" {
		t.Fatalf("softwareSourceCode.Repository = %q", softwareSourceCode.Repository)
	}
	if softwareSourceCode.License != "https://spdx.org/licenses/GPL-3.0-or-later.html" {
		t.Fatalf("softwareSourceCode.License = %q", softwareSourceCode.License)
	}
	if len(softwareSourceCode.Concepts) != 1 || softwareSourceCode.Concepts[0].Href != "v1/concept/category:software_jup_conc_multi/v1.0" {
		t.Fatalf("softwareSourceCode.Concepts = %#v", softwareSourceCode.Concepts)
	}
	if len(softwareSourceCode.Creator) != 1 || softwareSourceCode.Creator[0].Href != gatewayhref.Build(api.OrganizationType, "ROR:048a87296/v1.0") {
		t.Fatalf("softwareSourceCode.Creator = %#v", softwareSourceCode.Creator)
	}
	if len(softwareSourceCode.Identifiers) != 2 {
		t.Fatalf("softwareSourceCode.Identifiers = %#v", softwareSourceCode.Identifiers)
	}
	if softwareSourceCode.Identifiers[0].Type != "DDSS-ID" || softwareSourceCode.Identifiers[0].Value != "WP" {
		t.Fatalf("softwareSourceCode.Identifiers[0] = %#v", softwareSourceCode.Identifiers[0])
	}
	if softwareSourceCode.Identifiers[1].Type != "DOI" || softwareSourceCode.Identifiers[1].Value != "10.5281/zenodo.17598403" {
		t.Fatalf("softwareSourceCode.Identifiers[1] = %#v", softwareSourceCode.Identifiers[1])
	}

	wantKeywords := map[string]bool{
		"EPOS":            true,
		"JupyterLab":      true,
		"notebook":        true,
		"introduction":    true,
		"GIM":             true,
		"GNSS":            true,
		"seismology":      true,
		"OGC":             true,
		"OWS":             true,
		"EIDA":            true,
		"GLASS framework": true,
		"web mapping":     true,
		"web services":    true,
	}
	if len(softwareSourceCode.Keywords) != len(wantKeywords) {
		t.Fatalf("softwareSourceCode.Keywords = %#v", softwareSourceCode.Keywords)
	}
	for _, keyword := range softwareSourceCode.Keywords {
		if !wantKeywords[keyword] {
			t.Fatalf("unexpected keyword %q in %#v", keyword, softwareSourceCode.Keywords)
		}
	}
}

func TestMapSoftwareSourceCodeSkipsUnsupportedCreatorType(t *testing.T) {
	t.Parallel()

	softwareSourceCode, err := mapSoftwareSourceCode(context.Background(), &metadataclient.AccessDetailResponse{
		Id: "software-1",
		Properties: map[string]any{
			"schema:name": "Software title",
			"schema:creator": []any{
				map[string]any{"id": "org-1", "type": "organization", "href": "http://example.org/org-1"},
				map[string]any{"id": "person-1", "type": "person", "href": "http://example.org/person-1"},
			},
		},
	}, nil, logging.Get("test"))
	if err != nil {
		t.Fatalf("mapSoftwareSourceCode() error = %v", err)
	}

	if len(softwareSourceCode.Creator) != 1 || softwareSourceCode.Creator[0].Href != gatewayhref.Build(api.OrganizationType, "org-1") {
		t.Fatalf("softwareSourceCode.Creator = %#v", softwareSourceCode.Creator)
	}
}

func TestMapSoftwareSourceCodeMissingName(t *testing.T) {
	t.Parallel()

	_, err := mapSoftwareSourceCode(context.Background(), &metadataclient.AccessDetailResponse{Properties: map[string]any{}}, nil, logging.Get("test"))
	if err == nil {
		t.Fatal("mapSoftwareSourceCode() error = nil")
	}
}
