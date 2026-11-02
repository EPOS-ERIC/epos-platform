package eu.epos.semanticaggregator.identity;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import eu.epos.semanticaggregator.ingest.DatasetMapping;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

class IdentityStoreTest {
    @TempDir Path directory;

    @Test
    void scopesGroupsAndDoesNotReplaceExistingAssociations() throws IOException {
        var group = UUID.fromString("123e4567-e89b-12d3-a456-426614174000");
        var otherGroup = UUID.fromString("123e4567-e89b-12d3-a456-426614174001");
        var id = UUID.fromString("123e4567-e89b-12d3-a456-426614174002");
        var store = new IdentityStore(directory, "epos-dcat-common", 1, group);
        var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CONTACT_POINT, "https://example.org/contact");
        assertTrue(store.remember(reference, id).newAssociation());
        var again = store.remember(reference, id);
        assertFalse(again.newAssociation());
        assertEquals(id.toString(), again.resolved());
        assertThrows(IOException.class, () -> store.remember(reference, UUID.randomUUID()));
        var alias = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CONTACT_POINT, "https://example.org/alias");
        assertTrue(store.remember(alias, id).newAssociation());
        assertEquals(group, fileGroup(store.file()));
        var other = new IdentityStore(directory, "epos-dcat-common", 1, otherGroup);
        assertTrue(other.remember(reference, id).newAssociation());
    }

    @Test
    void rejectsCorruptRowsAndZeroUuid() throws IOException {
        var group = UUID.fromString("123e4567-e89b-12d3-a456-426614174000");
        var store = new IdentityStore(directory, "epos-dcat-common", 1, group);
        Files.createDirectories(store.file().getParent());
        Files.writeString(store.file(), "CONTACT_POINT\thttps://example.org/a\t00000000-0000-0000-0000-000000000000\n");
        assertThrows(IOException.class, store::snapshot);

        var valid = new IdentityStore(directory, "epos-dcat-common", 2, group);
        assertThrows(IllegalArgumentException.class, () -> valid.remember(new DatasetMapping.Reference(
                DatasetMapping.ReferenceKind.CONTACT_POINT, "https://example.org/a"), new UUID(0, 0)));
    }

    @Test
    void concurrentInstancesDoNotLoseAssociations() throws Exception {
        var group = UUID.randomUUID();
        var first = new IdentityStore(directory, "epos-dcat-common", 1, group);
        var second = new IdentityStore(directory, "epos-dcat-common", 1, group);
        try (var executor = java.util.concurrent.Executors.newFixedThreadPool(8)) {
            var tasks = new java.util.ArrayList<java.util.concurrent.Callable<Void>>();
            for (int index = 0; index < 40; index++) {
                int number = index;
                tasks.add(() -> {
                    var store = number % 2 == 0 ? first : second;
                    store.remember(new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CATEGORY,
                            "https://example.org/category/" + number), UUID.randomUUID());
                    return null;
                });
            }
            for (var future : executor.invokeAll(tasks)) future.get();
        }
        assertEquals(40, first.snapshot().size());
    }

    private static UUID fileGroup(Path file) {
        return UUID.fromString(file.getParent().getFileName().toString());
    }
}
