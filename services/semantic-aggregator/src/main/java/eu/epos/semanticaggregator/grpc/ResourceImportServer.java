package eu.epos.semanticaggregator.grpc;

import eu.epos.semanticaggregator.ingest.DatasetMapper;
import eu.epos.semanticaggregator.ingest.DatasetMapping;
import eu.epos.semanticaggregator.ingest.EntityDescription;
import eu.epos.semanticaggregator.ingest.EntityDescriptionMapper;
import eu.epos.semanticaggregator.ingest.LegacySchemeMapper;
import eu.epos.semanticaggregator.ingest.SoftwareMapper;
import eu.epos.semanticaggregator.ingest.SoftwareMapping;
import eu.epos.semanticaggregator.identity.IdentityStore;
import eu.epos.semanticaggregator.identity.SchemeIdentityStore;
import eu.epos.semanticaggregator.mapping.MappingRegistry;
import eu.epos.semanticaggregator.rdf.TurtleReader;
import eu.epos.semanticaggregator.upstream.MetadataLifecycleClient;
import eu.epos.semanticaggregator.upstream.ReferenceBindings;
import eu.epos.semanticaggregator.upstream.DatasetRequestMapper;
import eu.epos.semanticaggregator.upstream.SoftwareRequestMapper;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import java.net.URI;
import java.util.HashSet;
import java.util.List;
import java.util.NoSuchElementException;
import java.util.Set;
import java.util.Map;
import metadata_lifecycle.v1.Resource;
import org.apache.jena.riot.RiotException;
import semantic_aggregator.v1.Import;
import semantic_aggregator.v1.ResourceImportServiceGrpc;

/** Imports profile-supported datasets and software resources. */
public final class ResourceImportServer extends ResourceImportServiceGrpc.ResourceImportServiceImplBase {
    private static final int MAX_TURTLE_BYTES = 10 * 1024 * 1024;
    private final MappingRegistry mappings;
    private final MetadataLifecycleClient upstream;
    private final java.nio.file.Path identityStateDir;

    public ResourceImportServer(MappingRegistry mappings, MetadataLifecycleClient upstream, java.nio.file.Path identityStateDir) {
        this.mappings = mappings;
        this.upstream = upstream;
        this.identityStateDir = identityStateDir;
    }

