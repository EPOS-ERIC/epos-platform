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
    void rejectsLanguageTaggedTitles() {
        var model = read(PREFIXES + "ex:dataset a dcat:Dataset; dct:title \"One\"@en; dct:description \"D\" .");
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(model));
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
