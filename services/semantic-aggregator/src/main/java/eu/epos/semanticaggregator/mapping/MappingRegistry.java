package eu.epos.semanticaggregator.mapping;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.PropertyNamingStrategies;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import java.io.IOException;
import java.io.InputStream;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.NoSuchElementException;
import java.nio.file.Files;
import java.nio.file.Path;

/** Registry of explicitly registered mapping profiles, addressed by id and version. */
public final class MappingRegistry {
    private final ObjectMapper yaml = new ObjectMapper(new YAMLFactory().enable(JsonParser.Feature.STRICT_DUPLICATE_DETECTION))
            .setPropertyNamingStrategy(PropertyNamingStrategies.SNAKE_CASE)
            .disable(JsonParser.Feature.AUTO_CLOSE_SOURCE)
            .enable(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES);
    private final Map<Key, MappingProfile> profiles = new LinkedHashMap<>();

    /** The caller owns the input stream. Duplicate registrations are rejected. */
    public void register(InputStream input) throws IOException {
        MappingProfile profile = yaml.readValue(input, MappingProfile.class);
        if (profile == null) {
            throw new IllegalArgumentException("mapping document is empty");
        }
        Key key = new Key(profile.id(), profile.version());
        if (profiles.putIfAbsent(key, profile) != null) {
            throw new IllegalArgumentException("mapping already registered: " + key);
        }
    }

    public MappingProfile get(String id, int version) {
        MappingProfile profile = profiles.get(new Key(id, version));
        if (profile == null) {
            throw new NoSuchElementException("mapping not registered: " + id + " version " + version);
        }
        return profile;
    }

    /** Registers YAML files from a directory in filename order. */
    public void registerDirectory(Path directory) throws IOException {
        try (var paths = Files.list(directory)) {
            for (Path file : paths.filter(path -> path.getFileName().toString().endsWith(".yaml")
                    || path.getFileName().toString().endsWith(".yml")).sorted().toList()) {
                try (InputStream input = Files.newInputStream(file)) {
                    register(input);
                } catch (IOException | IllegalArgumentException exception) {
                    throw new IOException("invalid mapping file " + file, exception);
                }
            }
        }
    }

    public int size() {
        return profiles.size();
    }

    public static MappingRegistry withDefaults() throws IOException {
        MappingRegistry registry = new MappingRegistry();
        try (InputStream input = MappingRegistry.class.getResourceAsStream("/mappings/epos-dcat-common.yaml")) {
            if (input == null) {
                throw new IOException("bundled mapping epos-dcat-common.yaml not found");
            }
            registry.register(input);
        }
        return registry;
    }

    private record Key(String id, int version) {}
}
