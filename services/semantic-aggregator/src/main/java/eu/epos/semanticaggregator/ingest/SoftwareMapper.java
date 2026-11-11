package eu.epos.semanticaggregator.ingest;

import eu.epos.semanticaggregator.mapping.MappingProfile;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.Property;
import org.apache.jena.rdf.model.RDFNode;
import org.apache.jena.rdf.model.Resource;
import org.apache.jena.vocabulary.RDF;

/** Maps the supported profile-defined Schema.org software properties. */
public final class SoftwareMapper {
    private static final String SCHEMA = "http://schema.org/";
    private final MappingProfile profile;

    public SoftwareMapper(MappingProfile profile) {
        this.profile = java.util.Objects.requireNonNull(profile, "profile");
    }

    public List<SoftwareMapping> map(Model model) {
        List<SoftwareMapping> results = new ArrayList<>();
        mapKind(model, "application", SoftwareMapping.Kind.APPLICATION, results);
        mapKind(model, "source_code", SoftwareMapping.Kind.SOURCE_CODE, results);
        results.sort(java.util.Comparator.comparing(SoftwareMapping::subject));
        return List.copyOf(results);
    }

    private void mapKind(Model model, String mappingName, SoftwareMapping.Kind kind, List<SoftwareMapping> results) {
        var mapping = profile.softwareMappings().get(mappingName);
        if (mapping == null) return;
        var subjects = model.listResourcesWithProperty(RDF.type, model.createResource(mapping.rdfType()));
        try {
            while (subjects.hasNext()) results.add(mapResource(model, subjects.next(), mapping, kind));
        } finally {
            subjects.close();
        }
    }

    private SoftwareMapping mapResource(Model model, Resource subject, MappingProfile.SoftwareMapping mapping,
                                        SoftwareMapping.Kind kind) {
        Set<String> supported = new HashSet<>(mapping.fields().values());
        supported.add(RDF.type.getURI());
        var statements = subject.listProperties();
        try {
            while (statements.hasNext()) {
                String predicate = statements.next().getPredicate().getURI();
                if (!supported.contains(predicate)) throw invalid(subject, predicate, "unsupported property");
            }
        } finally {
            statements.close();
        }
        String name = singleString(subject, property(model, mapping, "name"), true);
        String description = singleString(subject, property(model, mapping, "description"), true);
        var normalizedKeywords = KeywordNormalizer.normalize(strings(subject, property(model, mapping, "keywords")));
        List<String> keywords = new ArrayList<>(normalizedKeywords.values());
        keywords.sort(String::compareTo);
        List<SoftwareMapping.Identifier> identifiers = identifiers(model, subject, mapping);
        identifiers.sort(java.util.Comparator.comparing(SoftwareMapping.Identifier::type)
                .thenComparing(SoftwareMapping.Identifier::value));
        return new SoftwareMapping(subjectName(subject), kind, name, description, List.copyOf(keywords),
                stringField(model, subject, mapping, "main_entity_of_page"), stringField(model, subject, mapping, "license"),
                stringField(model, subject, mapping, "version"), stringField(model, subject, mapping, "download_url"),
                stringField(model, subject, mapping, "requirements"), stringField(model, subject, mapping, "operating_system"),
                stringField(model, subject, mapping, "programming_language"), stringField(model, subject, mapping, "repository"),
                stringField(model, subject, mapping, "runtime_platform"), List.copyOf(identifiers),
                references(subject, property(model, mapping, "creators"), DatasetMapping.ReferenceKind.ORGANIZATION),
                references(subject, property(model, mapping, "categories"), DatasetMapping.ReferenceKind.CATEGORY),
                references(subject, property(model, mapping, "contact_points"), DatasetMapping.ReferenceKind.CONTACT_POINT),
                normalizedKeywords.warnings());
    }

