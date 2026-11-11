package eu.epos.semanticaggregator.grpc;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import eu.epos.semanticaggregator.mapping.MappingRegistry;
import eu.epos.semanticaggregator.upstream.MetadataLifecycleClient;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptors;
import io.grpc.Status;
import io.grpc.stub.MetadataUtils;
import io.grpc.stub.StreamObserver;
import java.nio.charset.StandardCharsets;
import metadata_lifecycle.v1.Resource;
import metadata_lifecycle.v1.ResourceServiceGrpc;
import metadata_lifecycle.v1.GroupOuterClass;
import metadata_lifecycle.v1.GroupServiceGrpc;
import metadata_lifecycle.v1.Category;
import metadata_lifecycle.v1.CategoryServiceGrpc;
import metadata_lifecycle.v1.OrganizationOuterClass;
import metadata_lifecycle.v1.OrganizationServiceGrpc;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import semantic_aggregator.v1.Import;
import semantic_aggregator.v1.ResourceImportServiceGrpc;

class ResourceImportServerTest {
    private Server upstreamServer;
    private Server importServer;
    private ManagedChannel upstreamChannel;
    private ManagedChannel importChannel;
    private final java.util.List<String> createdGroups = new java.util.ArrayList<>();
    private final java.util.List<String> createdSchemeNames = new java.util.ArrayList<>();
    private final java.util.List<String> createdCategories = new java.util.ArrayList<>();
    private final java.util.List<Category.CreateCategoryRequest> createdCategoryRequests = new java.util.ArrayList<>();
    private final java.util.List<Resource.CreateResourceRequest> createdResources = new java.util.ArrayList<>();
    private final java.util.Set<String> missingCategoryGroups = new java.util.HashSet<>();
    private final java.util.Set<String> missingCategoryIds = new java.util.HashSet<>();
    private final java.util.Set<String> rejectedDatasetTitles = new java.util.HashSet<>();
    private final java.util.Map<String, OrganizationOuterClass.Organization> organizations = new java.util.LinkedHashMap<>();

