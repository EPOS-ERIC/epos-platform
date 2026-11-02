package eu.epos.semanticaggregator.identity;

import java.io.IOException;
import java.net.URI;
import java.nio.channels.FileChannel;
import java.nio.channels.FileLock;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.StandardOpenOption;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.locks.ReentrantLock;

/** Group-scoped import ledger. An unresolved intent blocks automatic re-creation. */
public final class DatasetIdentityStore {
    private static final Map<Path, ReentrantLock> LOCKS = new ConcurrentHashMap<>();
    private final Path directory;

    public DatasetIdentityStore(Path baseDirectory, UUID groupId) {
        if (groupId == null || groupId.equals(new UUID(0, 0))) throw new IllegalArgumentException("group ID is required");
        directory = baseDirectory.toAbsolutePath().normalize().resolve("datasets").resolve(groupId.toString());
    }

    /** A lease serializes imports across mapping versions for this group. */
    public Lease acquire() throws IOException {
        Files.createDirectories(directory);
        Path lockPath = directory.resolve("imports.lock");
        var local = LOCKS.computeIfAbsent(lockPath, ignored -> new ReentrantLock());
        if (!local.tryLock()) throw new ImportBusyException();
        FileChannel channel = null;
        try {
            channel = FileChannel.open(lockPath, StandardOpenOption.CREATE, StandardOpenOption.WRITE);
            FileLock lock;
            try {
                lock = channel.tryLock();
            } catch (java.nio.channels.OverlappingFileLockException exception) {
                throw new ImportBusyException();
            }
            if (lock == null) throw new ImportBusyException();
            return new Lease(directory, local, channel, lock);
        } catch (IOException | RuntimeException exception) {
            if (channel != null) channel.close();
            local.unlock();
            throw exception;
        }
    }

    public record ResolvedIdentity(String resourceId, String versionId) {}

    public static final class ImportBusyException extends IOException {
        private static final long serialVersionUID = 1L;
        public ImportBusyException() { super("another dataset import is running for this group"); }
    }

    public static final class Lease implements AutoCloseable {
        private final Path directory;
        private final Path file;
        private final ReentrantLock local;
        private final FileChannel channel;
        private final FileLock lock;

        private Lease(Path directory, ReentrantLock local, FileChannel channel, FileLock lock) {
            this.directory = directory;
            this.file = directory.resolve("datasets.tsv");
            this.local = local;
            this.channel = channel;
            this.lock = lock;
        }

        public void requireNew(String iri) throws IOException {
            validateIri(iri);
            String state = read().get(iri);
            if (state != null) throw new ExistingDatasetException(state.equals("PENDING"));
        }

        public ResolvedIdentity existing(String iri) throws IOException {
            validateIri(iri);
            String state = read().get(iri);
            if (state == null || state.equals("PENDING")) return null;
            var parts = state.split("/", -1);
            if (parts.length != 2 || !isUuid(parts[0]) || !isUuid(parts[1])) return null;
            return new ResolvedIdentity(parts[0], parts[1]);
        }

        public void begin(String iri) throws IOException {
            requireNew(iri);
            var values = read();
            values.put(iri, "PENDING");
            write(values);
        }

        public void complete(String iri, UUID resourceId, UUID versionId) throws IOException {
            if (resourceId == null || resourceId.equals(new UUID(0, 0)) || versionId == null || versionId.equals(new UUID(0, 0))) {
                throw new IllegalArgumentException("dataset IDs are invalid");
            }
            var values = read();
            if (!"PENDING".equals(values.get(iri))) throw new IOException("dataset import intent is missing");
            values.put(iri, resourceId + "/" + versionId);
            write(values);
        }

        public void cancel(String iri) throws IOException {
            var values = read();
            if (!"PENDING".equals(values.get(iri))) throw new IOException("dataset import intent is missing");
            values.remove(iri);
            write(values);
        }

        private Map<String, String> read() throws IOException {
            var values = new LinkedHashMap<String, String>();
            if (!Files.exists(file)) return values;
            for (String line : Files.readAllLines(file, StandardCharsets.UTF_8)) {
                String[] parts = line.split("\t", -1);
                if (parts.length != 2) throw new IOException("invalid dataset identity ledger");
                try {
                    String iri = new String(Base64.getUrlDecoder().decode(parts[0]), StandardCharsets.UTF_8);
                    validateIri(iri);
                    if (!parts[1].equals("PENDING")) {
                        String[] identity = parts[1].split("/", -1);
                        if (identity.length != 2 || !isUuid(identity[0]) || !isUuid(identity[1])) throw new IllegalArgumentException();
                    }
                    if (values.putIfAbsent(iri, parts[1]) != null) throw new IllegalArgumentException();
                } catch (IllegalArgumentException exception) {
                    throw new IOException("invalid dataset identity ledger", exception);
                }
            }
            return values;
        }

        private void write(Map<String, String> values) throws IOException {
            Path temporary = Files.createTempFile(directory, "datasets-", ".tmp");
            try {
                var text = new StringBuilder();
                values.entrySet().stream().sorted(Map.Entry.comparingByKey()).forEach(entry -> text
                        .append(Base64.getUrlEncoder().withoutPadding().encodeToString(entry.getKey().getBytes(StandardCharsets.UTF_8)))
                        .append('\t').append(entry.getValue()).append('\n'));
                Files.writeString(temporary, text.toString(), StandardCharsets.UTF_8);
                try (var output = FileChannel.open(temporary, StandardOpenOption.WRITE)) { output.force(true); }
                Files.move(temporary, file, StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING);
            } finally {
                Files.deleteIfExists(temporary);
            }
        }

        @Override
        public void close() throws IOException {
            try {
                lock.release();
            } finally {
                try { channel.close(); } finally { local.unlock(); }
            }
        }
    }

    public static final class ExistingDatasetException extends IOException {
        private static final long serialVersionUID = 1L;
        private final boolean pending;
        ExistingDatasetException(boolean pending) {
            super(pending ? "dataset import outcome requires reconciliation" : "dataset IRI was already imported");
            this.pending = pending;
        }
        public boolean pending() { return pending; }
    }

    private static boolean isUuid(String value) {
        try {
            var parsed = UUID.fromString(value);
            return parsed.toString().equals(value) && !parsed.equals(new UUID(0, 0));
        } catch (IllegalArgumentException exception) {
            return false;
        }
    }

    private static void validateIri(String iri) {
        if (iri == null || !URI.create(iri).isAbsolute()) throw new IllegalArgumentException("dataset subject must be an absolute IRI");
    }
}