    private static List<SoftwareMapping.Identifier> identifiers(Model model, Resource subject, MappingProfile.SoftwareMapping mapping) {
        Property predicate = property(model, mapping, "identifier");
        if (predicate == null) return List.of();
        List<SoftwareMapping.Identifier> identifiers = new ArrayList<>();
        var statements = subject.listProperties(predicate);
        try {
            while (statements.hasNext()) {
                RDFNode node = statements.next().getObject();
                if (node.isLiteral()) {
                    String text = node.asLiteral().getString();
                    if (text.isBlank()) continue;
                    identifiers.add(new SoftwareMapping.Identifier("OTHER", text));
                    continue;
                }
                if (!node.isResource()) throw invalid(subject, predicate.getURI(), "expected a schema:PropertyValue resource");
                Resource value = node.asResource();
                requireOnly(value, Set.of(RDF.type.getURI(), SCHEMA + "propertyID", SCHEMA + "value"));
                if (!value.hasProperty(RDF.type, value.getModel().createResource(SCHEMA + "PropertyValue"))) {
                    throw invalid(value, RDF.type.getURI(), "expected schema:PropertyValue");
                }
                identifiers.add(new SoftwareMapping.Identifier(singleString(value, value.getModel().createProperty(SCHEMA + "propertyID"), true),
                        singleString(value, value.getModel().createProperty(SCHEMA + "value"), true)));
            }
        } finally {
            statements.close();
        }
        if (new HashSet<>(identifiers).size() != identifiers.size()) throw invalid(subject, predicate.getURI(), "duplicate identifier");
        return identifiers;
    }

    private static void requireOnly(Resource subject, Set<String> supported) {
        var statements = subject.listProperties();
        try {
            while (statements.hasNext()) {
                String predicate = statements.next().getPredicate().getURI();
                if (!supported.contains(predicate)) throw invalid(subject, predicate, "unsupported property");
            }
        } finally {
            statements.close();
        }
    }

    private static String stringField(Model model, Resource subject, MappingProfile.SoftwareMapping mapping, String name) {
        Property property = property(model, mapping, name);
        return property == null ? null : singleString(subject, property, false);
    }

    private static Property property(Model model, MappingProfile.SoftwareMapping mapping, String name) {
        String uri = mapping.fields().get(name);
        return uri == null ? null : model.createProperty(uri);
    }

    private static List<DatasetMapping.Reference> references(Resource subject, Property predicate, DatasetMapping.ReferenceKind kind) {
        if (predicate == null) return List.of();
        List<DatasetMapping.Reference> references = new ArrayList<>();
        var statements = subject.listProperties(predicate);
        try {
            while (statements.hasNext()) {
                RDFNode node = statements.next().getObject();
                if (!node.isURIResource()) throw invalid(subject, predicate.getURI(), "expected an IRI reference with an explicit binding");
                references.add(new DatasetMapping.Reference(kind, node.asResource().getURI()));
            }
        } finally {
            statements.close();
        }
        references.sort(java.util.Comparator.comparing(DatasetMapping.Reference::iri));
        return List.copyOf(references);
    }

    private static List<String> strings(Resource subject, Property predicate) {
        if (predicate == null) return List.of();
        List<String> values = new ArrayList<>();
        var statements = subject.listProperties(predicate);
        try {
            while (statements.hasNext()) values.add(string(statements.next().getObject(), subject, predicate));
        } finally {
            statements.close();
        }
        return values;
    }

    private static String singleString(Resource subject, Property predicate, boolean required) {
        List<String> values = strings(subject, predicate);
        if (values.size() > 1 || (required && values.isEmpty())) throw invalid(subject, predicate.getURI(), "expected exactly one value");
        return values.isEmpty() ? null : values.getFirst();
    }

    private static String string(RDFNode node, Resource subject, Property predicate) {
        if (node.isURIResource()) return node.asResource().getURI();
        if (node.isLiteral() && node.asLiteral().getLanguage().isEmpty() && !node.asLiteral().getString().isBlank()) {
            return node.asLiteral().getString();
        }
        if (node.isLiteral() && node.asLiteral().getString().isBlank()) return null;
        throw invalid(subject, predicate.getURI(), "expected a nonempty untagged literal or IRI");
    }

    private static String subjectName(Resource subject) {
        return subject.isURIResource() ? subject.getURI() : "_:" + subject.getId().getLabelString();
    }

    private static IllegalArgumentException invalid(Resource subject, String predicate, String reason) {
        return new IllegalArgumentException(subject + " " + predicate + ": " + reason);
    }
}
