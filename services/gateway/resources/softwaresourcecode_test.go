package resources

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapSoftwareSourceCode(t *testing.T) {
	t.Parallel()

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/softwaresourcecode/sgg.json")

	softwareSourceCode, err := mapSoftwareSourceCode(context.Background(), detail, logging.Get("test"))
	if err != nil {
		t.Fatalf("mapSoftwareSourceCode() error = %v", err)
	}

	if softwareSourceCode.ID != "http://www.epos-eu.org/epos-dcat-ap/software/Dataset/001/sgg/SSC/v1.0" {
		t.Fatalf("softwareSourceCode.ID = %q", softwareSourceCode.ID)
	}
	if softwareSourceCode.Name != "Analyze earthquakes, permanent displacement and geology" {
		t.Fatalf("softwareSourceCode.Name = %q", softwareSourceCode.Name)
	}
	if softwareSourceCode.Description == "" {
		t.Fatalf("softwareSourceCode.Description = %q", softwareSourceCode.Description)
	}
	if softwareSourceCode.ProgrammingLanguage != "Python" {
		t.Fatalf("softwareSourceCode.ProgrammingLanguage = %q", softwareSourceCode.ProgrammingLanguage)
	}
	if softwareSourceCode.Repository != "https://raw.githubusercontent.com/epos-eu/jupyter-notebooks/main/SEIS%2BGNSS%2BGIM_2025-06-24.ipynb" {
		t.Fatalf("softwareSourceCode.Repository = %q", softwareSourceCode.Repository)
	}
	if softwareSourceCode.License != "https://creativecommons.org/licenses/by/4.0/" {
		t.Fatalf("softwareSourceCode.License = %q", softwareSourceCode.License)
	}
	if len(softwareSourceCode.ContactPoints) != 1 || softwareSourceCode.ContactPoints[0].Href != "http://localhost:8080/api/v2/access/contactpoint/http%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2Fjupyter%2FContactPoint%2Fsgg%2F001%2Fv1.0" {
		t.Fatalf("softwareSourceCode.ContactPoints = %#v", softwareSourceCode.ContactPoints)
	}
	if len(softwareSourceCode.Concepts) != 1 || softwareSourceCode.Concepts[0].Href != "v1/concept/category:software_jup_conc_multi/v1.0" {
		t.Fatalf("softwareSourceCode.Concepts = %#v", softwareSourceCode.Concepts)
	}
	if len(softwareSourceCode.Creator) != 1 || softwareSourceCode.Creator[0].Href != "http://localhost:8080/api/v2/access/organization/PIC%3A999974456%2Fv1.0" {
		t.Fatalf("softwareSourceCode.Creator = %#v", softwareSourceCode.Creator)
	}

	wantKeywords := map[string]bool{
		"Jupyter notebook": true,
		"GNSS":             true,
		"GIM":              true,
		"seismology":       true,
		"geodesy":          true,
		"displacement":     true,
		"map view":         true,
		"geology":          true,
		"moment tensor":    true,
		"stations":         true,
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
	}, logging.Get("test"))
	if err != nil {
		t.Fatalf("mapSoftwareSourceCode() error = %v", err)
	}

	if len(softwareSourceCode.Creator) != 1 {
		t.Fatalf("softwareSourceCode.Creator = %#v", softwareSourceCode.Creator)
	}
	if softwareSourceCode.Creator[0].ID != "org-1" {
		t.Fatalf("softwareSourceCode.Creator[0] = %#v", softwareSourceCode.Creator[0])
	}
}

func TestMapSoftwareSourceCodeMissingName(t *testing.T) {
	t.Parallel()

	_, err := mapSoftwareSourceCode(context.Background(), &metadataclient.AccessDetailResponse{Properties: map[string]any{}}, logging.Get("test"))
	if err == nil {
		t.Fatal("mapSoftwareSourceCode() error = nil")
	}
}
