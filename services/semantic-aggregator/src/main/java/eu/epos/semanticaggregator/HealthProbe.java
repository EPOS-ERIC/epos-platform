package eu.epos.semanticaggregator;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.health.v1.HealthCheckRequest;
import io.grpc.health.v1.HealthCheckResponse;
import io.grpc.health.v1.HealthGrpc;
import java.util.concurrent.TimeUnit;

/** Container health check using the standard gRPC health protocol. */
public final class HealthProbe {
    private HealthProbe() {}

    public static void main(String[] args) {
        int port = Integer.parseInt(System.getenv().getOrDefault("GRPC_PORT", "8300"));
        ManagedChannel channel = ManagedChannelBuilder.forAddress("127.0.0.1", port).usePlaintext().build();
        try {
            HealthCheckResponse response = HealthGrpc.newBlockingStub(channel)
                    .withDeadlineAfter(3, TimeUnit.SECONDS)
                    .check(HealthCheckRequest.getDefaultInstance());
            if (response.getStatus() != HealthCheckResponse.ServingStatus.SERVING) {
                throw new IllegalStateException("gRPC server is not serving");
            }
        } finally {
            channel.shutdownNow();
        }
    }
}
