package eu.epos.semanticaggregator.ingest;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.mapping.MappingRegistry;
import eu.epos.semanticaggregator.rdf.TurtleReader;
import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import org.junit.jupiter.api.Test;

class EntityDescriptionMapperTest {
    @Test
    void tolerantMappingKeepsSupportedOrganizationFields() throws Exception {
        var model = read("""
                @prefix schema: <http://schema.org/> .
                @prefix ex: <https://example.org/> .
                ex:organization a schema:Organization;
                    schema:legalName "Example Organization";
                    schema:identifier [ a schema:PropertyValue; schema:propertyID "ROR"; schema:value "01" ];
                    schema:memberOf ex:parent .
                """);
        try {
            var mapper = mapper();
            var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.ORGANIZATION,
                    "https://example.org/organization");

            assertEquals("Example Organization", mapper.map(model, reference, true).description().fields().get("legal_name"));
            assertThrows(IllegalArgumentException.class, () -> mapper.map(model, reference));
        } finally {
            model.close();
        }
    }

    @Test
    void mapsLanguageTaggedOrganizationName() throws Exception {
        var model = read("""
                @prefix schema: <http://schema.org/> .
                @prefix ex: <https://example.org/> .
                ex:organization a schema:Organization;
                    schema:legalName "Example Organization"@en;
                    schema:url "www.example.org" .
                """);
        try {
            var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.ORGANIZATION,
                    "https://example.org/organization");

            assertEquals("Example Organization", mapper().map(model, reference).description().fields().get("legal_name"));
            assertNull(mapper().map(model, reference).description().fields().get("page_url"));
        } finally {
            model.close();
        }
    }

    @Test
    void mapsMailtoContactPointWithoutRdfDescription() throws Exception {
        var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CONTACT_POINT,
                "mailto:contact@example.org/contactPoint");
        var model = read("");
        try {
            assertEquals("contact@example.org", mapper().map(model, reference).description().fields().get("email"));
        } finally {
            model.close();
        }
    }

    @Test
    void mapsContactPointFromBaseResourceWithOneEmail() throws Exception {
        var model = read("""
                @prefix schema: <http://schema.org/> .
                @prefix ex: <https://example.org/> .
                ex:person schema:email "contact@example.org" .
                """);
        try {
            var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CONTACT_POINT,
                    "https://example.org/person/contactPoint");

            assertEquals("contact@example.org", mapper().map(model, reference).description().fields().get("email"));
        } finally {
            model.close();
        }
    }

    @Test
    void leavesContactPointWithoutBaseEmailUnmapped() throws Exception {
        var model = read("");
        try {
            var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CONTACT_POINT,
                    "https://example.org/person/contactPoint");

            assertNull(mapper().map(model, reference).description().fields().get("email"));
        } finally {
            model.close();
        }
    }

    @Test
    void removesInvisibleFormatCharactersFromLegacyEmails() throws Exception {
        var model = read("""
                @prefix schema: <http://schema.org/> .
                @prefix ex: <https://example.org/> .
                ex:organization a schema:Organization; schema:legalName "Example Organization";
                    schema:email "contact\u200b@example.org" .
                """);
        try {
            var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.ORGANIZATION,
                    "https://example.org/organization");

            assertEquals("contact@example.org", mapper().map(model, reference).description().fields().get("email"));
        } finally {
            model.close();
        }
    }

    private static EntityDescriptionMapper mapper() throws Exception {
        return new EntityDescriptionMapper(MappingRegistry.withDefaults().get("epos-dcat-ap", 1));
    }

    private static org.apache.jena.rdf.model.Model read(String turtle) {
        return new TurtleReader().read(new ByteArrayInputStream(turtle.getBytes(StandardCharsets.UTF_8)),
                "https://example.org/");
    }
}
