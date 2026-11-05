package eu.epos.semanticaggregator.ingest;

import eu.epos.semanticaggregator.mapping.MappingProfile;
import java.util.HashMap;
import java.util.Map;
import java.util.HashSet;
import java.util.Set;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.RDFNode;
import org.apache.jena.rdf.model.Resource;
import org.apache.jena.vocabulary.RDF;

/** Extracts supported attributes from a referenced RDF entity without mutating it. */
public final class EntityDescriptionMapper {
    private final MappingProfile profile;

    public EntityDescriptionMapper(MappingProfile profile) {
        this.profile = profile;
    }

    public Result map(Model model, DatasetMapping.Reference reference) {
        return map(model, reference, false);
    }

    public Result map(Model model, DatasetMapping.Reference reference, boolean tolerateUnsupported) {
        if (reference.kind() == DatasetMapping.ReferenceKind.CONTACT_POINT && reference.iri().startsWith("mailto:")) {
            String email = reference.iri().substring("mailto:".length()).split("/", 2)[0];
            email = email(email);
            if (email != null) {
                return new Result(new EntityDescription(reference, Map.of("email", email)), Set.of());
            }
        }
        if (reference.kind() == DatasetMapping.ReferenceKind.CONTACT_POINT && reference.iri().endsWith("/contactPoint")) {
            var source = model.createResource(reference.iri().substring(0, reference.iri().length() - "/contactPoint".length()));
            var emails = source.listProperties(model.createProperty("http://schema.org/email"));
            try {
                String email = null;
                while (emails.hasNext()) {
                    var value = emails.next().getObject();
                    if (!value.isLiteral() || email != null) {
                        email = null;
                        break;
                    }
                    email = value.asLiteral().getString();
                }
                email = email(email);
                if (email != null) {
                    return new Result(new EntityDescription(reference, Map.of("email", email)), Set.of(source));
                }
            } finally {
                emails.close();
            }
        }
        var subject = model.createResource(reference.iri());
        var statements = subject.listProperties();
        Map<String, String> values = new HashMap<>();
        Set<Resource> reached = new HashSet<>();
        reached.add(subject);
        Resource address = null;
        var mapping = profile.entityMappings().get(reference.kind().name().toLowerCase(java.util.Locale.ROOT));
        try {
            while (statements.hasNext()) {
                var statement = statements.next();
                String predicate = statement.getPredicate().getURI();
                RDFNode value = statement.getObject();
                if (mapping == null) throw invalid(reference, "entity descriptions are not configured");
                if (predicate.equals(RDF.type.getURI())) {
                    if (!value.isURIResource() || !mapping.rdfType().equals(value.asResource().getURI())) {
                        throw invalid(reference, "unsupported RDF type");
                    }
                    continue;
                }
                if (mapping.address() != null && predicate.equals(mapping.address().predicate())) {
                    if (address != null || !value.isResource()) throw invalid(reference, "expected exactly one address resource");
                    address = value.asResource();
                    reached.add(address);
                    continue;
                }
                String field = mapping.fields().entrySet().stream()
                        .filter(entry -> entry.getValue().equals(predicate)).map(Map.Entry::getKey).findFirst()
                        .orElse(null);
                if (field == null) {
                    if (tolerateUnsupported) continue;
                    throw invalid(reference, "unsupported entity predicate " + predicate);
                }
                String extracted = scalar(reference, field, predicate, value);
                if (extracted != null && !values.containsKey(field)) {
                    values.put(field, extracted);
                }
            }
        } finally {
            statements.close();
        }
        if (address != null) readAddress(reference, address, mapping.address(), values, tolerateUnsupported);
        return new Result(new EntityDescription(reference, values), Set.copyOf(reached));
    }

    private void readAddress(DatasetMapping.Reference reference, Resource address, MappingProfile.AddressMapping mapping,
                             Map<String, String> values, boolean tolerateUnsupported) {
        int count = 0;
        var statements = address.listProperties();
        try {
            while (statements.hasNext()) {
                var statement = statements.next();
                String predicate = statement.getPredicate().getURI();
                RDFNode value = statement.getObject();
                if (predicate.equals(RDF.type.getURI())) {
                    if (!value.isURIResource() || !mapping.rdfType().equals(value.asResource().getURI())) {
                        throw invalid(reference, "unsupported address RDF type");
                    }
                    continue;
                }
                String field = mapping.fields().entrySet().stream()
                        .filter(entry -> entry.getValue().equals(predicate)).map(Map.Entry::getKey).findFirst()
                        .orElse(null);
                if (field == null) {
                    if (tolerateUnsupported) continue;
                    throw invalid(reference, "unsupported address predicate " + predicate);
                }
                String parsed = scalar(reference, field, predicate, value);
                if (parsed != null && values.putIfAbsent(field, parsed) != null) {
                    throw invalid(reference, "multiple values for " + predicate);
                }
                if (parsed != null) count++;
            }
        } finally {
            statements.close();
        }
        if (count == 0) throw invalid(reference, "address must contain at least one supported field");
    }

    private static String scalar(DatasetMapping.Reference reference, String field, String predicate, RDFNode value) {
        boolean uriField = field.equals("logo_url") || field.equals("page_url");
        if (uriField && value.isURIResource()) return value.asResource().getURI();
        if (!value.isLiteral()) {
            throw invalid(reference, "expected an untagged value for " + predicate);
        }
        if (value.asLiteral().getString().isBlank()) return null;
        String datatype = value.asLiteral().getDatatypeURI();
        if (!"http://www.w3.org/2001/XMLSchema#string".equals(datatype)
                && !"http://www.w3.org/1999/02/22-rdf-syntax-ns#langString".equals(datatype)
                && !(uriField && "http://www.w3.org/2001/XMLSchema#anyURI".equals(datatype))) {
            throw invalid(reference, "unsupported datatype for " + predicate);
        }
        String text = value.asLiteral().getString().trim();
        if (field.equals("email")) return email(text);
        if (uriField) {
            try {
                if (!java.net.URI.create(text).isAbsolute()) return null;
            } catch (IllegalArgumentException exception) {
                return null;
            }
        }
        return text;
    }

    private static String email(String value) {
        if (value == null) return null;
        String normalized = value.replaceAll("\\p{Cf}", "");
        return normalized.matches("[^\\s@]+@[^\\s@]+\\.[^\\s@]+") ? normalized : null;
    }

    public record Result(EntityDescription description, Set<Resource> reachedSubjects) {}

    private static IllegalArgumentException invalid(DatasetMapping.Reference reference, String reason) {
        return new IllegalArgumentException(reference.iri() + ": " + reason);
    }
}