    @BeforeEach
    void startServers() throws Exception {
        upstreamServer = ServerBuilder.forPort(0).addService(new ResourceServiceGrpc.ResourceServiceImplBase() {
            @Override
            public void createResource(Resource.CreateResourceRequest request,
                                        StreamObserver<Resource.CreateResourceResponse> observer) {
                createdGroups.add(request.getGroupId());
                createdResources.add(request);
                if (request.hasDataset() && rejectedDatasetTitles.contains(request.getDataset().getTitle())) {
                    observer.onError(Status.INVALID_ARGUMENT.withDescription("dataset metadata is invalid").asRuntimeException());
                    return;
                }
                var resource = Resource.ResourceRecord.newBuilder().setResourceId("123e4567-e89b-12d3-a456-426614174001")
                        .setVersionId("123e4567-e89b-12d3-a456-426614174002").build();
                observer.onNext(Resource.CreateResourceResponse.newBuilder().setResource(resource).build());
                observer.onCompleted();
            }
        }).addService(new GroupServiceGrpc.GroupServiceImplBase() {
            @Override public void createGroup(GroupOuterClass.CreateGroupRequest request, StreamObserver<GroupOuterClass.CreateGroupResponse> observer) {
                createdSchemeNames.add(request.getName());
                String id = switch (request.getName()) {
                    case "Software" -> "123e4567-e89b-12d3-a456-426614174010";
                    case "Facilities" -> "123e4567-e89b-12d3-a456-426614174011";
                    case "Data" -> "123e4567-e89b-12d3-a456-426614174012";
                    default -> throw new IllegalArgumentException("unexpected group");
                };
                var group = GroupOuterClass.Group.newBuilder().setId(id).setName(request.getName())
                        .setPosition(request.getPosition()).setConcept(request.getConcept());
                if (request.hasDescription()) group.setDescription(request.getDescription());
                if (request.hasLabel()) group.setLabel(request.getLabel());
                if (request.hasLogo()) group.setLogo(request.getLogo());
                if (request.hasPage()) group.setPage(request.getPage());
                if (request.hasColor()) group.setColor(request.getColor());
                observer.onNext(GroupOuterClass.CreateGroupResponse.newBuilder().setGroup(group).build()); observer.onCompleted();
            }
        }).addService(new CategoryServiceGrpc.CategoryServiceImplBase() {
            @Override public void getCategory(Category.GetCategoryRequest request, StreamObserver<Category.GetCategoryResponse> observer) {
                if (missingCategoryGroups.contains(request.getGroupId()) || missingCategoryIds.contains(request.getCategoryId())) {
                    observer.onError(Status.NOT_FOUND.asRuntimeException());
                    return;
                }
                observer.onNext(Category.GetCategoryResponse.newBuilder().setCategory(Category.CanonicalCategory.newBuilder()
                        .setId(request.getCategoryId()).setGroupId(request.getGroupId()).setName("Category")).build()); observer.onCompleted();
            }
            @Override public void createCategory(Category.CreateCategoryRequest request, StreamObserver<Category.CreateCategoryResponse> observer) {
                createdCategories.add(request.getGroupId());
                createdCategoryRequests.add(request);
                var category = Category.CanonicalCategory.newBuilder().setId(categoryId(request.getGroupId()))
                        .setGroupId(request.getGroupId()).setName(request.getName());
                if (request.hasParentCategoryId()) category.setParentCategoryId(request.getParentCategoryId());
                observer.onNext(Category.CreateCategoryResponse.newBuilder().setCategory(category).build()); observer.onCompleted();
            }
        }).addService(new OrganizationServiceGrpc.OrganizationServiceImplBase() {
            @Override public void listOrganizations(OrganizationOuterClass.ListOrganizationsRequest request,
                                                    StreamObserver<OrganizationOuterClass.ListOrganizationsResponse> observer) {
                observer.onNext(OrganizationOuterClass.ListOrganizationsResponse.newBuilder()
                        .addAllOrganizations(organizations.values()).build());
                observer.onCompleted();
            }
            @Override public void getOrganization(OrganizationOuterClass.GetOrganizationRequest request,
                                                   StreamObserver<OrganizationOuterClass.GetOrganizationResponse> observer) {
                var organization = organizations.get(request.getOrganizationId());
                if (organization == null) {
                    observer.onError(Status.NOT_FOUND.asRuntimeException());
                    return;
                }
                observer.onNext(OrganizationOuterClass.GetOrganizationResponse.newBuilder().setOrganization(organization).build());
                observer.onCompleted();
            }
            @Override public void createOrganization(OrganizationOuterClass.CreateOrganizationRequest request,
                                                      StreamObserver<OrganizationOuterClass.CreateOrganizationResponse> observer) {
                String id = "123e4567-e89b-12d3-a456-426614174030";
                var organization = OrganizationOuterClass.Organization.newBuilder().setId(id).setName(request.getName());
                if (request.hasStreet()) organization.setStreet(request.getStreet());
                if (request.hasLocality()) organization.setLocality(request.getLocality());
                if (request.hasPostalCode()) organization.setPostalCode(request.getPostalCode());
                if (request.hasCountry()) organization.setCountry(request.getCountry());
                if (request.hasLogoUrl()) organization.setLogoUrl(request.getLogoUrl());
                if (request.hasPageUrl()) organization.setPageUrl(request.getPageUrl());
                if (request.hasEmail()) organization.setEmail(request.getEmail());
                if (request.hasLeiCode()) organization.setLeiCode(request.getLeiCode());
                organizations.put(id, organization.build());
                observer.onNext(OrganizationOuterClass.CreateOrganizationResponse.newBuilder().setOrganization(organization).build());
                observer.onCompleted();
            }
        }).build().start();
        upstreamChannel = ManagedChannelBuilder.forAddress("localhost", upstreamServer.getPort()).usePlaintext().build();
        var mappings = MappingRegistry.withDefaults();
        importServer = ServerBuilder.forPort(0).addService(ServerInterceptors.intercept(
                new ResourceImportServer(mappings, new MetadataLifecycleClient(upstreamChannel), java.nio.file.Files.createTempDirectory("resource-import-test")), new ActorInterceptor())).build().start();
        importChannel = ManagedChannelBuilder.forAddress("localhost", importServer.getPort()).usePlaintext().build();
    }

