package eu.epos.semanticaggregator.grpc;

import eu.epos.semanticaggregator.ingest.DatasetMapper;
import eu.epos.semanticaggregator.ingest.DatasetMapping;
import eu.epos.semanticaggregator.ingest.EntityDescription;
import eu.epos.semanticaggregator.ingest.EntityDescriptionMapper;
import eu.epos.semanticaggregator.identity.DatasetIdentityStore;
import eu.epos.semanticaggregator.identity.IdentityStore;
import eu.epos.semanticaggregator.mapping.MappingRegistry;
import eu.epos.semanticaggregator.mapping.MappingProfile;
import eu.epos.semanticaggregator.rdf.TurtleReader;
import eu.epos.semanticaggregator.upstream.DatasetRequestMapper;
import eu.epos.semanticaggregator.upstream.MetadataLifecycleClient;
import eu.epos.semanticaggregator.upstream.ReferenceBindings;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import java.net.URI;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.Set;
import metadata_lifecycle.v1.Resource;
import org.apache.jena.rdf.model.Model;
import org.apache.jena.rdf.model.ResourceFactory;
import org.apache.jena.riot.RiotException;
import semantic_aggregator.v1.DatasetImportServiceGrpc;
import semantic_aggregator.v1.Import;

public final class DatasetImportServer extends DatasetImportServiceGrpc.DatasetImportServiceImplBase {
    private static final int MAX_TURTLE_BYTES = 10 * 1024 * 1024;
    private final MappingRegistry mappings;
    private final MetadataLifecycleClient upstream;
    private final java.nio.file.Path identityStateDir;

    public DatasetImportServer(MappingRegistry mappings, MetadataLifecycleClient upstream, java.nio.file.Path identityStateDir) {
        this.mappings = mappings;
        this.upstream = upstream;
        this.identityStateDir = identityStateDir;
    }

