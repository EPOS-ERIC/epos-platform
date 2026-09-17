package turtle

import (
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

const importPrefixes = `
@prefix dcat: <http://www.w3.org/ns/dcat#> .
@prefix dct: <http://purl.org/dc/terms/> .
@prefix epos: <https://www.epos-eu.org/epos-dcat-ap#> .
@prefix http: <http://www.w3.org/2006/http#> .
@prefix schema: <http://schema.org/> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix hydra: <http://www.w3.org/ns/hydra/core#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
`

func TestParserUsesFallbackDescriptionAndKeepsDatasetWhenContactPointIsInvalid(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dcat:theme <category:one>;
    dcat:contactPoint <contact:bad>.
<contact:bad> a schema:ContactPoint;
    schema:contactType "manager".
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 {
		t.Fatalf("Datasets = %d, want 1", len(batch.Datasets))
	}
	if batch.Datasets[0].Description != "Dataset one description" {
		t.Fatalf("dataset description = %q, want %q", batch.Datasets[0].Description, "Dataset one description")
	}
	if len(batch.ContactPoints) != 0 {
		t.Fatalf("ContactPoints = %d, want 0", len(batch.ContactPoints))
	}
	assertDiagnostic(t, batch.Warnings, "dataset:one", dcatContactPoint, "omitted")
	assertDiagnostic(t, batch.Warnings, "dataset:one", dctDescription, `using default "Dataset one description"`)
}

func TestParserUsesTitleFallbackDescriptionsForSoftwareResources(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:software> a skos:ConceptScheme;
    dct:title "Software categories";
    skos:hasTopConcept <category:facets/software-theme>.
<category:software-item> a skos:Concept;
    skos:inScheme <category:software>;
    skos:prefLabel "Software".
<application:one> a schema:SoftwareApplication;
    schema:name "Application one";
    dcat:theme <category:software-item>.
<source:one> a schema:SoftwareSourceCode;
    schema:name "Source code one";
    schema:description "   ";
    dcat:theme <category:software-item>.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.SoftwareApplications) != 1 {
		t.Fatalf("SoftwareApplications = %d, want 1", len(batch.SoftwareApplications))
	}
	if batch.SoftwareApplications[0].Description != "Application one description" {
		t.Fatalf("software application description = %q, want %q", batch.SoftwareApplications[0].Description, "Application one description")
	}
	if len(batch.SoftwareSourceCodes) != 1 {
		t.Fatalf("SoftwareSourceCodes = %d, want 1", len(batch.SoftwareSourceCodes))
	}
	if batch.SoftwareSourceCodes[0].Description != "Source code one description" {
		t.Fatalf("software source code description = %q, want %q", batch.SoftwareSourceCodes[0].Description, "Source code one description")
	}
	assertDiagnostic(t, batch.Warnings, "application:one", schemaDescription, `using default "Application one description"`)
	assertDiagnostic(t, batch.Warnings, "source:one", schemaDescription, `using default "Source code one description"`)
}

func TestParserDropsDistributionWithInvalidWebService(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:distribution <distribution:one>.
<distribution:one> a dcat:Distribution;
    dct:title "Distribution one";
    dct:description "Distribution description";
    dct:format "application/json";
    dct:conformsTo <service:bad>.
<service:bad> a epos:WebService;
    schema:name "Service without endpoint".
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 {
		t.Fatalf("Datasets = %d, want 1", len(batch.Datasets))
	}
	if len(batch.Datasets[0].Distributions) != 0 {
		t.Fatalf("Distributions = %d, want 0", len(batch.Datasets[0].Distributions))
	}
	assertDiagnostic(t, batch.Warnings, "service:bad", "", "skipped")
	assertDiagnostic(t, batch.Warnings, "dataset:one", dcatDistributionLink, "omitted")
}

func TestParserAcceptsValidTurtleWithNoImportableEntities(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<contact:bad> a schema:ContactPoint;
    schema:contactType "manager".
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 0 || len(batch.ContactPoints) != 0 {
		t.Fatalf("batch contains importable entities: %+v", batch)
	}
	assertDiagnostic(t, batch.Warnings, "contact:bad", "", "skipped")
}

func TestParserDeduplicatesRepeatedRDFStatements(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:keyword "one";
    dcat:keyword "one";
    dcat:theme <category:one>.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 {
		t.Fatalf("Datasets = %d, want 1", len(batch.Datasets))
	}
	if len(batch.Datasets[0].Keywords) != 1 {
		t.Fatalf("Keywords = %d, want 1", len(batch.Datasets[0].Keywords))
	}
	assertDiagnostic(t, batch.Warnings, "dataset:one", dctTitle, "duplicate RDF statement")
}

