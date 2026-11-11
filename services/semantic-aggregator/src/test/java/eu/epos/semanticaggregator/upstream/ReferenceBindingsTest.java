package eu.epos.semanticaggregator.upstream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.ingest.DatasetMapping;
import java.util.List;
import org.junit.jupiter.api.Test;

class ReferenceBindingsTest {
    private static final DatasetMapping.ReferenceKind ORGANIZATION = DatasetMapping.ReferenceKind.ORGANIZATION;
    private static final String ID = "123e4567-e89b-12d3-a456-426614174001";

    @Test
    void validatesBindingsAndRejectsDuplicateReferences() {
        var bindings = new ReferenceBindings();
        bindings.register(ORGANIZATION, "https://example.org/a", ID);
        assertEquals(ID, bindings.resolve(new DatasetMapping.Reference(ORGANIZATION, "https://example.org/a")));
        assertThrows(IllegalArgumentException.class, () -> bindings.register(ORGANIZATION, "https://example.org/a", ID));
        assertThrows(IllegalArgumentException.class, () -> bindings.register(ORGANIZATION, "relative", ID));
        assertThrows(IllegalArgumentException.class, () -> bindings.register(ORGANIZATION, "https://example.org/b", "1-1-1-1-1"));
        assertThrows(IllegalArgumentException.class, () -> bindings.register(ORGANIZATION, "https://example.org/b", "00000000-0000-0000-0000-000000000000"));
        assertThrows(IllegalArgumentException.class, () -> bindings.resolve(new DatasetMapping.Reference(
                DatasetMapping.ReferenceKind.CONTACT_POINT, "https://example.org/a")));
    }

    @Test
    void deduplicatesTwoIrisBoundToTheSameEntityInOneList() {
        var bindings = new ReferenceBindings();
        bindings.register(ORGANIZATION, "https://example.org/a", ID);
        bindings.register(ORGANIZATION, "https://example.org/b", ID);
        assertEquals(List.of(ID), bindings.resolveAll(List.of(
                new DatasetMapping.Reference(ORGANIZATION, "https://example.org/a"),
                new DatasetMapping.Reference(ORGANIZATION, "https://example.org/b"))));
    }

    @Test
    void omitsUnresolvedContactPoints() {
        var bindings = new ReferenceBindings();
        bindings.register(ORGANIZATION, "https://example.org/organization", ID);

        assertEquals(List.of(ID), bindings.resolveAll(List.of(
                new DatasetMapping.Reference(ORGANIZATION, "https://example.org/organization"),
                new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CONTACT_POINT, "https://example.org/contact"))));
    }
}