    @Override
    public void importResources(Import.ImportResourcesRequest request, StreamObserver<Import.ImportResourcesResponse> observer) {
        String actor = ActorInterceptor.ACTOR.get();
        if (actor == null || actor.isBlank()) {
            observer.onError(Status.UNAUTHENTICATED.withDescription("Actor metadata is required").asRuntimeException());
            return;
        }
        try {
            var prepared = prepare(request);
            var response = Import.ImportResourcesResponse.newBuilder();
            var references = new ReferenceResolver();
            for (var scheme : prepared.schemeByIri().values().stream()
                    .sorted(java.util.Comparator.comparing(LegacySchemeMapper.Scheme::iri)).toList()) {
                try {
                    ensureSchemeAndCategories(scheme, prepared, actor);
                } catch (StatusRuntimeException exception) {
                    Status.Code code = exception.getStatus().getCode();
                    response.addWarnings(failure(scheme.iri(), code, outcomeUnknown(code), exception.getStatus().getDescription()));
                } catch (java.io.IOException exception) {
                    response.addWarnings(failure(scheme.iri(), Status.Code.UNAVAILABLE, true,
                            "Unable to manage scheme identities"));
                }
            }
            for (DatasetMapping dataset : prepared.datasets()) {
                List<List<DatasetMapping.Reference>> categorySets;
                try {
                    categorySets = categoriesByScheme(dataset.categories(), prepared);
                } catch (IllegalArgumentException exception) {
                    response.addWarnings(failure(dataset.subject(), Status.Code.INVALID_ARGUMENT, false, exception.getMessage()));
                    continue;
                }
                for (var categories : categorySets) {
                    var copy = withCategories(dataset, categories);
                    try {
                        String groupId = resolveGroupAndCategories(copy.subject(), copy.categories(), prepared, actor);
                        resolveAndVerifyReferences(datasetReferences(copy), groupId, prepared, references, actor);
                        Resource.CreateResourceResponse created = upstream.createResource(
                                new DatasetRequestMapper().toRequest(copy, groupId, prepared.bindings()), actor);
                         if (!created.hasResource() || !validId(created.getResource().getResourceId())
                                 || !validId(created.getResource().getVersionId())) {
                            response.addWarnings(failure(copy.subject(), Status.Code.INTERNAL, true));
                            continue;
                        }
                        response.addImported(Import.ImportedResource.newBuilder().setSubject(copy.subject())
                                .setResourceId(created.getResource().getResourceId()).setVersionId(created.getResource().getVersionId())
                                .setKind(Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_DATASET));
                    } catch (StatusRuntimeException exception) {
                        Status.Code code = exception.getStatus().getCode();
                        response.addWarnings(failure(copy.subject(), code, outcomeUnknown(code), exception.getStatus().getDescription()));
                    } catch (java.io.IOException exception) {
                        response.addWarnings(failure(copy.subject(), Status.Code.UNAVAILABLE, true,
                                "Unable to manage scheme identities"));
                    } catch (IllegalArgumentException exception) {
                        response.addWarnings(failure(copy.subject(), Status.Code.INVALID_ARGUMENT, false, exception.getMessage()));
                    }
                }
            }
            for (SoftwareMapping software : prepared.software()) {
                List<List<DatasetMapping.Reference>> categorySets;
                try {
                    categorySets = categoriesByScheme(software.categories(), prepared);
                } catch (IllegalArgumentException exception) {
                    response.addWarnings(failure(software.subject(), Status.Code.INVALID_ARGUMENT, false, exception.getMessage()));
                    continue;
                }
                for (var categories : categorySets) {
                    var copy = withCategories(software, categories);
                    try {
                        String groupId = resolveGroupAndCategories(copy.subject(), copy.categories(), prepared, actor);
                        resolveAndVerifyReferences(softwareReferences(copy), groupId, prepared, references, actor);
                        Resource.CreateResourceResponse created = upstream.createResource(
                                new SoftwareRequestMapper().toRequest(copy, groupId, prepared.bindings()), actor);
                        if (!created.hasResource() || !validId(created.getResource().getResourceId())
                                || !validId(created.getResource().getVersionId())) {
                            response.addWarnings(failure(copy.subject(), Status.Code.INTERNAL, true));
                            continue;
                        }
                        response.addImported(Import.ImportedResource.newBuilder().setSubject(copy.subject())
                                .setResourceId(created.getResource().getResourceId()).setVersionId(created.getResource().getVersionId())
                                .setKind(copy.kind() == SoftwareMapping.Kind.APPLICATION
                                        ? Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_SOFTWARE_APPLICATION
                                        : Import.ImportedResourceKind.IMPORTED_RESOURCE_KIND_SOFTWARE_SOURCE_CODE));
                    } catch (StatusRuntimeException exception) {
                        Status.Code code = exception.getStatus().getCode();
                        response.addWarnings(failure(copy.subject(), code, outcomeUnknown(code), exception.getStatus().getDescription()));
                    } catch (java.io.IOException exception) {
                        response.addWarnings(failure(copy.subject(), Status.Code.UNAVAILABLE, true,
                                "Unable to manage scheme identities"));
                    } catch (IllegalArgumentException exception) {
                        response.addWarnings(failure(copy.subject(), Status.Code.INVALID_ARGUMENT, false, exception.getMessage()));
                    }
                }
            }
            observer.onNext(response.build());
            observer.onCompleted();
        } catch (StatusRuntimeException exception) {
            String detail = switch (exception.getStatus().getCode()) {
                case NOT_FOUND -> "A referenced metadata entity was not found";
                case PERMISSION_DENIED -> "Not authorized to read a referenced metadata entity";
                default -> "Unable to verify a referenced metadata entity";
            };
            observer.onError(Status.fromCode(exception.getStatus().getCode()).withDescription(detail).asRuntimeException());
        } catch (IllegalArgumentException | NoSuchElementException | RiotException exception) {
            observer.onError(Status.INVALID_ARGUMENT.withDescription(exception.getMessage()).asRuntimeException());
        }
    }

