package eu.epos.semanticaggregator.mapping;

import java.net.URI;
import java.util.List;
import java.util.Map;
import java.util.Set;

/** A validated versioned dataset mapping definition. */
public record MappingProfile(String id, int version, String datasetType, Map<String, List<String>> fields,
                               Map<String, String> parameterRoles, Map<String, EntityMapping> entityMappings,
                               PrimaryIdentifierMapping primaryIdentifier, Map<String, SoftwareMapping> softwareMappings,
                               SchemeMapping schemeMapping) {
    private static final Set<String> SCALARS = Set.of("title", "description", "keywords");
    private static final Set<String> NESTED = Set.of("spatial_coverage", "temporal_start", "temporal_end",
            "identifier_type", "identifier_value");
    private static final Set<String> OPTIONAL_SCALARS = Set.of("accrual_periodicity", "access_right", "created", "updated", "providers", "contact_points", "categories");
    private static final Set<String> OPTIONAL_NESTED = Set.of("quality_annotation", "distribution_access_url");
    private static final Set<String> DISTRIBUTION = Set.of("distribution_title", "distribution_description", "distribution_type",
            "distribution_url", "distribution_format", "distribution_license", "distribution_issued", "distribution_modified", "distribution_byte_size");
    private static final Set<String> WEB_SERVICE = Set.of("web_service_name", "web_service_description", "web_service_operation",
            "operation_method", "operation_format", "operation_template");
    private static final Set<String> PARAMETERS = Set.of("parameter_name", "parameter_label", "parameter_required", "parameter_range",
            "parameter_property", "parameter_readonly", "parameter_multiple", "parameter_default", "parameter_pattern", "parameter_enum");
    private static final Set<String> PARAMETER_BOUNDS = Set.of("parameter_minimum", "parameter_maximum");
    private static final Set<String> SERVICE_TEMPORAL = Set.of("web_service_temporal_start", "web_service_temporal_end");
    private static final Set<String> SERVICE_DOCUMENTATION = Set.of("web_service_doc_uri", "web_service_doc_title", "web_service_doc_description");
    private static final Set<String> SERVICE_NESTED = Set.of("web_service_spatial", "web_service_temporal_start", "web_service_temporal_end",
            "web_service_doc_uri", "web_service_doc_title", "web_service_doc_description");
    private static final Set<String> SERVICE_REFERENCES = Set.of("web_service_provider", "web_service_contacts");

    public MappingProfile {
        if (id == null || !id.matches("[a-z0-9]+(?:-[a-z0-9]+)*") || version < 1) {
            throw new IllegalArgumentException("mapping id or version is invalid");
        }
        validateIri(datasetType);
        if (parameterRoles == null) {
            parameterRoles = Map.of();
        }
        parameterRoles.forEach((property, role) -> {
            validateIri(property);
            if (role == null || !Set.of("FORMAT", "START_DATE", "END_DATE", "MIN_LON", "MAX_LON", "MIN_LAT", "MAX_LAT").contains(role)) {
                throw new IllegalArgumentException("invalid parameter role: " + role);
            }
        });
        parameterRoles = Map.copyOf(parameterRoles);
        if (entityMappings == null) entityMappings = Map.of();
        for (var entry : entityMappings.entrySet()) {
            Set<String> expected = switch (entry.getKey()) {
                case "organization" -> Set.of("name", "legal_name");
                case "contact_point" -> Set.of("email");
                case "category" -> Set.of("name");
                default -> throw new IllegalArgumentException("unsupported entity mapping: " + entry.getKey());
            };
            EntityMapping mapping = entry.getValue();
            Set<String> optional = entry.getKey().equals("organization")
                    ? Set.of("logo_url", "page_url", "email", "lei_code") : Set.of();
            if (mapping == null || mapping.fields() == null || !mapping.fields().keySet().containsAll(expected)
                    || mapping.fields().keySet().stream().anyMatch(field -> !expected.contains(field) && !optional.contains(field))) {
                throw new IllegalArgumentException("invalid fields for entity mapping: " + entry.getKey());
            }
            validateIri(mapping.rdfType());
            var predicates = new java.util.HashSet<String>(Set.of(org.apache.jena.vocabulary.RDF.type.getURI()));
            for (String predicate : mapping.fields().values()) {
                validateIri(predicate);
                if (!predicates.add(predicate)) throw new IllegalArgumentException("entity predicates must be distinct");
            }
            if (mapping.address() != null) {
                if (!entry.getKey().equals("organization")) throw new IllegalArgumentException("address is only supported for organizations");
                AddressMapping address = mapping.address();
                validateIri(address.predicate());
                validateIri(address.rdfType());
                if (!predicates.add(address.predicate()) || address.fields() == null
                        || !address.fields().keySet().equals(Set.of("street", "locality", "postal_code", "country"))) {
                    throw new IllegalArgumentException("invalid organization address mapping");
                }
                var addressPredicates = new java.util.HashSet<String>(Set.of(org.apache.jena.vocabulary.RDF.type.getURI()));
                for (String predicate : address.fields().values()) {
                    validateIri(predicate);
                    if (!addressPredicates.add(predicate)) throw new IllegalArgumentException("address predicates must be distinct");
                }
            }
        }
        entityMappings = Map.copyOf(entityMappings);
        if (fields == null || !fields.keySet().containsAll(SCALARS) || !fields.keySet().containsAll(NESTED)
                || fields.keySet().stream().anyMatch(name -> !SCALARS.contains(name) && !NESTED.contains(name)
                        && !OPTIONAL_SCALARS.contains(name) && !DISTRIBUTION.contains(name) && !OPTIONAL_NESTED.contains(name)
                        && !WEB_SERVICE.contains(name) && !PARAMETERS.contains(name) && !PARAMETER_BOUNDS.contains(name)
                        && !SERVICE_NESTED.contains(name) && !SERVICE_REFERENCES.contains(name))) {
            throw new IllegalArgumentException("mapping fields must contain the required supported dataset fields");
        }
        if (fields.keySet().stream().anyMatch(DISTRIBUTION::contains) && !fields.keySet().containsAll(DISTRIBUTION)) {
            throw new IllegalArgumentException("distribution mapping must define all distribution fields");
        }
        if (fields.containsKey("distribution_access_url") && !fields.keySet().containsAll(DISTRIBUTION)) {
            throw new IllegalArgumentException("distribution_access_url requires a complete distribution mapping");
        }
        if (fields.keySet().stream().anyMatch(WEB_SERVICE::contains)
                && (!fields.keySet().containsAll(WEB_SERVICE) || !fields.containsKey("distribution_access_url"))) {
            throw new IllegalArgumentException("web service mapping requires all web service and distribution fields");
        }
        if (fields.keySet().stream().anyMatch(PARAMETERS::contains)
                && (!fields.keySet().containsAll(PARAMETERS) || !fields.keySet().containsAll(WEB_SERVICE))) {
            throw new IllegalArgumentException("parameter mapping requires all parameter and web service fields");
        }
        if (fields.keySet().stream().anyMatch(PARAMETER_BOUNDS::contains)
                && (!fields.keySet().containsAll(PARAMETER_BOUNDS) || !fields.keySet().containsAll(PARAMETERS))) {
            throw new IllegalArgumentException("parameter bounds require both bounds and all parameter fields");
        }
        if (fields.keySet().stream().anyMatch(name -> SERVICE_NESTED.contains(name) || SERVICE_REFERENCES.contains(name))
                && !fields.keySet().containsAll(WEB_SERVICE)) {
            throw new IllegalArgumentException("nested service fields require web service mapping");
        }
        if ((fields.keySet().stream().anyMatch(SERVICE_TEMPORAL::contains) && !fields.keySet().containsAll(SERVICE_TEMPORAL))
                || (fields.keySet().stream().anyMatch(SERVICE_DOCUMENTATION::contains) && !fields.keySet().containsAll(SERVICE_DOCUMENTATION))) {
            throw new IllegalArgumentException("service temporal and documentation fields must be configured as complete groups");
        }
        fields.forEach((name, path) -> {
            int expected = SCALARS.contains(name) || OPTIONAL_SCALARS.contains(name) ? 1
                    : PARAMETERS.contains(name) || PARAMETER_BOUNDS.contains(name) ? 5
                    : name.equals("operation_template") || SERVICE_NESTED.contains(name) ? 4
                    : WEB_SERVICE.contains(name) || SERVICE_REFERENCES.contains(name) ? 3 : 2;
            if (path == null || path.size() != expected) {
                throw new IllegalArgumentException("invalid RDF path for " + name);
            }
            path.forEach(MappingProfile::validateIri);
        });
        if (!fields.get("temporal_start").getFirst().equals(fields.get("temporal_end").getFirst())
                || !fields.get("identifier_type").getFirst().equals(fields.get("identifier_value").getFirst())) {
            throw new IllegalArgumentException("paired fields must use the same parent RDF predicate");
        }
        var roots = new java.util.HashSet<String>();
        roots.add(org.apache.jena.vocabulary.RDF.type.getURI());
        var rootFields = new java.util.ArrayList<>(List.of("title", "description", "keywords", "spatial_coverage", "temporal_start", "identifier_type"));
        OPTIONAL_SCALARS.stream().filter(fields::containsKey).forEach(rootFields::add);
        if (fields.containsKey("quality_annotation")) {
            rootFields.add("quality_annotation");
        }
        if (fields.containsKey("distribution_title")) {
            rootFields.add("distribution_title");
        }
        if (fields.containsKey("web_service_name")) {
            String distribution = fields.get("distribution_title").getFirst();
            String serviceLink = fields.get("web_service_name").get(1);
            String operationLink = fields.get("distribution_access_url").get(1);
            var serviceChildren = new java.util.HashSet<String>(Set.of(org.apache.jena.vocabulary.RDF.type.getURI()));
            var operationChildren = new java.util.HashSet<String>(Set.of(org.apache.jena.vocabulary.RDF.type.getURI()));
            for (String field : WEB_SERVICE) {
                List<String> path = fields.get(field);
                boolean service = field.startsWith("web_service_");
                if (!path.getFirst().equals(distribution) || !path.get(1).equals(service ? serviceLink : operationLink)
                        || !(service ? serviceChildren : operationChildren).add(path.get(2))) {
                    throw new IllegalArgumentException("web service paths must share parents and use distinct predicates");
                }
            }
            for (String field : SERVICE_REFERENCES) {
                if (!fields.containsKey(field)) continue;
                List<String> path = fields.get(field);
                if (!path.getFirst().equals(distribution) || !path.get(1).equals(serviceLink) || !serviceChildren.add(path.get(2))) {
                    throw new IllegalArgumentException("service reference paths must share the service parent and use distinct predicates");
                }
            }
            for (Set<String> group : List.of(Set.of("web_service_spatial"), SERVICE_TEMPORAL, SERVICE_DOCUMENTATION)) {
                if (!fields.keySet().containsAll(group)) continue;
                List<String> first = fields.get(group.iterator().next());
                if (!first.getFirst().equals(distribution) || !first.get(1).equals(serviceLink)
                        || !serviceChildren.add(first.get(2))) {
                    throw new IllegalArgumentException("nested service paths must share the service parent and use distinct predicates");
                }
                var nestedChildren = new java.util.HashSet<String>(Set.of(org.apache.jena.vocabulary.RDF.type.getURI()));
                for (String field : group) {
                    List<String> nested = fields.get(field);
                    if (!nested.subList(0, 3).equals(first.subList(0, 3)) || !nestedChildren.add(nested.get(3))) {
                        throw new IllegalArgumentException("nested service fields must share their parent and use distinct predicates");
                    }
                }
            }
            if (serviceLink.equals(operationLink) || serviceLink.equals(org.apache.jena.vocabulary.RDF.type.getURI())
                    || fields.get("operation_template").getLast().equals(org.apache.jena.vocabulary.RDF.type.getURI())) {
                throw new IllegalArgumentException("web service link or template predicate is invalid");
            }
            for (String field : DISTRIBUTION) {
                if (fields.get(field).get(1).equals(serviceLink)) {
                    throw new IllegalArgumentException("web service link must be distinct from distribution fields");
                }
            }
        }
        if (fields.containsKey("parameter_name")) {
            List<String> template = fields.get("operation_template");
            List<String> parameter = fields.get("parameter_name");
            var children = new java.util.HashSet<String>(Set.of(org.apache.jena.vocabulary.RDF.type.getURI()));
            if (!parameter.subList(0, 3).equals(template.subList(0, 3)) || parameter.get(3).equals(template.get(3))
                    || parameter.get(3).equals(org.apache.jena.vocabulary.RDF.type.getURI())) {
                throw new IllegalArgumentException("parameter paths must extend the operation template path");
            }
            var parameterFields = new java.util.HashSet<>(PARAMETERS);
            PARAMETER_BOUNDS.stream().filter(fields::containsKey).forEach(parameterFields::add);
            for (String field : parameterFields) {
                List<String> path = fields.get(field);
                if (!path.subList(0, 4).equals(parameter.subList(0, 4)) || !children.add(path.getLast())) {
                    throw new IllegalArgumentException("parameter paths must share a parent and use distinct predicates");
                }
            }
        }
        for (String field : rootFields) {
            if (!roots.add(fields.get(field).getFirst())) {
                throw new IllegalArgumentException("dataset root predicates must be distinct");
            }
        }
        if (primaryIdentifier != null) {
            validateIri(primaryIdentifier.predicate());
            if (!roots.add(primaryIdentifier.predicate()) || primaryIdentifier.stringType() == null
                    || primaryIdentifier.stringType().isBlank() || primaryIdentifier.uriType() == null
                    || primaryIdentifier.uriType().isBlank() || !primaryIdentifier.stringType().equals(primaryIdentifier.stringType().trim())
                    || !primaryIdentifier.uriType().equals(primaryIdentifier.uriType().trim())
                    || primaryIdentifier.stringType().equals(primaryIdentifier.uriType())) {
                throw new IllegalArgumentException("invalid primary identifier mapping");
            }
        }
        if (softwareMappings == null) softwareMappings = Map.of();
        for (var entry : softwareMappings.entrySet()) {
            if (!Set.of("application", "source_code").contains(entry.getKey())) {
                throw new IllegalArgumentException("unsupported software mapping: " + entry.getKey());
            }
            SoftwareMapping mapping = entry.getValue();
            if (mapping == null || mapping.fields() == null || !mapping.fields().keySet().containsAll(Set.of("name", "description"))) {
                throw new IllegalArgumentException("software mapping requires name and description");
            }
            validateIri(mapping.rdfType());
            mapping.fields().forEach((name, predicate) -> {
                if (!Set.of("name", "description", "identifier", "keywords", "main_entity_of_page", "license", "version",
                        "download_url", "requirements", "operating_system", "programming_language", "repository",
                        "runtime_platform", "creators", "categories", "contact_points").contains(name)) {
                    throw new IllegalArgumentException("unsupported software field: " + name);
                }
                validateIri(predicate);
            });
        }
        softwareMappings = Map.copyOf(softwareMappings);
        if (schemeMapping == null || schemeMapping.fields() == null
                || !schemeMapping.fields().keySet().equals(Set.of("name", "label", "description", "logo", "page", "color", "position", "concept"))
                || schemeMapping.category() == null || schemeMapping.category().fields() == null
                || !schemeMapping.category().fields().keySet().equals(Set.of("name", "scheme", "parent"))) {
            throw new IllegalArgumentException("invalid scheme mapping");
        }
        validateIri(schemeMapping.rdfType());
        schemeMapping.fields().values().forEach(MappingProfile::validateIri);
        validateIri(schemeMapping.category().rdfType());
        schemeMapping.category().fields().values().forEach(MappingProfile::validateIri);
        if (fields.containsKey("distribution_title")) {
            var children = new java.util.HashSet<String>();
            children.add(org.apache.jena.vocabulary.RDF.type.getURI());
            String parent = fields.get("distribution_title").getFirst();
            for (String field : DISTRIBUTION) {
                if (!fields.get(field).getFirst().equals(parent) || !children.add(fields.get(field).get(1))) {
                    throw new IllegalArgumentException("distribution paths must share a parent and have distinct child predicates");
                }
            }
            if (fields.containsKey("distribution_access_url")) {
                List<String> path = fields.get("distribution_access_url");
                if (!path.getFirst().equals(parent) || !children.add(path.get(1))) {
                    throw new IllegalArgumentException("distribution access URL path must use the shared parent and a distinct child");
                }
            }
        }
        for (List<String> pair : List.of(List.of("temporal_start", "temporal_end"), List.of("identifier_type", "identifier_value"))) {
            if (fields.get(pair.getFirst()).get(1).equals(fields.get(pair.getLast()).get(1))) {
                throw new IllegalArgumentException("paired fields must use distinct child predicates");
            }
        }
        for (String field : NESTED) {
            if (fields.get(field).get(1).equals(org.apache.jena.vocabulary.RDF.type.getURI())) {
                throw new IllegalArgumentException("child predicate cannot be rdf:type");
            }
        }
        if (fields.containsKey("quality_annotation") && fields.get("quality_annotation").get(1)
                .equals(org.apache.jena.vocabulary.RDF.type.getURI())) {
            throw new IllegalArgumentException("quality annotation child predicate cannot be rdf:type");
        }
        fields = fields.entrySet().stream().collect(java.util.stream.Collectors.toUnmodifiableMap(
                Map.Entry::getKey, entry -> List.copyOf(entry.getValue())));
    }

    public record EntityMapping(String rdfType, Map<String, String> fields, AddressMapping address) {
        public EntityMapping {
            if (fields != null) fields = Map.copyOf(fields);
        }
    }

    public record PrimaryIdentifierMapping(String predicate, String stringType, String uriType) {}

    public record SoftwareMapping(String rdfType, Map<String, String> fields) {
        public SoftwareMapping {
            if (fields != null) fields = Map.copyOf(fields);
        }
    }

    public record SchemeMapping(String rdfType, Map<String, String> fields, CategoryMapping category) {
        public SchemeMapping {
            if (fields != null) fields = Map.copyOf(fields);
        }
    }

    public record CategoryMapping(String rdfType, Map<String, String> fields) {
        public CategoryMapping {
            if (fields != null) fields = Map.copyOf(fields);
        }
    }

    public record AddressMapping(String predicate, String rdfType, Map<String, String> fields) {
        public AddressMapping {
            if (fields != null) fields = Map.copyOf(fields);
        }
    }

    private static void validateIri(String value) {
        try {
            if (value == null || !URI.create(value).isAbsolute()) {
                throw new IllegalArgumentException("RDF IRIs must be absolute: " + value);
            }
        } catch (IllegalArgumentException exception) {
            throw new IllegalArgumentException("Invalid RDF IRI: " + value, exception);
        }
    }
}
