package eu.epos.semanticaggregator.aggregation;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.google.protobuf.Timestamp;
import java.util.List;
import java.util.UUID;
import metadata_lifecycle.v1.Common;
import metadata_lifecycle.v1.Resource;
import org.apache.jena.rdf.model.ResourceFactory;
import org.junit.jupiter.api.Test;

class PublishedGraphBuilderTest {
    private static final PublishedResourceRef REF = new PublishedResourceRef(PublishedResourceRef.Kind.DATASET,
            UUID.fromString("123e4567-e89b-12d3-a456-426614174001"),
            UUID.fromString("123e4567-e89b-12d3-a456-426614174002"));

    @Test
    void mapsPublishedDatasetProtobufToStableDcatResource() throws Exception {
        var source = Resource.ResourceVersion.newBuilder()
                .setResourceId(REF.resourceId().toString()).setVersionId(REF.versionId().toString())
                .setRevision(7).setStatus(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED)
                .setCreatedAt(Timestamp.newBuilder().setSeconds(1_700_000_000L))
                .setDataset(Resource.DatasetResourceMetadata.newBuilder().setId("123e4567-e89b-12d3-a456-426614174003")
                        .setTitle("Earthquakes").setDescription("Observed earthquakes").addKeywords("seismic")
                        .addDistributions(Resource.Distribution.newBuilder()
                                .setId("123e4567-e89b-12d3-a456-426614174004")
                                .setTitle("CSV").setDescription("CSV download").setType("DOWNLOADABLE_FILE")
                                .setDownloadableFile(Resource.DownloadableFile.newBuilder()
                                        .setUrl("https://example.org/data.csv").setFormat("text/csv"))))
                .build();

        var graph = new PublishedGraphBuilder(reference -> source).build(AggregateRefresh.Version.V1, List.of(REF));
        try {
            var dataset = graph.model().getResource("urn:epos:metadata-lifecycle:resource:" + REF.resourceId()
                    + "/version/" + REF.versionId());
            assertTrue(graph.model().contains(dataset, ResourceFactory.createProperty("http://purl.org/dc/terms/title"), "Earthquakes"));
            assertTrue(graph.model().contains(dataset, ResourceFactory.createProperty("http://www.w3.org/ns/dcat#keyword"), "seismic"));
            assertTrue(graph.model().contains(dataset, ResourceFactory.createProperty("http://www.w3.org/ns/dcat#distribution")));
            assertEquals(java.util.Set.of(REF), graph.included());
        } finally {
            graph.model().close();
        }
    }

    @Test
    void mapsGenericWebServiceDistributionAndItsParameters() throws Exception {
        var source = Resource.ResourceVersion.newBuilder()
                .setResourceId(REF.resourceId().toString()).setVersionId(REF.versionId().toString())
                .setStatus(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED)
                .setDataset(Resource.DatasetResourceMetadata.newBuilder().setId("123e4567-e89b-12d3-a456-426614174003")
                        .setTitle("Earthquakes").setDescription("Observed earthquakes")
                        .addDistributions(Resource.Distribution.newBuilder()
                                .setId("123e4567-e89b-12d3-a456-426614174004")
                                .setTitle("API").setDescription("HTTP API").setType("WEB_SERVICE")
                                .setWebService(Resource.WebService.newBuilder()
                                        .setId("123e4567-e89b-12d3-a456-426614174005")
                                        .setName("Search").setType(Resource.WebServiceType.WEB_SERVICE_TYPE_HTTP)
                                        .setGeneric(Resource.GenericWebServiceConfiguration.newBuilder()
                                                .setTemplateUrl("https://example.org/search{?q}").setFormat("application/json")
                                                .setMethod(Resource.WebServiceMethod.WEB_SERVICE_METHOD_GET)
                                                .addParameters(Resource.Parameter.newBuilder()
                                                        .setId("123e4567-e89b-12d3-a456-426614174006")
                                                        .setName("q").setLabel("Query").setRequired(true)
                                                        .setString(Resource.StringParameter.getDefaultInstance()))))))
                .build();

        var graph = new PublishedGraphBuilder(reference -> source).build(AggregateRefresh.Version.V1, List.of(REF));
        try {
            var dataset = graph.model().getResource("urn:epos:metadata-lifecycle:resource:" + REF.resourceId()
                    + "/version/" + REF.versionId());
            assertTrue(graph.model().contains(dataset, ResourceFactory.createProperty("http://www.w3.org/ns/dcat#distribution")));
            assertTrue(graph.model().contains(null, ResourceFactory.createProperty("http://www.w3.org/ns/dcat#endpointURL"),
                    graph.model().createResource("https://example.org/search{?q}")));
            assertTrue(graph.model().contains(null, ResourceFactory.createProperty("http://www.w3.org/ns/hydra/core#variable"), "q"));
        } finally {
            graph.model().close();
        }
    }

    @Test
    void mapsSoftwareReferencesAndPlatformMetadata() throws Exception {
        var software = new PublishedResourceRef(PublishedResourceRef.Kind.SOFTWARE_SOURCE_CODE,
                UUID.fromString("123e4567-e89b-12d3-a456-426614174010"),
                UUID.fromString("123e4567-e89b-12d3-a456-426614174011"));
        var source = Resource.ResourceVersion.newBuilder()
                .setResourceId(software.resourceId().toString()).setVersionId(software.versionId().toString())
                .setStatus(Common.VersioningStatus.VERSIONING_STATUS_PUBLISHED)
                .setSoftwareSourceCode(Resource.SoftwareSourceCodeResourceMetadata.newBuilder()
                        .setId("123e4567-e89b-12d3-a456-426614174012").setName("Exporter")
                        .setDescription("RDF exporter").setRepository("https://example.org/repository")
                        .setProgrammingLanguage("Java").setRuntimePlatform("JVM")
                        .setMainEntityOfPage("https://example.org/exporter")
                        .addCreators(metadata_lifecycle.v1.Common.EntityReference.newBuilder()
                                .setId("123e4567-e89b-12d3-a456-426614174013").setName("EPOS"))
                        .addCategories(metadata_lifecycle.v1.Common.EntityReference.newBuilder()
                                .setId("123e4567-e89b-12d3-a456-426614174014").setName("Software"))
                        .addContactPoints("123e4567-e89b-12d3-a456-426614174015")
                        .addIdentifiers(metadata_lifecycle.v1.Common.Identifier.newBuilder()
                                .setId("123e4567-e89b-12d3-a456-426614174016").setType("DOI").setValue("10.1234/exporter")))
                .build();

        var graph = new PublishedGraphBuilder(reference -> source).build(AggregateRefresh.Version.V1, List.of(software));
        try {
            var resource = graph.model().getResource("urn:epos:metadata-lifecycle:resource:" + software.resourceId()
                    + "/version/" + software.versionId());
            assertTrue(graph.model().contains(resource, ResourceFactory.createProperty("https://schema.org/runtimePlatform"), "JVM"));
            assertTrue(graph.model().contains(resource, ResourceFactory.createProperty("http://purl.org/dc/terms/creator"),
                    graph.model().createResource("urn:epos:metadata-lifecycle:organization:123e4567-e89b-12d3-a456-426614174013")));
            assertTrue(graph.model().contains(resource, ResourceFactory.createProperty("http://www.w3.org/ns/adms#identifier")));
        } finally {
            graph.model().close();
        }
    }
}
