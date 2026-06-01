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
	descriptionKey         = "dct:description"
	accessURLKey           = "dcat:accessURL"
	operationID1           = "operation-1"
	operationType          = "operation"
	iriTemplateType        = "iritemplate"
	iriTemplateID          = "iri-template-1"
	iriTemplateMappingType = "iritemplatemapping"
	mappingID1             = "mapping-1"
	variableKey            = "variable"
	stationCodeLabel       = "Station code"
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
		fixtureKey(identifierShape, "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-identifier-34096b33/v1.0"):  "testdata/metadata-lifecycle/access/identifier/intermagnet-doi.json",
		fixtureKey(identifierShape, "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-identifier-70ea05a7/v1.0"):  "testdata/metadata-lifecycle/access/identifier/intermagnet-timespan-doi.json",
		fixtureKey(identifierShape, "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-identifier-b1f9aa83/v1.0"):  "testdata/metadata-lifecycle/access/identifier/intermagnet-ddss-id.json",
		fixtureKey("periodoftime", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-periodoftime-0f4b2f01/v1.0"): "testdata/metadata-lifecycle/access/periodoftime/intermagnet.json",
		fixtureKey("location", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-location-630d862a/v1.0"):         "testdata/metadata-lifecycle/access/location/intermagnet.json",
		fixtureKey("annotation", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-annotation-0970cbef/v1.0"):     "testdata/metadata-lifecycle/access/annotation/intermagnet.json",
	})

	dataset, err := mapDataset(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapDataset() error = %v", err)
	}

	if dataset.ID != "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Dataset/v1.0" {
		t.Fatalf("dataset.ID = %q", dataset.ID)
	}
	if dataset.Name != "INTERMAGNET Geomagnetic Observatory Data" {
		t.Fatalf("dataset.Name = %q", dataset.Name)
	}
	if dataset.Description == "" {
		t.Fatalf("dataset.Description = %q", dataset.Description)
	}
	if dataset.VersionInfo != "1.0.0" {
		t.Fatalf("dataset.VersionInfo = %q", dataset.VersionInfo)
	}
	if dataset.AccessRights != "https://publications.europa.eu/resource/authority/access-right/PUBLIC" {
		t.Fatalf("dataset.AccessRights = %q", dataset.AccessRights)
	}
	if dataset.AccrualPeriodicity != "http://purl.org/cld/freq/continuous" {
		t.Fatalf("dataset.AccrualPeriodicity = %q", dataset.AccrualPeriodicity)
	}
	if dataset.QualityAnnotation != "https://intermagnet.github.io/docs/Technical-Manual/technical_manual.pdf" {
		t.Fatalf("dataset.QualityAnnotation = %q", dataset.QualityAnnotation)
	}
	if dataset.Spatial != "POLYGON((180.0 -90.0, -180.0 -90.0, -180.0 90.0, 180.0 90.0, 180.0 -90.0))" {
		t.Fatalf("dataset.Spatial = %q", dataset.Spatial)
	}
	if len(dataset.Identifiers) != 3 {
		t.Fatalf("dataset.Identifiers = %#v", dataset.Identifiers)
	}
	if dataset.Identifiers[0] != (api.Identifier{Type: "DOI", Value: "10.5880/INTERMAGNET.2013"}) {
		t.Fatalf("dataset.Identifiers[0] = %#v", dataset.Identifiers[0])
	}
	if dataset.Temporal == nil {
		t.Fatal("dataset.Temporal = nil")
	}
	assertTimeEqual(t, dataset.Created, "2018-08-31T00:00:00Z")
	assertTimeEqual(t, dataset.Modified, "2018-08-31T00:00:00Z")
	assertTimeEqual(t, dataset.Issued, "2018-08-31T00:00:00Z")
	assertTimeEqual(t, dataset.Temporal.Start, "1991-01-01T00:00:00Z")
	if dataset.Temporal.End != nil {
		t.Fatalf("dataset.Temporal.End = %#v", dataset.Temporal.End)
	}
	if len(dataset.Concepts) != 1 || dataset.Concepts[0].Href != "v1/concept/category:geomagneticdata/v1.0" {
		t.Fatalf("dataset.Concepts = %#v", dataset.Concepts)
	}
	if len(dataset.Publisher) != 1 || dataset.Publisher[0].Href != "v1/organization/http://viaf.org/viaf/136554353//v1.0" {
		t.Fatalf("dataset.Publisher = %#v", dataset.Publisher)
	}
	if len(dataset.Distributions) != 1 || dataset.Distributions[0].Href != "v1/distribution/https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Distribution/v1.0" {
		t.Fatalf("dataset.Distributions = %#v", dataset.Distributions)
	}
	if len(dataset.ContactPoints) != 1 || dataset.ContactPoints[0].Href != "http://localhost:8080/api/v2/access/contactpoint/mailto%3Aochambers%40bgs.ac.uk%2FcontactPoint%2Fv1.0" {
		t.Fatalf("dataset.ContactPoints = %#v", dataset.ContactPoints)
	}
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
