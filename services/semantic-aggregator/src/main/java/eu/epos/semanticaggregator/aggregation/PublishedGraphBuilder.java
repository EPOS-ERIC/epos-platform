package eu.epos.semanticaggregator.aggregation;

import com.google.protobuf.Timestamp;
import eu.epos.semanticaggregator.upstream.PublishedCatalog;
import java.io.IOException;
import java.time.Instant;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import metadata_lifecycle.v1.Resource;
import org.apache.jena.datatypes.xsd.XSDDatatype;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.ModelFactory;
import org.apache.jena.rdf.model.Property;
import org.apache.jena.rdf.model.ResourceFactory;

/** Converts the published Metadata Lifecycle protobuf representation into an owned RDF graph. */
public final class PublishedGraphBuilder implements AggregateRefresh.Builder {
    private static final String DCAT = "http://www.w3.org/ns/dcat#";
    private static final String DCT = "http://purl.org/dc/terms/";
    private static final String RDF = "http://www.w3.org/1999/02/22-rdf-syntax-ns#";
    private static final String SCHEMA = "https://schema.org/";
    private static final String EPOS = "https://www.epos-eu.org/epos-dcat-ap#";
    private static final String RESOURCE_BASE = "urn:epos:metadata-lifecycle:resource:";

    @FunctionalInterface
    interface VersionReader {
        Resource.ResourceVersion fetch(PublishedResourceRef reference);
    }

    private final VersionReader versionReader;

    public PublishedGraphBuilder(PublishedCatalog catalog) {
        this(java.util.Objects.requireNonNull(catalog, "catalog")::fetchVersion);
    }

    PublishedGraphBuilder(VersionReader versionReader) {
        this.versionReader = java.util.Objects.requireNonNull(versionReader, "versionReader");
    }

    @Override
    public AggregateRefresh.BuiltGraph build(AggregateRefresh.Version version, List<PublishedResourceRef> resources)
            throws IOException, InterruptedException {
        Model model = ModelFactory.createDefaultModel();
        Set<PublishedResourceRef> included = new LinkedHashSet<>();
        try {
            for (PublishedResourceRef reference : resources) {
                map(model, reference, versionReader.fetch(reference));
                included.add(reference);
            }
            return new AggregateRefresh.BuiltGraph(model, included);
        } catch (RuntimeException exception) {
            model.close();
            throw exception;
        }
    }

    private static void map(Model model, PublishedResourceRef reference, Resource.ResourceVersion source) {
        var resource = model.createResource(resourceIri(reference));
        resource.addProperty(property(RDF + "type"), switch (reference.kind()) {
            case DATASET -> model.createResource(DCAT + "Dataset");
            case SOFTWARE_APPLICATION -> model.createResource(SCHEMA + "SoftwareApplication");
            case SOFTWARE_SOURCE_CODE -> model.createResource(SCHEMA + "SoftwareSourceCode");
        });
        resource.addLiteral(property(DCT + "identifier"), reference.resourceId().toString());
        resource.addLiteral(property(EPOS + "versionIdentifier"), reference.versionId().toString());
        resource.addLiteral(property(EPOS + "revision"), source.getRevision());
        addTimestamp(resource, DCT + "created", source.getCreatedAt());
        addTimestamp(resource, DCT + "modified", source.getUpdatedAt());

        switch (reference.kind()) {
            case DATASET -> mapDataset(model, resource, source.getDataset());
            case SOFTWARE_APPLICATION -> mapSoftwareApplication(resource, source.getSoftwareApplication());
            case SOFTWARE_SOURCE_CODE -> mapSoftwareSourceCode(resource, source.getSoftwareSourceCode());
        }
    }

