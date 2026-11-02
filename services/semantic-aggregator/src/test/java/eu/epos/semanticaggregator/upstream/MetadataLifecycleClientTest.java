package eu.epos.semanticaggregator.upstream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptors;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.ServerInterceptor;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;
import metadata_lifecycle.v1.Resource;
import metadata_lifecycle.v1.ResourceServiceGrpc;
import org.junit.jupiter.api.Test;

class MetadataLifecycleClientTest {
    @Test
    void forwardsActorAndRequestAndPropagatesGrpcFailures() throws Exception {
        var actor = new AtomicReference<String>();
        var group = new AtomicReference<String>();
        Metadata.Key<String> actorKey = Metadata.Key.of("x-epos-user-id", Metadata.ASCII_STRING_MARSHALLER);
        var service = new ResourceServiceGrpc.ResourceServiceImplBase() {
            @Override
            public void createResource(Resource.CreateResourceRequest request,
                                       StreamObserver<Resource.CreateResourceResponse> response) {
                group.set(request.getGroupId());
                response.onError(Status.PERMISSION_DENIED.asRuntimeException());
            }
        };
        Server server = ServerBuilder.forPort(0).addService(ServerInterceptors.intercept(service,
                new ServerInterceptor() {
                    @Override
                    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
                            ServerCall<ReqT, RespT> call, Metadata headers, ServerCallHandler<ReqT, RespT> next) {
                        actor.set(headers.get(actorKey));
                        return next.startCall(call, headers);
                    }
                })).build().start();
        var channel = ManagedChannelBuilder.forAddress("127.0.0.1", server.getPort()).usePlaintext().build();
        try (var client = new MetadataLifecycleClient(channel)) {
            var request = Resource.CreateResourceRequest.newBuilder().setGroupId("group-id").build();
            var error = assertThrows(StatusRuntimeException.class, () -> client.createResource(request, "actor-id"));
            assertEquals(Status.Code.PERMISSION_DENIED, error.getStatus().getCode());
            assertEquals("actor-id", actor.get());
            assertEquals("group-id", group.get());
            assertThrows(IllegalArgumentException.class, () -> client.createResource(request, " "));
        } finally {
            channel.shutdownNow();
            server.shutdownNow();
            server.awaitTermination(5, TimeUnit.SECONDS);
        }
    }
}
