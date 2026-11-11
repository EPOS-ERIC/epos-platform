package eu.epos.semanticaggregator.upstream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import eu.epos.semanticaggregator.ingest.DatasetMapper;
import eu.epos.semanticaggregator.rdf.TurtleReader;
import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import metadata_lifecycle.v1.Resource;
import org.junit.jupiter.api.Test;

class DatasetRequestMapperTest {
    private static final String PREFIXES = """
            @prefix dcat: <http://www.w3.org/ns/dcat#> .
            @prefix dct: <http://purl.org/dc/terms/> .
            @prefix schema: <http://schema.org/> .
            @prefix adms: <http://www.w3.org/ns/adms#> .
            @prefix skos: <http://www.w3.org/2004/02/skos/core#> .
            @prefix ex: <https://example.org/> .
            """;

    @Test
    void convertsNestedRdfValuesAndOpenTemporalBoundToCreateResource() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:keyword "earthquake";
                    dct:temporal [ schema:startDate "2020-02-03T12:30:00+01:00" ];
                    adms:identifier [ adms:schemeAgency "DOI"; skos:notation "10.123/example" ] .
                """);
        try {
            var mapping = mapper().map(model).getFirst();
            var request = new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000");
            assertEquals("Title", request.getDataset().getTitle());
            assertEquals("DOI", request.getDataset().getPersistentIdentifiers(0).getType());
            assertTrue(request.getDataset().getTemporalCoverage().hasStart());
            assertFalse(request.getDataset().getTemporalCoverage().hasEnd());
            assertEquals(Instant.parse("2020-02-03T11:30:00Z").getEpochSecond(),
                    request.getDataset().getTemporalCoverage().getStart().getSeconds());
            assertThrows(IllegalArgumentException.class, () -> new DatasetRequestMapper().toRequest(mapping, "1-1-1-1-1"));
        } finally {
            model.close();
        }
    }

    @Test
    void preservesPrimaryIdentifierValueAndDatatypeInConfiguredTypes() {
        var model = read(PREFIXES + """
                @prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
                ex:a a dcat:Dataset; dct:title "A"; dct:description "A";
                    dct:identifier "urn:example:first"^^xsd:anyURI .
                ex:b a dcat:Dataset; dct:title "B"; dct:description "B";
                    dct:identifier "dataset-b" .
                """);
        try {
            var datasets = mapper().map(model);
            var converter = new DatasetRequestMapper();
            var first = converter.toRequest(datasets.get(0), "123e4567-e89b-12d3-a456-426614174000").getDataset();
            var second = converter.toRequest(datasets.get(1), "123e4567-e89b-12d3-a456-426614174000").getDataset();
            assertEquals("EPOS-DCAT-AP:dct:identifier:xsd:anyURI", first.getPersistentIdentifiers(0).getType());
            assertEquals("urn:example:first", first.getPersistentIdentifiers(0).getValue());
            assertEquals("EPOS-DCAT-AP:dct:identifier:xsd:string", second.getPersistentIdentifiers(0).getType());
            assertEquals("dataset-b", second.getPersistentIdentifiers(0).getValue());
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsMultipleOrUnsupportedPrimaryIdentifiers() {
        var multiple = read(PREFIXES + "ex:a a dcat:Dataset; dct:title \"A\"; dct:description \"A\"; dct:identifier \"one\", \"two\" .");
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(multiple));
        } finally {
            multiple.close();
        }
        var unsupported = read(PREFIXES + "ex:a a dcat:Dataset; dct:title \"A\"; dct:description \"A\"; dct:identifier 42 .");
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(unsupported));
        } finally {
            unsupported.close();
        }
    }

    @Test
    void ignoresUnsupportedDatasetProperties() {
        var model = read(PREFIXES + "ex:dataset a dcat:Dataset; dct:title \"Title\"; "
                + "dct:description \"Description\"; ex:unknown \"value\" .");
        try {
            var mapping = mapper().map(model).getFirst();
            assertEquals(Resource.CreateResourceRequest.MetadataCase.DATASET,
                    new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000").getMetadataCase());
        } finally {
            model.close();
        }
    }

    @Test
    void mapsDownloadableDistributionAndDatasetMetadata() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dct:created "2020-01-01"; dct:accrualPeriodicity <https://example.org/daily>;
                    <http://www.w3.org/ns/dqv#hasQualityAnnotation>
                        [ <http://www.w3.org/ns/oa#hasBody> <https://example.org/quality> ];
                    dcat:distribution [ a dcat:Distribution; dct:title "CSV"; dct:description "Download";
                        dct:type <http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE>;
                        dcat:downloadURL <https://example.org/data.csv>; dct:format "text/csv";
                        dct:license <https://example.org/license>; dcat:byteSize 42 ] .
                """);
        try {
            var mapping = mapper().map(model).getFirst();
            var dataset = new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000").getDataset();
            assertEquals(Resource.DatasetAccrualPeriodicity.DATASET_ACCRUAL_PERIODICITY_DAILY, dataset.getAccrualPeriodicity());
            assertEquals("https://example.org/quality", dataset.getQualityAnnotation());
            assertEquals(Instant.parse("2020-01-01T00:00:00Z").getEpochSecond(), dataset.getCreated().getSeconds());
            assertEquals(Resource.DistributionType.DISTRIBUTION_TYPE_DOWNLOADABLE_FILE, dataset.getDistributions(0).getType());
            assertEquals("https://example.org/data.csv", dataset.getDistributions(0).getDownloadableFile().getUrl());
            assertEquals("text/csv", dataset.getDistributions(0).getDownloadableFile().getFormat());
            assertEquals(42, dataset.getDistributions(0).getByteSize());
        } finally {
            model.close();
        }
    }

    @Test
    void acceptsAccessUrlForDownloadableFilesAndRejectsConflictingUrls() {
        String prefix = PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:distribution [ dct:title "CSV"; dct:description "Download";
                        dct:type "DOWNLOADABLE_FILE"; dcat:accessURL <https://example.org/a.csv>;
                        dct:format "text/csv"%s ] .
                """;
        var model = read(prefix.formatted(""));
        try {
            var mapping = mapper().map(model).getFirst();
            var request = new DatasetRequestMapper().toRequest(mapping, "123e4567-e89b-12d3-a456-426614174000");
            assertEquals("https://example.org/a.csv", request.getDataset().getDistributions(0).getDownloadableFile().getUrl());
        } finally {
            model.close();
        }
        var conflicting = read(prefix.formatted("; dcat:downloadURL <https://example.org/b.csv>"));
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(conflicting));
        } finally {
            conflicting.close();
        }
    }

    @Test
    void rejectsWebServiceRatherThanTreatingItAsDownload() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:distribution [ dct:type "WEB_SERVICE" ] .
                """);
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(model));
        } finally {
            model.close();
        }
    }

    @Test
    void mapsMultipleDefaultsForStringIntegerAndFloatParameters() {
        var model = read(PREFIXES + """
                @prefix hydra: <http://www.w3.org/ns/hydra/core#> .
                @prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
                @prefix http: <http://www.w3.org/2006/http#> .
                @prefix schema: <http://schema.org/> .
                @prefix epos: <https://www.epos-eu.org/epos-dcat-ap#> .
                ex:dataset a dcat:Dataset; dct:title "Dataset"; dct:description "Description"; dcat:distribution ex:distribution .
                ex:distribution a dcat:Distribution; dct:title "Distribution"; dct:description "Description";
                    dct:type "WEB_SERVICE"; dct:format "application/json"; dct:conformsTo ex:service; dcat:accessURL ex:operation .
                ex:service a epos:WebService; schema:name "Service"; hydra:supportedOperation ex:operation .
                ex:operation a hydra:Operation; hydra:method "GET"; hydra:property [ a hydra:IriTemplate;
                    hydra:template "https://example.org/query{?labels,ids,weights}";
                    hydra:mapping [ a hydra:IriTemplateMapping; hydra:variable "labels"; rdfs:range "xsd:string";
                        schema:multipleValues true; schema:defaultValue "one", "two"; http:paramValue "one", "two" ];
                    hydra:mapping [ a hydra:IriTemplateMapping; hydra:variable "ids"; rdfs:range "xsd:integer";
                        schema:multipleValues true; schema:defaultValue "1", "2"; http:paramValue "1", "2" ];
                    hydra:mapping [ a hydra:IriTemplateMapping; hydra:variable "weights"; rdfs:range "xsd:float";
                        schema:multipleValues true; schema:defaultValue "1.5", "2.5"; http:paramValue "1.5", "2.5" ] ] .
                """);
        try {
            var request = new DatasetRequestMapper().toRequest(mapper().map(model).getFirst(),
                    "123e4567-e89b-12d3-a456-426614174000");
            var parameters = request.getDataset().getDistributions(0).getWebService().getGeneric().getParametersList();
            assertEquals(java.util.Set.of("one", "two"), java.util.Set.copyOf(parameters.get(0).getString().getDefaultList()));
            assertEquals(java.util.Set.of(1L, 2L), java.util.Set.copyOf(parameters.get(1).getInteger().getDefaultList()));
            assertEquals(java.util.Set.of(1.5, 2.5), java.util.Set.copyOf(parameters.get(2).getFloat().getDefaultList()));
        } finally {
            model.close();
        }
    }

    @Test
    void mapsWmsConfigurationAndMapVisibility() {
        var model = webService("""
                hydra:template "https://example.org/wms{?service,request,layers,format}";
                hydra:mapping [ hydra:variable "service"; rdfs:range "xsd:string"; schema:defaultValue "WMS" ];
                hydra:mapping [ hydra:variable "request"; rdfs:range "xsd:string"; schema:defaultValue "GetMap" ];
                hydra:mapping [ hydra:variable "layers"; rdfs:range "xsd:string"; schema:defaultValue "geology"; http:paramValue "geology" ];
                hydra:mapping [ hydra:variable "format"; rdfs:range "xsd:string"; schema:defaultValue "image/png"; hydra:property schema:encodingFormat ]
                """, "image/png");
        try {
            var service = request(model).getDataset().getDistributions(0).getWebService();
            assertEquals(Resource.WebServiceType.WEB_SERVICE_TYPE_WMS, service.getType());
            assertEquals(java.util.List.of(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP), service.getVisibleOnList());
            assertEquals("https://example.org/wms?service=WMS&request=GetCapabilities", service.getWms().getCapabilitiesUrl());
            assertEquals("geology", service.getWms().getDefaultLayer());
        } finally {
            model.close();
        }
    }

    @Test
    void mapsWmtsConfigurationAndMapTableVisibility() {
        var model = webService("""
                hydra:template "https://example.org/wmts{?service,request,layer,format}";
                hydra:mapping [ hydra:variable "service"; rdfs:range "xsd:string"; schema:defaultValue "WMTS" ];
                hydra:mapping [ hydra:variable "request"; rdfs:range "xsd:string"; schema:defaultValue "GetTile" ];
                hydra:mapping [ hydra:variable "layer"; rdfs:range "xsd:string"; schema:defaultValue "displacement"; http:paramValue "displacement" ];
                hydra:mapping [ hydra:variable "format"; rdfs:range "xsd:string"; schema:defaultValue "image/png"; hydra:property schema:encodingFormat ]
                """, "image/png");
        try {
            var service = request(model).getDataset().getDistributions(0).getWebService();
            assertEquals(Resource.WebServiceType.WEB_SERVICE_TYPE_WMTS, service.getType());
            assertEquals(java.util.List.of(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP,
                    Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE), service.getVisibleOnList());
            assertEquals("displacement", service.getWmts().getDefaultLayer());
        } finally {
            model.close();
        }
    }

    @Test
    void normalizesWfsTwoDotZeroDotTwoAndMapsMapTableVisibility() {
        var model = webService("""
                hydra:template "https://example.org/wfs{?service,request,typenames,version,outputFormat,count}";
                hydra:mapping [ hydra:variable "service"; rdfs:range "xsd:string"; schema:defaultValue "WFS" ];
                hydra:mapping [ hydra:variable "request"; rdfs:range "xsd:string"; schema:defaultValue "GetFeature" ];
                hydra:mapping [ hydra:variable "typenames"; rdfs:range "xsd:string"; schema:defaultValue "boreholes"; http:paramValue "boreholes" ];
                hydra:mapping [ hydra:variable "version"; rdfs:range "xsd:string"; schema:defaultValue "2.0.2" ];
                hydra:mapping [ hydra:variable "outputFormat"; rdfs:range "xsd:string"; schema:defaultValue "json"; http:paramValue "json"; hydra:property schema:encodingFormat ];
                hydra:mapping [ hydra:variable "count"; rdfs:range "xsd:string"; schema:defaultValue "100" ]
                """, "application/gml+xml");
        try {
            var service = request(model).getDataset().getDistributions(0).getWebService();
            assertEquals(Resource.WebServiceType.WEB_SERVICE_TYPE_WFS, service.getType());
            assertEquals("2.0.0", service.getWfs().getVersion());
            assertEquals(100, service.getWfs().getResultLimit());
            assertEquals(java.util.List.of(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP,
                    Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE), service.getVisibleOnList());
        } finally {
            model.close();
        }
    }

    @Test
    void mapsLegacyWfsCqlFilterToAQueryParameter() {
        var model = webService("""
                hydra:template "https://example.org/wfs{?service,request,typenames,version,outputFormat,gml:identifier}";
                hydra:mapping [ hydra:variable "service"; rdfs:range "xsd:string"; schema:defaultValue "WFS" ];
                hydra:mapping [ hydra:variable "request"; rdfs:range "xsd:string"; schema:defaultValue "GetFeature" ];
                hydra:mapping [ hydra:variable "typenames"; rdfs:range "xsd:string"; schema:defaultValue "boreholes"; http:paramValue "boreholes" ];
                hydra:mapping [ hydra:variable "version"; rdfs:range "xsd:string"; schema:defaultValue "2.0.0" ];
                hydra:mapping [ hydra:variable "outputFormat"; rdfs:range "xsd:string"; schema:defaultValue "json"; http:paramValue "json" ];
                hydra:mapping [ hydra:variable "CQL_FILTER=gml:identifier"; rdfs:range "xsd:string" ]
                """, "application/gml+xml");
        try {
            var parameters = request(model).getDataset().getDistributions(0).getWebService().getWfs().getParametersList();
            assertEquals(1, parameters.size());
            assertEquals("CQL_FILTER", parameters.getFirst().getName());
            assertEquals(Resource.WebServiceParameterLocation.WEB_SERVICE_PARAMETER_LOCATION_QUERY,
                    parameters.getFirst().getLocation());
            assertTrue(parameters.getFirst().getTemplate().getTemplate().contains("gml:identifier = {{cql .}}"));
            assertEquals("CQL_FILTER=gml:identifier", parameters.getFirst().getTemplate().getParameters(0).getLabel());
        } finally {
            model.close();
        }
    }

    @Test
    void mapsMultipleLegacyCqlFiltersIntoOneTemplateParameter() {
        var model = webService("""
                hydra:template "https://example.org/wfs{?service,request,typenames,version,outputFormat,gml:identifier,gml:name}";
                hydra:mapping [ hydra:variable "service"; rdfs:range "xsd:string"; schema:defaultValue "WFS" ];
                hydra:mapping [ hydra:variable "request"; rdfs:range "xsd:string"; schema:defaultValue "GetFeature" ];
                hydra:mapping [ hydra:variable "typenames"; rdfs:range "xsd:string"; schema:defaultValue "boreholes"; http:paramValue "boreholes" ];
                hydra:mapping [ hydra:variable "version"; rdfs:range "xsd:string"; schema:defaultValue "2.0.0" ];
                hydra:mapping [ hydra:variable "outputFormat"; rdfs:range "xsd:string"; schema:defaultValue "json"; http:paramValue "json" ];
                hydra:mapping [ hydra:variable "CQL_FILTER=gml:identifier"; rdfs:range "xsd:string" ];
                hydra:mapping [ hydra:variable "CQL_FILTER=gml:name"; rdfs:range "xsd:string" ]
                """, "application/gml+xml");
        try {
            var template = request(model).getDataset().getDistributions(0).getWebService().getWfs().getParameters(0).getTemplate();
            assertEquals(2, template.getParametersCount());
            assertTrue(template.getTemplate().contains("gml:identifier = {{cql .}}"));
            assertTrue(template.getTemplate().contains("gml:name = {{cql .}}"));
        } finally {
            model.close();
        }
    }

    @Test
    void infersVisibilityFromGeoJsonAndCovJsonFormats() {
        for (var expected : java.util.Map.of(
                "application/geo+json", java.util.List.of(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP,
                        Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE),
                "application/geo%2Bjson", java.util.List.of(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP,
                        Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE),
                "application/geo-json", java.util.List.of(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP,
                        Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE),
                "application/vnd.cov+json", java.util.List.of(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP,
                        Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_GRAPH)).entrySet()) {
            var model = webService("hydra:template \"https://example.org/data\"", expected.getKey());
            try {
                assertEquals(expected.getValue(), request(model).getDataset().getDistributions(0).getWebService().getVisibleOnList());
            } finally {
                model.close();
            }
        }
    }

    @Test
    void rejectsReversedTemporalRange() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dct:temporal [ schema:startDate "2021-01-01"; schema:endDate "2020-01-01" ] .
                """);
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(model));
        } finally {
            model.close();
        }
    }

    private static org.apache.jena.rdf.model.Model read(String turtle) {
        return new TurtleReader().read(new ByteArrayInputStream(turtle.getBytes(StandardCharsets.UTF_8)),
                "https://example.org/");
    }

    private static org.apache.jena.rdf.model.Model webService(String template, String format) {
        return read(PREFIXES + """
                @prefix hydra: <http://www.w3.org/ns/hydra/core#> .
                @prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
                @prefix http: <http://www.w3.org/2006/http#> .
                @prefix epos: <https://www.epos-eu.org/epos-dcat-ap#> .
                ex:dataset a dcat:Dataset; dct:title "Dataset"; dct:description "Description"; dcat:distribution ex:distribution .
                ex:distribution a dcat:Distribution; dct:title "Distribution"; dct:description "Description";
                    dct:type "WEB_SERVICE"; dct:format "%s"; dct:conformsTo ex:service; dcat:accessURL ex:operation .
                ex:service a epos:WebService; schema:name "Service"; hydra:supportedOperation ex:operation .
                ex:operation a hydra:Operation; hydra:method "GET"; hydra:property [ a hydra:IriTemplate; %s ] .
                """.formatted(format, template));
    }

    private static Resource.CreateResourceRequest request(org.apache.jena.rdf.model.Model model) {
        return new DatasetRequestMapper().toRequest(mapper().map(model).getFirst(), "123e4567-e89b-12d3-a456-426614174000");
    }

    private static DatasetMapper mapper() {
        try {
            return new DatasetMapper(eu.epos.semanticaggregator.mapping.MappingRegistry.withDefaults()
                    .get("epos-dcat-common", 1));
        } catch (java.io.IOException exception) {
            throw new java.io.UncheckedIOException(exception);
        }
    }
}
