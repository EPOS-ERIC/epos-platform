package eu.epos.semanticaggregator.ingest;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.mapping.MappingRegistry;
import eu.epos.semanticaggregator.rdf.TurtleReader;
import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import org.junit.jupiter.api.Test;

class SoftwareMapperTest {
    private static final String PREFIXES = """
            @prefix schema: <http://schema.org/> .
            @prefix dcat: <http://www.w3.org/ns/dcat#> .
            @prefix ex: <https://example.org/> .
            """;

    @Test
    void mapsApplicationAndSourceCodeUsingProfilePredicates() {
        var model = read(PREFIXES + """
                ex:app a schema:SoftwareApplication; schema:name "Client"; schema:description "Desktop client";
                    schema:keywords "seismic", "client"; schema:softwareVersion "1.2";
                    schema:identifier [ a schema:PropertyValue; schema:propertyID "doi"; schema:value "10.1/client" ];
                    schema:creator ex:org; dcat:theme ex:category; schema:contactPoint ex:contact .
                ex:source a schema:SoftwareSourceCode; schema:name "Client source"; schema:description "Source";
                    schema:programmingLanguage "Java"; schema:codeRepository <https://example.org/repo> .
                """);
        try {
            var mapped = mapper().map(model);
            assertEquals(2, mapped.size());
            var application = mapped.getFirst();
            assertEquals(SoftwareMapping.Kind.APPLICATION, application.kind());
            assertEquals(java.util.List.of("client", "seismic"), application.keywords());
            assertEquals(new SoftwareMapping.Identifier("doi", "10.1/client"), application.identifiers().getFirst());
            assertEquals(new DatasetMapping.Reference(DatasetMapping.ReferenceKind.ORGANIZATION, "https://example.org/org"),
                    application.creators().getFirst());
            var source = mapped.get(1);
            assertEquals(SoftwareMapping.Kind.SOURCE_CODE, source.kind());
            assertEquals("Java", source.programmingLanguage());
            assertEquals("https://example.org/repo", source.repository());
        } finally {
            model.close();
        }
    }

    @Test
    void acceptsNonPropertyValueIdentifiersAndRejectsUnsupportedProperties() {
        var invalidIdentifier = read(PREFIXES + """
                ex:app a schema:SoftwareApplication; schema:name "Client"; schema:description "Description";
                    schema:identifier "10.1/client" .
                """);
        try {
            var result = mapper().map(invalidIdentifier);
            assertEquals(1, result.size());
            assertEquals(new SoftwareMapping.Identifier("OTHER", "10.1/client"), result.getFirst().identifiers().getFirst());
        } finally {
            invalidIdentifier.close();
        }
        var unsupported = read(PREFIXES + """
                ex:app a schema:SoftwareApplication; schema:name "Client"; schema:description "Description";
                    schema:author "not supported" .
                """);
        try {
            assertThrows(IllegalArgumentException.class, () -> mapper().map(unsupported));
        } finally {
            unsupported.close();
        }
    }

    private static org.apache.jena.rdf.model.Model read(String turtle) {
        return new TurtleReader().read(new ByteArrayInputStream(turtle.getBytes(StandardCharsets.UTF_8)), "https://example.org/");
    }

    private static SoftwareMapper mapper() {
        try {
            return new SoftwareMapper(MappingRegistry.withDefaults().get("epos-dcat-common", 1));
        } catch (java.io.IOException exception) {
            throw new java.io.UncheckedIOException(exception);
        }
    }
}
