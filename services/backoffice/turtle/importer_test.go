package turtle

import (
	"os"
	"strings"
	"testing"

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
		for _, identifier := range dataset.PersistentIdentifiers {
			if strings.ContainsRune(identifier.SourceIRI, '\x00') {
				t.Errorf("identifier %q contains NUL byte", identifier.SourceIRI)
			}
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
