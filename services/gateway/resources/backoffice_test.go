package resources

import (
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/google/uuid"
)

func testUUID(value string) uuid.UUID {
	return uuid.MustParse(value)
}

func TestMapDatasetFromResources(t *testing.T) {
	created := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	updated := created.Add(time.Hour)
	sourceUpdated := created.Add(2 * time.Hour)
	dataset, err := mapDataset(clients.Dataset{
		ID: testUUID("550e8400-e29b-41d4-a716-446655440000"), SourceIRI: "dataset-1", Title: "Dataset", Keywords: "one,two", Created: created, Updated: updated, SourceCreated: &created, SourceUpdated: &sourceUpdated, OriginalDatasetID: testUUID("550e8400-e29b-41d4-a716-446655440000"), Status: "PUBLISHED",
		Categories:      []clients.Category{{ID: testUUID("550e8400-e29b-41d4-a716-446655440001"), SourceIRI: "concept-1", Name: "Concept"}},
		Providers:       []clients.Organization{{ID: testUUID("550e8400-e29b-41d4-a716-446655440002"), SourceIRI: "organization-1", Name: "Organization"}},
		Distributions:   []clients.Distribution{{ID: testUUID("550e8400-e29b-41d4-a716-446655440003"), SourceIRI: "distribution-1", Title: "Distribution"}},
		SpatialCoverage: &clients.Spatial{Point: &clients.Coordinate{Latitude: 45.1, Longitude: 7.2}},
	})
	if err != nil {
		t.Fatalf("mapDataset() error = %v", err)
	}
	if dataset.Status != api.PublishedStatus || dataset.Spatial != "POINT(7.2 45.1)" || dataset.Created == nil || !dataset.Created.Equal(created) || dataset.Modified == nil || !dataset.Modified.Equal(updated) || dataset.SourceUpdated == nil || !dataset.SourceUpdated.Equal(sourceUpdated) || dataset.IsVersionOf != nil {
		t.Fatalf("dataset = %#v", dataset)
	}
	if dataset.ID != "550e8400-e29b-41d4-a716-446655440000" || dataset.Concepts[0].ID != "550e8400-e29b-41d4-a716-446655440001" || dataset.Concepts[0].Href != "v1/resources/concept/550e8400-e29b-41d4-a716-446655440001" || len(dataset.Keywords) != 2 || len(dataset.Concepts) != 1 || len(dataset.Publisher) != 1 || len(dataset.Distributions) != 1 {
		t.Fatalf("dataset refs = %#v", dataset)
	}
}

func TestMapDatasetVersion(t *testing.T) {
	originalID := testUUID("550e8400-e29b-41d4-a716-446655440001")
	dataset, err := mapDataset(clients.Dataset{
		ID:                testUUID("550e8400-e29b-41d4-a716-446655440002"),
		OriginalDatasetID: originalID,
		Status:            "PUBLISHED",
	})
	if err != nil {
		t.Fatalf("mapDataset() error = %v", err)
	}
	if dataset.IsVersionOf == nil || dataset.IsVersionOf.ID != originalID.String() || dataset.IsVersionOf.Href != "v1/resources/dataset/550e8400-e29b-41d4-a716-446655440001" {
		t.Fatalf("is_version_of = %#v", dataset.IsVersionOf)
	}
}

func TestMapDistributionWebservice(t *testing.T) {
	parameter := clients.Parameter{
		Metadata: clients.ParameterMetadata{SourceIRI: "parameter-1", Kind: "STRING", Variable: "author", Label: "Author", Property: "author", Required: false, MultipleValues: false, ReadOnly: false},
	}
	distribution, err := mapDistribution(clients.Distribution{
		ID: testUUID("550e8400-e29b-41d4-a716-446655440004"), SourceIRI: "distribution-1", Title: "Distribution", Type: string(api.DistributionKindWebService),
		WebService: &clients.WebService{ID: testUUID("550e8400-e29b-41d4-a716-446655440005"), SourceIRI: "webservice-1", Name: "Webservice", TemplateURL: "https://example.org/{id}", Parameters: []clients.Parameter{parameter}, Provider: &clients.Organization{ID: testUUID("550e8400-e29b-41d4-a716-446655440006"), Name: "Organization"}},
	})
	if err != nil {
		t.Fatalf("mapDistribution() error = %v", err)
	}
	if distribution.Webservice == nil || distribution.Webservice.ID != "550e8400-e29b-41d4-a716-446655440005" || distribution.Operation == nil || distribution.Operation.IriTemplate.Template == "" || distribution.Operation.IriTemplate.Mappings[0].Range != "STRING" {
		t.Fatalf("distribution = %#v", distribution)
	}
}