    private PreparedImport prepare(Import.ImportResourcesRequest request) {
        if (request.getTurtle().isEmpty() || request.getTurtle().size() > MAX_TURTLE_BYTES) {
            throw new IllegalArgumentException("Turtle must contain between 1 byte and 10 MiB");
        }
        if (!URI.create(request.getBaseUri()).isAbsolute()) throw new IllegalArgumentException("base_uri must be an absolute URI");
        var profile = mappings.get(request.getMappingId(), request.getMappingVersion());
        var bindings = new ReferenceBindings();
        for (var binding : request.getReferencesList()) {
            bindings.register(referenceKind(binding.getKind()), binding.getIri(), binding.getId());
        }
        var model = new TurtleReader().read(request.getTurtle().newInput(), request.getBaseUri());
        try {
            var datasets = new DatasetMapper(profile).map(model);
            var software = new SoftwareMapper(profile).map(model);
            if (datasets.isEmpty() && software.isEmpty()) throw new IllegalArgumentException("No resources match the selected mapping");
            Set<String> subjects = new HashSet<>();
            for (SoftwareMapping item : software) {
                if (!subjects.add(item.subject())) throw new IllegalArgumentException("Software subject has multiple supported types: " + item.subject());
            }
            var schemes = new LegacySchemeMapper(profile);
            Map<String, LegacySchemeMapper.Scheme> schemeByIri = new java.util.HashMap<>();
            for (var scheme : schemes.map(model)) schemeByIri.put(scheme.iri(), scheme);
            Map<String, LegacySchemeMapper.Category> categoryByIri = new java.util.HashMap<>();
            for (DatasetMapping item : datasets) for (var category : item.categories()) {
                var mapped = schemes.category(model, category.iri());
                if (mapped == null || !schemeByIri.containsKey(mapped.schemeIri())) continue;
                categoryByIri.put(mapped.iri(), mapped);
            }
            for (SoftwareMapping item : software) for (var category : item.categories()) {
                var mapped = schemes.category(model, category.iri());
                if (mapped == null || !schemeByIri.containsKey(mapped.schemeIri())) continue;
                categoryByIri.put(mapped.iri(), mapped);
            }
            for (var category : schemes.categories(model, schemeByIri.keySet())) categoryByIri.put(category.iri(), category);
            Map<DatasetMapping.Reference, EntityDescription> descriptions = new java.util.HashMap<>();
            var entityMapper = new EntityDescriptionMapper(profile);
            for (var reference : entityReferences(datasets, software)) {
                var description = entityMapper.map(model, reference, true).description();
                descriptions.put(reference, description);
                if (reference.kind() == DatasetMapping.ReferenceKind.CONTACT_POINT
                        && (!description.fields().containsKey("email") || !description.fields().containsKey("role"))) continue;
                if (!bindings.contains(reference)) validateEntityDescription(reference, description);
            }
            return new PreparedImport(datasets, software, bindings, profile.id(), profile.version(), schemeByIri, categoryByIri,
                    Map.copyOf(descriptions));
        } finally {
            model.close();
        }
    }

