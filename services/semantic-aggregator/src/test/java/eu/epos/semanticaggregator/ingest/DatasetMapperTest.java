package eu.epos.semanticaggregator.ingest;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.rdf.TurtleReader;
import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import org.junit.jupiter.api.Test;

class DatasetMapperTest {
    private static final String PREFIXES = """
            @prefix dcat: <http://www.w3.org/ns/dcat#> .
            @prefix dct: <http://purl.org/dc/terms/> .
            @prefix adms: <http://www.w3.org/ns/adms#> .
            @prefix skos: <http://www.w3.org/2004/02/skos/core#> .
            @prefix locn: <http://www.w3.org/ns/locn#> .
            @prefix ex: <https://example.org/> .
            """;

    @Test
    void flattensBlankNodesAndReportsUnsupportedProperties() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:keyword "z", "a"; ex:unmapped "value";
                    dct:spatial [ locn:geometry "POINT (12.5 41.9)"; ex:extra "x" ];
                    adms:identifier [ adms:schemeAgency "DOI"; skos:notation "10.123/test" ] .
                """);
        try {
            var result = mapper().map(model).getFirst();
            assertEquals("https://example.org/dataset", result.subject());
            assertEquals("POINT (12.5 41.9)", result.spatialCoverage());
            assertEquals(java.util.List.of("a", "z"), result.keywords());
            assertEquals(new DatasetMapping.Identifier("DOI", "10.123/test"), result.identifiers().getFirst());
            assertEquals(2, result.unsupportedPredicates().size());
        } finally {
            model.close();
        }
    }

    @Test
    void acceptsLanguageTaggedTitles() {
        var model = read(PREFIXES + "ex:dataset a dcat:Dataset; dct:title \"One\"@en; dct:description \"D\" .");
        try {
            assertEquals("One", mapper().map(model).getFirst().title());
        } finally {
            model.close();
        }
    }

    @Test
    void omitsKeywordsOutsideTheLifecycleContract() {
        var model = read(PREFIXES + """
                ex:dataset a dcat:Dataset; dct:title "Title"; dct:description "Description";
                    dcat:keyword "Valid keyword", "3D/4D Model" .
                """);
        try {
            assertEquals(java.util.List.of("Valid keyword"), mapper().map(model).getFirst().keywords());
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsMultipleTitlesRatherThanChoosingOne() {
        var model = read(PREFIXES + "ex:dataset a dcat:Dataset; dct:title \"One\", \"Two\"; dct:description \"D\" .");
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(model));
        } finally {
            model.close();
        }
    }

    @Test
    void makesLegacyFormatSelectorsReadOnly() {
        var model = read(PREFIXES + """
                @prefix hydra: <http://www.w3.org/ns/hydra/core#> .
                @prefix schema: <http://schema.org/> .
                @prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
                @prefix http: <http://www.w3.org/2006/http#> .
                @prefix epos: <https://www.epos-eu.org/epos-dcat-ap#> .
                ex:dataset a dcat:Dataset; dct:title "Dataset"; dct:description "Description"; dcat:distribution ex:distribution .
                ex:distribution a dcat:Distribution; dct:title "Distribution"; dct:description "Description";
                    dct:type "WEB_SERVICE"; dct:format "application/json"; dct:conformsTo ex:service; dcat:accessURL ex:operation .
                ex:service a epos:WebService; schema:name "Service"; hydra:supportedOperation ex:operation .
                ex:operation a hydra:Operation; hydra:method "GET"; hydra:property [ a hydra:IriTemplate;
                    hydra:template "https://example.org/query{?format}";
                    hydra:mapping [ a hydra:IriTemplateMapping; hydra:variable "format"; hydra:property schema:encodingFormat;
                        rdfs:range "xsd:string"; schema:defaultValue "json"; http:paramValue "xml" ] ] .
                """);
        try {
            var parameter = mapper().map(model).getFirst().distributions().getFirst().webService().parameters().getFirst();
            assertEquals(DatasetMapping.ParameterRole.FORMAT, parameter.role());
            assertEquals(true, parameter.readOnly());
            assertEquals(java.util.List.of("json", "xml"), ((DatasetMapping.StringSchema) parameter.schema()).allowedValues());
        } finally {
            model.close();
        }
    }

    @Test
    void normalizesLegacyTwentyFourHourDateTimeLayouts() {
        var model = read(PREFIXES + """
                @prefix hydra: <http://www.w3.org/ns/hydra/core#> .
                @prefix schema: <http://schema.org/> .
                @prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
                @prefix epos: <https://www.epos-eu.org/epos-dcat-ap#> .
                ex:dataset a dcat:Dataset; dct:title "Dataset"; dct:description "Description"; dcat:distribution ex:distribution .
                ex:distribution a dcat:Distribution; dct:title "Distribution"; dct:description "Description";
                    dct:type "WEB_SERVICE"; dct:format "application/json"; dct:conformsTo ex:service; dcat:accessURL ex:operation .
                ex:service a epos:WebService; schema:name "Service"; hydra:supportedOperation ex:operation .
                ex:operation a hydra:Operation; hydra:method "GET"; hydra:property [ a hydra:IriTemplate;
                    hydra:template "https://example.org/query{?start}";
                    hydra:mapping [ a hydra:IriTemplateMapping; hydra:variable "start"; hydra:property schema:startDate;
                        rdfs:range "xsd:dateTime"; schema:valuePattern "YYYY-MM-DD HH:MM:SS.SSSZ" ] ] .
                """);
        try {
            var schema = (DatasetMapping.DateTimeSchema) mapper().map(model).getFirst().distributions()
                    .getFirst().webService().parameters().getFirst().schema();
            assertEquals("YYYY-MM-DD hh:mm:ss.SSSZ", schema.layout());
        } finally {
            model.close();
        }
    }

    private static org.apache.jena.rdf.model.Model read(String turtle) {
        return new TurtleReader().read(new ByteArrayInputStream(turtle.getBytes(StandardCharsets.UTF_8)),
                "https://example.org/");
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
