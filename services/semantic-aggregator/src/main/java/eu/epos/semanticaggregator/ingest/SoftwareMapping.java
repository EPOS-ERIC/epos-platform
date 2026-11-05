package eu.epos.semanticaggregator.ingest;

import java.util.List;

/** Values extracted from one software RDF resource before upstream references are resolved. */
public record SoftwareMapping(String subject, Kind kind, String name, String description, List<String> keywords,
                              String mainEntityOfPage, String license, String version, String downloadUrl,
                              String requirements, String operatingSystem, String programmingLanguage,
                              String repository, String runtimePlatform, List<Identifier> identifiers,
                              List<DatasetMapping.Reference> creators, List<DatasetMapping.Reference> categories,
                              List<DatasetMapping.Reference> contactPoints) {
    public enum Kind { APPLICATION, SOURCE_CODE }
    public record Identifier(String type, String value) {}
}
