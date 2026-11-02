package eu.epos.semanticaggregator.mapping;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.ingest.DatasetMapper;
import eu.epos.semanticaggregator.rdf.TurtleReader;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

class MappingRegistryTest {
    @Test
    void registersMultipleProfilesAndUsesTheirConfiguredPaths() throws IOException {
        String original;
        try (var input = MappingRegistryTest.class.getResourceAsStream("/mappings/epos-dcat-common.yaml")) {
            original = new String(input.readAllBytes(), StandardCharsets.UTF_8);
        }
        var registry = MappingRegistry.withDefaults();
        String alternate = original.replace("id: epos-dcat-common", "id: alternate")
                .replace("http://purl.org/dc/terms/title", "https://example.org/customTitle");
        registry.register(bytes(alternate));
        assertThrows(IllegalArgumentException.class, () -> registry.register(bytes(alternate)));
        assertThrows(java.util.NoSuchElementException.class, () -> registry.get("alternate", 2));

        String turtle = """
                @prefix dcat: <http://www.w3.org/ns/dcat#> .
                @prefix dct: <http://purl.org/dc/terms/> .
                @prefix ex: <https://example.org/> .
                ex:s a dcat:Dataset; ex:customTitle "Custom"; dct:description "Description" .
                """;
        var model = new TurtleReader().read(bytes(turtle), "https://example.org/");
        try {
            var dataset = new DatasetMapper(registry.get("alternate", 1)).map(model).getFirst();
            assertEquals("Custom", dataset.title());
            assertEquals(0, dataset.unsupportedPredicates().size());
        } finally {
            model.close();
        }
    }

    @Test
    void rejectsInvalidProfiles() {
        var registry = new MappingRegistry();
        assertThrows(IOException.class, () -> registry.register(bytes("id: bad\nversion: 1\ndataset_type: relative\nfields: {}\n")));
        assertThrows(IOException.class, () -> registry.register(bytes("id: one\nid: two\n")));
    }

    @Test
    void rejectsAmbiguousPrimaryIdentifierTypes() throws IOException {
        String original;
        try (var input = MappingRegistryTest.class.getResourceAsStream("/mappings/epos-dcat-common.yaml")) {
            original = new String(input.readAllBytes(), StandardCharsets.UTF_8);
        }
        String invalid = original.replace("EPOS-DCAT-AP:dct:identifier:xsd:anyURI", "EPOS-DCAT-AP:dct:identifier:xsd:string");
        assertThrows(IOException.class, () -> new MappingRegistry().register(bytes(invalid)));
    }

    @Test
    void rejectsIncompleteNestedServiceMappings() throws IOException {
        String original;
        try (var input = MappingRegistryTest.class.getResourceAsStream("/mappings/epos-dcat-common.yaml")) {
            original = new String(input.readAllBytes(), StandardCharsets.UTF_8);
        }
        String incomplete = original.lines().filter(line -> !line.startsWith("  web_service_temporal_end:"))
                .collect(java.util.stream.Collectors.joining("\n"));
        assertThrows(IOException.class, () -> new MappingRegistry().register(bytes(incomplete)));
    }

    @Test
    void loadsExternalProfilesFromDirectory(@TempDir Path directory) throws IOException {
        String original;
        try (var input = MappingRegistryTest.class.getResourceAsStream("/mappings/epos-dcat-common.yaml")) {
            original = new String(input.readAllBytes(), StandardCharsets.UTF_8);
        }
        Files.writeString(directory.resolve("custom.yaml"), original.replace("id: epos-dcat-common", "id: custom"));
        Files.writeString(directory.resolve("ignored.txt"), "not yaml");
        var registry = MappingRegistry.withDefaults();
        registry.registerDirectory(directory);
        assertEquals(2, registry.size());
        assertEquals("custom", registry.get("custom", 1).id());
    }

    @Test
    void acceptsEarlierProfilesWithoutOptionalFields() throws IOException {
        String original;
        try (var input = MappingRegistryTest.class.getResourceAsStream("/mappings/epos-dcat-common.yaml")) {
            original = new String(input.readAllBytes(), StandardCharsets.UTF_8);
        }
        String legacy = original.replaceFirst("(?s)primary_identifier:.*?entity_mappings:", "entity_mappings:")
                .replaceFirst("(?s)entity_mappings:.*?parameter_roles:", "parameter_roles:")
                .replaceFirst("(?s)parameter_roles:.*?fields:", "fields:").lines().filter(line -> !line.matches(
                "  (distribution_.*|web_service_.*|operation_.*|parameter_.*|accrual_periodicity|access_right|quality_annotation|created|updated|providers|contact_points|categories):.*"))
                .collect(java.util.stream.Collectors.joining("\n"));
        var registry = new MappingRegistry();
        registry.register(bytes(legacy));
        assertEquals("epos-dcat-common", registry.get("epos-dcat-common", 1).id());
        assertEquals(0, registry.get("epos-dcat-common", 1).parameterRoles().size());
        assertEquals(0, registry.get("epos-dcat-common", 1).entityMappings().size());
        org.junit.jupiter.api.Assertions.assertNull(registry.get("epos-dcat-common", 1).primaryIdentifier());
    }

    @Test
    void acceptsOlderOrganizationMappingsWithoutAddress() throws IOException {
        String original;
        try (var input = MappingRegistryTest.class.getResourceAsStream("/mappings/epos-dcat-common.yaml")) {
            original = new String(input.readAllBytes(), StandardCharsets.UTF_8);
        }
        String oldOrganization = """
                  organization:
                    rdf_type: "http://schema.org/Organization"
                    fields:
                      name: "http://schema.org/name"
                      legal_name: "http://schema.org/legalName"
                  contact_point:
                """;
        String legacy = original.replaceFirst("(?s)  organization:.*?  contact_point:\\n", oldOrganization);
        var registry = new MappingRegistry();
        registry.register(bytes(legacy));
        org.junit.jupiter.api.Assertions.assertNull(registry.get("epos-dcat-common", 1).entityMappings().get("organization").address());
    }

    private static ByteArrayInputStream bytes(String value) {
        return new ByteArrayInputStream(value.getBytes(StandardCharsets.UTF_8));
    }
}
