package eu.epos.semanticaggregator.upstream;

import io.grpc.ManagedChannel;
import io.grpc.Metadata;
import io.grpc.Status;
import io.grpc.stub.MetadataUtils;
import eu.epos.semanticaggregator.ingest.EntityDescription;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.TimeUnit;
import metadata_lifecycle.v1.Resource;
import metadata_lifecycle.v1.ResourceServiceGrpc;
import metadata_lifecycle.v1.Common;
import metadata_lifecycle.v1.OrganizationOuterClass;
import metadata_lifecycle.v1.OrganizationServiceGrpc;
import metadata_lifecycle.v1.ContactPointOuterClass;
import metadata_lifecycle.v1.ContactPointServiceGrpc;
import metadata_lifecycle.v1.Category;
import metadata_lifecycle.v1.CategoryServiceGrpc;
import metadata_lifecycle.v1.Common;

/** Client for resource creation using the shared Metadata Lifecycle contract. */
public final class MetadataLifecycleClient implements AutoCloseable {
    private static final Metadata.Key<String> ACTOR_KEY =
            Metadata.Key.of("x-epos-user-id", Metadata.ASCII_STRING_MARSHALLER);

    private final ManagedChannel channel;

    public MetadataLifecycleClient(ManagedChannel channel) {
        this.channel = Objects.requireNonNull(channel, "channel");
    }

    /** Fetches the identity/version currently resolvable for a resolved dataset identity. */
    public Resource.ResourceVersion getVersion(String resourceId, String versionId, String actorId) {
        var request = Resource.GetResourceVersionRequest.newBuilder().setResourceType(Common.ResourceType.RESOURCE_TYPE_DATASET)
                .setResourceId(resourceId).setVersionId(versionId).build();
        return ResourceServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(10, TimeUnit.SECONDS).getResourceVersion(request).getResource();
    }

    public Resource.CreateResourceResponse createResource(Resource.CreateResourceRequest request, String actorId) {
        Objects.requireNonNull(request, "request");
        return ResourceServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(30, TimeUnit.SECONDS)
                .createResource(request);
    }