    private static void mapDataset(Model model, org.apache.jena.rdf.model.Resource target,
                                   Resource.DatasetResourceMetadata source) {
        text(target, DCT + "title", source.getTitle());
        text(target, DCT + "description", source.getDescription());
        for (String keyword : source.getKeywordsList()) text(target, DCAT + "keyword", keyword);
        optional(target, DCT + "accrualPeriodicity", source.hasAccrualPeriodicity(), source.getAccrualPeriodicity().name());
        optional(target, DCT + "accessRights", source.hasAccessRight(), source.getAccessRight().name());
        optional(target, "http://www.w3.org/ns/dqv#hasQualityAnnotation", source.hasQualityAnnotation(), source.getQualityAnnotation());
        optional(target, "http://purl.org/dc/terms/spatial", source.hasSpatialCoverage(), source.getSpatialCoverage());
        addTimestamp(target, DCT + "created", source.getCreated());
        addTimestamp(target, DCT + "modified", source.getUpdated());
        if (source.hasTemporalCoverage()) {
            addTimestamp(target, SCHEMA + "startDate", source.getTemporalCoverage().getStart());
            addTimestamp(target, SCHEMA + "endDate", source.getTemporalCoverage().getEnd());
        }
        if (source.hasGeologicalTemporalCoverage()) {
            throw new IllegalArgumentException("geological temporal coverage is not supported by the RDF exporter");
        }
        for (String contactId : source.getContactPointsList()) {
            target.addProperty(property(DCAT + "contactPoint"), entity(target.getModel(), "contact-point", contactId));
        }
        for (var provider : source.getProvidersList()) {
            target.addProperty(property(DCT + "publisher"), entity(target.getModel(), "organization", provider.getId()));
        }
        for (var category : source.getCategoriesList()) {
            target.addProperty(property(DCAT + "theme"), entity(target.getModel(), "category", category.getId()));
        }
        mapIdentifiers(model, target, source.getPersistentIdentifiersList());
        for (var distribution : source.getDistributionsList()) mapDistribution(model, target, distribution);
    }

    private static void mapDistribution(Model model, org.apache.jena.rdf.model.Resource dataset, Resource.Distribution source) {
        var distribution = model.createResource(dataset.getURI() + "/distribution/" + source.getId());
        distribution.addProperty(property(RDF + "type"), model.createResource(DCAT + "Distribution"));
        text(distribution, DCT + "title", source.getTitle());
        text(distribution, DCT + "description", source.getDescription());
        optional(distribution, DCT + "license", source.hasLicense(), source.getLicense());
        if (source.hasByteSize()) distribution.addLiteral(property(DCAT + "byteSize"), source.getByteSize());
        if (source.hasDownloadableFile()) {
            distribution.addProperty(property(DCAT + "downloadURL"), model.createResource(source.getDownloadableFile().getUrl()));
            text(distribution, DCT + "format", source.getDownloadableFile().getFormat());
        }
        if (source.hasWebService()) {
            mapWebService(model, distribution, source.getWebService());
        }
        for (var format : source.getNativeFormatsList()) text(distribution, DCT + "format", format.getOutputFormat());
        dataset.addProperty(property(DCAT + "distribution"), distribution);
    }

