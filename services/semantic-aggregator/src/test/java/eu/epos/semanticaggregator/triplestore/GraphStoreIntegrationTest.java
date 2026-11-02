package eu.epos.semanticaggregator.triplestore;

import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

import java.net.URI;
import java.util.UUID;
import java.util.List;
import java.util.Set;
import eu.epos.semanticaggregator.aggregation.AggregateRefresh;
import eu.epos.semanticaggregator.aggregation.PublishedResourceRef;
import org.apache.jena.rdf.model.ModelFactory;
import org.junit.jupiter.api.Test;

class GraphStoreIntegrationTest {
    @Test
    void replacesOnlyTheSelectedGraph() throws Exception {
        String repository = System.getenv("TRIPLESTORE_TEST_ENDPOINT");
        assumeTrue(repository != null && !repository.isBlank());
        String username = System.getenv("TRIPLESTORE_TEST_USERNAME");
        var store = username == null ? new GraphStore(URI.create(repository), null)
                : GraphStore.withBasicAuth(URI.create(repository), username, System.getenv("TRIPLESTORE_TEST_PASSWORD"));
        String graph = "urn:epos:semantic-aggregator:test:" + UUID.randomUUID();
        var first = ModelFactory.createDefaultModel();
        var second = ModelFactory.createDefaultModel();
        AutoCloseable cleanup = () -> store.delete(graph);
        try (cleanup) {
            first.add(first.createResource("urn:test:first"), first.createProperty("urn:test:value"), "first");
            second.add(second.createResource("urn:test:second"), second.createProperty("urn:test:value"), "second");
            store.replace(graph, first);
            var actualFirst = store.read(graph).orElseThrow();
            try {
                assertTrue(first.isIsomorphicWith(actualFirst));
            } finally {
                actualFirst.close();
            }
            store.replace(graph, second);
            var actualSecond = store.read(graph).orElseThrow();
            try {
                assertTrue(second.isIsomorphicWith(actualSecond));
            } finally {
                actualSecond.close();
            }
        } finally {
            first.close();
            second.close();
        }
    }

    @Test
    void failedRefreshDoesNotReplaceExistingGraph() throws Exception {
        String repository = System.getenv("TRIPLESTORE_TEST_ENDPOINT");
        assumeTrue(repository != null && !repository.isBlank());
        String username = System.getenv("TRIPLESTORE_TEST_USERNAME");
        var store = username == null ? new GraphStore(URI.create(repository), null)
                : GraphStore.withBasicAuth(URI.create(repository), username, System.getenv("TRIPLESTORE_TEST_PASSWORD"));
        String graph = "urn:epos:semantic-aggregator:test:" + UUID.randomUUID();
        var original = ModelFactory.createDefaultModel();
        var resource = new PublishedResourceRef(PublishedResourceRef.Kind.DATASET, UUID.randomUUID(), UUID.randomUUID());
        AutoCloseable cleanup = () -> store.delete(graph);
        try (cleanup) {
            original.add(original.createResource("urn:test:original"), original.createProperty("urn:test:value"), "original");
            store.replace(graph, original);
            var refresh = new AggregateRefresh(() -> List.of(resource),
                    (version, resources) -> new AggregateRefresh.BuiltGraph(ModelFactory.createDefaultModel(), Set.of()),
                    (ignored, model) -> store.replace(graph, model));
            org.junit.jupiter.api.Assertions.assertThrows(IllegalStateException.class,
                    () -> refresh.refresh(AggregateRefresh.Version.V1));
            var actual = store.read(graph).orElseThrow();
            try {
                assertTrue(original.isIsomorphicWith(actual));
            } finally {
                actual.close();
            }
        } finally {
            original.close();
        }
    }
}
