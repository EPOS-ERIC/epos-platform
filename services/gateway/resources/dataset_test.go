package resources

import (
	"context"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

const (
	titleKey               = "dct:title"
	typeKey                = "type"
	accessURLKey           = "dcat:accessURL"
	operationID1           = "operation-1"
	operationType          = "operation"
	iriTemplateType        = "iritemplate"
	iriTemplateID          = "iri-template-1"
	iriTemplateMappingType = "iritemplatemapping"
	mappingID1             = "mapping-1"
	variableKey            = "variable"
	rangeKey               = "range"
	xsdStringRange         = "xsd:string"
	requiredKey            = "required"
	distributionTitle      = "Distribution title"
	identifierShape        = "identifier"
)

func TestMapDataset(t *testing.T) {
	t.Parallel()

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/dataset/intermagnet.json")
	fetch := newFixtureAccessDetailFetcher(t, map[string]string{
		fixtureKey("organization", currentAHEADPublisherID):            "testdata/metadata-lifecycle/access/organization/pic.json",
		fixtureKey(identifierShape, currentAHEADDatasetIdentifierDDSS): "testdata/metadata-lifecycle/access/identifier/intermagnet-ddss-id.json",
		fixtureKey(identifierShape, currentAHEADDatasetIdentifierDOI):  "testdata/metadata-lifecycle/access/identifier/intermagnet-doi.json",
		fixtureKey("periodoftime", currentAHEADPeriodOfTimeID):         "testdata/metadata-lifecycle/access/periodoftime/intermagnet.json",
		fixtureKey("location", currentAHEADLocationID):                 "testdata/metadata-lifecycle/access/location/intermagnet.json",
		fixtureKey("annotation", currentAHEADAnnotationID):             "testdata/metadata-lifecycle/access/annotation/intermagnet.json",
	})

	dataset, err := mapDataset(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapDataset() error = %v", err)
	}

	if dataset.ID != currentAHEADDatasetID {
		t.Fatalf("dataset.ID = %q", dataset.ID)
	}
	if dataset.Name != "TEST1 - Historical earthquakes 1000-1899" {
		t.Fatalf("dataset.Name = %q", dataset.Name)
	}
	if dataset.Description == "" {
		t.Fatalf("dataset.Description = %q", dataset.Description)
	}
	if dataset.VersionInfo != "" {
		t.Fatalf("dataset.VersionInfo = %q", dataset.VersionInfo)
	}
	if dataset.AccessRights != "https://publications.europa.eu/resource/authority/access-right/PUBLIC" {
		t.Fatalf("dataset.AccessRights = %q", dataset.AccessRights)
	}
	if dataset.AccrualPeriodicity != "http://purl.org/cld/freq/irregular" {
		t.Fatalf("dataset.AccrualPeriodicity = %q", dataset.AccrualPeriodicity)
	}
	if dataset.QualityAnnotation != "https://www.emidius.eu/AHEAD/data_quality_assurance.php" {
		t.Fatalf("dataset.QualityAnnotation = %q", dataset.QualityAnnotation)
	}
	if dataset.Spatial != "POLYGON((-34 74,45 74,45 33,-34 33,-34 74))" {
		t.Fatalf("dataset.Spatial = %q", dataset.Spatial)
	}
	if len(dataset.Identifiers) != 2 {
		t.Fatalf("dataset.Identifiers = %#v", dataset.Identifiers)
	}
	if dataset.Identifiers[0] != (api.Identifier{Type: "DDSS-ID", Value: "WP08-DDSS-024"}) {
		t.Fatalf("dataset.Identifiers[0] = %#v", dataset.Identifiers[0])
	}
	if dataset.Identifiers[1] != (api.Identifier{Type: "DOI", Value: "10.6092/INGV.IT-AHEAD"}) {
		t.Fatalf("dataset.Identifiers[1] = %#v", dataset.Identifiers[1])
	}
	if dataset.Temporal == nil {
		t.Fatal("dataset.Temporal = nil")
	}
	assertTimeEqual(t, dataset.Created, "2010-06-01T00:00:00Z")
	assertTimeEqual(t, dataset.Modified, "2018-09-10T00:00:00Z")
	if dataset.Issued != nil {
		t.Fatalf("dataset.Issued = %#v", dataset.Issued)
	}
	assertTimeEqual(t, dataset.Temporal.Start, "1000-01-01T00:00:00Z")
	assertTimeEqual(t, dataset.Temporal.End, "1899-12-31T23:59:59Z")
	if len(dataset.Concepts) != 1 || dataset.Concepts[0].Href != "v1/concept/category:earthquakeparameters/v1.0" {
		t.Fatalf("dataset.Concepts = %#v", dataset.Concepts)
	}
	if len(dataset.Publisher) != 1 || dataset.Publisher[0].Href != currentAHEADPublisherHref || dataset.Publisher[0].Name != "INGV - Istituto Nazionale di Geofisica e Vulcanologia" {
		t.Fatalf("dataset.Publisher = %#v", dataset.Publisher)
	}
	if len(dataset.Distributions) != 1 || dataset.Distributions[0].Href != currentAHEADDistributionHref {
		t.Fatalf("dataset.Distributions = %#v", dataset.Distributions)
	}
	if len(dataset.ContactPoints) != 2 {
		t.Fatalf("dataset.ContactPoints = %#v", dataset.ContactPoints)
	}
	assertContactPointHrefs(t, dataset.ContactPoints,
		"http://localhost:8080/api/v2/access/contactpoint/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FSeismology%2FFacility%2FAHEAD%2FcontactPoint%2Fv1.0",
		"http://localhost:8080/api/v2/access/contactpoint/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FSeismology%2FFacility%2FAHEAD%2FscientificContact%2Fv1.0",
	)
	if dataset.IsVersionOf != nil {
		t.Fatalf("dataset.IsVersionOf = %#v", dataset.IsVersionOf)
	}
}

func assertTimeEqual(t *testing.T, got *time.Time, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("time is nil, want %s", want)
	}

	parsedWant, err := time.Parse(time.RFC3339, want)
	if err != nil {
		t.Fatalf("parse want time: %v", err)
	}

	if !got.Equal(parsedWant) {
		t.Fatalf("time = %s, want %s", got.Format(time.RFC3339), parsedWant.Format(time.RFC3339))
	}
}
