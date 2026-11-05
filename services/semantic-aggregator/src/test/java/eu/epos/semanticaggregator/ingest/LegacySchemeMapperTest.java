package eu.epos.semanticaggregator.ingest;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.rdf.TurtleReader;
import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import org.junit.jupiter.api.Test;

class LegacySchemeMapperTest {
    private static final String PREFIXES = """
            @prefix skos: <http://www.w3.org/2004/02/skos/core#> .
            @prefix dct: <http://purl.org/dc/terms/> .
            @prefix ex: <https://example.org/> .
            @prefix category: <https://www.epos-eu.org/epos-dcat-ap/category/> .
            """;

    @Test
    void derivesTopLevelConceptFromSkosHasTopConcept() {
        var model = read(PREFIXES + """
                ex:data a skos:ConceptScheme; dct:title "Data"; skos:hasTopConcept <category:facets/dataset-theme> .
                ex:facilities a skos:ConceptScheme; dct:title "Facilities"; skos:hasTopConcept <category:facets/facility-theme> .
                ex:software a skos:ConceptScheme; dct:title "Software"; skos:hasTopConcept <category:facets/software-theme> .
                """);
        try {
            var schemes = new LegacySchemeMapper().map(model);
            assertEquals(3, schemes.size());
            assertEquals("DATA", schemes.get(0).topLevelConcept());
            assertEquals("FACILITIES", schemes.get(1).topLevelConcept());
            assertEquals("SOFTWARE", schemes.get(2).topLevelConcept());
        } finally {
            model.close();
        }
    }

    @Test
    void mapsTitleAsNameAndPrefLabelAsLabel() {
        var model = read(PREFIXES + """
                ex:software a skos:ConceptScheme; dct:title "Software"; skos:prefLabel "SW";
                    skos:hasTopConcept <category:facets/software-theme> .
                """);
        try {
            var schemes = new LegacySchemeMapper().map(model);
            assertEquals("SOFTWARE", schemes.getFirst().topLevelConcept());
            assertEquals("Software", schemes.getFirst().name());
            assertEquals("SW", schemes.getFirst().label());
        } finally {
            model.close();
        }
    }

    @Test
    void mapsCompleteLegacySchemeMetadata() {
        var model = read(PREFIXES + """
                ex:data a skos:ConceptScheme; dct:title "Seismology"; skos:prefLabel "SEI";
                    dct:description "TCS Domain"; <http://xmlns.com/foaf/0.1/logo> <https://example.org/logo>;
                    <http://xmlns.com/foaf/0.1/homepage> <https://example.org/page>; <http://schema.org/color> "#476f57";
                    <http://schema.org/orderItemNumber> "1"; skos:hasTopConcept <category:facets/dataset-theme> .
                """);
        try {
            var scheme = new LegacySchemeMapper().map(model).getFirst();
            assertEquals("Seismology", scheme.name());
            assertEquals("SEI", scheme.label());
            assertEquals("TCS Domain", scheme.description());
            assertEquals("https://example.org/logo", scheme.logo());
            assertEquals("https://example.org/page", scheme.page());
            assertEquals("#476f57", scheme.color());
            assertEquals(1, scheme.position());
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsConflictingTopConcepts() {
        var model = read(PREFIXES + """
                ex:mixed a skos:ConceptScheme; dct:title "Mixed";
                    skos:hasTopConcept <category:facets/dataset-theme>, <category:facets/software-theme> .
                """);
        try {
            assertThrows(IllegalArgumentException.class, () -> new LegacySchemeMapper().map(model));
        } finally {
            model.close();
        }
    }

    @Test
    void ignoresVocabularySchemesWithoutTopLevelConcept() {
        var model = read(PREFIXES + """
                ex:vocabulary a skos:ConceptScheme; dct:title "Vocabulary" .
                ex:data a skos:ConceptScheme; dct:title "Data"; skos:hasTopConcept <category:facets/dataset-theme> .
                """);
        try {
            var schemes = new LegacySchemeMapper().map(model);
            assertEquals(1, schemes.size());
            assertEquals("DATA", schemes.getFirst().topLevelConcept());
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsUnsupportedTopLevelConceptValue() {
        var model = read(PREFIXES + """
                ex:unknown a skos:ConceptScheme; dct:title "Unknown"; skos:hasTopConcept <category:facets/equipment-theme> .
                """);
        try {
            assertThrows(IllegalArgumentException.class, () -> new LegacySchemeMapper().map(model));
        } finally {
            model.close();
        }
    }

    @Test
    void mapsAllSupportedSchemeCategoriesAndTheirParent() {
        var model = read(PREFIXES + """
                ex:data a skos:ConceptScheme; dct:title "Data"; skos:hasTopConcept <category:facets/dataset-theme> .
                ex:root a skos:Concept; skos:inScheme ex:data; skos:prefLabel "Root" .
                ex:child a skos:Concept; skos:inScheme ex:data; skos:prefLabel "Child"; skos:broader ex:root .
                ex:other a skos:Concept; skos:inScheme ex:vocabulary; skos:prefLabel "Other" .
                ex:unscoped a skos:Concept; skos:prefLabel "Unscoped vocabulary value" .
                """);
        try {
            var mapper = new LegacySchemeMapper();
            var categories = mapper.categories(model, java.util.Set.of("https://example.org/data"));
            assertEquals(2, categories.size());
            assertEquals("https://example.org/root", categories.stream()
                    .filter(category -> category.iri().equals("https://example.org/child")).findFirst().orElseThrow().parentIri());
        } finally {
            model.close();
        }
    }

    private static org.apache.jena.rdf.model.Model read(String turtle) {
        return new TurtleReader().read(new ByteArrayInputStream(turtle.getBytes(StandardCharsets.UTF_8)),
                "https://example.org/");
    }
}
