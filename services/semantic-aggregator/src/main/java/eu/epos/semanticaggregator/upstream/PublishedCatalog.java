package eu.epos.semanticaggregator.upstream;

import eu.epos.semanticaggregator.aggregation.PublishedResourceRef;
import io.grpc.ManagedChannel;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashSet;
import java.util.List;
import java.util.Objects;
import java.util.UUID;
import java.util.concurrent.TimeUnit;
import metadata_lifecycle.v1.Common;
import metadata_lifecycle.v1.Discovery;
import metadata_lifecycle.v1.DiscoveryServiceGrpc;
import metadata_lifecycle.v1.Resource;
import metadata_lifecycle.v1.ResourceServiceGrpc;

/** Reads the complete public published inventory from Metadata Lifecycle. */
public final class PublishedCatalog {
    private final ManagedChannel channel;

    public PublishedCatalog(ManagedChannel channel) {
        this.channel = Objects.requireNonNull(channel, "channel");
    }

    /** Discovery currently returns a single unpaginated response; failures are never treated as an empty catalog. */
    public List<PublishedResourceRef> fetch() {
        var request = Discovery.SearchResourcesRequest.newBuilder()
                .addVersioningStatuses(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED).build();
        var response = DiscoveryServiceGrpc.newBlockingStub(channel)
                .withMaxInboundMessageSize(64 * 1024 * 1024)
                .withDeadlineAfter(60, TimeUnit.SECONDS).searchResources(request);
        var resources = new ArrayList<PublishedResourceRef>();
        var versions = new HashSet<String>();
        var identities = new HashSet<String>();
        for (var item : response.getItemsList()) {
            PublishedResourceRef resource = switch (item.getResourceCase()) {
                case DATASET -> published(PublishedResourceRef.Kind.DATASET, item.getDataset().getResourceId(),
                        item.getDataset().getVersionId(), item.getDataset().getStatus());
                case SOFTWARE_APPLICATION -> published(PublishedResourceRef.Kind.SOFTWARE_APPLICATION,
                        item.getSoftwareApplication().getResourceId(), item.getSoftwareApplication().getVersionId(),
                        item.getSoftwareApplication().getStatus());
                case SOFTWARE_SOURCE_CODE -> published(PublishedResourceRef.Kind.SOFTWARE_SOURCE_CODE,
                        item.getSoftwareSourceCode().getResourceId(), item.getSoftwareSourceCode().getVersionId(),
                        item.getSoftwareSourceCode().getStatus());
                default -> throw new IllegalStateException("Discovery returned an item without a supported resource");
            };
            if (!versions.add(resource.kind() + ":" + resource.versionId())
                    || !identities.add(resource.kind() + ":" + resource.resourceId())) {
                throw new IllegalStateException("Discovery returned duplicate published resources or versions");
            }
            resources.add(resource);
        }
        resources.sort(Comparator.comparing(PublishedResourceRef::kind)
                .thenComparing(item -> item.resourceId().toString())
                .thenComparing(item -> item.versionId().toString()));
        return List.copyOf(resources);
    }

    /** Fails closed when the published inventory differs from the earlier read. */
    public void requireUnchanged(List<PublishedResourceRef> earlier) {
        if (!fetch().equals(earlier)) throw new IllegalStateException("Published inventory changed during aggregation");
    }

    /** Fetches the complete wire representation for conversion at the upstream boundary. */
    public Resource.ResourceVersion fetchVersion(PublishedResourceRef reference) {
        Objects.requireNonNull(reference, "reference");
        var type = switch (reference.kind()) {
            case DATASET -> Common.ResourceType.RESOURCE_TYPE_DATASET;
            case SOFTWARE_APPLICATION -> Common.ResourceType.RESOURCE_TYPE_SOFTWARE_APPLICATION;
            case SOFTWARE_SOURCE_CODE -> Common.ResourceType.RESOURCE_TYPE_SOFTWARE_SOURCE_CODE;
        };
        var request = Resource.GetResourceVersionRequest.newBuilder().setResourceType(type)
                .setResourceId(reference.resourceId().toString()).setVersionId(reference.versionId().toString()).build();
        var result = ResourceServiceGrpc.newBlockingStub(channel).withMaxInboundMessageSize(64 * 1024 * 1024)
                .withDeadlineAfter(30, TimeUnit.SECONDS).getResourceVersion(request);
        if (!result.hasResource()) throw new IllegalStateException("Metadata Lifecycle returned no resource version");
        var version = result.getResource();
        if (version.getStatus() != Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED
                || !uuid(version.getResourceId()).equals(reference.resourceId())
                || !uuid(version.getVersionId()).equals(reference.versionId())) {
            throw new IllegalStateException("Published resource version changed or does not match the requested identity");
        }
        boolean kindMatches = switch (reference.kind()) {
            case DATASET -> version.hasDataset();
            case SOFTWARE_APPLICATION -> version.hasSoftwareApplication();
            case SOFTWARE_SOURCE_CODE -> version.hasSoftwareSourceCode();
        };
        if (!kindMatches) throw new IllegalStateException("Published resource type does not match its metadata");
        return version;
    }

    private static PublishedResourceRef published(PublishedResourceRef.Kind kind, String resourceId, String versionId,
                                                  Common.VersioningStatus status) {
        if (status != Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED) {
            throw new IllegalStateException("Discovery returned a non-published resource");
        }
        return new PublishedResourceRef(kind, uuid(resourceId), uuid(versionId));
    }

    private static UUID uuid(String value) {
        try {
            UUID parsed = UUID.fromString(value);
            if (!parsed.toString().equalsIgnoreCase(value) || parsed.equals(new UUID(0, 0))) {
                throw new IllegalArgumentException("noncanonical UUID");
            }
            return parsed;
        } catch (IllegalArgumentException exception) {
            throw new IllegalStateException("Discovery returned an invalid resource identifier", exception);
        }
    }
}
