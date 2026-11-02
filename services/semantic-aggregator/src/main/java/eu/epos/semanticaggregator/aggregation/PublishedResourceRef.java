package eu.epos.semanticaggregator.aggregation;

import java.util.UUID;

/** Identifies a published resource version without exposing transport types. */
public record PublishedResourceRef(Kind kind, UUID resourceId, UUID versionId) {
    public enum Kind { DATASET, SOFTWARE_APPLICATION, SOFTWARE_SOURCE_CODE }

    public PublishedResourceRef {
        java.util.Objects.requireNonNull(kind, "kind");
        java.util.Objects.requireNonNull(resourceId, "resourceId");
        java.util.Objects.requireNonNull(versionId, "versionId");
        if (resourceId.equals(new UUID(0, 0)) || versionId.equals(new UUID(0, 0))) {
            throw new IllegalArgumentException("published resource IDs must be nonzero");
        }
    }
}
