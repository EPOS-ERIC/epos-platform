package turtle

import (
	"os"
	"strings"
	"testing"
	"time"

	domain "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
)

func TestParseMetadataOne(t *testing.T) {
	file, err := os.Open("../../../packages/assets/metadatone.ttl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close fixture: %v", err)
		}
	}()

	batch, err := Parse(file, Options{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(batch.Datasets) != 3 {
		t.Fatalf("datasets = %d, want 3", len(batch.Datasets))
	}
	if len(batch.Categories) == 0 || len(batch.CategorySchemes) == 0 {
		t.Fatalf("categories = %d, schemes = %d", len(batch.Categories), len(batch.CategorySchemes))
	}
	if len(batch.Organizations) == 0 || len(batch.ContactPoints) == 0 {
		t.Fatalf("organizations = %d, contacts = %d", len(batch.Organizations), len(batch.ContactPoints))
	}
	for _, dataset := range batch.Datasets {
		if len(dataset.Distributions) == 0 {
			t.Errorf("dataset %q has no distributions", dataset.SourceIRI)
		}
		if dataset.SourceIRI == "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/bibliography" {
			created := time.Date(2010, 6, 1, 0, 0, 0, 0, time.UTC)
			updated := time.Date(2018, 9, 10, 0, 0, 0, 0, time.UTC)
			if dataset.SourceCreated == nil || !dataset.SourceCreated.Equal(created) {
				t.Errorf("bibliography SourceCreated = %v, want %v", dataset.SourceCreated, created)
			}
			if dataset.SourceUpdated == nil || !dataset.SourceUpdated.Equal(updated) {
				t.Errorf("bibliography SourceUpdated = %v, want %v", dataset.SourceUpdated, updated)
			}
		}
		if dataset.SourceIRI == "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events" {
			keywords := make(map[string]bool)
			for keyword := range strings.SplitSeq(dataset.Keywords, ",") {
				keywords[keyword] = true
			}
			for _, keyword := range []string{"event", "earthquake", "seismicity", "catalogue", "magnitude"} {
				if !keywords[keyword] {
					t.Errorf("events keywords = %q", dataset.Keywords)
				}
			}
			if dataset.AccrualPeriodicity == nil || *dataset.AccrualPeriodicity != "http://purl.org/cld/freq/irregular" {
				t.Errorf("events accrual periodicity = %v", dataset.AccrualPeriodicity)
			}
			if dataset.QualityAnnotation == nil || *dataset.QualityAnnotation != "https://www.emidius.eu/AHEAD/data_quality_assurance.php" {
				t.Errorf("events quality annotation = %v", dataset.QualityAnnotation)
			}
			if dataset.AccessRight == nil || *dataset.AccessRight != "https://publications.europa.eu/resource/authority/access-right/PUBLIC" {
				t.Errorf("events access right = %v", dataset.AccessRight)
			}
		}
		for _, distribution := range dataset.Distributions {
			if distribution.WebService != nil && distribution.Type != domain.DistributionTypeWebService {
				t.Errorf("web service distribution type = %q", distribution.Type)
			}
			if distribution.DownloadableFile != nil && distribution.Type != domain.DistributionTypeDownloadableFile {
				t.Errorf("downloadable distribution type = %q", distribution.Type)
			}
			if distribution.WebService != nil && (distribution.Issued == nil || distribution.Modified == nil || !distribution.Issued.Equal(time.Date(2017, 10, 25, 0, 0, 0, 0, time.UTC)) || !distribution.Modified.Equal(time.Date(2017, 10, 25, 0, 0, 0, 0, time.UTC))) {
				t.Errorf("web service distribution dates = (%v, %v)", distribution.Issued, distribution.Modified)
			}
			if distribution.DownloadableFile != nil && (distribution.ByteSize == nil || *distribution.ByteSize != 72981156) {
				t.Errorf("downloadable distribution byte size = %v", distribution.ByteSize)
			}
		}
		for _, identifier := range dataset.PersistentIdentifiers {
			if strings.ContainsRune(identifier.SourceIRI, '\x00') {
				t.Errorf("identifier %q contains NUL byte", identifier.SourceIRI)
			}
		}
	}

	schemes := make(map[string]domain.CategoryScheme, len(batch.CategorySchemes))
	for _, scheme := range batch.CategorySchemes {
		schemes[scheme.SourceIRI] = scheme
	}
	if scheme := schemes["category:seismology"]; scheme.Label == nil || *scheme.Label != "SEI" || scheme.Logo == nil || scheme.Page == nil || scheme.Color == nil || scheme.Position != 1 {
		t.Errorf("seismology scheme fields = %#v", scheme)
	}
	if scheme := schemes["category:Volcanology"]; scheme.Description == nil || *scheme.Description != "Facility Domain" {
		t.Errorf("volcanology scheme description = %v", scheme.Description)
	}

	organizations := make(map[string]domain.Organization, len(batch.Organizations))
	for _, organization := range batch.Organizations {
		organizations[organization.SourceIRI] = organization
	}
	for source, want := range map[string]struct {
		street, locality, postalCode, country string
	}{
		"pic:999472675": {
			street: "Via di Vigna Murata 605", locality: "Rome", postalCode: "00143", country: "IT",
		},
		"https://doi.org/10.7914/SN/IT": {
			street: "Via Roma 30", locality: "Roma", postalCode: "00143", country: "IT",
		},
	} {
		organization, ok := organizations[source]
		if !ok || organization.Street == nil || organization.Locality == nil || organization.PostalCode == nil || organization.Country == nil {
			t.Errorf("organization %q address is incomplete", source)
			continue
		}
		if *organization.Street != want.street || *organization.Locality != want.locality || *organization.PostalCode != want.postalCode || *organization.Country != want.country {
			t.Errorf("organization %q address = (%q, %q, %q, %q)", source, *organization.Street, *organization.Locality, *organization.PostalCode, *organization.Country)
		}
	}
	if organization := organizations["pic:999472675"]; organization.LEICode == nil || !strings.Contains(*organization.LEICode, "Legal Entity Identifier") {
		t.Errorf("organization lei code = %v", organization.LEICode)
	}

	for _, dataset := range batch.Datasets {
		for _, distribution := range dataset.Distributions {
			if distribution.WebService == nil {
				continue
			}
			service := distribution.WebService
			if service.Provider == nil || service.Provider.SourceIRI != "pic:999472675" {
				t.Errorf("web service provider = %#v", service.Provider)
			}
			if len(service.ContactPoints) != 2 {
				t.Errorf("web service contacts = %d, want 2", len(service.ContactPoints))
			}
		}
	}

	var eventService *domain.WebService
	var bibliographyService *domain.WebService
	for _, dataset := range batch.Datasets {
		switch dataset.SourceIRI {
		case "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events":
			eventService = dataset.Distributions[0].WebService
		case "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/bibliography":
			bibliographyService = dataset.Distributions[0].WebService
		}
	}
	if eventService == nil || bibliographyService == nil {
		t.Fatal("expected web services not found")
	}
	parameters := make(map[string]domain.Parameter, len(eventService.Parameters))
	for _, parameter := range eventService.Parameters {
		parameters[parameter.GetMetadata().Variable] = parameter
	}
	for variable, kind := range map[string]domain.ParameterKind{
		"starttime":         domain.ParameterKindDateTime,
		"minlatitude":       domain.ParameterKindFloat,
		"includeallorigins": domain.ParameterKindBoolean,
		"limit":             domain.ParameterKindInteger,
	} {
		parameter, ok := parameters[variable]
		if !ok {
			t.Errorf("parameter %q not found", variable)
			continue
		}
		if parameter.Kind() != kind {
			t.Errorf("parameter %q kind = %v, want %v", variable, parameter.Kind(), kind)
		}
	}
	startTime, ok := parameters["starttime"].(domain.DateTimeParameter)
	start := time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(1899, 12, 31, 23, 59, 59, 0, time.UTC)
	if !ok || startTime.Minimum == nil || startTime.Maximum == nil || startTime.Default == nil || !startTime.Minimum.Equal(start) || !startTime.Maximum.Equal(end) || !startTime.Default.Equal(start) {
		t.Errorf("starttime bounds = %#v", parameters["starttime"])
	}
	bibliographyParameters := make(map[string]domain.Parameter, len(bibliographyService.Parameters))
	for _, parameter := range bibliographyService.Parameters {
		bibliographyParameters[parameter.GetMetadata().Variable] = parameter
	}
	yearStart, ok := bibliographyParameters["yearstart"].(domain.DateTimeParameter)
	year := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	if !ok || yearStart.Minimum == nil || yearStart.Default == nil || !yearStart.Minimum.Equal(year) || !yearStart.Default.Equal(year) {
		t.Errorf("yearstart bounds = %#v", bibliographyParameters["yearstart"])
	}
	minLatitude, ok := parameters["minlatitude"].(domain.FloatParameter)
	if !ok || minLatitude.Minimum == nil || minLatitude.Maximum == nil || *minLatitude.Minimum != 33 || *minLatitude.Maximum != 74 {
		t.Errorf("minlatitude bounds = %#v", parameters["minlatitude"])
	}
}

func TestOwnedIDsAreDifferentForDifferentParents(t *testing.T) {
	first := ownedID("dataset-a", "web-service", "service")
	second := ownedID("dataset-b", "web-service", "service")
	if first == second {
		t.Fatal("owned IDs reused across parents")
	}
	firstShared := sharedID("organization", "org")
	secondShared := sharedID("organization", "org")
	if firstShared != secondShared {
		t.Fatal("shared IDs are not deterministic")
	}
}

func TestParseStandaloneSoftware(t *testing.T) {
	const ttl = `
@prefix dcat: <http://www.w3.org/ns/dcat#> .
@prefix schema: <http://schema.org/> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
<scheme> skos:hasTopConcept <category:facets/software-theme>; skos:prefLabel "Software" .
<concept> skos:inScheme <scheme>; skos:prefLabel "Tools" .
<org> a schema:Organization; schema:name "EPOS" .
<app> a schema:SoftwareApplication; schema:name "App"; schema:softwareVersion "1"; schema:creator <org>; dcat:theme <concept> .
<source> a schema:SoftwareSourceCode; schema:name "Source"; schema:codeRepository "https://example.test/repo"; schema:creator <org>; dcat:theme <concept> .
`
	batch, err := Parse(strings.NewReader(ttl), Options{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(batch.Datasets) != 0 || len(batch.SoftwareApplications) != 1 || len(batch.SoftwareSourceCodes) != 1 {
		t.Fatalf("resource counts = datasets:%d applications:%d source codes:%d", len(batch.Datasets), len(batch.SoftwareApplications), len(batch.SoftwareSourceCodes))
	}
	application := batch.SoftwareApplications[0]
	if application.Version == nil || *application.Version != "1" || len(application.Creators) != 1 || len(application.Categories) != 1 {
		t.Errorf("application = %#v", application)
	}
	source := batch.SoftwareSourceCodes[0]
	if source.Repository == nil || *source.Repository != "https://example.test/repo" || len(source.Creators) != 1 || len(source.Categories) != 1 {
		t.Errorf("source = %#v", source)
	}
}

func TestParseClonesOwnedWebServicesPerDataset(t *testing.T) {
	const ttl = `
@prefix dcat: <http://www.w3.org/ns/dcat#> .
@prefix dct: <http://purl.org/dc/terms/> .
@prefix epos: <https://www.epos-eu.org/epos-dcat-ap#> .
@prefix hydra: <http://www.w3.org/ns/hydra/core#> .
@prefix schema: <http://schema.org/> .
<provider> a schema:Organization; schema:name "Provider" .
<dataset-a> a dcat:Dataset; dct:title "A"; dcat:distribution <distribution-a> .
<dataset-b> a dcat:Dataset; dct:title "B"; dcat:distribution <distribution-b> .
<distribution-a> a dcat:Distribution; dct:title "A"; dct:format "json"; dct:conformsTo <service> .
<distribution-b> a dcat:Distribution; dct:title "B"; dct:format "json"; dct:conformsTo <service> .
<service> a epos:WebService; schema:name "Shared source"; schema:provider <provider>; hydra:supportedOperation <operation> .
<operation> hydra:property <template> .
<template> hydra:template "https://example.test" .
`

	batch, err := Parse(strings.NewReader(ttl), Options{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(batch.Datasets) != 2 || batch.Datasets[0].Distributions[0].WebService == nil || batch.Datasets[1].Distributions[0].WebService == nil {
		t.Fatalf("expected two datasets with web services")
	}
	if batch.Datasets[0].Distributions[0].WebService.ID == batch.Datasets[1].Distributions[0].WebService.ID {
		t.Fatal("owned web service was reused")
	}
}

func TestParseDatasetSourceTimestamps(t *testing.T) {
	const ttl = `
@prefix dcat: <http://www.w3.org/ns/dcat#> .
@prefix dct: <http://purl.org/dc/terms/> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
<dataset-with-dates> a dcat:Dataset;
  dct:title "With dates";
  dct:created "2020-01-02"^^xsd:date;
  dct:modified "2021-02-03T04:05:06Z"^^xsd:dateTime .
<dataset-without-dates> a dcat:Dataset; dct:title "Without dates" .
`

	batch, err := Parse(strings.NewReader(ttl), Options{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	for _, dataset := range batch.Datasets {
		switch dataset.SourceIRI {
		case "dataset-with-dates":
			created := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
			updated := time.Date(2021, 2, 3, 4, 5, 6, 0, time.UTC)
			if dataset.SourceCreated == nil || !dataset.SourceCreated.Equal(created) {
				t.Errorf("SourceCreated = %v, want %v", dataset.SourceCreated, created)
			}
			if dataset.SourceUpdated == nil || !dataset.SourceUpdated.Equal(updated) {
				t.Errorf("SourceUpdated = %v, want %v", dataset.SourceUpdated, updated)
			}
			if !dataset.Created.IsZero() || !dataset.Updated.IsZero() {
				t.Errorf("service timestamps = (%v, %v), want zero values", dataset.Created, dataset.Updated)
			}
		case "dataset-without-dates":
			if dataset.SourceCreated != nil || dataset.SourceUpdated != nil {
				t.Errorf("source timestamps = (%v, %v), want nil", dataset.SourceCreated, dataset.SourceUpdated)
			}
		}
	}
}
