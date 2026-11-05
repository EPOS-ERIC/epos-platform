package eu.epos.semanticaggregator.upstream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.ingest.DatasetMapping;
import eu.epos.semanticaggregator.ingest.SoftwareMapping;
import java.util.List;
import org.junit.jupiter.api.Test;

class SoftwareRequestMapperTest {
    private static final String GROUP = "123e4567-e89b-12d3-a456-426614174000";
    private static final String ORGANIZATION = "123e4567-e89b-12d3-a456-426614174001";
    private static final String CATEGORY = "123e4567-e89b-12d3-a456-426614174002";
    private static final String CONTACT = "123e4567-e89b-12d3-a456-426614174003";

    @Test
    void createsSourceCodeInputWithExplicitReferencesAndIdentifiers() {
        var organization = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.ORGANIZATION, "https://example.org/org");
        var category = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CATEGORY, "https://example.org/category");
        var contact = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CONTACT_POINT, "https://example.org/contact");
        var bindings = new ReferenceBindings();
        bindings.register(organization.kind(), organization.iri(), ORGANIZATION);
        bindings.register(category.kind(), category.iri(), CATEGORY);
        bindings.register(contact.kind(), contact.iri(), CONTACT);
        var mapping = new SoftwareMapping("https://example.org/source", SoftwareMapping.Kind.SOURCE_CODE, "Source", "Description",
                List.of("java"), "https://example.org/page", "Apache-2.0", "1.0", null, "JDK", null, "Java",
                "https://example.org/repo", "JVM", List.of(new SoftwareMapping.Identifier("doi", "10.1/source")),
                List.of(organization), List.of(category), List.of(contact));

        var request = new SoftwareRequestMapper().toRequest(mapping, GROUP, bindings);
        var source = request.getSoftwareSourceCode();
        assertEquals("Source", source.getName());
        assertEquals("Java", source.getProgrammingLanguage());
        assertEquals("JDK", source.getSoftwareRequirements());
        assertEquals(ORGANIZATION, source.getCreators(0));
        assertEquals(CATEGORY, source.getCategories(0));
        assertEquals(CONTACT, source.getContactPoints(0));
        assertEquals("doi", source.getIdentifiers(0).getType());
    }

    @Test
    void requiresBindingsInsteadOfCreatingReferencedEntities() {
        var creator = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.ORGANIZATION, "https://example.org/org");
        var mapping = new SoftwareMapping("https://example.org/app", SoftwareMapping.Kind.APPLICATION, "App", "Description",
                List.of(), null, null, null, null, null, null, null, null, null, List.of(), List.of(creator), List.of(), List.of());
        assertThrows(IllegalArgumentException.class, () -> new SoftwareRequestMapper().toRequest(mapping, GROUP, new ReferenceBindings()));
    }

    @Test
    void acceptsSourceCodeWithoutDescription() {
        var mapping = new SoftwareMapping("https://example.org/source", SoftwareMapping.Kind.SOURCE_CODE, "Source", null,
                List.of(), null, null, null, null, null, null, null, null, null, List.of(), List.of(), List.of(), List.of());

        var request = new SoftwareRequestMapper().toRequest(mapping, GROUP, new ReferenceBindings());

        assertEquals("Source", request.getSoftwareSourceCode().getName());
        assertEquals("", request.getSoftwareSourceCode().getDescription());
    }
}
