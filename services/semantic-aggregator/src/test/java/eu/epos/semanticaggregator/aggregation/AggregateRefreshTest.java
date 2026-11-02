package eu.epos.semanticaggregator.aggregation;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.util.List;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;
import org.apache.jena.rdf.model.ModelFactory;
import org.junit.jupiter.api.Test;

class AggregateRefreshTest {
    private static final PublishedResourceRef RESOURCE = new PublishedResourceRef(PublishedResourceRef.Kind.DATASET,
            UUID.fromString("123e4567-e89b-12d3-a456-426614174001"),
            UUID.fromString("123e4567-e89b-12d3-a456-426614174002"));

    @Test
    void publishesCompleteStableGraphToTheRequestedVersion() throws Exception {
        var writes = new AtomicInteger();
        var graphIri = new AtomicReference<String>();
        var refresh = new AggregateRefresh(() -> List.of(RESOURCE),
                (version, resources) -> new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.copyOf(resources)),
                (iri, model) -> { graphIri.set(iri); writes.incrementAndGet(); });
        refresh.refresh(AggregateRefresh.Version.V2);
        assertEquals(1, writes.get());
        assertEquals(AggregateRefresh.Version.V2.graphIri(), graphIri.get());
    }

    @Test
    void doesNotPublishWhenBuilderOmitsAResourceOrInventoryChanges() {
        var writes = new AtomicInteger();
        var incomplete = new AggregateRefresh(() -> List.of(RESOURCE),
                (version, resources) -> new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.of()),
                (iri, model) -> writes.incrementAndGet());
        assertThrows(IllegalStateException.class, () -> incomplete.refresh(AggregateRefresh.Version.V1));

        var reads = new AtomicInteger();
        var changed = new AggregateRefresh(() -> reads.incrementAndGet() == 1 ? List.of(RESOURCE) : List.of(),
                (version, resources) -> new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.copyOf(resources)),
                (iri, model) -> writes.incrementAndGet());
        assertThrows(IllegalStateException.class, () -> changed.refresh(AggregateRefresh.Version.V1));
        assertEquals(0, writes.get());
    }

    @Test
    void doesNotPublishWhenBuildingFailsAndPropagatesStorageFailure() {
        var writes = new AtomicInteger();
        var failedBuild = new AggregateRefresh(() -> List.of(RESOURCE), (version, resources) -> {
            throw new IOException("upstream failed");
        }, (iri, model) -> writes.incrementAndGet());
        assertThrows(IOException.class, () -> failedBuild.refresh(AggregateRefresh.Version.V3));
        assertEquals(0, writes.get());

        var failedStore = new AggregateRefresh(() -> List.of(RESOURCE),
                (version, resources) -> new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.copyOf(resources)),
                (iri, model) -> { throw new IOException("storage failed"); });
        assertThrows(IOException.class, () -> failedStore.refresh(AggregateRefresh.Version.V3));
    }

    @Test
    void preventsConcurrentRefreshesOfTheSameVersionWithinTheProcess() throws Exception {
        var started = new CountDownLatch(1);
        var release = new CountDownLatch(1);
        var refresh = new AggregateRefresh(() -> List.of(RESOURCE), (version, resources) -> {
            started.countDown();
            if (!release.await(5, TimeUnit.SECONDS)) throw new IOException("test timed out");
            return new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.copyOf(resources));
        }, (iri, model) -> {});
        var another = new AggregateRefresh(() -> List.of(RESOURCE),
                (version, resources) -> new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.copyOf(resources)),
                (iri, model) -> {});
        try (var executor = Executors.newSingleThreadExecutor()) {
            var first = executor.submit(() -> { refresh.refresh(AggregateRefresh.Version.V1); return null; });
            try {
                assertTrue(started.await(5, TimeUnit.SECONDS));
                assertThrows(IllegalStateException.class, () -> another.refresh(AggregateRefresh.Version.V1));
            } finally {
                release.countDown();
            }
            first.get(5, TimeUnit.SECONDS);
        }
    }
}
