package eu.epos.semanticaggregator.ingest;

import eu.epos.semanticaggregator.mapping.MappingProfile;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import java.time.Instant;
import java.time.LocalDate;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.time.format.DateTimeParseException;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.Property;
import org.apache.jena.rdf.model.RDFNode;
import org.apache.jena.rdf.model.Resource;
import org.apache.jena.vocabulary.RDF;

/** Extracts a deliberately limited subset; unsupported predicates are reported. */
public final class DatasetMapper {
    private final MappingProfile profile;

    public DatasetMapper(MappingProfile profile) {
        this.profile = java.util.Objects.requireNonNull(profile, "profile");
    }

    public List<DatasetMapping> map(Model model) {
        Resource datasetType = model.createResource(profile.datasetType());
        List<DatasetMapping> results = new ArrayList<>();
        var subjects = model.listResourcesWithProperty(RDF.type, datasetType);
        try {
            while (subjects.hasNext()) {
                results.add(mapDataset(model, subjects.next()));
            }
        } finally {
            subjects.close();
        }
        results.sort((left, right) -> left.subject().compareTo(right.subject()));
        return List.copyOf(results);
    }

    private DatasetMapping mapDataset(Model model, Resource subject) {
        String name = subject.isURIResource() ? subject.getURI() : "_:" + subject.getId().getLabelString();
        String title = singleLiteral(subject, property(model, path("title", 0)), true);
        String description = singleLiteral(subject, property(model, path("description", 0)), true);
        List<String> keywords = literals(subject, property(model, path("keywords", 0)));
        keywords.sort(String::compareTo);
        List<String> unsupported = new ArrayList<>();
        String quality = null;
        if (profile.fields().containsKey("quality_annotation")) {
            String parent = path("quality_annotation", 0);
            String child = path("quality_annotation", 1);
            Resource annotation = singleResource(subject, property(model, parent), false);
            if (annotation != null) {
                addUnsupported(annotation, Set.of(RDF.type.getURI(), child), parent + "/", unsupported);
                quality = singleValue(annotation, property(model, child), true);
            }
        }
        var types = subject.listProperties(RDF.type);
        try {
            while (types.hasNext()) {
                RDFNode type = types.next().getObject();
                if (!type.isURIResource() || !type.asResource().getURI().equals(profile.datasetType())) {
                    unsupported.add(RDF.type.getURI() + "=" + type);
                }
            }
        } finally {
            types.close();
        }

        String spatialPredicate = path("spatial_coverage", 0);
        String geometryPredicate = path("spatial_coverage", 1);
        Resource spatial = singleResource(subject, property(model, spatialPredicate), false);
        String geometry = spatial == null ? null : singleLiteral(spatial, property(model, geometryPredicate), true);
        if (spatial != null) {
            addUnsupported(spatial, Set.of(RDF.type.getURI(), geometryPredicate), spatialPredicate + "/", unsupported);
        }
        String temporalPredicate = path("temporal_start", 0);
        String startPredicate = path("temporal_start", 1);
        String endPredicate = path("temporal_end", 1);
        Resource temporal = singleResource(subject, property(model, temporalPredicate), false);
        DatasetMapping.TemporalCoverage coverage = null;
        if (temporal != null) {
            addUnsupported(temporal, Set.of(RDF.type.getURI(), startPredicate, endPredicate),
                    temporalPredicate + "/", unsupported);
            Instant start = parseInstant(temporal, property(model, startPredicate));
            Instant end = parseInstant(temporal, property(model, endPredicate));
            if (start == null && end == null) {
                throw invalid(temporal, temporalPredicate, "expected at least one temporal bound");
            }
            if (start != null && end != null && start.isAfter(end)) {
                throw invalid(temporal, temporalPredicate, "start must not be after end");
            }
            coverage = new DatasetMapping.TemporalCoverage(start, end);
        }
        List<DatasetMapping.Identifier> identifiers = new ArrayList<>();
        if (profile.primaryIdentifier() != null) {
            var primary = primaryIdentifier(model, subject);
            if (primary != null) identifiers.add(primary);
        }
        String identifierPredicate = path("identifier_type", 0);
        String identifierType = path("identifier_type", 1);
        String identifierValue = path("identifier_value", 1);
        var idNodes = subject.listProperties(property(model, identifierPredicate));
        try {
            while (idNodes.hasNext()) {
                RDFNode node = idNodes.next().getObject();
                if (!node.isResource()) {
                    throw invalid(subject, identifierPredicate, "expected a resource");
                }
                Resource id = node.asResource();
                addUnsupported(id, Set.of(RDF.type.getURI(), identifierType, identifierValue),
                        identifierPredicate + "/", unsupported);
                identifiers.add(new DatasetMapping.Identifier(
                        singleLiteral(id, property(model, identifierType), true),
                        singleLiteral(id, property(model, identifierValue), true)));
            }
        } finally {
            idNodes.close();
        }
        if (new java.util.HashSet<>(identifiers).size() != identifiers.size()) {
            throw invalid(subject, identifierPredicate, "duplicate identifier type and value");
        }
        List<DatasetMapping.Distribution> distributions = new ArrayList<>();
        if (profile.fields().containsKey("distribution_title")) {
            var links = subject.listProperties(property(model, path("distribution_title", 0)));
            try {
                while (links.hasNext()) {
                    RDFNode node = links.next().getObject();
                    if (!node.isResource()) {
                        throw invalid(subject, path("distribution_title", 0), "expected a distribution resource");
                    }
                    distributions.add(distribution(model, node.asResource(), unsupported));
                }
            } finally {
                links.close();
            }
        }
        distributions.sort(java.util.Comparator.comparing(DatasetMapping.Distribution::title)
                .thenComparing(value -> value.webService() == null ? value.url() : value.webService().templateUrl()));
        identifiers.sort((left, right) -> {
            int type = left.type().compareTo(right.type());
            return type != 0 ? type : left.value().compareTo(right.value());
        });
        Set<String> rootPredicates = new java.util.HashSet<>(Set.of(RDF.type.getURI(), path("title", 0),
                path("description", 0), path("keywords", 0), spatialPredicate, temporalPredicate, identifierPredicate));
        for (String field : List.of("accrual_periodicity", "access_right", "created", "updated", "distribution_title", "quality_annotation",
                "providers", "contact_points", "categories")) {
            if (profile.fields().containsKey(field)) {
                rootPredicates.add(path(field, 0));
            }
        }
        if (profile.primaryIdentifier() != null) rootPredicates.add(profile.primaryIdentifier().predicate());
        addUnsupported(subject, rootPredicates, "", unsupported);
        unsupported.sort(String::compareTo);
        return new DatasetMapping(name, title, description, List.copyOf(keywords), geometry, coverage,
                optionalValue(model, subject, "accrual_periodicity"), optionalValue(model, subject, "access_right"),
                quality, optionalDate(model, subject, "created"), optionalDate(model, subject, "updated"),
                List.copyOf(identifiers), references(model, subject, "providers", 0, DatasetMapping.ReferenceKind.ORGANIZATION),
                references(model, subject, "contact_points", 0, DatasetMapping.ReferenceKind.CONTACT_POINT),
                references(model, subject, "categories", 0, DatasetMapping.ReferenceKind.CATEGORY),
                List.copyOf(distributions), List.copyOf(unsupported));
    }

