package eu.epos.semanticaggregator.identity;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.nio.file.Path;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

class DatasetIdentityStoreTest {
    @TempDir Path directory;

    @Test
    void pendingBlocksAndCompleteAllowsCancellingOnlyPending() throws Exception {
        var store = new DatasetIdentityStore(directory, UUID.randomUUID());
        String iri = "https://example.org/dataset";
        try (var first = store.acquire()) {
            first.begin(iri);
            var pending = assertThrows(DatasetIdentityStore.ExistingDatasetException.class, () -> first.requireNew(iri));
            assertTrue(pending.pending());
            assertThrows(IllegalArgumentException.class, () -> first.complete(iri, new UUID(0, 0), new UUID(0, 0)));
            assertThrows(IOException.class, () -> first.cancel("https://example.org/other"));
            first.complete(iri, UUID.randomUUID(), UUID.randomUUID());
            var completed = assertThrows(DatasetIdentityStore.ExistingDatasetException.class, () -> first.requireNew(iri));
            assertFalse(completed.pending());
        }
    }

    @Test
    void secondAcquisitionBlocksWhileLeaseIsOpen() throws Exception {
        var store = new DatasetIdentityStore(directory, UUID.randomUUID());
        try (var ignored = store.acquire()) {
            assertThrows(DatasetIdentityStore.ImportBusyException.class, store::acquire);
        }
        try (var second = store.acquire()) {
            second.begin("https://example.org/second");
        }
    }

    @Test
    void rejectsInvalidIrisAndDoesNotPersistCancellationWithoutPending() throws Exception {
        var store = new DatasetIdentityStore(directory, UUID.randomUUID());
        try (var lease = store.acquire()) {
            assertThrows(IllegalArgumentException.class, () -> lease.begin("relative"));
            assertThrows(IOException.class, () -> lease.complete("https://example.org/missing", UUID.randomUUID(), UUID.randomUUID()));
        }
    }
}
