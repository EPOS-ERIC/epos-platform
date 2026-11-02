package eu.epos.semanticaggregator.identity;

import eu.epos.semanticaggregator.ingest.DatasetMapping;
import java.io.IOException;
import java.net.URI;
import java.nio.channels.FileChannel;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.charset.StandardCharsets;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;

/**
 * Durable namespace-scoped IRI→UUID associations.
 *
 * <p>Files are isolated per mapping profile and group, so different profiles and groups never share identity ownership.
 * Values are TSV rows to avoid trusting JSON parsing for untrusted IRIs.
 */
public final class IdentityStore {
    private static final java.util.concurrent.ConcurrentMap<Path, Object> LOCKS = new java.util.concurrent.ConcurrentHashMap<>();
    private final Path directory;
    private final Path file;

    public IdentityStore(Path baseDirectory, String mappingId, int mappingVersion, UUID groupId) {
        Objects.requireNonNull(baseDirectory, "baseDirectory");
        validate(mappingId, mappingVersion, groupId);
        this.directory = baseDirectory.toAbsolutePath().normalize().resolve(mappingId + "-v" + mappingVersion).resolve(groupId.toString());
        this.file = directory.resolve("references.tsv");
    }

    public Path file() { return file; }

    /** Durable registry used during import planning. */
    public Map<DatasetMapping.Reference, UUID> snapshot() throws IOException {
        return Map.copyOf(read());
    }

    public record Resolution(String resolved, boolean newAssociation) {}

    /** Registers IRI→ID without replacing an existing association. */
    public Resolution remember(DatasetMapping.Reference reference, UUID id) throws IOException {
        Objects.requireNonNull(reference, "reference");
        Objects.requireNonNull(id, "id");
        validateReference(reference);
        if (id.equals(new UUID(0, 0))) throw new IllegalArgumentException("zero UUID is not allowed");
        Files.createDirectories(directory);
        synchronized (LOCKS.computeIfAbsent(file, ignored -> new Object())) {
            try (var channel = FileChannel.open(directory.resolve("references.lock"),
                    java.nio.file.StandardOpenOption.CREATE, java.nio.file.StandardOpenOption.WRITE);
                 var lock = channel.lock()) {
                var values = new LinkedHashMap<>(read());
                UUID existing = values.get(reference);
                if (existing != null) {
                    if (!existing.equals(id)) throw new IOException("identity conflicts with an existing association");
                    return new Resolution(existing.toString(), false);
                }
                values.put(reference, id);
                write(values);
                return new Resolution(id.toString(), true);
            }
        }
    }

    private Map<DatasetMapping.Reference, UUID> read() throws IOException {
        if (!Files.exists(file)) return Map.of();
        LinkedHashMap<DatasetMapping.Reference, UUID> values = new LinkedHashMap<>();
        int lineNumber = 0;
        for (String line : Files.readAllLines(file, StandardCharsets.UTF_8)) {
            lineNumber++;
            if (line.isBlank()) continue;
            String[] parts = line.split("\t", -1);
            if (parts.length != 3 || !isUuid(parts[2])) {
                throw new IOException("invalid identity store row " + lineNumber);
            }
            DatasetMapping.Reference reference;
            try {
                reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.valueOf(parts[0]), parts[1]);
                validateReference(reference);
            } catch (IllegalArgumentException exception) {
                throw new IOException("invalid identity store row " + lineNumber, exception);
            }
            UUID id = UUID.fromString(parts[2]);
            if (values.put(reference, id) != null) throw new IOException("duplicate IRI in identity store");
        }
        return values;
    }

    private void write(Map<DatasetMapping.Reference, UUID> values) throws IOException {
        Path temporary = Files.createTempFile(directory, "references-", ".tmp");
        try {
            var output = new StringBuilder();
            values.entrySet().stream().sorted(Map.Entry.comparingByKey(java.util.Comparator.comparing((DatasetMapping.Reference r) -> r.kind().name())
                    .thenComparing(DatasetMapping.Reference::iri))).forEach(entry ->
                    output.append(entry.getKey().kind()).append('\t').append(entry.getKey().iri()).append('\t').append(entry.getValue()).append('\n'));
            Files.writeString(temporary, output.toString(), StandardCharsets.UTF_8);
            Files.move(temporary, file, StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING);
        } finally {
            Files.deleteIfExists(temporary);
        }
    }

    private static void validateReference(DatasetMapping.Reference reference) {
        Objects.requireNonNull(reference.kind(), "reference kind");
        String iri = Objects.requireNonNull(reference.iri(), "reference IRI");
        if (iri.indexOf('\t') >= 0 || iri.indexOf('\n') >= 0 || iri.indexOf('\r') >= 0 || !URI.create(iri).isAbsolute()) {
            throw new IllegalArgumentException("reference IRI must be absolute and contain no control characters");
        }
    }

    private static boolean isUuid(String value) {
        try {
            var parsed = UUID.fromString(value);
            return parsed.toString().equalsIgnoreCase(value) && !parsed.equals(new UUID(0, 0));
        } catch (IllegalArgumentException exception) {
            return false;
        }
    }

    private static void validate(String mappingId, int mappingVersion, UUID groupId) {
        if (mappingId == null || !mappingId.matches("[a-z0-9]+(?:-[a-z0-9]+)*") || mappingVersion < 1 || groupId == null || groupId.equals(new UUID(0, 0))) {
            throw new IllegalArgumentException("identity scope is invalid");
        }
    }
}