    private DatasetMapping.Identifier primaryIdentifier(Model model, Resource subject) {
        var config = profile.primaryIdentifier();
        var statements = subject.listProperties(property(model, config.predicate()));
        try {
            if (!statements.hasNext()) return null;
            RDFNode value = statements.next().getObject();
            if (statements.hasNext() || !value.isLiteral() || !value.asLiteral().getLanguage().isEmpty()
                    || value.asLiteral().getString().isBlank()) {
                throw invalid(subject, config.predicate(), "expected exactly one nonempty untagged literal");
            }
            if (!value.asLiteral().getString().equals(value.asLiteral().getString().strip())) {
                throw invalid(subject, config.predicate(), "identifier must not have surrounding whitespace");
            }
            String datatype = value.asLiteral().getDatatypeURI();
            String type;
            if ("http://www.w3.org/2001/XMLSchema#string".equals(datatype)) {
                type = config.stringType();
            } else if ("http://www.w3.org/2001/XMLSchema#anyURI".equals(datatype)) {
                type = config.uriType();
            } else {
                throw invalid(subject, config.predicate(), "unsupported primary identifier datatype");
            }
            return new DatasetMapping.Identifier(type, value.asLiteral().getString());
        } finally {
            statements.close();
        }
    }

    private DatasetMapping.Distribution distribution(Model model, Resource subject, List<String> unsupported) {
        List<String> fields = List.of("distribution_title", "distribution_description", "distribution_type",
                "distribution_url", "distribution_format", "distribution_license", "distribution_issued",
                "distribution_modified", "distribution_byte_size");
        Set<String> predicates = new java.util.HashSet<>();
        predicates.add(RDF.type.getURI());
        fields.forEach(field -> predicates.add(path(field, 1)));
        if (profile.fields().containsKey("distribution_access_url")) {
            predicates.add(path("distribution_access_url", 1));
        }
        String type = singleValue(subject, property(model, path("distribution_type", 1)), true);
        boolean web = type.equals("WEB_SERVICE")
                || type.equals("http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE")
                || type.equals("https://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE");
        if (web && profile.fields().containsKey("web_service_name")) {
            predicates.add(path("web_service_name", 1));
        }
        addUnsupported(subject, predicates, path("distribution_title", 0) + "/", unsupported);
        var types = subject.listProperties(RDF.type);
        try {
            while (types.hasNext()) {
                RDFNode rdfType = types.next().getObject();
                if (!rdfType.isURIResource() || !rdfType.asResource().getURI().equals("http://www.w3.org/ns/dcat#Distribution")) {
                    unsupported.add(path("distribution_title", 0) + "/" + RDF.type.getURI() + "=" + rdfType);
                }
            }
        } finally {
            types.close();
        }
        if (!web && !type.equals("DOWNLOADABLE_FILE")
                && !type.equals("http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE")
                && !type.equals("https://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE")) {
            throw invalid(subject, path("distribution_type", 1), "unsupported distribution type");
        }
        String downloadUrl = singleValue(subject, property(model, path("distribution_url", 1)), false);
        String accessUrl = !web && profile.fields().containsKey("distribution_access_url")
                ? singleValue(subject, property(model, path("distribution_access_url", 1)), false) : null;
        if (!web && downloadUrl != null && accessUrl != null && !downloadUrl.equals(accessUrl)) {
            throw invalid(subject, path("distribution_url", 1), "download and access URLs differ");
        }
        String url = downloadUrl != null ? downloadUrl : accessUrl;
        if (!web && url == null) {
            throw invalid(subject, path("distribution_url", 1), "download or access URL is required");
        }
        if (!web && !java.net.URI.create(url).isAbsolute()) {
            throw invalid(subject, path("distribution_url", 1), "expected an absolute URI");
        }
        String bytes = singleLiteral(subject, property(model, path("distribution_byte_size", 1)), false);
        Long byteSize = null;
        if (bytes != null) {
            try {
                byteSize = Long.valueOf(bytes);
                if (byteSize < 1) {
                    throw new NumberFormatException();
                }
            } catch (NumberFormatException exception) {
                throw invalid(subject, path("distribution_byte_size", 1), "expected a positive integer");
            }
        }
        String format = singleValue(subject, property(model, path("distribution_format", 1)), !web);
        DatasetMapping.WebService service = null;
        if (web) {
            if (!profile.fields().containsKey("web_service_name")) {
                throw invalid(subject, path("distribution_type", 1), "web service mapping is not configured");
            }
            if (downloadUrl != null) {
                throw invalid(subject, path("distribution_url", 1), "web service download URL is not supported");
            }
            service = webService(model, subject, unsupported);
            if (format != null && !format.equals(service.format())) {
                throw invalid(subject, path("distribution_format", 1), "distribution and operation formats differ");
            }
        }
        return new DatasetMapping.Distribution(
                singleLiteral(subject, property(model, path("distribution_title", 1)), true),
                singleLiteral(subject, property(model, path("distribution_description", 1)), true),
                web ? null : url, format,
                singleValue(subject, property(model, path("distribution_license", 1)), false),
                parseInstant(subject, property(model, path("distribution_issued", 1))),
                parseInstant(subject, property(model, path("distribution_modified", 1))), byteSize, service);
    }

