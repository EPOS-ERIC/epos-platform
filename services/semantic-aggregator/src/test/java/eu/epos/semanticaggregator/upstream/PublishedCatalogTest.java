package eu.epos.semanticaggregator.upstream;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.aggregation.PublishedResourceRef;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;
import metadata_lifecycle.v1.Common;
import metadata_lifecycle.v1.Discovery;
import metadata_lifecycle.v1.DiscoveryServiceGrpc;
import metadata_lifecycle.v1.Resource;
import metadata_lifecycle.v1.ResourceServiceGrpc;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

class PublishedCatalogTest {
    private static final String RESOURCE = "123e4567-e89b-12d3-a456-426614174001";
    private static final String VERSION = "123e4567-e89b-12d3-a456-426614174002";
    private final AtomicReference<Discovery.SearchResourcesResponse> response = new AtomicReference<>();
    private final AtomicReference<Discovery.SearchResourcesRequest> received = new AtomicReference<>();
    private final AtomicReference<Status> error = new AtomicReference<>();
    private final AtomicReference<Resource.GetResourceVersionResponse> version = new AtomicReference<>();
    private final AtomicReference<Resource.GetResourceVersionRequest> versionRequest = new AtomicReference<>();
    private Server server;
    private ManagedChannel channel;
    private PublishedCatalog catalog;

    @BeforeEach
    void start() throws Exception {
        server = ServerBuilder.forPort(0).addService(new DiscoveryServiceGrpc.DiscoveryServiceImplBase() {
            @Override
            public void searchResources(Discovery.SearchResourcesRequest request,
                                        StreamObserver<Discovery.SearchResourcesResponse> observer) {
                received.set(request);
                if (error.get() != null) {
                    observer.onError(error.get().asRuntimeException());
                } else {
                    observer.onNext(response.get());
                    observer.onCompleted();
                }
            }
        }).addService(new ResourceServiceGrpc.ResourceServiceImplBase() {
            @Override
            public void getResourceVersion(Resource.GetResourceVersionRequest request,
                                           StreamObserver<Resource.GetResourceVersionResponse> observer) {
                versionRequest.set(request);
                observer.onNext(version.get());
                observer.onCompleted();
            }
        }).build().start();
        channel = ManagedChannelBuilder.forAddress("127.0.0.1", server.getPort()).usePlaintext().build();
        catalog = new PublishedCatalog(channel);
    }

    @AfterEach
    void stop() throws InterruptedException {
        channel.shutdownNow();
        server.shutdownNow();
        server.awaitTermination(5, TimeUnit.SECONDS);
    }

    @Test
    void requestsOnlyPublishedAndDetectsInventoryChanges() {
        response.set(Discovery.SearchResourcesResponse.newBuilder().addItems(dataset(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED)).build());
        var snapshot = catalog.fetch();
        assertEquals(PublishedResourceRef.Kind.DATASET, snapshot.getFirst().kind());
        assertEquals(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED, received.get().getVersioningStatuses(0));
        catalog.requireUnchanged(snapshot);
        response.set(Discovery.SearchResourcesResponse.getDefaultInstance());
        assertThrows(IllegalStateException.class, () -> catalog.requireUnchanged(snapshot));
    }

    @Test
    void rejectsNonPublishedOrDuplicateRecords() {
        response.set(Discovery.SearchResourcesResponse.newBuilder().addItems(dataset(Common.VersioningStatus.VERSIONING_STATUS_DRAFT)).build());
        assertThrows(IllegalStateException.class, catalog::fetch);
        var item = dataset(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED);
        response.set(Discovery.SearchResourcesResponse.newBuilder().addItems(item).addItems(item).build());
        assertThrows(IllegalStateException.class, catalog::fetch);
    }

    @Test
    void propagatesUpstreamFailureInsteadOfReturningAnEmptyCatalog() {
        error.set(Status.UNAVAILABLE);
        assertEquals(Status.Code.UNAVAILABLE, assertThrows(StatusRuntimeException.class, catalog::fetch).getStatus().getCode());
    }

    @Test
    void fetchesFullPublishedVersionAndRejectsChangedOrMismatchedVersions() {
        response.set(Discovery.SearchResourcesResponse.newBuilder().addItems(dataset(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED)).build());
        var reference = catalog.fetch().getFirst();
        var data = Resource.ResourceVersion.newBuilder().setResourceId(RESOURCE).setVersionId(VERSION)
                .setStatus(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED)
                .setDataset(Resource.DatasetResourceMetadata.newBuilder().setTitle("Dataset")).build();
        version.set(Resource.GetResourceVersionResponse.newBuilder().setResource(data).build());
        assertEquals("Dataset", catalog.fetchVersion(reference).getDataset().getTitle());
        assertEquals(Common.ResourceType.RESOURCE_TYPE_DATASET, versionRequest.get().getResourceType());

        version.set(Resource.GetResourceVersionResponse.newBuilder().setResource(data.toBuilder()
                .setStatus(Common.VersioningStatus.VERSIONING_STATUS_ARCHIVED)).build());
        assertThrows(IllegalStateException.class, () -> catalog.fetchVersion(reference));
        version.set(Resource.GetResourceVersionResponse.newBuilder().setResource(data.toBuilder()
                .clearDataset().setSoftwareApplication(Resource.SoftwareApplicationResourceMetadata.getDefaultInstance())).build());
        assertThrows(IllegalStateException.class, () -> catalog.fetchVersion(reference));
        version.set(Resource.GetResourceVersionResponse.newBuilder().setResource(data.toBuilder()
                .setVersionId(RESOURCE)).build());
        assertThrows(IllegalStateException.class, () -> catalog.fetchVersion(reference));
    }

    private static Discovery.DiscoveryItem dataset(Common.VersioningStatus status) {
        return Discovery.DiscoveryItem.newBuilder().setDataset(Discovery.DatasetDiscoveryItem.newBuilder()
                .setResourceId(RESOURCE).setVersionId(VERSION).setStatus(status)).build();
    }
}