func TestParserSkipsDatasetWithConflictingSingleValue(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<dataset:one> a dcat:Dataset;
    dct:title "First title";
    dct:title "Second title";
    dct:description "Dataset description".
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 0 {
		t.Fatalf("Datasets = %d, want 0", len(batch.Datasets))
	}
	assertDiagnostic(t, batch.Warnings, "dataset:one", "", "conflicting values")
}

func TestParserSkipsDatasetWithMultipleCategorySchemes(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:facility> a skos:ConceptScheme;
    dct:title "Facility categories";
    skos:hasTopConcept <category:facets/facility-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<category:two> a skos:Concept;
    skos:inScheme <category:facility>;
    skos:prefLabel "Two".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>, <category:two>.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 0 {
		t.Fatalf("Datasets = %d, want 0", len(batch.Datasets))
	}
	assertDiagnostic(t, batch.Warnings, "dataset:one", "", "exactly one scheme")
}

func TestParserMergesCanonicalContactPoints(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<contact:one> a schema:ContactPoint;
    schema:email "owner@example.com";
    schema:contactType "owner".
<contact:two> a schema:ContactPoint;
    schema:email "owner@example.com";
    schema:contactType "owner".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:contactPoint <contact:one>, <contact:two>.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.ContactPoints) != 1 {
		t.Fatalf("ContactPoints = %d, want 1", len(batch.ContactPoints))
	}
	if len(batch.Datasets) != 1 || len(batch.Datasets[0].ContactPoints) != 1 {
		t.Fatalf("dataset contact points = %+v, want one canonical contact point", batch.Datasets)
	}
	assertDiagnostic(t, batch.Warnings, "contact:two", "", "merged")
}

func TestParserUsesDistributionTypeForWebService(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:distribution <distribution:one>.
<distribution:one> a dcat:Distribution;
    dct:title "Distribution one";
    dct:description "Distribution description";
    dct:type "https://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"^^xsd:anyURI;
    dct:conformsTo <service:one>;
    dcat:accessURL <operation:one>;
    dct:format "application/json".
<service:one> schema:name "Service one";
    hydra:supportedOperation <operation:one>.
<operation:one> hydra:template "https://example.com/data{?id}".
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 || len(batch.Datasets[0].Distributions) != 1 {
		t.Fatalf("dataset distributions = %+v, want one distribution", batch.Datasets)
	}
	distribution := batch.Datasets[0].Distributions[0]
	if distribution.Type != domain.DistributionTypeWebService || distribution.WebService == nil {
		t.Fatalf("distribution = %+v, want web service delivery", distribution)
	}
	if distribution.DownloadableFile != nil {
		t.Fatal("web-service distribution has downloadable file delivery")
	}
}

func TestParserAllowsNumericParametersWithoutAllowedValues(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:distribution <distribution:one>.
<distribution:one> a dcat:Distribution;
    dct:title "Distribution one";
    dct:description "Distribution description";
    dct:type "https://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"^^xsd:anyURI;
    dct:format "application/json";
    dct:conformsTo <service:one>;
    dcat:accessURL <operation:one>.
<service:one> schema:name "Service one";
    hydra:supportedOperation <operation:one>.
<operation:one> hydra:template "https://example.com/data{?count,ratio}";
    hydra:property <property:one>.
<property:one> hydra:mapping <parameter:count>, <parameter:ratio>.
<parameter:count> hydra:variable "count";
    hydra:property "count";
    rdfs:range xsd:integer.
<parameter:ratio> hydra:variable "ratio";
    hydra:property "ratio";
    rdfs:range xsd:float.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 || len(batch.Datasets[0].Distributions) != 1 {
		t.Fatalf("dataset distributions = %+v, want one distribution", batch.Datasets)
	}
	service := batch.Datasets[0].Distributions[0].WebService
	if service == nil || len(service.Parameters) != 2 {
		t.Fatalf("service parameters = %+v, want two parameters; warnings = %+v", service, batch.Warnings)
	}
	for _, parameter := range service.Parameters {
		switch schema := parameter.Schema.(type) {
		case domain.IntegerSchema:
			if schema.Enum != nil {
				t.Fatalf("integer enum = %v, want nil", schema.Enum)
			}
		case domain.FloatSchema:
			if schema.Enum != nil {
				t.Fatalf("float enum = %v, want nil", schema.Enum)
			}
		default:
			t.Fatalf("parameter %q schema = %T, want numeric schema", parameter.Name, parameter.Schema)
		}
	}
}