    @AfterEach
    void stopServers() {
        if (importChannel != null) importChannel.shutdownNow();
        if (importServer != null) importServer.shutdownNow();
        if (upstreamChannel != null) upstreamChannel.shutdownNow();
        if (upstreamServer != null) upstreamServer.shutdownNow();
    }

    @Test
    void requiresActorAndCreatesSoftwareApplication() {
        var request = request(false);
        var stub = ResourceImportServiceGrpc.newBlockingStub(importChannel);
        var unauthenticated = assertThrows(io.grpc.StatusRuntimeException.class, () -> stub.importResources(request));
        assertEquals(Status.Code.UNAUTHENTICATED, unauthenticated.getStatus().getCode());

        Metadata actor = new Metadata();
        actor.put(Metadata.Key.of("x-epos-user-id", Metadata.ASCII_STRING_MARSHALLER), "actor-1");
        var response = stub.withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actor)).importResources(request);
        assertEquals(2, response.getImportedCount());
        assertEquals(Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_SOFTWARE_APPLICATION, response.getImported(0).getKind());
        assertEquals("123e4567-e89b-12d3-a456-426614174001", response.getImported(0).getResourceId());
        assertEquals(java.util.List.of("123e4567-e89b-12d3-a456-426614174010", "123e4567-e89b-12d3-a456-426614174011"), createdGroups);
    }

    @Test
    void importsDatasetsAndSoftwareIntoTheirSchemeGroups() {
        Metadata actor = new Metadata();
        actor.put(Metadata.Key.of("x-epos-user-id", Metadata.ASCII_STRING_MARSHALLER), "actor-1");

        var response = ResourceImportServiceGrpc.newBlockingStub(importChannel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actor)).importResources(request(true));

        assertEquals(4, response.getImportedCount());
        assertEquals(Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_DATASET, response.getImported(0).getKind());
        assertEquals(Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_DATASET, response.getImported(1).getKind());
        assertEquals(Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_SOFTWARE_APPLICATION, response.getImported(2).getKind());
        assertEquals(Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_SOFTWARE_SOURCE_CODE, response.getImported(3).getKind());
        assertEquals(4, createdResources.size());
        assertEquals("123e4567-e89b-12d3-a456-426614174011", createdResources.get(0).getGroupId());
        assertEquals("123e4567-e89b-12d3-a456-426614174012", createdResources.get(1).getGroupId());
        assertEquals("123e4567-e89b-12d3-a456-426614174010", createdResources.get(2).getGroupId());
        assertEquals("123e4567-e89b-12d3-a456-426614174011", createdResources.get(3).getGroupId());
        assertEquals(Resource.CreateResourceRequest.MetadataCase.DATASET, createdResources.get(0).getMetadataCase());
        assertEquals(Resource.CreateResourceRequest.MetadataCase.DATASET, createdResources.get(1).getMetadataCase());
    }

    @Test
    void duplicatesResourcesWithCategoriesFromMultipleSchemes() {
        var response = authenticatedStub().importResources(multiSchemeRequest());

        assertEquals(5, response.getImportedCount());
        assertEquals(2, response.getImportedList().stream()
                .filter(item -> item.getSubject().equals("https://example.org/dataset-one")).count());
        assertEquals(java.util.List.of(
                "123e4567-e89b-12d3-a456-426614174012",
                "123e4567-e89b-12d3-a456-426614174011"), createdGroups.subList(0, 2));
        assertEquals(1, createdResources.get(0).getDataset().getCategoriesCount());
        assertEquals(1, createdResources.get(1).getDataset().getCategoriesCount());
    }

    @Test
    void retryReusesMappingsForEveryReferencedScheme() {
        var stub = authenticatedStub();

        stub.importResources(request(true));
        var retried = stub.importResources(request(true));

        assertEquals(4, retried.getImportedCount());
        assertEquals(java.util.List.of("Data", "Facilities", "Software"), createdSchemeNames);
        assertEquals(java.util.List.of(
                "123e4567-e89b-12d3-a456-426614174012",
                "123e4567-e89b-12d3-a456-426614174011",
                "123e4567-e89b-12d3-a456-426614174010"), createdCategories);
        assertEquals(8, createdResources.size());
    }

    @Test
    void doesNotRecreateMappingsWhenStoredSchemeGroupIsStale() {
        var stub = authenticatedStub();
        stub.importResources(request(true));
        missingCategoryGroups.add("123e4567-e89b-12d3-a456-426614174011");

        var retried = stub.importResources(request(true));

        assertEquals("https://example.org/facilities", retried.getErrors(0).getSubject());
        assertEquals("NOT_FOUND", retried.getErrors(0).getCode());
        org.junit.jupiter.api.Assertions.assertTrue(retried.getErrors(0).getMessage()
                .contains("verify category https://example.org/facility-category"));
        assertEquals(java.util.List.of("Data", "Facilities", "Software"), createdSchemeNames);
        assertEquals(3, createdCategories.size());
        assertEquals(6, createdResources.size());
    }

    @Test
    void doesNotRecreateMappingsWhenStoredCategoryIsStale() {
        var stub = authenticatedStub();
        stub.importResources(request(true));
        missingCategoryIds.add(categoryId("123e4567-e89b-12d3-a456-426614174011"));

        var retried = stub.importResources(request(true));

        assertEquals("https://example.org/facilities", retried.getErrors(0).getSubject());
        assertEquals("NOT_FOUND", retried.getErrors(0).getCode());
        org.junit.jupiter.api.Assertions.assertTrue(retried.getErrors(0).getMessage()
                .contains("verify category https://example.org/facility-category"));
        assertEquals(java.util.List.of("Data", "Facilities", "Software"), createdSchemeNames);
        assertEquals(3, createdCategories.size());
        assertEquals(6, createdResources.size());
    }

    @Test
    void skipsRejectedResourcesAndReportsWarnings() {
        rejectedDatasetTitles.add("Dataset One");

        var response = authenticatedStub().importResources(request(true));

        assertEquals(3, response.getImportedCount());
        assertEquals(1, response.getErrorsCount());
        assertEquals("https://example.org/dataset-one", response.getErrors(0).getSubject());
        assertEquals("INVALID_ARGUMENT", response.getErrors(0).getCode());
        assertEquals("dataset metadata is invalid", response.getErrors(0).getMessage());
        assertEquals(4, createdResources.size());
    }

    @Test
    void importsEverySchemeCategoryWithItsHierarchy() {
        var response = authenticatedStub().importResources(hierarchicalCategoryRequest());

        assertEquals(1, response.getImportedCount());
        assertEquals(3, createdCategoryRequests.size());
        var child = createdCategoryRequests.stream().filter(item -> item.getName().equals("Child")).findFirst().orElseThrow();
        assertEquals(categoryId("123e4567-e89b-12d3-a456-426614174012"), child.getParentCategoryId());
        assertEquals(1, createdCategoryRequests.stream().filter(item -> item.getName().equals("Unused")).count());
    }

    @Test
    void reusesCompatibleOrganizationsWithTheSameName() {
        String turtle = request(true).getTurtle().toStringUtf8()
                .replace("ex:app a", "ex:organization-one a <http://schema.org/Organization>; <http://schema.org/legalName> \"Example Organization\"; <http://schema.org/email> \"info@example.org\" .\n"
                        + "ex:organization-two a <http://schema.org/Organization>; <http://schema.org/legalName> \"Example Organization\"; <http://schema.org/email> \"info@example.org\" .\n"
                        + "ex:app a")
                .replace("<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:facility-category .",
                        "<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:facility-category; <http://purl.org/dc/terms/publisher> ex:organization-one .")
                .replace("<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:data-category .",
                        "<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:data-category; <http://purl.org/dc/terms/publisher> ex:organization-two .");
        var request = Import.ImportResourcesRequest.newBuilder().setTurtle(com.google.protobuf.ByteString.copyFrom(turtle, StandardCharsets.UTF_8))
                .setBaseUri("https://example.org/").setMappingId("epos-dcat-common").setMappingVersion(1).build();

        var response = authenticatedStub().importResources(request);

        assertEquals(4, response.getImportedCount());
        assertEquals(0, response.getWarningsCount());
        assertEquals(1, organizations.size());
    }

    @Test
    void rejectsOrganizationsWithTheSameNameAndConflictingFields() {
        String turtle = request(true).getTurtle().toStringUtf8()
                .replace("ex:app a", "ex:organization-one a <http://schema.org/Organization>; <http://schema.org/legalName> \"Example Organization\"; <http://schema.org/email> \"one@example.org\" .\n"
                        + "ex:organization-two a <http://schema.org/Organization>; <http://schema.org/legalName> \"Example Organization\"; <http://schema.org/email> \"two@example.org\" .\n"
                        + "ex:app a")
                .replace("<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:facility-category .",
                        "<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:facility-category; <http://purl.org/dc/terms/publisher> ex:organization-one .")
                .replace("<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:data-category .",
                        "<http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:data-category; <http://purl.org/dc/terms/publisher> ex:organization-two .");
        var request = Import.ImportResourcesRequest.newBuilder().setTurtle(com.google.protobuf.ByteString.copyFrom(turtle, StandardCharsets.UTF_8))
                .setBaseUri("https://example.org/").setMappingId("epos-dcat-common").setMappingVersion(1).build();

        var response = authenticatedStub().importResources(request);

        assertEquals(3, response.getImportedCount());
        assertEquals(1, response.getErrorsCount());
        assertEquals(Status.Code.ALREADY_EXISTS.name(), response.getErrors(0).getCode());
        assertEquals(1, organizations.size());
    }

    private ResourceImportServiceGrpc.ResourceImportServiceBlockingStub authenticatedStub() {
        Metadata actor = new Metadata();
        actor.put(Metadata.Key.of("x-epos-user-id", Metadata.ASCII_STRING_MARSHALLER), "actor-1");
        return ResourceImportServiceGrpc.newBlockingStub(importChannel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actor));
    }

    private static String categoryId(String groupId) {
        return switch (groupId) {
            case "123e4567-e89b-12d3-a456-426614174010" -> "123e4567-e89b-12d3-a456-426614174020";
            case "123e4567-e89b-12d3-a456-426614174011" -> "123e4567-e89b-12d3-a456-426614174021";
            case "123e4567-e89b-12d3-a456-426614174012" -> "123e4567-e89b-12d3-a456-426614174022";
            default -> throw new IllegalArgumentException("unexpected category group");
        };
    }

    private static Import.ImportResourcesRequest request(boolean includeDatasets) {
        String turtle = """
                @prefix schema: <http://schema.org/> .
                @prefix ex: <https://example.org/> .
                @prefix skos: <http://www.w3.org/2004/02/skos/core#> .
                @prefix category: <https://www.epos-eu.org/epos-dcat-ap/category/> .
                ex:software a skos:ConceptScheme; skos:prefLabel "Software"; skos:hasTopConcept <category:facets/software-theme> .
                ex:category a skos:Concept; skos:prefLabel "Category"; skos:inScheme ex:software .
                ex:facilities a skos:ConceptScheme; skos:prefLabel "Facilities"; skos:hasTopConcept <category:facets/facility-theme> .
                ex:facility-category a skos:Concept; skos:prefLabel "Facility Category"; skos:inScheme ex:facilities .
                ex:data a skos:ConceptScheme; skos:prefLabel "Data"; skos:hasTopConcept <category:facets/dataset-theme> .
                ex:data-category a skos:Concept; skos:prefLabel "Data Category"; skos:inScheme ex:data .
                %s
                ex:app a schema:SoftwareApplication; schema:name "Application"; schema:description "Description"; <http://www.w3.org/ns/dcat#theme> ex:category .
                ex:source a schema:SoftwareSourceCode; schema:name "Source"; schema:description "Description"; <http://www.w3.org/ns/dcat#theme> ex:facility-category .
                """.formatted(includeDatasets ? """
                ex:dataset-one a <http://www.w3.org/ns/dcat#Dataset>; <http://purl.org/dc/terms/title> "Dataset One"; <http://purl.org/dc/terms/description> "Description"; <http://www.w3.org/ns/dcat#theme> ex:facility-category .
                ex:dataset-two a <http://www.w3.org/ns/dcat#Dataset>; <http://purl.org/dc/terms/title> "Dataset Two"; <http://purl.org/dc/terms/description> "Description"; <http://www.w3.org/ns/dcat#theme> ex:data-category .
                """ : "");
        return Import.ImportResourcesRequest.newBuilder().setTurtle(com.google.protobuf.ByteString.copyFrom(turtle, StandardCharsets.UTF_8))
                .setBaseUri("https://example.org/").setMappingId("epos-dcat-common").setMappingVersion(1).build();
    }

    private static Import.ImportResourcesRequest multiSchemeRequest() {
        var request = request(true);
        String turtle = request.getTurtle().toStringUtf8().replace(
                "ex:dataset-one a <http://www.w3.org/ns/dcat#Dataset>; <http://purl.org/dc/terms/title> \"Dataset One\"; <http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:facility-category .",
                "ex:dataset-one a <http://www.w3.org/ns/dcat#Dataset>; <http://purl.org/dc/terms/title> \"Dataset One\"; <http://purl.org/dc/terms/description> \"Description\"; <http://www.w3.org/ns/dcat#theme> ex:facility-category, ex:data-category .");
        return request.toBuilder().setTurtle(com.google.protobuf.ByteString.copyFrom(turtle, StandardCharsets.UTF_8)).build();
    }

    private static Import.ImportResourcesRequest hierarchicalCategoryRequest() {
        String turtle = """
                @prefix dcat: <http://www.w3.org/ns/dcat#> .
                @prefix dct: <http://purl.org/dc/terms/> .
                @prefix ex: <https://example.org/> .
                @prefix skos: <http://www.w3.org/2004/02/skos/core#> .
                ex:data a skos:ConceptScheme; skos:prefLabel "Data"; skos:hasTopConcept <https://www.epos-eu.org/epos-dcat-ap/category/facets/dataset-theme> .
                ex:root a skos:Concept; skos:prefLabel "Root"; skos:inScheme ex:data .
                ex:child a skos:Concept; skos:prefLabel "Child"; skos:inScheme ex:data; skos:broader ex:root .
                ex:unused a skos:Concept; skos:prefLabel "Unused"; skos:inScheme ex:data .
                ex:dataset a dcat:Dataset; dct:title "Dataset"; dct:description "Description"; dcat:theme ex:child .
                """;
        return Import.ImportResourcesRequest.newBuilder().setTurtle(com.google.protobuf.ByteString.copyFrom(turtle, StandardCharsets.UTF_8))
                .setBaseUri("https://example.org/").setMappingId("epos-dcat-common").setMappingVersion(1).build();
    }
}