func TestMapWebserviceWithoutProvider(t *testing.T) {
	webservice := mapWebservice(clients.WebService{
		ID:          testUUID("550e8400-e29b-41d4-a716-446655440014"),
		Name:        "Webservice",
		TemplateURL: "https://example.org/{id}",
	})
	if webservice.Provider != nil {
		t.Fatalf("provider = %#v, want nil", webservice.Provider)
	}
}

func TestMapConcepts(t *testing.T) {
	scheme := clients.CategoryScheme{ID: testUUID("550e8400-e29b-41d4-a716-446655440006"), SourceIRI: "scheme-1", Name: "Scheme"}
	concept := mapConcept(clients.Category{ID: testUUID("550e8400-e29b-41d4-a716-446655440007"), SourceIRI: "concept-1", Name: "Concept", Scheme: scheme})
	mappedScheme := mapConceptScheme(clients.CategoryScheme{ID: testUUID("550e8400-e29b-41d4-a716-446655440006"), SourceIRI: "scheme-1", Name: "Scheme", TopConcepts: []clients.Category{{ID: testUUID("550e8400-e29b-41d4-a716-446655440007"), SourceIRI: "concept-1", Name: "Concept"}}})
	if concept.ID != "550e8400-e29b-41d4-a716-446655440007" || concept.InScheme == nil || concept.InScheme.ID != "550e8400-e29b-41d4-a716-446655440006" || len(mappedScheme.TopConcepts) != 1 {
		t.Fatalf("concept = %#v, scheme = %#v", concept, mappedScheme)
	}
}

func TestMapSoftwareApplication(t *testing.T) {
	mainEntityOfPage := "https://example.org/software"
	software := mapSoftwareApplication(clients.SoftwareApplication{
		ID: testUUID("550e8400-e29b-41d4-a716-446655440008"), SourceIRI: "software-1", Name: "Software", Keywords: "one,two", MainEntityOfPage: &mainEntityOfPage,
		Creators:    []clients.Organization{{ID: testUUID("550e8400-e29b-41d4-a716-446655440009"), SourceIRI: "organization-1", Name: "Organization"}},
		Identifiers: []clients.Identifier{{Type: "DOI", Value: "10.1/example"}},
		Categories:  []clients.Category{{ID: testUUID("550e8400-e29b-41d4-a716-446655440010"), SourceIRI: "concept-1", Name: "Concept"}},
	})
	if software.ID != "550e8400-e29b-41d4-a716-446655440008" || len(software.Keywords) != 2 || len(software.Creator) != 1 || len(software.Identifiers) != 1 || len(software.Concepts) != 1 {
		t.Fatalf("software = %#v", software)
	}
}

func TestMapSoftwareSourceCode(t *testing.T) {
	repository := "https://example.org/repository"
	software := mapSoftwareSourceCode(clients.SoftwareSourceCode{
		ID: testUUID("550e8400-e29b-41d4-a716-446655440011"), SourceIRI: "source-1", Name: "Source", Keywords: "one,two", Repository: &repository,
		Creators:    []clients.Organization{{ID: testUUID("550e8400-e29b-41d4-a716-446655440012"), SourceIRI: "organization-1", Name: "Organization"}},
		Identifiers: []clients.Identifier{{Type: "DOI", Value: "10.1/example"}},
		Categories:  []clients.Category{{ID: testUUID("550e8400-e29b-41d4-a716-446655440013"), SourceIRI: "concept-1", Name: "Concept"}},
	})
	if software.ID != "550e8400-e29b-41d4-a716-446655440011" || software.Repository != repository || len(software.Keywords) != 2 || len(software.Creator) != 1 || len(software.Identifiers) != 1 || len(software.Concepts) != 1 {
		t.Fatalf("software = %#v", software)
	}
}