    private DatasetMapping.WebService webService(Model model, Resource distribution, List<String> unsupported) {
        String serviceLink = path("web_service_name", 1);
        Resource service = singleResource(distribution, property(model, serviceLink), true);
        String operationLink = path("distribution_access_url", 1);
        Resource operation = singleResource(distribution, property(model, operationLink), true);
        String operationReference = path("web_service_operation", 2);
        Resource declaredOperation = singleResource(service, property(model, operationReference), false);
        if (declaredOperation != null && !declaredOperation.equals(operation)) {
            throw invalid(service, operationReference, "declared operation differs from distribution access URL");
        }
        String name = path("web_service_name", 2);
        String description = path("web_service_description", 2);
        Set<String> servicePredicates = new java.util.HashSet<>(Set.of(RDF.type.getURI(), name, description, operationReference));
        for (String field : List.of("web_service_spatial", "web_service_temporal_start", "web_service_doc_uri",
                "web_service_provider", "web_service_contacts")) {
            if (profile.fields().containsKey(field)) servicePredicates.add(path(field, 2));
        }
        addUnsupported(service, servicePredicates, serviceLink + "/", unsupported);
        checkType(service, "https://www.epos-eu.org/epos-dcat-ap#WebService", serviceLink, unsupported);
        String spatial = serviceSpatial(model, service, unsupported);
        DatasetMapping.TemporalCoverage temporal = serviceTemporal(model, service, unsupported);
        DatasetMapping.Documentation documentation = serviceDocumentation(model, service, unsupported);
        var providers = references(model, service, "web_service_provider", 2, DatasetMapping.ReferenceKind.ORGANIZATION);
        if (providers.size() > 1) throw invalid(service, path("web_service_provider", 2), "expected at most one provider");
        var contacts = references(model, service, "web_service_contacts", 2, DatasetMapping.ReferenceKind.CONTACT_POINT);

        String method = path("operation_method", 2);
        String format = path("operation_format", 2);
        String templateLink = path("operation_template", 2);
        addUnsupported(operation, Set.of(RDF.type.getURI(), method, format, templateLink), operationLink + "/", unsupported);
        checkType(operation, "http://www.w3.org/ns/hydra/core#Operation", operationLink, unsupported);
        Resource template = singleResource(operation, property(model, templateLink), true);
        String templatePredicate = path("operation_template", 3);
        Set<String> templatePredicates = new java.util.HashSet<>(Set.of(RDF.type.getURI(), templatePredicate));
        if (profile.fields().containsKey("parameter_name")) {
            templatePredicates.add(path("parameter_name", 3));
        }
        addUnsupported(template, templatePredicates, operationLink + "/" + templateLink + "/", unsupported);
        checkType(template, "http://www.w3.org/ns/hydra/core#IriTemplate", templateLink, unsupported);
        String rawUrl = singleLiteral(template, property(model, templatePredicate), true);
        var mappedTemplate = mapUrlTemplate(model, template, rawUrl, unsupported);
        String httpMethod = singleLiteral(operation, property(model, method), true).toUpperCase(java.util.Locale.ROOT);
        if (!Set.of("GET", "POST", "PUT", "DELETE").contains(httpMethod)) {
            throw invalid(operation, method, "unsupported HTTP method");
        }
        return new DatasetMapping.WebService(singleLiteral(service, property(model, name), true),
                singleLiteral(service, property(model, description), false), mappedTemplate.url(),
                singleValue(operation, property(model, format), true), httpMethod, mappedTemplate.parameters(),
                documentation, spatial, temporal, providers.isEmpty() ? null : providers.getFirst(), contacts);
    }