    private static void mapWebService(Model model, org.apache.jena.rdf.model.Resource distribution, Resource.WebService source) {
        var service = model.createResource(distribution.getURI() + "/service/" + source.getId());
        service.addProperty(property(RDF + "type"), model.createResource(DCAT + "DataService"));
        service.addLiteral(property(EPOS + "webServiceType"), source.getType().name());
        if (!source.getProviderReference().getId().isEmpty()) {
            service.addProperty(property(DCT + "publisher"), entity(model, "organization", source.getProviderReference().getId()));
        }
        for (String contactId : source.getContactPointsList()) {
            service.addProperty(property(DCAT + "contactPoint"), entity(model, "contact-point", contactId));
        }
        if (source.hasDocumentationUri()) {
            service.addProperty(property("http://xmlns.com/foaf/0.1/page"), model.createResource(source.getDocumentationUri()));
        }
        distribution.addProperty(property(DCAT + "accessService"), service);
        switch (source.getConfigurationCase()) {
            case GENERIC -> mapGenericConfiguration(model, service, source.getGeneric());
            case DYNAMIC -> mapGenericConfiguration(model, service, source.getDynamic());
            case WFS -> {
                var config = source.getWfs();
                service.addProperty(property(DCAT + "endpointURL"), model.createResource(config.getGetFeatureTemplateUrl()));
                text(service, EPOS + "wfsVersion", config.getVersion());
                text(service, EPOS + "defaultFeatureType", config.getDefaultFeatureType());
                text(service, DCT + "format", config.getDefaultOutputFormat());
                mapParameters(model, service, config.getParametersList());
            }
            case WMS -> {
                var config = source.getWms();
                service.addProperty(property(DCAT + "endpointURL"), model.createResource(config.getCapabilitiesUrl()));
                text(service, EPOS + "defaultLayer", config.getDefaultLayer());
                mapParameters(model, service, config.getGetMapParametersList());
                mapParameters(model, service, config.getGetFeatureInfoParametersList());
            }
            case WMTS -> {
                var config = source.getWmts();
                service.addProperty(property(DCAT + "endpointURL"), model.createResource(config.getCapabilitiesUrl()));
                text(service, EPOS + "defaultLayer", config.getDefaultLayer());
                mapParameters(model, service, config.getGetTileParametersList());
                mapParameters(model, service, config.getGetFeatureInfoParametersList());
            }
            case CONFIGURATION_NOT_SET -> throw new IllegalArgumentException("web service configuration is missing");
        }
    }

    private static void mapGenericConfiguration(Model model, org.apache.jena.rdf.model.Resource service,
                                                Resource.GenericWebServiceConfiguration source) {
        service.addProperty(property(DCAT + "endpointURL"), model.createResource(source.getTemplateUrl()));
        text(service, DCT + "format", source.getFormat());
        service.addLiteral(property(EPOS + "httpMethod"), source.getMethod().name());
        mapParameters(model, service, source.getParametersList());
    }

    private static void mapParameters(Model model, org.apache.jena.rdf.model.Resource service,
                                      List<Resource.Parameter> parameters) {
        for (var source : parameters) {
            var parameter = model.createResource(service.getURI() + "/parameter/" + source.getId());
            parameter.addProperty(property(RDF + "type"), model.createResource("http://www.w3.org/ns/hydra/core#IriTemplateMapping"));
            text(parameter, "http://www.w3.org/ns/hydra/core#variable", source.getName());
            text(parameter, "http://www.w3.org/2000/01/rdf-schema#label", source.getLabel());
            parameter.addLiteral(property("http://www.w3.org/ns/hydra/core#required"), source.getRequired());
            parameter.addLiteral(property(EPOS + "readOnly"), source.getReadOnly());
            parameter.addLiteral(property(EPOS + "position"), source.getPosition());
            parameter.addLiteral(property(EPOS + "location"), source.getLocation().name());
            parameter.addLiteral(property(EPOS + "role"), source.getRole().name());
            optional(parameter, DCT + "description", source.hasDescription(), source.getDescription());
            optional(parameter, EPOS + "property", source.hasProperty(), source.getProperty());
            service.addProperty(property("http://www.w3.org/ns/hydra/core#mapping"), parameter);
        }
    }

    private static void mapSoftwareApplication(org.apache.jena.rdf.model.Resource target,
                                               Resource.SoftwareApplicationResourceMetadata source) {
        Model model = target.getModel();
        text(target, "https://schema.org/name", source.getName());
        text(target, DCT + "description", source.getDescription());
        for (String keyword : source.getKeywordsList()) text(target, DCAT + "keyword", keyword);
        optional(target, "https://schema.org/softwareVersion", source.hasVersion(), source.getVersion());
        optional(target, "https://schema.org/downloadUrl", source.hasDownloadUrl(), source.getDownloadUrl());
        optional(target, DCT + "license", source.hasLicense(), source.getLicense());
        optional(target, "https://schema.org/mainEntityOfPage", source.hasMainEntityOfPage(), source.getMainEntityOfPage());
        optional(target, "https://schema.org/softwareRequirements", source.hasRequirements(), source.getRequirements());
        optional(target, "https://schema.org/operatingSystem", source.hasOperatingSystem(), source.getOperatingSystem());
        for (var creator : source.getCreatorsList()) {
            target.addProperty(property(DCT + "creator"), entity(model, "organization", creator.getId()));
        }
        for (var category : source.getCategoriesList()) {
            target.addProperty(property(DCAT + "theme"), entity(model, "category", category.getId()));
        }
        for (String contact : source.getContactPointsList()) {
            target.addProperty(property(DCAT + "contactPoint"), entity(model, "contact-point", contact));
        }
        mapIdentifiers(model, target, source.getIdentifiersList());
    }

