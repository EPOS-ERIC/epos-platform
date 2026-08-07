package turtle

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	domain "github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
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

	batch, err := Parse(file, Options{
		Editor: domain.User{ID: uuid.New(), AuthIdentifier: "importer", Status: domain.UserStatusActive},
		Group:  domain.Group{ID: uuid.New(), Slug: "all", Name: "ALL"},
	})
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
		for _, identifier := range dataset.PersistentIdentifiers {
			if strings.ContainsRune(identifier.SourceIRI, '\x00') {
				t.Errorf("identifier %q contains NUL byte", identifier.SourceIRI)
			}
		}
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

	for _, dataset := range batch.Datasets {
		for _, distribution := range dataset.Distributions {
			if distribution.WebService == nil {
				continue
			}
			service := distribution.WebService
			if len(service.Providers) != 1 || service.Providers[0].SourceIRI != "pic:999472675" {
				t.Errorf("web service %q providers = %#v", service.SourceIRI, service.Providers)
			}
			if len(service.ContactPoints) != 2 {
				t.Errorf("web service %q contacts = %d, want 2", service.SourceIRI, len(service.ContactPoints))
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
	} else {
		record, err := domain.EncodeParameter(startTime)
		if err != nil {
			t.Fatalf("encode starttime parameter: %v", err)
		}
		var config map[string]string
		if err := json.Unmarshal(record.Config, &config); err != nil {
			t.Fatalf("decode starttime config: %v", err)
		}
		if config["minimum"] != "1000-01-01T00:00:00Z" || config["maximum"] != "1899-12-31T23:59:59Z" || config["default"] != "1000-01-01T00:00:00Z" {
			t.Errorf("starttime config = %s", record.Config)
		}
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
	} else {
		record, err := domain.EncodeParameter(minLatitude)
		if err != nil {
			t.Fatalf("encode minlatitude parameter: %v", err)
		}
		var config map[string]float64
		if err := json.Unmarshal(record.Config, &config); err != nil {
			t.Fatalf("decode minlatitude config: %v", err)
		}
		if config["minimum"] != 33 || config["maximum"] != 74 || config["default"] != 33 {
			t.Errorf("minlatitude config = %s", record.Config)
		}
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

func TestParseClonesOwnedWebServicesPerDataset(t *testing.T) {
	const ttl = `
@prefix dcat: <http://www.w3.org/ns/dcat#> .
@prefix dct: <http://purl.org/dc/terms/> .
@prefix epos: <https://www.epos-eu.org/epos-dcat-ap#> .
@prefix hydra: <http://www.w3.org/ns/hydra/core#> .
@prefix schema: <http://schema.org/> .
<dataset-a> a dcat:Dataset; dct:title "A"; dcat:distribution <distribution-a> .
<dataset-b> a dcat:Dataset; dct:title "B"; dcat:distribution <distribution-b> .
<distribution-a> a dcat:Distribution; dct:title "A"; dct:format "json"; dct:conformsTo <service> .
<distribution-b> a dcat:Distribution; dct:title "B"; dct:format "json"; dct:conformsTo <service> .
<service> a epos:WebService; schema:name "Shared source"; hydra:supportedOperation <operation> .
<operation> hydra:property <template> .
<template> hydra:template "https://example.test" .
`

	batch, err := Parse(strings.NewReader(ttl), Options{
		Editor: domain.User{ID: uuid.New(), AuthIdentifier: "importer", Status: domain.UserStatusActive},
		Group:  domain.Group{ID: uuid.New(), Slug: "all", Name: "ALL"},
	})
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

	batch, err := Parse(strings.NewReader(ttl), Options{
		Editor: domain.User{ID: uuid.New(), AuthIdentifier: "importer", Status: domain.UserStatusActive},
		Group:  domain.Group{ID: uuid.New(), Slug: "all", Name: "ALL"},
	})
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