    @Override
    public void importDatasets(Import.ImportDatasetsRequest request,
                               StreamObserver<Import.ImportDatasetsResponse> observer) {
        String actor = ActorInterceptor.ACTOR.get();
        if (actor == null || actor.isBlank()) {
            observer.onError(Status.UNAUTHENTICATED.withDescription("Actor metadata is required").asRuntimeException());
            return;
        }
        try {
            var prepared = prepare(request, actor);
            for (var reference : prepared.references()) {
                if (prepared.pending().stream().anyMatch(candidate -> candidate.placeholder().equals(reference.id()))
                        || prepared.categories().stream().anyMatch(candidate -> candidate.placeholder().equals(reference.id()))
                        || prepared.organizations().stream().anyMatch(candidate -> candidate.placeholder().equals(reference.id()))) continue;
                upstream.verifyReference(reference, request.getGroupId(), actor,
                        prepared.descriptions().getOrDefault(reference, List.of()));
            }
            try {
                for (var entry : prepared.bindings().usedIris().entrySet()) {
                    String id = entry.getValue();
                    if (prepared.pending().stream().anyMatch(candidate -> candidate.placeholder().equals(id))
                            || prepared.categories().stream().anyMatch(candidate -> candidate.placeholder().equals(id))
                            || prepared.organizations().stream().anyMatch(candidate -> candidate.placeholder().equals(id))) continue;
                    prepared.identity().remember(entry.getKey(), java.util.UUID.fromString(id));
                }
            } catch (java.io.IOException exception) {
                observer.onError(Status.UNAVAILABLE.withDescription("Unable to persist reference identities").asRuntimeException());
                return;
            }
            var response = Import.ImportDatasetsResponse.newBuilder();
            response.addAllContacts(prepared.reused());
            for (var organization : prepared.organizations()) {
                try {
                    String id = upstream.createOrganization(organization.description(), actor);
                    response.addOrganizations(Import.CreatedOrganization.newBuilder().setIri(organization.reference().iri()).setId(id));
                    try {
                        prepared.identity().remember(organization.reference(), java.util.UUID.fromString(id));
                        prepared.bindings().replacePlaceholder(organization.reference(), organization.placeholder(), id);
                    } catch (java.io.IOException | IllegalArgumentException exception) {
                        response.setFailure(failure(organization.reference().iri(), Status.Code.INTERNAL, false,
                                Import.ImportPhase.IMPORT_PHASE_ORGANIZATION_CREATION));
                        observer.onNext(response.build());
                        observer.onCompleted();
                        return;
                    }
                } catch (StatusRuntimeException exception) {
                    Status.Code code = exception.getStatus().getCode();
                    response.setFailure(failure(organization.reference().iri(), code, outcomeUnknown(code),
                            Import.ImportPhase.IMPORT_PHASE_ORGANIZATION_CREATION));
                    observer.onNext(response.build());
                    observer.onCompleted();
                    return;
                }
            }
            for (var contact : prepared.pending()) {
                try {
                    var resolved = createOrReuseContact(contact.key(), actor);
                    for (var reference : contact.references()) {
                        response.addContacts(Import.ResolvedContact.newBuilder().setIri(reference.iri())
                                .setId(resolved.id()).setCreated(resolved.created()));
                        try {
                            prepared.identity().remember(reference, java.util.UUID.fromString(resolved.id()));
                        } catch (java.io.IOException exception) {
                            response.setFailure(failure(reference.iri(), Status.Code.INTERNAL, false,
                                    Import.ImportPhase.IMPORT_PHASE_CONTACT_RESOLUTION));
                            observer.onNext(response.build());
                            observer.onCompleted();
                            return;
                        }
                        try {
                            prepared.bindings().replacePlaceholder(reference, contact.placeholder(), resolved.id());
                        } catch (IllegalArgumentException exception) {
                            response.setFailure(failure(reference.iri(), Status.Code.INTERNAL, false,
                                    Import.ImportPhase.IMPORT_PHASE_CONTACT_RESOLUTION));
                            observer.onNext(response.build());
                            observer.onCompleted();
                            return;
                        }
                    }
                } catch (StatusRuntimeException exception) {
                    Status.Code code = exception.getStatus().getCode();
                    response.setFailure(failure(contact.references().getFirst().iri(), code, outcomeUnknown(code),
                            Import.ImportPhase.IMPORT_PHASE_CONTACT_RESOLUTION));
                    observer.onNext(response.build());
                    observer.onCompleted();
                    return;
                }
            }
            for (var category : prepared.categories()) {
                try {
                    String id = upstream.createRootCategory(request.getGroupId(), category.name(), actor);
                    response.addCategories(Import.CreatedCategory.newBuilder().setIri(category.reference().iri()).setId(id));
                    try {
                        prepared.identity().remember(category.reference(), java.util.UUID.fromString(id));
                    } catch (java.io.IOException exception) {
                        response.setFailure(failure(category.reference().iri(), Status.Code.INTERNAL, false,
                                Import.ImportPhase.IMPORT_PHASE_CATEGORY_CREATION));
                        observer.onNext(response.build());
                        observer.onCompleted();
                        return;
                    }
                    try {
                        prepared.bindings().replacePlaceholder(category.reference(), category.placeholder(), id);
                    } catch (IllegalArgumentException exception) {
                        response.setFailure(failure(category.reference().iri(), Status.Code.INTERNAL, false,
                                Import.ImportPhase.IMPORT_PHASE_CATEGORY_CREATION));
                        observer.onNext(response.build());
                        observer.onCompleted();
                        return;
                    }
                } catch (StatusRuntimeException exception) {
                    Status.Code code = exception.getStatus().getCode();
                    response.setFailure(failure(category.reference().iri(), code, outcomeUnknown(code),
                            Import.ImportPhase.IMPORT_PHASE_CATEGORY_CREATION));
                    observer.onNext(response.build());
                    observer.onCompleted();
                    return;
                }
            }
            var converter = new DatasetRequestMapper();
            var groupLedger = new DatasetIdentityStore(identityStateDir, java.util.UUID.fromString(request.getGroupId()));
            try (var lease = groupLedger.acquire()) {
                for (var dataset : prepared.datasets()) {
                    try {
                        lease.requireNew(dataset.subject());
                    } catch (DatasetIdentityStore.ExistingDatasetException existing) {
                        if (existing.pending()) {
                            response.setFailure(failure(dataset.subject(), Status.Code.UNKNOWN, true,
                                    Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                        } else {
                            var existingIdentity = lease.existing(dataset.subject());
                            if (existingIdentity != null) {
                                if (existingIdentity.versionId() != null) {
                                    response.addImported(Import.ImportedDataset.newBuilder().setSubject(dataset.subject())
                                            .setResourceId(existingIdentity.resourceId())
                                            .setVersionId(existingIdentity.versionId())
                                            .setOutcome(Import.DatasetImportOutcome.DATASET_IMPORT_OUTCOME_REUSED));
                                } else {
                                    var resolved = upstream.getVersion(existingIdentity.resourceId(), existingIdentity.versionId(), actor);
                                    response.addImported(Import.ImportedDataset.newBuilder().setSubject(dataset.subject())
                                            .setResourceId(existingIdentity.resourceId())
                                            .setVersionId(resolved.getId())
                                            .setOutcome(Import.DatasetImportOutcome.DATASET_IMPORT_OUTCOME_REUSED));
                                }
                            } else {
                                response.setFailure(failure(dataset.subject(), Status.Code.ALREADY_EXISTS, false,
                                        Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                            }
                        }
                        break;
                    }
                    boolean intent = false;
                    try {
                        lease.begin(dataset.subject());
                        intent = true;
                    var input = converter.toRequest(dataset, request.getGroupId(), prepared.bindings());
                    Resource.CreateResourceResponse result = upstream.createResource(input, actor);
                    if (!result.hasResource() || result.getResource().getResourceId().isBlank()
                            || result.getResource().getVersionId().isBlank()) {
                        response.setFailure(failure(dataset.subject(), Status.Code.INTERNAL, true,
                                Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                        break;
                    }
                    java.util.UUID resultId;
                    java.util.UUID resultVersion;
                    try {
                        resultId = java.util.UUID.fromString(result.getResource().getResourceId());
                        resultVersion = java.util.UUID.fromString(result.getResource().getVersionId());
                        if (resultId.equals(new java.util.UUID(0, 0)) || resultVersion.equals(new java.util.UUID(0, 0))) {
                            throw new IllegalArgumentException("response identity is incomplete");
                        }
                    } catch (IllegalArgumentException exception) {

                        response.setFailure(failure(dataset.subject(), Status.Code.INTERNAL, true,
                                Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                        break;
                    }
                    lease.complete(dataset.subject(), resultId, resultVersion);

                        intent = false;
                        response.addImported(Import.ImportedDataset.newBuilder().setSubject(dataset.subject())
                                .setResourceId(result.getResource().getResourceId())
                                .setVersionId(result.getResource().getVersionId())
                                .setOutcome(Import.DatasetImportOutcome.DATASET_IMPORT_OUTCOME_CREATED));
                    } catch (java.io.IOException exception) {
                        boolean leavePending = exception instanceof DatasetIdentityStore.ExistingDatasetException
                                && ((DatasetIdentityStore.ExistingDatasetException) exception).pending();
                        if (intent && !leavePending) {
                            try {
                                lease.cancel(dataset.subject());
                                intent = false;
                            } catch (java.io.IOException ignored) {
                                // Preserve the uncertain state; an unresolved intent blocks automatic re-creation.
                            }
                        }
                        response.setFailure(failure(dataset.subject(), Status.Code.UNAVAILABLE, true,
                                Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                        break;
                    } catch (StatusRuntimeException exception) {
                        Status.Code code = exception.getStatus().getCode();
                        boolean certain = switch (code) {
                            case INVALID_ARGUMENT, UNAUTHENTICATED, PERMISSION_DENIED, NOT_FOUND,
                                    ALREADY_EXISTS, FAILED_PRECONDITION -> true;
                            default -> false;
                        };
                        if (intent && certain) {
                            try {
                                lease.cancel(dataset.subject());
                                intent = false;
                            } catch (java.io.IOException ignored) {
                                response.setFailure(failure(dataset.subject(), Status.Code.UNAVAILABLE, true,
                                        Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                                break;
                            }
                        }
                        response.setFailure(failure(dataset.subject(), code, !certain,
                                Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                        break;
                    } catch (IllegalArgumentException exception) {
                        if (intent) {
                            try {
                                lease.cancel(dataset.subject());
                                intent = false;
                            } catch (java.io.IOException ignored) {
                                response.setFailure(failure(dataset.subject(), Status.Code.UNAVAILABLE, true,
                                        Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                                break;
                            }
                        }
                        response.setFailure(failure(dataset.subject(), Status.Code.FAILED_PRECONDITION, false,
                                Import.ImportPhase.IMPORT_PHASE_DATASET_CREATION));
                        break;
                    }
                }
            }
            observer.onNext(response.build());
            observer.onCompleted();
        } catch (MetadataLifecycleClient.EntityMismatchException exception) {
            observer.onError(Status.FAILED_PRECONDITION.withDescription("Referenced entity does not match the RDF metadata")
                    .asRuntimeException());
        } catch (StatusRuntimeException exception) {
            String detail = switch (exception.getStatus().getCode()) {
                case NOT_FOUND -> "A referenced metadata entity was not found";
                case PERMISSION_DENIED -> "Not authorized to read a referenced metadata entity";
                case ALREADY_EXISTS -> "An entity with this name exists; provide an explicit binding";
                default -> "Unable to verify a referenced metadata entity";
            };
            observer.onError(Status.fromCode(exception.getStatus().getCode())
                    .withDescription(detail).asRuntimeException());
        } catch (DatasetIdentityStore.ImportBusyException exception) {
            observer.onError(Status.UNAVAILABLE.withDescription("Another dataset import is running for this group").asRuntimeException());
        } catch (java.io.IOException exception) {
            observer.onError(Status.UNAVAILABLE.withDescription("Unable to manage dataset import state").asRuntimeException());
        } catch (IllegalArgumentException | NoSuchElementException | RiotException exception) {
            observer.onError(Status.INVALID_ARGUMENT.withDescription(exception.getMessage()).asRuntimeException());
        }
    }

    private PreparedImport prepare(Import.ImportDatasetsRequest request, String actor) {
        if (request.getTurtle().isEmpty() || request.getTurtle().size() > MAX_TURTLE_BYTES) {
            throw new IllegalArgumentException("Turtle must contain between 1 byte and 10 MiB");
        }
        if (!URI.create(request.getBaseUri()).isAbsolute()) {
            throw new IllegalArgumentException("base_uri must be an absolute URI");
        }
        var groupId = java.util.UUID.fromString(request.getGroupId());
        if (!groupId.toString().equalsIgnoreCase(request.getGroupId()) || groupId.equals(new java.util.UUID(0, 0))) {
            throw new IllegalArgumentException("group_id must be a nonzero canonical UUID");
        }
        var profile = mappings.get(request.getMappingId(), request.getMappingVersion());
        var bindings = new ReferenceBindings();
        for (var binding : request.getReferencesList()) {
            DatasetMapping.ReferenceKind kind = switch (binding.getKind()) {
                case REFERENCE_KIND_ORGANIZATION -> DatasetMapping.ReferenceKind.ORGANIZATION;
                case REFERENCE_KIND_CONTACT_POINT -> DatasetMapping.ReferenceKind.CONTACT_POINT;
                case REFERENCE_KIND_CATEGORY -> DatasetMapping.ReferenceKind.CATEGORY;
                default -> throw new IllegalArgumentException("reference kind is required");
            };
            bindings.register(kind, binding.getIri(), binding.getId());
        }
        IdentityStore identity = new IdentityStore(identityStateDir, profile.id(), profile.version(), groupId);
        Model model = new TurtleReader().read(request.getTurtle().newInput(), request.getBaseUri());
        try {
            var datasets = new DatasetMapper(profile).map(model);
            if (datasets.isEmpty()) {
                throw new IllegalArgumentException("No datasets match the selected mapping");
            }
            var references = new HashSet<DatasetMapping.Reference>();
            for (var dataset : datasets) {
                if (!dataset.unsupportedPredicates().isEmpty()) {
                    throw new IllegalArgumentException("Unsupported RDF properties for " + dataset.subject() + ": "
                            + String.join(", ", dataset.unsupportedPredicates()));
                }
                references.addAll(dataset.providers());
                references.addAll(dataset.contactPoints());
                references.addAll(dataset.categories());
                for (var distribution : dataset.distributions()) {
                    var service = distribution.webService();
                    if (service != null) {
                        if (service.provider() != null) references.add(service.provider());
                        references.addAll(service.contactPoints());
                    }
                }
            }
            var entityMapper = new EntityDescriptionMapper(profile);
            Set<org.apache.jena.rdf.model.Resource> entitySubjects = new HashSet<>();
            Map<DatasetMapping.Reference, EntityDescription> byReference = new java.util.HashMap<>();
            for (var reference : references) {
                var result = entityMapper.map(model, reference);
                byReference.put(reference, result.description());
                entitySubjects.addAll(result.reachedSubjects());
            }
            rejectDisconnectedSubjects(model, profile, entitySubjects);
            applyIdentity(bindings, identity, references);
            validateMissingReferences(references, byReference, bindings);
            var organizationPlan = planOrganizations(references, byReference, bindings, actor);
            var contactPlan = planContacts(references, byReference, bindings, actor);
            var categoryPlan = planCategories(references, byReference, bindings, request.getGroupId(), actor);
            int converted = 0;
            var converter = new DatasetRequestMapper();
            for (var dataset : datasets) {
                converter.toRequest(dataset, request.getGroupId(), bindings);
                converted++;
            }
            var descriptions = new java.util.HashMap<ReferenceBindings.ResolvedReference, List<EntityDescription>>();
            for (var entry : bindings.usedIris().entrySet()) {
                var resolved = new ReferenceBindings.ResolvedReference(entry.getKey().kind(), entry.getValue());
                descriptions.computeIfAbsent(resolved, ignored -> new ArrayList<>()).add(byReference.get(entry.getKey()));
            }
            return new PreparedImport(List.copyOf(datasets), bindings, bindings.used(), Map.copyOf(descriptions),
                    contactPlan.pending(), contactPlan.reused(), categoryPlan, identity, organizationPlan, converted);
        } finally {
            model.close();
        }
    }

    private void applyIdentity(ReferenceBindings bindings, IdentityStore identity, Set<DatasetMapping.Reference> references) {
        try {
            var snapshot = identity.snapshot();
            for (var reference : references) {
                var id = snapshot.get(reference);
                if (id == null) continue;
                if (bindings.contains(reference)) {
                    if (!bindings.resolve(reference, false).equals(id.toString())) {
                        throw new IllegalArgumentException("explicit binding conflicts with a stored identity");
                    }
                } else {
                    bindings.register(reference.kind(), reference.iri(), id.toString());
                }
            }
        } catch (java.io.IOException exception) {
            throw Status.UNAVAILABLE.withDescription("Unable to read identity storage").withCause(exception).asRuntimeException();
        }
    }

    private void validateMissingReferences(Set<DatasetMapping.Reference> references,
                                           Map<DatasetMapping.Reference, EntityDescription> descriptions,
                                           ReferenceBindings bindings) {
        for (var reference : references) {
            if (bindings.contains(reference)) continue;
            var fields = descriptions.get(reference).fields();
            boolean resolvable = switch (reference.kind()) {
                case CONTACT_POINT -> fields.containsKey("email");
                case CATEGORY -> fields.containsKey("name");
                case ORGANIZATION -> fields.containsKey("name") || fields.containsKey("legal_name");
            };
            if (!resolvable) throw new IllegalArgumentException("missing reference binding: " + reference.kind() + " " + reference.iri());
        }
    }

    private List<PendingOrganization> planOrganizations(Set<DatasetMapping.Reference> references,
            Map<DatasetMapping.Reference, EntityDescription> descriptions, ReferenceBindings bindings, String actor) {
        var missing = references.stream().filter(reference -> reference.kind() == DatasetMapping.ReferenceKind.ORGANIZATION
                && !bindings.contains(reference)).sorted(java.util.Comparator.comparing(DatasetMapping.Reference::iri)).toList();
        if (missing.isEmpty()) return List.of();
        var existing = upstream.listOrganizationNames(actor);
        List<PendingOrganization> pending = new ArrayList<>();
        List<String> planned = new ArrayList<>();
        for (var reference : missing) {
            var description = descriptions.get(reference);
            var fields = description.fields();
            String name = fields.getOrDefault("legal_name", fields.get("name"));
            if (fields.containsKey("legal_name") && fields.containsKey("name")
                    && !fields.get("legal_name").equals(fields.get("name"))) {
                throw new IllegalArgumentException("organization name and legal name differ");
            }
            if (existing.stream().anyMatch(value -> value.equalsIgnoreCase(name))) {
                throw Status.ALREADY_EXISTS.withDescription("organization name already exists").asRuntimeException();
            }
            if (planned.stream().anyMatch(value -> value.equalsIgnoreCase(name))) {
                throw new IllegalArgumentException("multiple organization IRIs have the same name");
            }
            String placeholder;
            do {
                placeholder = java.util.UUID.randomUUID().toString();
            } while (bindings.containsId(placeholder));
            bindings.register(reference.kind(), reference.iri(), placeholder);
            pending.add(new PendingOrganization(reference, description, placeholder));
            planned.add(name);
        }
        return List.copyOf(pending);
    }

    private ContactPlan planContacts(Set<DatasetMapping.Reference> references,
                                         Map<DatasetMapping.Reference, EntityDescription> descriptions,
                                         ReferenceBindings bindings, String actor) {
        List<DatasetMapping.Reference> missing = references.stream()
                .filter(reference -> reference.kind() == DatasetMapping.ReferenceKind.CONTACT_POINT && !bindings.contains(reference))
                .sorted(java.util.Comparator.comparing(DatasetMapping.Reference::kind).thenComparing(DatasetMapping.Reference::iri))
                .toList();
        if (missing.isEmpty()) return new ContactPlan(List.of(), List.of());
        var contacts = upstream.listContactKeys(actor);
        var pending = new java.util.LinkedHashMap<MetadataLifecycleClient.ContactKey, PendingContact>();
        List<Import.ResolvedContact> reused = new ArrayList<>();
        for (var reference : missing) {
            var fields = descriptions.get(reference).fields();
            var key = new MetadataLifecycleClient.ContactKey(fields.get("email"));
            String id = contacts.get(key);
            if (id != null) {
                bindings.register(reference.kind(), reference.iri(), id);
                reused.add(Import.ResolvedContact.newBuilder().setIri(reference.iri()).setId(id).setCreated(false).build());
            } else {
                var candidate = pending.get(key);
                if (candidate == null) {
                    String placeholder;
                    do {
                        placeholder = java.util.UUID.randomUUID().toString();
                    } while (bindings.containsId(placeholder) || contacts.containsValue(placeholder));
                    candidate = new PendingContact(key, placeholder, new ArrayList<>());
                    pending.put(key, candidate);
                }
                candidate.references().add(reference);
                bindings.register(reference.kind(), reference.iri(), candidate.placeholder());
            }
        }
        return new ContactPlan(List.copyOf(pending.values()), List.copyOf(reused));
    }

    private List<PendingCategory> planCategories(Set<DatasetMapping.Reference> references,
                                                  Map<DatasetMapping.Reference, EntityDescription> descriptions,
                                                  ReferenceBindings bindings, String groupId, String actor) {
        List<DatasetMapping.Reference> missing = references.stream()
                .filter(reference -> reference.kind() == DatasetMapping.ReferenceKind.CATEGORY && !bindings.contains(reference))
                .sorted(java.util.Comparator.comparing(DatasetMapping.Reference::iri)).toList();
        if (missing.isEmpty()) return List.of();
        var existing = upstream.listRootCategoryNames(groupId, actor);
        List<PendingCategory> pending = new ArrayList<>();
        List<String> plannedNames = new ArrayList<>();
        for (var reference : missing) {
            String name = descriptions.get(reference).fields().get("name");
            if (existing.stream().anyMatch(value -> value.equalsIgnoreCase(name))) {
                throw Status.ALREADY_EXISTS.withDescription("category name already exists").asRuntimeException();
            }
            if (plannedNames.stream().anyMatch(value -> value.equalsIgnoreCase(name))) {
                throw new IllegalArgumentException("multiple category IRIs have the same root name");
            }
            String placeholder;
            do {
                placeholder = java.util.UUID.randomUUID().toString();
            } while (bindings.containsId(placeholder));
            bindings.register(reference.kind(), reference.iri(), placeholder);
            pending.add(new PendingCategory(reference, name, placeholder));
            plannedNames.add(name);
        }
        return List.copyOf(pending);
    }

    private ContactResult createOrReuseContact(MetadataLifecycleClient.ContactKey key, String actor) {
        try {
            return new ContactResult(upstream.createContact(key, actor), true);
        } catch (StatusRuntimeException exception) {
            if (exception.getStatus().getCode() != Status.Code.ALREADY_EXISTS
                    && exception.getStatus().getCode() != Status.Code.FAILED_PRECONDITION) throw exception;
            String existing = upstream.listContactKeys(actor).get(key);
            if (existing == null) throw exception;
            return new ContactResult(existing, false);
        }
    }

    private static void rejectDisconnectedSubjects(Model model, MappingProfile profile, Set<org.apache.jena.rdf.model.Resource> entitySubjects) {
        Set<org.apache.jena.rdf.model.Resource> reached = new HashSet<>();
        reached.addAll(entitySubjects);
        var roots = model.listResourcesWithProperty(org.apache.jena.vocabulary.RDF.type,
                ResourceFactory.createResource(profile.datasetType()));
        try {
            while (roots.hasNext()) {
                var root = roots.next();
                reached.add(root);
                for (List<String> path : profile.fields().values()) {
                    Set<org.apache.jena.rdf.model.Resource> current = Set.of(root);
                    for (int index = 0; index < path.size() - 1; index++) {
                        Set<org.apache.jena.rdf.model.Resource> next = new HashSet<>();
                        for (var parent : current) {
                            var links = parent.listProperties(model.createProperty(path.get(index)));
                            try {
                                while (links.hasNext()) {
                                    var node = links.next().getObject();
                                    if (node.isResource()) {
                                        next.add(node.asResource());
                                        reached.add(node.asResource());
                                    }
                                }
                            } finally {
                                links.close();
                            }
                        }
                        current = next;
                    }
                }
            }
        } finally {
            roots.close();
        }
        var subjects = model.listSubjects();
        try {
            while (subjects.hasNext()) {
                if (!reached.contains(subjects.next())) {
                    throw new IllegalArgumentException("Turtle contains subjects not connected to a dataset");
                }
            }
        } finally {
            subjects.close();
        }
    }

    private static Import.ImportFailure failure(String subject, Status.Code code, boolean uncertain, Import.ImportPhase phase) {
        return Import.ImportFailure.newBuilder().setSubject(subject).setCode(code.name()).setOutcomeUnknown(uncertain).setPhase(phase).build();
    }

    private static boolean outcomeUnknown(Status.Code code) {
        return switch (code) {
            case INVALID_ARGUMENT, UNAUTHENTICATED, PERMISSION_DENIED, NOT_FOUND, ALREADY_EXISTS, FAILED_PRECONDITION -> false;
            default -> true;
        };
    }

    private record ContactResult(String id, boolean created) {}
    private record PendingContact(MetadataLifecycleClient.ContactKey key, String placeholder, List<DatasetMapping.Reference> references) {}
    private record PendingCategory(DatasetMapping.Reference reference, String name, String placeholder) {}
    private record PendingOrganization(DatasetMapping.Reference reference, EntityDescription description, String placeholder) {}
    private record ContactPlan(List<PendingContact> pending, List<Import.ResolvedContact> reused) {}
    private record PreparedImport(List<DatasetMapping> datasets, ReferenceBindings bindings,
                                  List<ReferenceBindings.ResolvedReference> references,
                                  Map<ReferenceBindings.ResolvedReference, List<EntityDescription>> descriptions,
                                  List<PendingContact> pending, List<Import.ResolvedContact> reused,
                                  List<PendingCategory> categories, IdentityStore identity,
                                  List<PendingOrganization> organizations, int converted) {}
}