func TestParserMapsMultipleValuesForEnumParameters(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:distribution <distribution:one>.
<distribution:one> a dcat:Distribution;
    dct:title "Distribution one";
    dct:description "Distribution description";
    dct:type "https://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"^^xsd:anyURI;
    dct:format "application/json";
    dct:conformsTo <service:one>;
    dcat:accessURL <operation:one>.
<service:one> schema:name "Service one";
    hydra:supportedOperation <operation:one>.
<operation:one> hydra:template "https://example.com/data{?value}";
    hydra:property <property:one>.
<property:one> hydra:mapping <parameter:one>.
<parameter:one> hydra:variable "value";
    hydra:property "value";
    schema:multipleValues true;
    http:paramValue "one", "two";
    rdfs:range xsd:string.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 || len(batch.Datasets[0].Distributions) != 1 {
		t.Fatalf("dataset distributions = %+v, want one distribution", batch.Datasets)
	}
	service := batch.Datasets[0].Distributions[0].WebService
	if service == nil || len(service.Parameters) != 1 {
		t.Fatalf("service parameters = %+v, want one parameter", service)
	}
	schema, ok := service.Parameters[0].Schema.(domain.StringSchema)
	if !ok || !schema.MultipleValues {
		t.Fatal("MultipleValues = false, want true")
	}
	if service.Parameters[0].Explode {
		t.Fatal("Explode = true, want false for imported parameters")
	}
	if service.Type != domain.WebServiceTypeGeoJSON {
		t.Fatalf("web service type = %q, want GEOJSON", service.Type)
	}
	if service.VisibleOn != nil {
		t.Fatalf("visible on = %v, want nil", service.VisibleOn)
	}
	schema, ok = service.Parameters[0].Schema.(domain.StringSchema)
	if !ok || len(schema.Enum) != 2 {
		t.Fatalf("schema = %#v, want two enum values", service.Parameters[0].Schema)
	}
}

func TestParserUsesDistributionOperationOnly(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:distribution <distribution:one>.
<distribution:one> a dcat:Distribution;
    dct:title "Distribution one";
    dct:description "Distribution description";
    dct:type "https://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"^^xsd:anyURI;
    dct:conformsTo <service:one>;
    dcat:accessURL <operation:selected>;
    dct:format "application/json".
<service:one> a epos:WebService;
    schema:name "Service one";
    hydra:supportedOperation <operation:other>, <operation:selected>.
<operation:selected> hydra:template "https://example.com/selected{?selected}";
    hydra:property <property:selected>.
<property:selected> hydra:mapping <parameter:selected>.
<parameter:selected> hydra:variable "selected";
    hydra:property "selected";
    rdfs:range xsd:string.
<operation:other> hydra:template "https://example.com/other{?other}";
    hydra:property <property:other>.
<property:other> hydra:mapping <parameter:other>.
<parameter:other> hydra:variable "other";
    hydra:property "other";
    rdfs:range xsd:string.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 || len(batch.Datasets[0].Distributions) != 1 {
		t.Fatalf("dataset distributions = %+v, want one distribution", batch.Datasets)
	}
	service := batch.Datasets[0].Distributions[0].WebService
	if service == nil {
		t.Fatal("distribution web service = nil")
	}
	if service.TemplateURL != "https://example.com/selected" {
		t.Fatalf("template URL = %q, want selected operation template", service.TemplateURL)
	}
	if len(service.Parameters) != 1 || service.Parameters[0].Name != "selected" {
		t.Fatalf("parameters = %+v, want selected operation parameter only", service.Parameters)
	}
}

func TestParserDropsWebServiceDistributionWithoutOperation(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:distribution <distribution:one>.
<distribution:one> a dcat:Distribution;
    dct:title "Distribution one";
    dct:description "Distribution description";
    dct:type "https://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE"^^xsd:anyURI;
    dct:conformsTo <service:one>;
    dct:format "application/json".
<service:one> a epos:WebService;
    schema:name "Service one".
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(batch.Datasets) != 1 || len(batch.Datasets[0].Distributions) != 0 {
		t.Fatalf("dataset distributions = %+v, want distribution skipped", batch.Datasets)
	}
	assertDiagnostic(t, batch.Warnings, "dataset:one", dcatDistributionLink, dcatAccessURL)
}

