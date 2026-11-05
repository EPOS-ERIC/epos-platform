package eu.epos.semanticaggregator.ingest;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import eu.epos.semanticaggregator.mapping.MappingProfile;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.RDFNode;
import org.apache.jena.rdf.model.Resource;
import org.apache.jena.vocabulary.RDF;

/** Maps the legacy SKOS group-scheme shape needed to select lifecycle groups. */
public final class LegacySchemeMapper {
    private static final MappingProfile.SchemeMapping DEFAULT_MAPPING = new MappingProfile.SchemeMapping(
            "http://www.w3.org/2004/02/skos/core#ConceptScheme", Map.of(
                    "name", "http://purl.org/dc/terms/title", "label", "http://www.w3.org/2004/02/skos/core#prefLabel",
                    "description", "http://purl.org/dc/terms/description", "logo", "http://xmlns.com/foaf/0.1/logo",
                    "page", "http://xmlns.com/foaf/0.1/homepage", "color", "http://schema.org/color",
                    "position", "http://schema.org/orderItemNumber", "concept", "http://www.w3.org/2004/02/skos/core#hasTopConcept"),
            new MappingProfile.CategoryMapping("http://www.w3.org/2004/02/skos/core#Concept", Map.of(
                    "name", "http://www.w3.org/2004/02/skos/core#prefLabel", "scheme", "http://www.w3.org/2004/02/skos/core#inScheme",
                    "parent", "http://www.w3.org/2004/02/skos/core#broader")));
    private final MappingProfile.SchemeMapping mapping;

    public LegacySchemeMapper() { this(DEFAULT_MAPPING); }

    public LegacySchemeMapper(MappingProfile profile) { this(profile.schemeMapping()); }

    private LegacySchemeMapper(MappingProfile.SchemeMapping mapping) { this.mapping = mapping; }

    /** Maps group schemes; vocabulary schemes without a group top-level concept are ignored. */
    public List<Scheme> map(Model model) {
        List<Scheme> schemes = new ArrayList<>();
        var iterator = model.listResourcesWithProperty(RDF.type, model.createResource(mapping.rdfType()));
        try { while (iterator.hasNext()) {
            var scheme = groupScheme(iterator.next());
            if (scheme != null) schemes.add(scheme);
        } } finally { iterator.close(); }
        schemes.sort(java.util.Comparator.comparing(Scheme::iri));
        return List.copyOf(schemes);
    }

    public Category category(Model model, String iri) {
        Resource concept = model.getResource(iri);
        if (!concept.hasProperty(RDF.type, model.createResource(mapping.category().rdfType()))) return null;
        String scheme = singleResource(concept, mapping.category().fields().get("scheme"), "category must have exactly one skos:inScheme");
        return new Category(iri, scheme, singleString(concept, mapping.category().fields().get("name"), "category must have exactly one skos:prefLabel"),
                optionalResource(concept, mapping.category().fields().get("parent"), "category must have at most one skos:broader"));
    }

    /** Maps every concept belonging to a supported group scheme. */
    public List<Category> categories(Model model, java.util.Set<String> schemeIris) {
        List<Category> categories = new ArrayList<>();
        var iterator = model.listResourcesWithProperty(RDF.type, model.createResource(mapping.category().rdfType()));
        try { while (iterator.hasNext()) {
            Resource concept = iterator.next();
            if (!concept.isURIResource()) throw new IllegalArgumentException("skos:Concept must have an IRI");
            // Legacy vocabularies also use skos:Concept but are not lifecycle
            // categories unless they declare their owning concept scheme.
            if (!concept.hasProperty(model.createProperty(mapping.category().fields().get("scheme")))) continue;
            var category = category(model, concept.getURI());
            if (schemeIris.contains(category.schemeIri())) categories.add(category);
        } } finally { iterator.close(); }
        categories.sort(java.util.Comparator.comparing(Category::iri));
        return List.copyOf(categories);
    }