    private List<DatasetMapping.Reference> references(Model model, Resource subject, String field, int position,
                                                       DatasetMapping.ReferenceKind kind) {
        if (!profile.fields().containsKey(field)) return List.of();
        var predicate = property(model, path(field, position));
        List<DatasetMapping.Reference> references = new ArrayList<>();
        var statements = subject.listProperties(predicate);
        try {
            while (statements.hasNext()) {
                RDFNode value = statements.next().getObject();
                if (!value.isURIResource()) {
                    throw invalid(subject, predicate.getURI(), "expected an IRI reference to an existing entity");
                }
                references.add(new DatasetMapping.Reference(kind, value.asResource().getURI()));
            }
        } finally {
            statements.close();
        }
        references.sort(java.util.Comparator.comparing(DatasetMapping.Reference::iri));
        return List.copyOf(references);
    }

    private String serviceSpatial(Model model, Resource service, List<String> unsupported) {
        if (!profile.fields().containsKey("web_service_spatial")) return null;
        String link = path("web_service_spatial", 2);
        String geometry = path("web_service_spatial", 3);
        Resource location = singleResource(service, property(model, link), false);
        if (location == null) return null;
        addUnsupported(location, Set.of(RDF.type.getURI(), geometry), link + "/", unsupported);
        checkType(location, "http://purl.org/dc/terms/Location", link, unsupported);
        return singleLiteral(location, property(model, geometry), true);
    }

    private DatasetMapping.TemporalCoverage serviceTemporal(Model model, Resource service, List<String> unsupported) {
        if (!profile.fields().containsKey("web_service_temporal_start")) return null;
        String link = path("web_service_temporal_start", 2);
        String startPredicate = path("web_service_temporal_start", 3);
        String endPredicate = path("web_service_temporal_end", 3);
        Resource period = singleResource(service, property(model, link), false);
        if (period == null) return null;
        addUnsupported(period, Set.of(RDF.type.getURI(), startPredicate, endPredicate), link + "/", unsupported);
        checkType(period, "http://purl.org/dc/terms/PeriodOfTime", link, unsupported);
        Instant start = parseInstant(period, property(model, startPredicate));
        Instant end = parseInstant(period, property(model, endPredicate));
        if (start == null && end == null) throw invalid(period, link, "expected at least one temporal bound");
        if (start != null && end != null && start.isAfter(end)) throw invalid(period, link, "start must not be after end");
        return new DatasetMapping.TemporalCoverage(start, end);
    }

