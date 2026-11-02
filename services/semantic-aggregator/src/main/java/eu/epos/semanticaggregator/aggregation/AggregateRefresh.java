package eu.epos.semanticaggregator.aggregation;

import java.io.IOException;
import java.util.EnumMap;
import java.util.HashSet;
import java.util.List;
import java.util.Objects;
import java.util.Set;
import java.util.concurrent.locks.ReentrantLock;
import org.apache.jena.rdf.model.Model;

/** Builds a complete graph before replacing the corresponding published graph. */
public final class AggregateRefresh {
    public enum Version {
        V1("urn:epos:semantic-aggregator:aggregate:v1"),
        V2("urn:epos:semantic-aggregator:aggregate:v2"),
        V3("urn:epos:semantic-aggregator:aggregate:v3");

        private final String graphIri;
        Version(String graphIri) { this.graphIri = graphIri; }
        public String graphIri() { return graphIri; }
    }

    @FunctionalInterface
    public interface Inventory {
        List<PublishedResourceRef> fetch();
    }

    @FunctionalInterface
    public interface Builder {
        /** Returns an owned model and the exact published versions it contains. */
        BuiltGraph build(Version version, List<PublishedResourceRef> resources) throws IOException, InterruptedException;
    }

    public record BuiltGraph(Model model, Set<PublishedResourceRef> included) {
        public BuiltGraph {
            Objects.requireNonNull(model, "model");
            included = Set.copyOf(included);
        }
    }

    private static final EnumMap<Version, ReentrantLock> LOCKS = new EnumMap<>(Version.class);
    static {
        for (Version version : Version.values()) LOCKS.put(version, new ReentrantLock());
    }

    private final Inventory inventory;
    private final Builder builder;
    private final GraphPublisher publisher;

    public AggregateRefresh(Inventory inventory, Builder builder, GraphPublisher publisher) {
        this.inventory = Objects.requireNonNull(inventory, "inventory");
        this.builder = Objects.requireNonNull(builder, "builder");
        this.publisher = Objects.requireNonNull(publisher, "publisher");
    }

    public void refresh(Version version) throws IOException, InterruptedException {
        Objects.requireNonNull(version, "version");
        var lock = LOCKS.get(version);
        if (!lock.tryLock()) throw new IllegalStateException("refresh already running for " + version);
        try {
            List<PublishedResourceRef> before = List.copyOf(inventory.fetch());
            Set<PublishedResourceRef> expected = unique(before);
            BuiltGraph graph = builder.build(version, before);
            try {
                if (!graph.included().equals(expected)) {
                    throw new IllegalStateException("rendered graph does not cover the complete published inventory");
                }
                if (!unique(inventory.fetch()).equals(expected)) {
                    throw new IllegalStateException("published inventory changed during refresh");
                }
                publisher.replace(version.graphIri(), graph.model());
            } finally {
                graph.model().close();
            }
        } finally {
            lock.unlock();
        }
    }

    private static Set<PublishedResourceRef> unique(List<PublishedResourceRef> resources) {
        var unique = new HashSet<>(resources);
        if (unique.size() != resources.size() || unique.contains(null)) {
            throw new IllegalStateException("published inventory contains duplicate or null entries");
        }
        return Set.copyOf(unique);
    }
}