    /** Checks the canonical entity exists and is visible to the importing actor. */
    public void verifyReference(ReferenceBindings.ResolvedReference reference, String groupId, String actorId,
                                List<EntityDescription> descriptions) {
        var interceptor = MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId));
        switch (reference.kind()) {
            case ORGANIZATION -> {
                var response = OrganizationServiceGrpc.newBlockingStub(channel).withInterceptors(interceptor)
                        .withDeadlineAfter(5, TimeUnit.SECONDS)
                        .getOrganization(OrganizationOuterClass.GetOrganizationRequest.newBuilder().setOrganizationId(reference.id()).build());
                if (!response.hasOrganization() || !reference.id().equalsIgnoreCase(response.getOrganization().getId())) {
                    throw Status.INTERNAL.withDescription("invalid organization response").asRuntimeException();
                }
                verifyDescriptions(descriptions, organizationFields(response.getOrganization()));
            }
            case CONTACT_POINT -> {
                var response = ContactPointServiceGrpc.newBlockingStub(channel).withInterceptors(interceptor)
                        .withDeadlineAfter(5, TimeUnit.SECONDS)
                        .getContactPoint(ContactPointOuterClass.GetContactPointRequest.newBuilder().setContactPointId(reference.id()).build());
                if (!response.hasContactPoint() || !reference.id().equalsIgnoreCase(response.getContactPoint().getId())) {
                    throw Status.INTERNAL.withDescription("invalid contact response").asRuntimeException();
                }
                verifyDescriptions(descriptions, Map.of("email", response.getContactPoint().getEmail(),
                        "role", response.getContactPoint().getRole()));
            }
            case CATEGORY -> {
                var response = CategoryServiceGrpc.newBlockingStub(channel).withInterceptors(interceptor)
                        .withDeadlineAfter(5, TimeUnit.SECONDS)
                        .getCategory(Category.GetCategoryRequest.newBuilder().setGroupId(groupId).setCategoryId(reference.id()).build());
                if (!response.hasCategory() || !reference.id().equalsIgnoreCase(response.getCategory().getId())
                        || !groupId.equalsIgnoreCase(response.getCategory().getGroupId())) {
                    throw Status.INTERNAL.withDescription("invalid category response").asRuntimeException();
                }
                verifyDescriptions(descriptions, Map.of("name", response.getCategory().getName()));
            }
        }
    }

    /** Returns canonical contacts indexed by the upstream unique email/role key. */
    public Map<ContactKey, String> listContactKeys(String actorId) {
        var response = ContactPointServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .listContactPoints(ContactPointOuterClass.ListContactPointsRequest.getDefaultInstance());
        Map<ContactKey, String> contacts = new java.util.HashMap<>();
        for (var contact : response.getContactPointsList()) {
            try {
                var id = java.util.UUID.fromString(contact.getId());
                if (!id.toString().equalsIgnoreCase(contact.getId()) || id.equals(new java.util.UUID(0, 0))
                        || contact.getEmail().isBlank() || contact.getRole().isBlank()) {
                    throw new IllegalArgumentException("invalid contact point");
                }
                var key = new ContactKey(contact.getEmail(), contact.getRole());
                if (contacts.putIfAbsent(key, id.toString()) != null) {
                    throw new IllegalArgumentException("duplicate contact key");
                }
            } catch (IllegalArgumentException exception) {
                throw Status.INTERNAL.withDescription("invalid contact list response").asRuntimeException();
            }
        }
        return Map.copyOf(contacts);
    }

    public record ContactKey(String email, String role) {}

    /** Lists existing canonical organization names; callers must not infer RDF identity from a match. */
    public java.util.Set<String> listOrganizationNames(String actorId) {
        var response = OrganizationServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .listOrganizations(OrganizationOuterClass.ListOrganizationsRequest.getDefaultInstance());
        var names = new java.util.HashSet<String>();
        for (var organization : response.getOrganizationsList()) {
            if (organization.getName().isBlank() || !names.add(organization.getName())) {
                throw Status.INTERNAL.withDescription("invalid organization list response").asRuntimeException();
            }
        }
        return java.util.Set.copyOf(names);
    }

    /** Creates a canonical organization with the supported RDF attributes. */
    public String createOrganization(EntityDescription description, String actorId) {
        var fields = description.fields();
        String name = fields.getOrDefault("legal_name", fields.get("name"));
        var input = OrganizationOuterClass.CreateOrganizationRequest.newBuilder().setName(name);
        if (fields.containsKey("street")) input.setStreet(fields.get("street"));
        if (fields.containsKey("locality")) input.setLocality(fields.get("locality"));
        if (fields.containsKey("postal_code")) input.setPostalCode(fields.get("postal_code"));
        if (fields.containsKey("country")) input.setCountry(fields.get("country"));
        if (fields.containsKey("logo_url")) input.setLogoUrl(fields.get("logo_url"));
        if (fields.containsKey("page_url")) input.setPageUrl(fields.get("page_url"));
        if (fields.containsKey("email")) input.setEmail(fields.get("email"));
        if (fields.containsKey("lei_code")) input.setLeiCode(fields.get("lei_code"));
        var response = OrganizationServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(10, TimeUnit.SECONDS).createOrganization(input.build());
        if (!response.hasOrganization()) throw Status.INTERNAL.withDescription("invalid organization creation response").asRuntimeException();
        var organization = response.getOrganization();
        try {
            var id = java.util.UUID.fromString(organization.getId());
            if (!id.toString().equalsIgnoreCase(organization.getId()) || id.equals(new java.util.UUID(0, 0))) {
                throw new IllegalArgumentException("invalid organization id");
            }
            verifyDescriptions(List.of(description), organizationFields(organization));
            return id.toString();
        } catch (IllegalArgumentException | EntityMismatchException exception) {
            throw Status.INTERNAL.withDescription("invalid organization creation response").asRuntimeException();
        }
    }

    private static Map<String, String> organizationFields(OrganizationOuterClass.Organization organization) {
        String name = organization.getName();
        return Map.of("name", name, "legal_name", name,
                "street", organization.getStreet(), "locality", organization.getLocality(),
                "postal_code", organization.getPostalCode(), "country", organization.getCountry(),
                "logo_url", organization.getLogoUrl(), "page_url", organization.getPageUrl(),
                "email", organization.getEmail(), "lei_code", organization.getLeiCode());
    }

    /** Lists names of existing root categories in the destination group. */
    public java.util.Set<String> listRootCategoryNames(String groupId, String actorId) {
        var response = CategoryServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .listCategories(Category.ListCategoriesRequest.newBuilder().setGroupId(groupId).build());
        var names = new java.util.HashSet<String>();
        for (var category : response.getCategoriesList()) {
            if (!groupId.equalsIgnoreCase(category.getGroupId()) || category.getName().isBlank()) {
                throw Status.INTERNAL.withDescription("invalid category list response").asRuntimeException();
            }
            if (!category.hasParentCategoryId() && !names.add(category.getName())) {
                throw Status.INTERNAL.withDescription("duplicate root category name").asRuntimeException();
            }
        }
        return java.util.Set.copyOf(names);
    }

    /** Creates a root category; the upstream enforces group/name uniqueness. */
    public String createRootCategory(String groupId, String name, String actorId) {
        var response = CategoryServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .createCategory(Category.CreateCategoryRequest.newBuilder().setGroupId(groupId).setName(name).build());
        if (!response.hasCategory() || !groupId.equalsIgnoreCase(response.getCategory().getGroupId())
                || !name.equals(response.getCategory().getName()) || response.getCategory().hasParentCategoryId()) {
            throw Status.INTERNAL.withDescription("invalid category creation response").asRuntimeException();
        }
        String id = response.getCategory().getId();
        try {
            var parsed = java.util.UUID.fromString(id);
            if (!parsed.toString().equalsIgnoreCase(id) || parsed.equals(new java.util.UUID(0, 0))) {
                throw new IllegalArgumentException("invalid category id");
            }
            return parsed.toString();
        } catch (IllegalArgumentException exception) {
            throw Status.INTERNAL.withDescription("invalid category creation response").asRuntimeException();
        }
    }

    /** Creates a contact; the upstream enforces uniqueness of email and role. */
    public String createContact(ContactKey key, String actorId) {
        var response = ContactPointServiceGrpc.newBlockingStub(channel)
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(actorMetadata(actorId)))
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .createContactPoint(ContactPointOuterClass.CreateContactPointRequest.newBuilder()
                        .setEmail(key.email()).setRole(key.role()).build());
        if (!response.hasContactPoint() || !key.email().equals(response.getContactPoint().getEmail())
                || !key.role().equals(response.getContactPoint().getRole())) {
            throw Status.INTERNAL.withDescription("invalid contact creation response").asRuntimeException();
        }
        String id = response.getContactPoint().getId();
        try {
            var parsed = java.util.UUID.fromString(id);
            if (!parsed.toString().equalsIgnoreCase(id) || parsed.equals(new java.util.UUID(0, 0))) {
                throw new IllegalArgumentException("invalid contact id");
            }
            return parsed.toString();
        } catch (IllegalArgumentException exception) {
            throw Status.INTERNAL.withDescription("invalid contact creation response").asRuntimeException();
        }
    }

    private static void verifyDescriptions(List<EntityDescription> descriptions, Map<String, String> actual) {
        for (var description : descriptions) {
            for (var field : description.fields().entrySet()) {
                if (!field.getValue().equals(actual.get(field.getKey()))) {
                    throw new EntityMismatchException();
                }
            }
        }
    }

    public static final class EntityMismatchException extends RuntimeException {
        private static final long serialVersionUID = 1L;

        private EntityMismatchException() {
            super("RDF entity metadata conflicts with the existing entity");
        }
    }

    private static Metadata actorMetadata(String actorId) {
        if (actorId == null || actorId.isBlank()) throw new IllegalArgumentException("actorId is required");
        Metadata metadata = new Metadata();
        metadata.put(ACTOR_KEY, actorId);
        return metadata;
    }

    @Override
    public void close() {
        channel.shutdown();
    }
}
