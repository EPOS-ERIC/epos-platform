package eu.epos.semanticaggregator.upstream;

import eu.epos.semanticaggregator.ingest.DatasetMapping;
import java.net.URI;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;

/** Explicit links from RDF IRIs to existing Metadata Lifecycle entities. */
public final class ReferenceBindings {
    private final Map<DatasetMapping.Reference, String> ids = new HashMap<>();
    private final java.util.Set<ResolvedReference> used = new HashSet<>();
    private final Map<DatasetMapping.Reference, String> usedIris = new HashMap<>();

    public void register(DatasetMapping.ReferenceKind kind, String iri, String id) {
        Objects.requireNonNull(kind, "kind");
        if (!URI.create(iri).isAbsolute()) throw new IllegalArgumentException("reference IRI must be absolute");
        UUID parsed = UUID.fromString(id);
        if (!parsed.toString().equalsIgnoreCase(id) || parsed.equals(new UUID(0, 0))) {
            throw new IllegalArgumentException("reference ID must be a nonzero canonical UUID");
        }
        var reference = new DatasetMapping.Reference(kind, iri);
        if (ids.putIfAbsent(reference, parsed.toString()) != null) {
            throw new IllegalArgumentException("duplicate reference binding: " + kind + " " + iri);
        }
    }

    public String resolve(DatasetMapping.Reference reference) {
        return resolve(reference, true);
    }

    public String resolve(DatasetMapping.Reference reference, boolean track) {
        String id = ids.get(reference);
        if (id == null) throw new IllegalArgumentException("missing reference binding: " + reference.kind() + " " + reference.iri());
        if (track) {
            used.add(new ResolvedReference(reference.kind(), id));
            usedIris.put(reference, id);
        }
        return id;
    }

    public boolean contains(DatasetMapping.Reference reference) {
        return ids.containsKey(reference);
    }

    public boolean containsId(String id) {
        return ids.containsValue(id);
    }

    /** Replaces a local planning placeholder after the upstream entity is resolved. */
    public void replacePlaceholder(DatasetMapping.Reference reference, String placeholder, String actualId) {
        if (!placeholder.equals(ids.get(reference))) throw new IllegalArgumentException("reference placeholder changed");
        UUID parsed = UUID.fromString(actualId);
        if (!parsed.toString().equalsIgnoreCase(actualId) || parsed.equals(new UUID(0, 0))) {
            throw new IllegalArgumentException("reference ID must be a nonzero canonical UUID");
        }
        ids.put(reference, parsed.toString());
        if (usedIris.containsKey(reference)) usedIris.put(reference, parsed.toString());
        used.clear();
        usedIris.forEach((key, id) -> used.add(new ResolvedReference(key.kind(), id)));
    }

    public List<String> resolveAll(List<DatasetMapping.Reference> references) {
        List<String> result = new ArrayList<>();
        var seen = new HashSet<String>();
        for (var reference : references) {
            if (reference.kind() == DatasetMapping.ReferenceKind.CONTACT_POINT && !contains(reference)) continue;
            String id = resolve(reference);
            if (!seen.add(id)) throw new IllegalArgumentException("multiple references map to the same entity: " + id);
            result.add(id);
        }
        return List.copyOf(result);
    }

    public List<ResolvedReference> used() {
        return used.stream().sorted(java.util.Comparator.comparing(ResolvedReference::kind)
                .thenComparing(ResolvedReference::id)).toList();
    }

    public Map<DatasetMapping.Reference, String> usedIris() {
        return Map.copyOf(usedIris);
    }

    public record ResolvedReference(DatasetMapping.ReferenceKind kind, String id) {}
}
