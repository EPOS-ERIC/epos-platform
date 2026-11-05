package eu.epos.semanticaggregator.identity;

import java.io.IOException;
import java.net.URI;
import java.nio.channels.FileChannel;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;
import java.util.function.Supplier;

/** Durable ownership of legacy ConceptScheme IRIs by lifecycle groups. */
public final class SchemeIdentityStore {
    private final Path directory;
    private final Path file;

    public SchemeIdentityStore(Path baseDirectory, String mappingId, int mappingVersion) {
        directory = baseDirectory.toAbsolutePath().normalize().resolve(mappingId + "-v" + mappingVersion);
        file = directory.resolve("schemes.tsv");
    }

    /** Runs creation while holding the durable registry lock, preventing duplicate groups across processes. */
    public UUID resolveOrCreate(String schemeIri, Supplier<UUID> create) throws IOException {
        if (!URI.create(schemeIri).isAbsolute() || schemeIri.contains("\t") || schemeIri.contains("\n")) {
            throw new IllegalArgumentException("scheme IRI must be an absolute IRI without control characters");
        }
        Files.createDirectories(directory);
        try (var channel = FileChannel.open(directory.resolve("schemes.lock"),
                java.nio.file.StandardOpenOption.CREATE, java.nio.file.StandardOpenOption.WRITE);
             var lock = channel.lock()) {
            var values = read();
            UUID existing = values.get(schemeIri);
            if (existing != null) return existing;
            UUID created = create.get();
            if (created == null || created.equals(new UUID(0, 0))) throw new IllegalArgumentException("group creation returned an invalid ID");
            values.put(schemeIri, created);
            write(values);
            return created;
        }
    }

    private Map<String, UUID> read() throws IOException {
        if (!Files.exists(file)) return new LinkedHashMap<>();
        Map<String, UUID> result = new LinkedHashMap<>();
        for (String line : Files.readAllLines(file, StandardCharsets.UTF_8)) {
            if (line.isBlank()) continue;
            String[] parts = line.split("\t", -1);
            if (parts.length != 2 || !URI.create(parts[0]).isAbsolute()) throw new IOException("invalid scheme identity row");
            UUID id;
            try { id = UUID.fromString(parts[1]); } catch (IllegalArgumentException exception) { throw new IOException("invalid scheme identity row", exception); }
            if (id.equals(new UUID(0, 0)) || result.put(parts[0], id) != null) throw new IOException("invalid scheme identity row");
        }
        return result;
    }

    private void write(Map<String, UUID> values) throws IOException {
        Path temporary = Files.createTempFile(directory, "schemes-", ".tmp");
        try {
            StringBuilder output = new StringBuilder();
            values.entrySet().stream().sorted(Map.Entry.comparingByKey()).forEach(entry ->
                    output.append(entry.getKey()).append('\t').append(entry.getValue()).append('\n'));
            Files.writeString(temporary, output, StandardCharsets.UTF_8);
            Files.move(temporary, file, StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING);
        } finally { Files.deleteIfExists(temporary); }
    }
}