func assertDiagnostic(t *testing.T, diagnostics []domain.ImportDiagnostic, subject, predicate, message string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Subject == subject && diagnostic.Predicate == predicate && strings.Contains(diagnostic.Message, message) {
			return
		}
	}
	t.Fatalf("diagnostic (%q, %q, %q) not found in %+v", subject, predicate, message, diagnostics)
}

func TestAppendUniqueIdentifierDeduplicatesTypeAndValue(t *testing.T) {
	identifier := domain.Identifier{Type: "DOI", Value: "10.1234/example"}
	values := appendUniqueIdentifier([]domain.Identifier{identifier}, identifier)
	if len(values) != 1 {
		t.Fatalf("identifiers = %d, want 1", len(values))
	}
}

func TestAppendUniqueParameterMergesConflictingDefaults(t *testing.T) {
	firstDefault := "json"
	secondDefault := "xml"
	first := domain.WebServiceParameter{
		Name: "format",
		Schema: domain.StringSchema{
			Default:        &firstDefault,
			Enum:           []string{"json"},
			MultipleValues: true,
		},
	}
	second := domain.WebServiceParameter{
		Name:     "format",
		Required: true,
		Schema: domain.StringSchema{
			Default: &secondDefault,
			Enum:    []string{"xml"},
		},
	}

	values, err := appendUniqueParameter([]domain.WebServiceParameter{first}, second)
	if err != nil {
		t.Fatalf("appendUniqueParameter() error = %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("parameters = %d, want 1", len(values))
	}
	merged := values[0]
	schema := merged.Schema.(domain.StringSchema)
	if schema.Default != nil {
		t.Fatalf("default = %q, want nil after conflict", *schema.Default)
	}
	if len(schema.Enum) != 2 || schema.Enum[0] != "json" || schema.Enum[1] != "xml" {
		t.Fatalf("allowed values = %v, want [json xml]", schema.Enum)
	}
	if !merged.Required {
		t.Fatal("required = false, want true")
	}
}

func TestAppendUniqueParameterRejectsIncompatibleKinds(t *testing.T) {
	first := domain.WebServiceParameter{
		Name: "format", Schema: domain.StringSchema{},
	}
	second := domain.WebServiceParameter{
		Name: "format", Schema: domain.BooleanSchema{},
	}

	values, err := appendUniqueParameter([]domain.WebServiceParameter{first}, second)
	if err == nil {
		t.Fatal("appendUniqueParameter() error = nil, want incompatible kind error")
	}
	if len(values) != 1 || values[0].Schema.Kind() != domain.ParameterSchemaKindString {
		t.Fatalf("parameters = %v, want original string parameter", values)
	}
	if !strings.Contains(err.Error(), "incompatible kinds") {
		t.Fatalf("error = %v, want incompatible kinds diagnostic", err)
	}
}

func TestParserDiagnosesIncompatibleParameterKinds(t *testing.T) {
	batch, err := (Parser{}).Parse(strings.NewReader(importPrefixes + `
<category:dataset> a skos:ConceptScheme;
    dct:title "Dataset categories";
    skos:hasTopConcept <category:facets/dataset-theme>.
<category:one> a skos:Concept;
    skos:inScheme <category:dataset>;
    skos:prefLabel "One".
<dataset:one> a dcat:Dataset;
    dct:title "Dataset one";
    dct:description "Dataset description";
    dcat:theme <category:one>;
    dcat:distribution <distribution:one>.
<distribution:one> a dcat:Distribution;
    dct:title "Distribution one";
    dct:description "Distribution description";
    dct:format "application/json";
    dct:conformsTo <service:one>;
    dcat:accessURL <operation:one>.
<service:one> a epos:WebService;
    schema:name "Service one";
    hydra:supportedOperation <operation:one>.
<operation:one> hydra:template "https://example.com/data{?value}";
    hydra:property <property:one>.
<property:one> hydra:mapping <parameter:string>, <parameter:boolean>.
<parameter:string> hydra:variable "value";
    hydra:property "value";
    rdfs:range xsd:string.
<parameter:boolean> hydra:variable "value";
    hydra:property "value";
    rdfs:range xsd:boolean.
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(batch.Datasets) != 1 || len(batch.Datasets[0].Distributions) != 1 {
		t.Fatalf("datasets = %+v, want one dataset and distribution", batch.Datasets)
	}
	service := batch.Datasets[0].Distributions[0].WebService
	if service == nil || len(service.Parameters) != 1 {
		t.Fatalf("service parameters = %+v, want one retained parameter", service)
	}
	assertDiagnostic(t, batch.Warnings, "service:one", hydraMapping, "incompatible kinds")
}
