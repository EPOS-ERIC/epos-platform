package eu.epos.semanticaggregator.upstream;

import eu.epos.semanticaggregator.ingest.SoftwareMapping;
import metadata_lifecycle.v1.Resource;

/** Maps validated software RDF extraction to Metadata Lifecycle resource inputs. */
public final class SoftwareRequestMapper {
    private static final java.util.regex.Pattern KEYWORD = java.util.regex.Pattern.compile("^[A-Za-z0-9 _:-]+$");

    public Resource.CreateResourceRequest toRequest(SoftwareMapping mapping, String groupId, ReferenceBindings bindings) {
        validateGroupId(groupId);
        return switch (mapping.kind()) {
            case APPLICATION -> Resource.CreateResourceRequest.newBuilder().setGroupId(groupId)
                    .setSoftwareApplication(application(mapping, bindings)).build();
            case SOURCE_CODE -> Resource.CreateResourceRequest.newBuilder().setGroupId(groupId)
                    .setSoftwareSourceCode(sourceCode(mapping, bindings)).build();
        };
    }

    private static Resource.SoftwareApplicationInput application(SoftwareMapping mapping, ReferenceBindings bindings) {
        var input = Resource.SoftwareApplicationInput.newBuilder().setName(mapping.name())
                .addAllKeywords(validatedKeywords(mapping.keywords())).addAllCreators(bindings.resolveAll(mapping.creators()))
                .addAllCategories(bindings.resolveAll(mapping.categories())).addAllContactPoints(bindings.resolveAll(mapping.contactPoints()));
        if (mapping.description() != null) input.setDescription(mapping.description());
        if (mapping.mainEntityOfPage() != null) input.setMainEntityOfPage(mapping.mainEntityOfPage());
        if (mapping.license() != null) input.setLicense(mapping.license());
        if (mapping.version() != null) input.setVersion(mapping.version());
        if (mapping.downloadUrl() != null) input.setDownloadUrl(mapping.downloadUrl());
        if (mapping.requirements() != null) input.setRequirements(mapping.requirements());
        if (mapping.operatingSystem() != null) input.setOperatingSystem(mapping.operatingSystem());
        mapping.identifiers().forEach(identifier -> input.addIdentifiers(Resource.VersionedIdentifierInput.newBuilder()
                .setType(identifier.type()).setValue(identifier.value())));
        return input.build();
    }

    private static Resource.SoftwareSourceCodeInput sourceCode(SoftwareMapping mapping, ReferenceBindings bindings) {
        var input = Resource.SoftwareSourceCodeInput.newBuilder().setName(mapping.name())
                .addAllKeywords(validatedKeywords(mapping.keywords())).addAllCreators(bindings.resolveAll(mapping.creators()))
                .addAllCategories(bindings.resolveAll(mapping.categories())).addAllContactPoints(bindings.resolveAll(mapping.contactPoints()));
        if (mapping.description() != null) input.setDescription(mapping.description());
        if (mapping.programmingLanguage() != null) input.setProgrammingLanguage(mapping.programmingLanguage());
        if (mapping.repository() != null) input.setRepository(mapping.repository());
        if (mapping.license() != null) input.setLicense(mapping.license());
        if (mapping.requirements() != null) input.setSoftwareRequirements(mapping.requirements());
        if (mapping.runtimePlatform() != null) input.setRuntimePlatform(mapping.runtimePlatform());
        if (mapping.mainEntityOfPage() != null) input.setMainEntityOfPage(mapping.mainEntityOfPage());
        if (mapping.version() != null) input.setVersion(mapping.version());
        mapping.identifiers().forEach(identifier -> input.addIdentifiers(Resource.VersionedIdentifierInput.newBuilder()
                .setType(identifier.type()).setValue(identifier.value())));
        return input.build();
    }

    private static void validateGroupId(String groupId) {
        var parsed = java.util.UUID.fromString(groupId);
        if (!parsed.toString().equalsIgnoreCase(groupId) || parsed.equals(new java.util.UUID(0, 0))) {
            throw new IllegalArgumentException("group_id must be a nonzero canonical UUID");
        }
    }

    private static java.util.List<String> validatedKeywords(java.util.List<String> keywords) {
        for (String keyword : keywords) {
            if (!KEYWORD.matcher(keyword).matches()) {
                throw new IllegalArgumentException("invalid keyword: " + keyword);
            }
        }
        return keywords;
    }
}
