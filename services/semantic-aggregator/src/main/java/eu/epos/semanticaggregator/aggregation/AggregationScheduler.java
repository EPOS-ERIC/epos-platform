package eu.epos.semanticaggregator.aggregation;

import java.time.Duration;
import java.util.Objects;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.logging.Level;
import java.util.logging.Logger;

/** Opt-in background refresher that keeps all aggregate graph versions independent. */
public final class AggregationScheduler implements AutoCloseable {
    private static final Logger LOGGER = Logger.getLogger(AggregationScheduler.class.getName());

    private final AggregateRefresh refresh;
    private final ScheduledExecutorService executor;
    private final AutoCloseable resource;

    public AggregationScheduler(AggregateRefresh refresh, Duration interval) {
        this(refresh, interval, () -> {}, Executors.newSingleThreadScheduledExecutor(Thread.ofVirtual()
                .name("semantic-aggregation-refresh-", 0).factory()));
    }

    public AggregationScheduler(AggregateRefresh refresh, Duration interval, AutoCloseable resource) {
        this(refresh, interval, resource, Executors.newSingleThreadScheduledExecutor(Thread.ofVirtual()
                .name("semantic-aggregation-refresh-", 0).factory()));
    }

    AggregationScheduler(AggregateRefresh refresh, Duration interval, AutoCloseable resource, ScheduledExecutorService executor) {
        this.refresh = Objects.requireNonNull(refresh, "refresh");
        Objects.requireNonNull(interval, "interval");
        if (interval.isNegative() || interval.isZero()) throw new IllegalArgumentException("refresh interval must be positive");
        this.resource = Objects.requireNonNull(resource, "resource");
        this.executor = Objects.requireNonNull(executor, "executor");
        executor.scheduleWithFixedDelay(this::refreshAll, 0, interval.toMillis(), TimeUnit.MILLISECONDS);
    }

    private void refreshAll() {
        for (var version : AggregateRefresh.Version.values()) {
            try {
                refresh.refresh(version);
            } catch (Exception exception) {
                // Each graph is refreshed independently; a later interval retries failures without replacing it.
                LOGGER.log(Level.WARNING, "Failed to refresh aggregate graph " + version, exception);
            }
        }
    }

    @Override
    public void close() {
        executor.shutdown();
        try {
            if (!executor.awaitTermination(10, TimeUnit.SECONDS)) executor.shutdownNow();
        } catch (InterruptedException exception) {
            executor.shutdownNow();
            Thread.currentThread().interrupt();
        } finally {
            try {
                resource.close();
            } catch (Exception exception) {
                LOGGER.log(Level.WARNING, "Failed to close aggregate refresh resource", exception);
            }
        }
    }
}
