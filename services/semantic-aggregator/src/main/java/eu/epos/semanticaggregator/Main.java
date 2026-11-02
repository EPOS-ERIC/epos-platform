package eu.epos.semanticaggregator;

import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptors;
import io.grpc.ManagedChannelBuilder;
import io.grpc.protobuf.services.HealthStatusManager;
import eu.epos.semanticaggregator.grpc.ActorInterceptor;
import eu.epos.semanticaggregator.grpc.DatasetImportServer;
import eu.epos.semanticaggregator.mapping.MappingRegistry;
import eu.epos.semanticaggregator.upstream.MetadataLifecycleClient;
import java.io.IOException;
import java.nio.file.Path;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

public final class Main {
    private static final Logger LOGGER = Logger.getLogger(Main.class.getName());

    private Main() {}

    public static void main(String[] args) throws IOException, InterruptedException {
        int port = Integer.parseInt(System.getenv().getOrDefault("GRPC_PORT", "8300"));
        MappingRegistry mappings = MappingRegistry.withDefaults();
        String mappingDir = System.getenv("MAPPINGS_DIR");
        if (mappingDir != null && !mappingDir.isBlank()) {
            mappings.registerDirectory(Path.of(mappingDir));
        }
        LOGGER.info(() -> "Registered " + mappings.size() + " mapping profiles");
        String upstreamAddress = System.getenv().getOrDefault("METADATA_LIFECYCLE_ADDR", "localhost:5000");
        var upstream = new MetadataLifecycleClient(ManagedChannelBuilder.forTarget(upstreamAddress).usePlaintext().build());
        java.nio.file.Path identityState = Path.of(System.getenv().getOrDefault("IDENTITY_STATE_DIR", "/tmp/epos-semantic-aggregator-state"));
        HealthStatusManager health = new HealthStatusManager();
        Server server = ServerBuilder.forPort(port).maxInboundMessageSize(10 * 1024 * 1024 + 65536)
                .addService(health.getHealthService())
                .addService(ServerInterceptors.intercept(new DatasetImportServer(mappings, upstream, identityState), new ActorInterceptor()))
                .build().start();
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            health.enterTerminalState();
            server.shutdown();
            try {
                if (!server.awaitTermination(10, TimeUnit.SECONDS)) {
                    server.shutdownNow();
                }
            } catch (InterruptedException exception) {
                server.shutdownNow();
                Thread.currentThread().interrupt();
            }
            upstream.close();
        }));
        LOGGER.info(() -> "Semantic Aggregator gRPC listening on port " + port);
        server.awaitTermination();
    }
}
