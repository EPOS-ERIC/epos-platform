package eu.epos.semanticaggregator.ingest;

import java.util.List;
import java.time.Instant;

/** Values extracted from one RDF dataset, before resolving upstream references. */
public record DatasetMapping(
        String subject,
        String title,
        String description,
        List<String> keywords,
        String spatialCoverage,
        TemporalCoverage temporalCoverage,
        String accrualPeriodicity,
        String accessRight,
        String qualityAnnotation,
        Instant created,
        Instant updated,
        List<Identifier> identifiers,
        List<Reference> providers,
        List<Reference> contactPoints,
        List<Reference> categories,
        List<Distribution> distributions,
        List<String> unsupportedPredicates,
        List<String> warnings) {
    public DatasetMapping(String subject, String title, String description, List<String> keywords, String spatialCoverage,
                          TemporalCoverage temporalCoverage, String accrualPeriodicity, String accessRight,
                          String qualityAnnotation, Instant created, Instant updated, List<Identifier> identifiers,
                          List<Reference> providers, List<Reference> contactPoints, List<Reference> categories,
                          List<Distribution> distributions, List<String> unsupportedPredicates) {
        this(subject, title, description, keywords, spatialCoverage, temporalCoverage, accrualPeriodicity, accessRight,
                qualityAnnotation, created, updated, identifiers, providers, contactPoints, categories, distributions,
                unsupportedPredicates, List.of());
    }
    public record Identifier(String type, String value) {}
    public record TemporalCoverage(Instant start, Instant end) {}
    public record Distribution(String title, String description, String url, String format, String license,
                               Instant issued, Instant modified, Long byteSize, WebService webService) {}
    public record WebService(String name, String description, String templateUrl, String format, String method,
                             List<Parameter> parameters, Documentation documentation, String spatialCoverage,
                             TemporalCoverage temporalCoverage, Reference provider, List<Reference> contactPoints) {}
    public record Reference(ReferenceKind kind, String iri) {}
    public enum ReferenceKind { ORGANIZATION, CONTACT_POINT, CATEGORY }
    public record Documentation(String uri, String title, String description) {}
    public record Parameter(String name, String label, String property, boolean required, boolean readOnly,
                            ParameterLocation location, ParameterRole role, ParameterSchema schema) {}
    public enum ParameterLocation { PATH, QUERY }
    public enum ParameterRole { FORMAT, START_DATE, END_DATE, MIN_LON, MAX_LON, MIN_LAT, MAX_LAT }
    public sealed interface ParameterSchema permits StringSchema, BooleanSchema, IntegerSchema, FloatSchema, DateTimeSchema, TemplateSchema {}
    public record StringSchema(boolean multipleValues, List<String> defaultValues, String pattern, List<String> allowedValues) implements ParameterSchema {}
    public record BooleanSchema(Boolean defaultValue) implements ParameterSchema {}
    public record IntegerSchema(boolean multipleValues, Long minimum, Long maximum, List<Long> defaultValues,
                                 List<Long> allowedValues) implements ParameterSchema {}
    public record FloatSchema(boolean multipleValues, Double minimum, Double maximum, List<Double> defaultValues,
                               List<Double> allowedValues) implements ParameterSchema {}
    public record DateTimeSchema(Instant minimum, Instant maximum, Instant defaultValue, String layout) implements ParameterSchema {}
    public record TemplateSchema(String template, List<Parameter> parameters) implements ParameterSchema {}
}