    private static void mapSoftwareSourceCode(org.apache.jena.rdf.model.Resource target,
                                              Resource.SoftwareSourceCodeResourceMetadata source) {
        Model model = target.getModel();
        text(target, "https://schema.org/name", source.getName());
        text(target, DCT + "description", source.getDescription());
        for (String keyword : source.getKeywordsList()) text(target, DCAT + "keyword", keyword);
        optional(target, "https://schema.org/codeRepository", source.hasRepository(), source.getRepository());
        optional(target, "https://schema.org/programmingLanguage", source.hasProgrammingLanguage(), source.getProgrammingLanguage());
        optional(target, "https://schema.org/softwareVersion", source.hasVersion(), source.getVersion());
        optional(target, DCT + "license", source.hasLicense(), source.getLicense());
        optional(target, "https://schema.org/softwareRequirements", source.hasSoftwareRequirements(), source.getSoftwareRequirements());
        optional(target, "https://schema.org/runtimePlatform", source.hasRuntimePlatform(), source.getRuntimePlatform());
        optional(target, "https://schema.org/mainEntityOfPage", source.hasMainEntityOfPage(), source.getMainEntityOfPage());
        for (var creator : source.getCreatorsList()) {
            target.addProperty(property(DCT + "creator"), entity(model, "organization", creator.getId()));
        }
        for (var category : source.getCategoriesList()) {
            target.addProperty(property(DCAT + "theme"), entity(model, "category", category.getId()));
        }
        for (String contact : source.getContactPointsList()) {
            target.addProperty(property(DCAT + "contactPoint"), entity(model, "contact-point", contact));
        }
        mapIdentifiers(model, target, source.getIdentifiersList());
    }

    private static String resourceIri(PublishedResourceRef reference) {
        return RESOURCE_BASE + reference.resourceId() + "/version/" + reference.versionId();
    }

    private static Property property(String iri) { return ResourceFactory.createProperty(iri); }

    private static org.apache.jena.rdf.model.Resource entity(Model model, String kind, String id) {
        return model.createResource("urn:epos:metadata-lifecycle:" + kind + ":" + id);
    }

    private static void mapIdentifiers(Model model, org.apache.jena.rdf.model.Resource target,
                                       List<Resource.VersionedIdentifier> identifiers) {
        for (var identifier : identifiers) {
            var node = model.createResource();
            node.addLiteral(property("http://www.w3.org/2004/02/skos/core#notation"), identifier.getValue());
            node.addLiteral(property(DCAT + "qualifiedRelation"), identifier.getType());
            target.addProperty(property("http://www.w3.org/ns/adms#identifier"), node);
        }
    }

    private static void text(org.apache.jena.rdf.model.Resource target, String predicate, String value) {
        if (!value.isEmpty()) target.addLiteral(property(predicate), value);
    }

    private static void optional(org.apache.jena.rdf.model.Resource target, String predicate, boolean present, String value) {
        if (present) text(target, predicate, value);
    }

    private static void addTimestamp(org.apache.jena.rdf.model.Resource target, String predicate, Timestamp timestamp) {
        if (timestamp.getSeconds() != 0 || timestamp.getNanos() != 0) {
            target.addLiteral(property(predicate), ResourceFactory.createTypedLiteral(
                    Instant.ofEpochSecond(timestamp.getSeconds(), timestamp.getNanos()).toString(), XSDDatatype.XSDdateTime));
        }
    }
}