    private DatasetMapping.Documentation serviceDocumentation(Model model, Resource service, List<String> unsupported) {
        if (!profile.fields().containsKey("web_service_doc_uri")) return null;
        String link = path("web_service_doc_uri", 2);
        String uriPredicate = path("web_service_doc_uri", 3);
        String title = path("web_service_doc_title", 3);
        String description = path("web_service_doc_description", 3);
        Resource documentation = singleResource(service, property(model, link), false);
        if (documentation == null) return null;
        addUnsupported(documentation, Set.of(RDF.type.getURI(), uriPredicate, title, description), link + "/", unsupported);
        checkType(documentation, "http://www.w3.org/ns/hydra/core#ApiDocumentation", link, unsupported);
        String uri = singleValue(documentation, property(model, uriPredicate), false);
        if (uri == null && documentation.isURIResource()) uri = documentation.getURI();
        if (uri == null || !java.net.URI.create(uri).isAbsolute()) throw invalid(documentation, uriPredicate, "expected an absolute documentation URI");
        return new DatasetMapping.Documentation(uri, singleLiteral(documentation, property(model, title), false),
                singleLiteral(documentation, property(model, description), false));
    }

    private MappedTemplate mapUrlTemplate(Model model, Resource template, String rawUrl, List<String> unsupported) {
        var expression = java.util.regex.Pattern.compile("\\{([?&])([A-Za-z0-9_,-]+)\\}$").matcher(rawUrl);
        List<String> queryNames = new ArrayList<>();
        String url = rawUrl;
        String queryOperator = null;
        if (expression.find()) {
            queryOperator = expression.group(1);
            queryNames.addAll(List.of(expression.group(2).split(",", -1)));
            url = rawUrl.substring(0, expression.start());
        }
        var pathMatcher = java.util.regex.Pattern.compile("\\{([A-Za-z_][A-Za-z0-9_.-]*)\\}").matcher(url);
        List<String> pathNames = new ArrayList<>();
        int authorityStart = url.indexOf("://");
        int pathStart = authorityStart < 0 ? -1 : url.indexOf('/', authorityStart + 3);
        int queryStart = url.indexOf('?');
        int fragmentStart = url.indexOf('#');
        while (pathMatcher.find()) {
            if (pathStart < 0 || pathMatcher.start() < pathStart
                    || (queryStart >= 0 && pathMatcher.start() >= queryStart)
                    || (fragmentStart >= 0 && pathMatcher.start() >= fragmentStart)) {
                throw invalid(template, path("operation_template", 3), "placeholders are only supported in the URL path");
            }
            if (!pathNames.contains(pathMatcher.group(1))) pathNames.add(pathMatcher.group(1));
        }
        String validationUrl = pathMatcher.replaceAll("placeholder");
        java.net.URI parsed = java.net.URI.create(validationUrl);
        if (validationUrl.contains("{") || validationUrl.contains("}") || parsed.getHost() == null
                || !("http".equalsIgnoreCase(parsed.getScheme()) || "https".equalsIgnoreCase(parsed.getScheme()))
                || parsed.getRawFragment() != null
                || ("?".equals(queryOperator) && parsed.getRawQuery() != null)
                || ("&".equals(queryOperator) && (parsed.getRawQuery() == null || parsed.getRawQuery().isEmpty()))
                || queryNames.stream().anyMatch(name -> !name.matches("[A-Za-z0-9_-]+"))
                || new java.util.HashSet<>(queryNames).size() != queryNames.size()
                || queryNames.stream().anyMatch(pathNames::contains)) {
            throw invalid(template, path("operation_template", 3), "unsupported URL template");
        }
        var parameters = new java.util.HashMap<String, DatasetMapping.Parameter>();
        if (profile.fields().containsKey("parameter_name")) {
            String link = path("parameter_name", 3);
            var mappings = template.listProperties(property(model, link));
            try {
                while (mappings.hasNext()) {
                    RDFNode node = mappings.next().getObject();
                    if (!node.isResource()) {
                        throw invalid(template, link, "expected a parameter resource");
                    }
                    var parameter = parameter(model, node.asResource(), unsupported);
                    if (parameters.putIfAbsent(parameter.name(), parameter) != null) {
                        throw invalid(template, link, "duplicate parameter " + parameter.name());
                    }
                }
            } finally {
                mappings.close();
            }
        }
        List<String> names = new ArrayList<>(pathNames);
        names.addAll(queryNames);
        if (!parameters.keySet().equals(new java.util.HashSet<>(names))) {
            throw invalid(template, path("operation_template", 3), "URL variables and parameter mappings must match");
        }
        List<DatasetMapping.Parameter> ordered = new ArrayList<>();
        Set<DatasetMapping.ParameterRole> roles = new java.util.HashSet<>();
        for (String name : names) {
            var parameter = parameters.get(name);
            var location = pathNames.contains(name) ? DatasetMapping.ParameterLocation.PATH : DatasetMapping.ParameterLocation.QUERY;
            if (location == DatasetMapping.ParameterLocation.PATH && !parameter.required()) {
                throw invalid(template, path("parameter_required", 4), "path parameters must be required");
            }
            if (parameter.role() != null && !roles.add(parameter.role())) {
                throw invalid(template, path("parameter_property", 4), "multiple parameters have the same role");
            }
            ordered.add(new DatasetMapping.Parameter(parameter.name(), parameter.label(), parameter.property(),
                    parameter.required(), parameter.readOnly(), location, parameter.role(), parameter.schema()));
        }
        return new MappedTemplate(url, List.copyOf(ordered));
    }