    private static DatasetMapping.ReferenceKind referenceKind(Import.ReferenceKind kind) {
        return switch (kind) {
            case REFERENCE_KIND_ORGANIZATION -> DatasetMapping.ReferenceKind.ORGANIZATION;
            case REFERENCE_KIND_CONTACT_POINT -> DatasetMapping.ReferenceKind.CONTACT_POINT;
            case REFERENCE_KIND_CATEGORY -> DatasetMapping.ReferenceKind.CATEGORY;
            default -> throw new IllegalArgumentException("reference kind is required");
        };
    }

    private static List<List<DatasetMapping.Reference>> categoriesByScheme(List<DatasetMapping.Reference> categories,
                                                                            PreparedImport prepared) {
        var byScheme = new java.util.TreeMap<String, List<DatasetMapping.Reference>>();
        for (var category : categories) {
            var mapped = prepared.categoryByIri().get(category.iri());
            if (mapped == null) throw new IllegalArgumentException("missing supported category scheme: " + category.iri());
            byScheme.computeIfAbsent(mapped.schemeIri(), ignored -> new java.util.ArrayList<>()).add(category);
        }
        return List.copyOf(byScheme.values());
    }

    private static DatasetMapping withCategories(DatasetMapping dataset, List<DatasetMapping.Reference> categories) {
        return new DatasetMapping(dataset.subject(), dataset.title(), dataset.description(), dataset.keywords(), dataset.spatialCoverage(),
                dataset.temporalCoverage(), dataset.accrualPeriodicity(), dataset.accessRight(), dataset.qualityAnnotation(), dataset.created(),
                dataset.updated(), dataset.identifiers(), dataset.providers(), dataset.contactPoints(), categories, dataset.distributions(),
                dataset.unsupportedPredicates());
    }

    private static SoftwareMapping withCategories(SoftwareMapping software, List<DatasetMapping.Reference> categories) {
        return new SoftwareMapping(software.subject(), software.kind(), software.name(), software.description(), software.keywords(),
                software.mainEntityOfPage(), software.license(), software.version(), software.downloadUrl(), software.requirements(),
                software.operatingSystem(), software.programmingLanguage(), software.repository(), software.runtimePlatform(),
                software.identifiers(), software.creators(), categories, software.contactPoints());
    }

    private void verifyReferences(java.util.List<DatasetMapping.Reference> references, String groupId,
                                   ReferenceBindings bindings, String actor) {
        for (var reference : references) {
            upstream.verifyReference(new ReferenceBindings.ResolvedReference(reference.kind(), bindings.resolve(reference)),
                    groupId, actor, java.util.List.of());
        }
    }

    private void resolveAndVerifyReferences(java.util.List<DatasetMapping.Reference> references, String groupId,
                                            PreparedImport prepared, ReferenceResolver resolver, String actor) throws java.io.IOException {
        if (references.isEmpty()) return;
        var identities = new IdentityStore(identityStateDir, prepared.mappingId(), prepared.mappingVersion(), java.util.UUID.fromString(groupId));
        var resolvedReferences = new java.util.ArrayList<DatasetMapping.Reference>();
        for (var reference : references) {
            var description = prepared.descriptions().get(reference);
            if (reference.kind() == DatasetMapping.ReferenceKind.CONTACT_POINT
                    && !prepared.bindings().contains(reference)
                    && (description == null || !description.fields().containsKey("email") || !description.fields().containsKey("role"))) {
                continue;
            }
            String id;
            if (prepared.bindings().contains(reference)) {
                id = prepared.bindings().resolve(reference, false);
            } else {
                if (description == null) throw new IllegalArgumentException("missing reference binding: " + reference.kind() + " " + reference.iri());
                id = resolver.resolve(reference, description, identities, actor);
                prepared.bindings().register(reference.kind(), reference.iri(), id);
            }
            identities.remember(reference, java.util.UUID.fromString(id));
            resolvedReferences.add(reference);
        }
        verifyReferences(resolvedReferences, groupId, prepared.bindings(), actor);
    }

