package eu.epos.semanticaggregator.aggregation;

import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.List;
import java.util.Set;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import org.apache.jena.rdf.model.ModelFactory;
import org.junit.jupiter.api.Test;

class AggregationSchedulerTest {
    @Test
    void refreshesEveryVersionAndClosesItsOwnedResource() throws Exception {
        var writes = new CountDownLatch(AggregateRefresh.Version.values().length);
        var closed = new AtomicBoolean();
        var refresh = new AggregateRefresh(List::of,
                (version, resources) -> new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.of()),
                (graph, model) -> writes.countDown());
        var executor = Executors.newSingleThreadScheduledExecutor();
        var scheduler = new AggregationScheduler(refresh, Duration.ofHours(1), () -> closed.set(true), executor);
        try {
            assertTrue(writes.await(5, TimeUnit.SECONDS));
        } finally {
            scheduler.close();
        }
        assertTrue(closed.get());
    }
}