    private DatasetMapping.Parameter parameter(Model model, Resource parameter, List<String> unsupported) {
        List<String> fields = List.of("parameter_name", "parameter_label", "parameter_required", "parameter_range",
                "parameter_property", "parameter_readonly", "parameter_multiple", "parameter_default", "parameter_pattern", "parameter_enum");
        Set<String> predicates = new java.util.HashSet<>(Set.of(RDF.type.getURI()));
        fields.forEach(field -> predicates.add(path(field, 4)));
        for (String field : List.of("parameter_minimum", "parameter_maximum")) {
            if (profile.fields().containsKey(field)) {
                predicates.add(path(field, 4));
            }
        }
        addUnsupported(parameter, predicates, path("parameter_name", 3) + "/", unsupported);
        checkType(parameter, "http://www.w3.org/ns/hydra/core#IriTemplateMapping", path("parameter_name", 3), unsupported);
        String name = singleLiteral(parameter, property(model, path("parameter_name", 4)), true);
        String label = singleLiteral(parameter, property(model, path("parameter_label", 4)), false);
        String range = singleValue(parameter, property(model, path("parameter_range", 4)), true);
        String defaultValue = singleLiteral(parameter, property(model, path("parameter_default", 4)), false);
        List<String> values = literals(parameter, property(model, path("parameter_enum", 4)));
        String pattern = singleLiteral(parameter, property(model, path("parameter_pattern", 4)), false);
        String minimum = optionalParameterLiteral(model, parameter, "parameter_minimum");
        String maximum = optionalParameterLiteral(model, parameter, "parameter_maximum");
        boolean multiple = optionalBoolean(model, parameter, "parameter_multiple");
        DatasetMapping.ParameterSchema schema = parameterSchema(parameter, range, defaultValue, values, pattern, minimum, maximum, multiple);
        boolean readOnly = optionalBoolean(model, parameter, "parameter_readonly");
        String propertyValue = singleValue(parameter, property(model, path("parameter_property", 4)), false);
        String roleValue = propertyValue == null ? null : profile.parameterRoles().get(propertyValue);
        DatasetMapping.ParameterRole role = roleValue == null ? null : DatasetMapping.ParameterRole.valueOf(roleValue);
        validateRole(parameter, role, schema, readOnly);
        return new DatasetMapping.Parameter(name, label == null ? name : label,
                propertyValue, optionalBoolean(model, parameter, "parameter_required"), readOnly,
                DatasetMapping.ParameterLocation.QUERY, role, schema);
    }

    private void validateRole(Resource subject, DatasetMapping.ParameterRole role, DatasetMapping.ParameterSchema schema, boolean readOnly) {
        if (role == null) return;
        boolean valid = switch (role) {
            case FORMAT -> schema instanceof DatasetMapping.StringSchema string
                    && !string.multipleValues() && string.defaultValue() != null && readOnly;
            case START_DATE, END_DATE -> schema instanceof DatasetMapping.DateTimeSchema;
            case MIN_LON, MAX_LON, MIN_LAT, MAX_LAT ->
                    (schema instanceof DatasetMapping.IntegerSchema integer && !integer.multipleValues())
                    || (schema instanceof DatasetMapping.FloatSchema floating && !floating.multipleValues());
        };
        if (!valid) {
            throw invalid(subject, path("parameter_property", 4), "parameter type or options do not satisfy role " + role);
        }
    }