    private static java.util.List<DatasetMapping.Reference> datasetReferences(DatasetMapping dataset) {
        var references = new java.util.ArrayList<DatasetMapping.Reference>();
        references.addAll(dataset.providers());
        references.addAll(dataset.contactPoints());
        for (var distribution : dataset.distributions()) {
            var service = distribution.webService();
            if (service == null) continue;
            if (service.provider() != null) references.add(service.provider());
            references.addAll(service.contactPoints());
        }
        return List.copyOf(references);
    }

    private static java.util.List<DatasetMapping.Reference> softwareReferences(SoftwareMapping software) {
        var references = new java.util.ArrayList<DatasetMapping.Reference>();
        references.addAll(software.creators());
        references.addAll(software.contactPoints());
        return List.copyOf(references);
    }

    private static Set<DatasetMapping.Reference> entityReferences(java.util.List<DatasetMapping> datasets,
                                                                    java.util.List<SoftwareMapping> software) {
        Set<DatasetMapping.Reference> references = new HashSet<>();
        for (var dataset : datasets) references.addAll(datasetReferences(dataset));
        for (var item : software) references.addAll(softwareReferences(item));
        return references;
    }

    private static void validateEntityDescription(DatasetMapping.Reference reference, EntityDescription description) {
        boolean valid = switch (reference.kind()) {
            case ORGANIZATION -> description.fields().containsKey("name") || description.fields().containsKey("legal_name");
            case CONTACT_POINT -> description.fields().containsKey("email");
            case CATEGORY -> false;
        };
        if (!valid) throw new IllegalArgumentException("missing reference binding: " + reference.kind() + " " + reference.iri());
    }

    private String resolveGroupAndCategories(String subject, java.util.List<DatasetMapping.Reference> categories,
                                             PreparedImport prepared, String actor) throws java.io.IOException {
        if (categories.isEmpty()) throw new IllegalArgumentException("resource must reference a category with skos:inScheme: " + subject);
        String schemeIri = null;
        for (var reference : categories) {
            var category = prepared.categoryByIri().get(reference.iri());
            if (schemeIri == null) schemeIri = category.schemeIri();
            else if (!schemeIri.equals(category.schemeIri())) throw new IllegalArgumentException("resource categories belong to multiple schemes: " + subject);
        }
        var scheme = prepared.schemeByIri().get(schemeIri);
        return ensureSchemeAndCategories(scheme, prepared, actor);
    }

    private String ensureSchemeAndCategories(LegacySchemeMapper.Scheme scheme, PreparedImport prepared, String actor)
            throws java.io.IOException {
        String schemeIri = scheme.iri();
        java.util.UUID groupId;
        try {
            groupId = new SchemeIdentityStore(identityStateDir, prepared.mappingId(), prepared.mappingVersion()).resolveOrCreate(schemeIri, () ->
                    java.util.UUID.fromString(upstream.createGroup(scheme, actor)));
        } catch (StatusRuntimeException exception) {
            throw contextualize(exception, "create group for scheme " + schemeIri);
        }
        var identities = new IdentityStore(identityStateDir, prepared.mappingId(), prepared.mappingVersion(), groupId);
        var visiting = new java.util.HashSet<String>();
        for (var category : prepared.categoryByIri().values()) {
            if (category.schemeIri().equals(schemeIri)) {
                ensureCategory(category, groupId, identities, prepared, actor, visiting);
            }
        }
        return groupId.toString();
    }