    private Scheme groupScheme(Resource scheme) {
        if (!scheme.isURIResource()) throw new IllegalArgumentException("skos:ConceptScheme must have an IRI");
        String concept = topLevelConcept(scheme, mapping.fields().get("concept"));
        if (concept == null) return null;
        String label = optionalString(scheme, mapping.fields().get("label"));
        String name = optionalString(scheme, mapping.fields().get("name"));
        if (name == null) name = label;
        if (name == null) throw new IllegalArgumentException("scheme must have dct:title or skos:prefLabel: " + scheme.getURI());
        String position = optionalString(scheme, mapping.fields().get("position"));
        try {
            return new Scheme(scheme.getURI(), name, optionalString(scheme, mapping.fields().get("description")), label,
                    optionalString(scheme, mapping.fields().get("logo")), optionalString(scheme, mapping.fields().get("page")),
                    optionalString(scheme, mapping.fields().get("color")), position == null ? 0 : Integer.parseInt(position), concept);
        } catch (NumberFormatException exception) {
            throw new IllegalArgumentException("scheme position must be an integer: " + scheme.getURI(), exception);
        }
    }

    private static String optionalString(Resource resource, String property) {
        var statements = resource.listProperties();
        String result = null;
        try {
            while (statements.hasNext()) {
                var statement = statements.next();
                if (!statement.getPredicate().getURI().equals(property)) continue;
                RDFNode value = statement.getObject();
                String candidate = value.isLiteral() ? value.asLiteral().getString() : value.isURIResource() ? value.asResource().getURI() : null;
                if (candidate == null || candidate.isBlank() || result != null) throw new IllegalArgumentException("scheme metadata must have one value");
                result = candidate;
            }
        } finally { statements.close(); }
        return result;
    }

    private static String singleString(Resource resource, String property, String message) {
        String value = optionalString(resource, property);
        if (value == null) throw new IllegalArgumentException(message + ": " + resource);
        return value;
    }

    private static String singleResource(Resource resource, String property, String message) {
        var values = resource.listProperties(resource.getModel().createProperty(property));
        try {
            if (!values.hasNext()) throw new IllegalArgumentException(message + ": " + resource);
            RDFNode value = values.next().getObject();
            if (values.hasNext() || !value.isURIResource()) throw new IllegalArgumentException(message + ": " + resource);
            return value.asResource().getURI();
        } finally { values.close(); }
    }

    private static String optionalResource(Resource resource, String property, String message) {
        var values = resource.listProperties(resource.getModel().createProperty(property));
        try {
            if (!values.hasNext()) return null;
            RDFNode value = values.next().getObject();
            if (values.hasNext() || !value.isURIResource()) throw new IllegalArgumentException(message + ": " + resource);
            return value.asResource().getURI();
        } finally { values.close(); }
    }

    private static String topLevelConcept(Resource scheme, String predicate) {
        String result = null;
        var topConcepts = scheme.listProperties(scheme.getModel().createProperty(predicate));
        try {
            while (topConcepts.hasNext()) {
                RDFNode value = topConcepts.next().getObject();
                if (!value.isURIResource()) throw new IllegalArgumentException("scheme skos:hasTopConcept must be an IRI: " + scheme.getURI());
                String candidate = normalizeTopLevelConcept(value.asResource().getURI(), scheme.getURI());
                if (result != null && !result.equals(candidate)) {
                    throw new IllegalArgumentException("scheme has top concepts of multiple kinds: " + scheme.getURI());
                }
                result = candidate;
            }
        } finally { topConcepts.close(); }
        return result;
    }

    private static String normalizeTopLevelConcept(String value, String schemeIri) {
        String normalized = value.substring(value.lastIndexOf('#') + 1).substring(value.lastIndexOf('/') + 1)
                .replace('-', '_').toUpperCase(java.util.Locale.ROOT);
        String canonical = switch (normalized) {
            case "DATA", "DATASET", "DATASET_THEME" -> "DATA";
            case "FACILITIES", "FACILITY", "FACILITY_THEME" -> "FACILITIES";
            case "SOFTWARE", "SOFTWARE_THEME" -> "SOFTWARE";
            default -> null;
        };
        if (canonical == null) throw new IllegalArgumentException("unsupported legacy topLevelConcept " + value + " on " + schemeIri);
        return canonical;
    }

    public record Scheme(String iri, String name, String description, String label, String logo, String page, String color,
                         int position, String topLevelConcept) {}
    public record Category(String iri, String schemeIri, String name, String parentIri) {}
}