    private DatasetMapping.ParameterSchema parameterSchema(Resource subject, String range, String defaultValue,
            List<String> values, String pattern, String minimum, String maximum, boolean multiple) {
        String type = range.startsWith("http://www.w3.org/2001/XMLSchema#")
                ? range.substring("http://www.w3.org/2001/XMLSchema#".length())
                : range.startsWith("xsd:") ? range.substring(4) : "";
        if (type.equals("string")) {
            if (minimum != null || maximum != null) {
                throw invalid(subject, path("parameter_range", 4), "string parameters do not support numeric bounds");
            }
            values.sort(String::compareTo);
            validateValues(subject, defaultValue, null, null, values);
            return new DatasetMapping.StringSchema(multiple, defaultValue, pattern, List.copyOf(values));
        }
        if (type.equals("dateTime")) {
            if (multiple || !values.isEmpty()) {
                throw invalid(subject, path("parameter_range", 4), "date-time parameters do not support enum or multiple values");
            }
            Instant min = parameterDate(subject, "parameter_minimum", minimum);
            Instant max = parameterDate(subject, "parameter_maximum", maximum);
            Instant def = parameterDate(subject, "parameter_default", defaultValue);
            validateValues(subject, def, min, max, List.of());
            String layout = pattern == null ? null : pattern.trim();
            if (layout != null && (layout.isEmpty() || !layout.matches("(?:(?:YYYY|MM|DD|hh|mm|ss)|[^YyMDHhms])+"))) {
                throw invalid(subject, path("parameter_pattern", 4), "unsupported date-time layout");
            }
            return new DatasetMapping.DateTimeSchema(min, max, def, layout);
        }
        if (pattern != null) {
            throw invalid(subject, path("parameter_pattern", 4), "pattern is supported only for string and date-time parameters");
        }
        if (type.equals("boolean")) {
            if (multiple || minimum != null || maximum != null || !values.isEmpty()) {
                throw invalid(subject, path("parameter_range", 4), "boolean parameters do not support bounds, enum or multiple values");
            }
            Boolean value = defaultValue == null ? null : booleanValue(subject, path("parameter_default", 4), defaultValue);
            return new DatasetMapping.BooleanSchema(value);
        }
        if (type.equals("integer") || type.equals("long")) {
            Long min = integerValue(subject, minimum);
            Long max = integerValue(subject, maximum);
            Long def = integerValue(subject, defaultValue);
            List<Long> options = values.stream().map(value -> integerValue(subject, value)).sorted().toList();
            validateValues(subject, def, min, max, options);
            return new DatasetMapping.IntegerSchema(multiple, min, max, def, options);
        }
        if (type.equals("double") || type.equals("decimal") || type.equals("float")) {
            Double min = floatValue(subject, minimum);
            Double max = floatValue(subject, maximum);
            Double def = floatValue(subject, defaultValue);
            List<Double> options = values.stream().map(value -> floatValue(subject, value)).sorted().toList();
            validateValues(subject, def, min, max, options);
            return new DatasetMapping.FloatSchema(multiple, min, max, def, options);
        }
        throw invalid(subject, path("parameter_range", 4), "unsupported parameter type " + range);
    }

    private <T extends Comparable<T>> void validateValues(Resource subject, T value, T minimum, T maximum, List<T> options) {
        if (minimum != null && maximum != null && minimum.compareTo(maximum) > 0) {
            throw invalid(subject, path("parameter_range", 4), "minimum exceeds maximum");
        }
        if (new java.util.HashSet<>(options).size() != options.size()) {
            throw invalid(subject, path("parameter_enum", 4), "duplicate allowed values");
        }
        List<T> all = new ArrayList<>(options);
        if (value != null) {
            if (!options.isEmpty() && !options.contains(value)) {
                throw invalid(subject, path("parameter_default", 4), "default must be an allowed value");
            }
            all.add(value);
        }
        if (all.stream().anyMatch(item -> (minimum != null && item.compareTo(minimum) < 0)
                || (maximum != null && item.compareTo(maximum) > 0))) {
            throw invalid(subject, path("parameter_range", 4), "value outside parameter bounds");
        }
    }

    private Long integerValue(Resource subject, String value) {
        if (value == null) return null;
        try {
            return Long.valueOf(value);
        } catch (NumberFormatException exception) {
            throw invalid(subject, path("parameter_range", 4), "invalid 64-bit integer");
        }
    }

    private Instant parameterDate(Resource subject, String field, String value) {
        if (value == null) return null;
        try {
            Instant instant = OffsetDateTime.parse(value).toInstant();
            if (instant.isBefore(Instant.parse("0001-01-01T00:00:00Z"))
                    || instant.isAfter(Instant.parse("9999-12-31T23:59:59.999999999Z"))) {
                throw new IllegalArgumentException("timestamp outside protobuf range");
            }
            return instant;
        } catch (DateTimeParseException | IllegalArgumentException exception) {
            throw invalid(subject, path(field, 4), "expected a date-time with an explicit timezone within the protobuf range");
        }
    }

    private Double floatValue(Resource subject, String value) {
        if (value == null) return null;
        try {
            double result = Double.parseDouble(value);
            if (!Double.isFinite(result) || new java.math.BigDecimal(value).compareTo(java.math.BigDecimal.valueOf(result)) != 0) {
                throw new NumberFormatException();
            }
            return result == 0 ? 0.0 : result;
        } catch (NumberFormatException exception) {
            throw invalid(subject, path("parameter_range", 4), "invalid or imprecise floating-point value");
        }
    }

    private String optionalParameterLiteral(Model model, Resource subject, String field) {
        return profile.fields().containsKey(field) ? singleLiteral(subject, property(model, path(field, 4)), false) : null;
    }

    private boolean optionalBoolean(Model model, Resource subject, String field) {
        String value = singleLiteral(subject, property(model, path(field, 4)), false);
        if (value == null) {
            return false;
        }
        return booleanValue(subject, path(field, 4), value);
    }