    private String ensureCategory(LegacySchemeMapper.Category category, java.util.UUID groupId, IdentityStore identities,
                                  PreparedImport prepared, String actor, Set<String> visiting) throws java.io.IOException {
        if (!visiting.add(category.iri())) {
            throw new IllegalArgumentException("category hierarchy contains a cycle at " + category.iri());
        }
        try {
            String parentId = null;
            if (category.parentIri() != null) {
                var parent = prepared.categoryByIri().get(category.parentIri());
                if (parent == null || !parent.schemeIri().equals(category.schemeIri())) {
                    throw new IllegalArgumentException("category parent must belong to the same supported scheme: " + category.iri());
                }
                parentId = ensureCategory(parent, groupId, identities, prepared, actor, visiting);
            }
            var reference = new DatasetMapping.Reference(DatasetMapping.ReferenceKind.CATEGORY, category.iri());
            if (prepared.bindings().contains(reference)) {
                String id = prepared.bindings().resolve(reference);
                verifyCategory(reference, id, groupId, actor);
                return id;
            }
            final String resolvedParentId = parentId;
            String id;
            try {
                id = identities.resolveOrCreate(reference, () -> java.util.UUID.fromString(createCategory(
                        groupId.toString(), category, resolvedParentId, actor))).toString();
            } catch (StatusRuntimeException exception) {
                throw contextualize(exception, "create category " + category.iri() + " in group " + groupId);
            }
            verifyCategory(reference, id, groupId, actor);
            prepared.bindings().register(reference.kind(), reference.iri(), id);
            return id;
        } finally {
            visiting.remove(category.iri());
        }
    }

    private void verifyCategory(DatasetMapping.Reference reference, String id, java.util.UUID groupId, String actor) {
        try {
            upstream.verifyReference(new ReferenceBindings.ResolvedReference(reference.kind(), id), groupId.toString(), actor, java.util.List.of());
        } catch (StatusRuntimeException exception) {
            throw contextualize(exception, "verify category " + reference.iri() + " in group " + groupId);
        }
    }

    private String createCategory(String groupId, LegacySchemeMapper.Category category, String parentId, String actor) {
        try {
            return upstream.createCategory(groupId, categoryName(category), parentId, actor);
        } catch (StatusRuntimeException exception) {
            if (exception.getStatus().getCode() != Status.Code.FAILED_PRECONDITION) throw exception;
            // Lifecycle requires sibling names to be unique, but legacy RDF can
            // define distinct concept IRIs with the same prefLabel.
            String suffix = java.util.UUID.nameUUIDFromBytes(category.iri()
                    .getBytes(java.nio.charset.StandardCharsets.UTF_8)).toString();
            return upstream.createCategory(groupId, categoryName(category) + " (" + suffix + ")", parentId, actor);
        }
    }

    private static String categoryName(LegacySchemeMapper.Category category) { return category.name(); }

    private static boolean validId(String value) {
        try {
            var id = java.util.UUID.fromString(value);
            return id.toString().equalsIgnoreCase(value) && !id.equals(new java.util.UUID(0, 0));
        } catch (IllegalArgumentException exception) {
            return false;
        }
    }

    private static StatusRuntimeException contextualize(StatusRuntimeException exception, String action) {
        String detail = exception.getStatus().getDescription();
        String message = detail == null || detail.isBlank() ? action : action + ": " + detail;
        return Status.fromCode(exception.getStatus().getCode()).withDescription(message).asRuntimeException();
    }

    private static Import.ImportFailure failure(String subject, Status.Code code, boolean unknown) {
        return failure(subject, code, unknown, null);
    }

    private static Import.ImportFailure failure(String subject, Status.Code code, boolean unknown, String message) {
        var result = Import.ImportFailure.newBuilder().setSubject(subject).setCode(code.name()).setOutcomeUnknown(unknown);
        if (message != null && !message.isBlank()) result.setMessage(message);
        return result.build();
    }

    private static boolean outcomeUnknown(Status.Code code) {
        return switch (code) {
            case INVALID_ARGUMENT, UNAUTHENTICATED, PERMISSION_DENIED, NOT_FOUND, ALREADY_EXISTS, FAILED_PRECONDITION -> false;
            default -> true;
        };
    }

    private final class ReferenceResolver {
        private Map<String, MetadataLifecycleClient.Organization> organizationsByName;
        private Map<String, String> organizationLeiIds;
        private Map<MetadataLifecycleClient.ContactKey, String> contactKeys;