    private boolean booleanValue(Resource subject, String predicate, String value) {
        if (value.equals("false") || value.equals("0")) return false;
        if (value.equals("true") || value.equals("1")) return true;
        throw invalid(subject, predicate, "expected an RDF boolean");
    }

    private record MappedTemplate(String url, List<DatasetMapping.Parameter> parameters) {}

    private static void checkType(Resource subject, String expected, String path, List<String> unsupported) {
        var types = subject.listProperties(RDF.type);
        try {
            while (types.hasNext()) {
                RDFNode type = types.next().getObject();
                if (!type.isURIResource() || !type.asResource().getURI().equals(expected)) {
                    unsupported.add(path + "/" + RDF.type.getURI() + "=" + type);
                }
            }
        } finally {
            types.close();
        }
    }

    private String optionalValue(Model model, Resource subject, String field) {
        return profile.fields().containsKey(field) ? singleValue(subject, property(model, path(field, 0)), false) : null;
    }

    private Instant optionalDate(Model model, Resource subject, String field) {
        return profile.fields().containsKey(field) ? parseInstant(subject, property(model, path(field, 0))) : null;
    }

    private static String singleValue(Resource subject, Property predicate, boolean required) {
        var statements = subject.listProperties(predicate);
        try {
            if (!statements.hasNext()) {
                if (required) {
                    throw invalid(subject, predicate.getURI(), "expected a value");
                }
                return null;
            }
            RDFNode node = statements.next().getObject();
            if (statements.hasNext()) {
                throw invalid(subject, predicate.getURI(), "expected exactly one value");
            }
            if (node.isURIResource()) {
                return node.asResource().getURI();
            }
            if (node.isLiteral() && node.asLiteral().getLanguage().isEmpty()
                    && !node.asLiteral().getString().isBlank()) {
                return node.asLiteral().getString();
            }
            throw invalid(subject, predicate.getURI(), "expected a nonempty literal or IRI");
        } finally {
            statements.close();
        }
    }

    private String path(String field, int index) {
        return profile.fields().get(field).get(index);
    }

    private static void addUnsupported(Resource subject, Set<String> supported, String prefix, List<String> unsupported) {
        var statements = subject.listProperties();
        try {
            while (statements.hasNext()) {
                String predicate = statements.next().getPredicate().getURI();
                String path = prefix + predicate;
                if (!supported.contains(predicate) && !unsupported.contains(path)) {
                    unsupported.add(path);
                }
            }
        } finally {
            statements.close();
        }
    }

    private static Property property(Model model, String uri) {
        return model.createProperty(uri);
    }

    private static Instant parseInstant(Resource subject, Property predicate) {
        String value = singleLiteral(subject, predicate, false);
        if (value == null) {
            return null;
        }
        try {
            if (value.matches("\\d{4}-\\d{2}-\\d{2}")) {
                return LocalDate.parse(value).atStartOfDay().toInstant(ZoneOffset.UTC);
            }
            return OffsetDateTime.parse(value).toInstant();
        } catch (DateTimeParseException exception) {
            throw invalid(subject, predicate.getURI(), "expected an ISO date or a date-time with timezone");
        }
    }

    private static List<String> literals(Resource subject, Property predicate) {
        List<String> values = new ArrayList<>();
        var statements = subject.listProperties(predicate);
        try {
            while (statements.hasNext()) {
                RDFNode node = statements.next().getObject();
                if (!node.isLiteral() || node.asLiteral().getString().isBlank()) {
                    throw invalid(subject, predicate.getURI(), "expected a nonempty literal");
                }
                if (!node.asLiteral().getLanguage().isEmpty()) {
                    throw invalid(subject, predicate.getURI(), "language-tagged literals are not yet supported");
                }
                values.add(node.asLiteral().getString());
            }
        } finally {
            statements.close();
        }
        return values;
    }

    private static String singleLiteral(Resource subject, Property predicate, boolean required) {
        List<String> values = literals(subject, predicate);
        if (values.size() > 1 || (required && values.isEmpty())) {
            throw invalid(subject, predicate.getURI(), "expected exactly one value");
        }
        return values.isEmpty() ? null : values.getFirst();
    }

    private static Resource singleResource(Resource subject, Property predicate, boolean required) {
        var statements = subject.listProperties(predicate);
        try {
            if (!statements.hasNext()) {
                if (required) {
                    throw invalid(subject, predicate.getURI(), "expected a resource");
                }
                return null;
            }
            RDFNode node = statements.next().getObject();
            if (!node.isResource() || statements.hasNext()) {
                throw invalid(subject, predicate.getURI(), "expected exactly one resource");
            }
            return node.asResource();
        } finally {
            statements.close();
        }
    }

    private static IllegalArgumentException invalid(Resource subject, String predicate, String reason) {
        return new IllegalArgumentException(subject + " " + predicate + ": " + reason);
    }
}