        private String resolve(DatasetMapping.Reference reference, EntityDescription description, IdentityStore identities,
                               String actor) throws java.io.IOException {
            var existing = identities.snapshot().get(reference);
            if (existing != null) return existing.toString();
            String id = switch (reference.kind()) {
                case ORGANIZATION -> createOrganization(reference, description, actor);
                case CONTACT_POINT -> createOrReuseContact(description, actor);
                case CATEGORY -> throw new IllegalArgumentException("missing reference binding: " + reference.kind() + " " + reference.iri());
            };
            return identities.remember(reference, java.util.UUID.fromString(id)).resolved();
        }

        private String createOrganization(DatasetMapping.Reference reference, EntityDescription description, String actor) {
            if (organizationsByName == null) organizationsByName = new java.util.HashMap<>(upstream.listOrganizations(actor));
            if (organizationLeiIds == null) organizationLeiIds = upstream.listOrganizationLeiIds(actor);
            String lei = description.fields().get("lei_code");
            if (lei != null && organizationLeiIds.containsKey(lei)) return organizationLeiIds.get(lei);
            String name = description.fields().getOrDefault("legal_name", description.fields().get("name"));
            var existing = organizationsByName.get(name.toLowerCase(java.util.Locale.ROOT));
            if (existing != null && compatibleOrganization(description, existing)) return existing.id();
            if (existing != null) {
                throw Status.ALREADY_EXISTS.withDescription("organization name already exists; provide an explicit binding for "
                        + reference.iri()).asRuntimeException();
            }
            String id = upstream.createOrganization(description, actor);
            organizationsByName.put(name.toLowerCase(java.util.Locale.ROOT),
                    new MetadataLifecycleClient.Organization(id, canonicalOrganizationFields(description)));
            if (lei != null) organizationLeiIds.put(lei, id);
            return id;
        }

        private static boolean compatibleOrganization(EntityDescription description, MetadataLifecycleClient.Organization existing) {
            for (var entry : canonicalOrganizationFields(description).entrySet()) {
                if (!entry.getValue().equals(existing.fields().getOrDefault(entry.getKey(), ""))) return false;
            }
            return true;
        }

        private static Map<String, String> canonicalOrganizationFields(EntityDescription description) {
            var fields = new java.util.HashMap<>(description.fields());
            String name = fields.getOrDefault("legal_name", fields.get("name"));
            fields.put("name", name);
            fields.put("legal_name", name);
            return Map.copyOf(fields);
        }

        private String createOrReuseContact(EntityDescription description, String actor) {
            if (contactKeys == null) contactKeys = new java.util.HashMap<>(upstream.listContactKeys(actor));
            var key = new MetadataLifecycleClient.ContactKey(description.fields().get("email"), description.fields().get("role"));
            String existing = contactKeys.get(key);
            if (existing != null) return existing;
            try {
                String id = upstream.createContact(key, actor);
                contactKeys.put(key, id);
                return id;
            } catch (StatusRuntimeException exception) {
                if (exception.getStatus().getCode() != Status.Code.ALREADY_EXISTS
                        && exception.getStatus().getCode() != Status.Code.FAILED_PRECONDITION) throw exception;
                String id = upstream.listContactKeys(actor).get(key);
                if (id == null) throw exception;
                contactKeys.put(key, id);
                return id;
            }
        }
    }

    private record PreparedImport(java.util.List<DatasetMapping> datasets, java.util.List<SoftwareMapping> software,
                                  ReferenceBindings bindings, String mappingId, int mappingVersion,
                                  Map<String, LegacySchemeMapper.Scheme> schemeByIri,
                                  Map<String, LegacySchemeMapper.Category> categoryByIri,
                                  Map<DatasetMapping.Reference, EntityDescription> descriptions) {}
}
