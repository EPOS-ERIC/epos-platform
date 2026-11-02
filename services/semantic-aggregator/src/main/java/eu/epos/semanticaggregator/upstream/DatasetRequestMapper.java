package eu.epos.semanticaggregator.upstream;

import com.google.protobuf.Timestamp;
import eu.epos.semanticaggregator.ingest.DatasetMapping;
import java.time.Instant;
import java.util.UUID;
import metadata_lifecycle.v1.Common;
import metadata_lifecycle.v1.Resource;

/** Maps validated RDF extraction to the Metadata Lifecycle transport contract. */
public final class DatasetRequestMapper {
    public Resource.CreateResourceRequest toRequest(DatasetMapping mapping, String groupId) {
        return toRequest(mapping, groupId, new ReferenceBindings());
    }

    public Resource.CreateResourceRequest toRequest(DatasetMapping mapping, String groupId, ReferenceBindings bindings) {
        if (!mapping.unsupportedPredicates().isEmpty()) {
            throw new IllegalArgumentException("Unsupported RDF properties for " + mapping.subject() + ": "
                    + String.join(", ", mapping.unsupportedPredicates()));
        }
        UUID group = UUID.fromString(groupId);
        if (!group.toString().equalsIgnoreCase(groupId)) {
            throw new IllegalArgumentException("groupId must be a canonical UUID");
        }
        var dataset = Resource.DatasetInput.newBuilder()
                .setTitle(mapping.title())
                .setDescription(mapping.description())
                .addAllKeywords(mapping.keywords());
        dataset.addAllProviders(bindings.resolveAll(mapping.providers()));
        dataset.addAllContactPoints(bindings.resolveAll(mapping.contactPoints()));
        dataset.addAllCategories(bindings.resolveAll(mapping.categories()));
        if (mapping.spatialCoverage() != null) {
            dataset.setSpatialCoverage(mapping.spatialCoverage());
        }
        if (mapping.accrualPeriodicity() != null) {
            dataset.setAccrualPeriodicity(mapping.accrualPeriodicity());
        }
        if (mapping.accessRight() != null) {
            dataset.setAccessRight(mapping.accessRight());
        }
        if (mapping.qualityAnnotation() != null) {
            dataset.setQualityAnnotation(mapping.qualityAnnotation());
        }
        if (mapping.created() != null) {
            dataset.setCreated(timestamp(mapping.created()));
        }
        if (mapping.updated() != null) {
            dataset.setUpdated(timestamp(mapping.updated()));
        }
        if (mapping.temporalCoverage() != null) {
            var temporal = Resource.Temporal.newBuilder();
            if (mapping.temporalCoverage().start() != null) {
                temporal.setStart(timestamp(mapping.temporalCoverage().start()));
            }
            if (mapping.temporalCoverage().end() != null) {
                temporal.setEnd(timestamp(mapping.temporalCoverage().end()));
            }
            dataset.setTemporalCoverage(temporal);
        }
        for (var identifier : mapping.identifiers()) {
            dataset.addPersistentIdentifiers(Common.IdentifierInput.newBuilder()
                    .setType(identifier.type()).setValue(identifier.value()));
        }
        for (var distribution : mapping.distributions()) {
            var input = Resource.DistributionInput.newBuilder().setTitle(distribution.title())
                    .setDescription(distribution.description());
            if (distribution.webService() == null) {
                input.setType("DOWNLOADABLE_FILE").setDownloadableFile(Resource.DownloadableFile.newBuilder()
                        .setUrl(distribution.url()).setFormat(distribution.format()));
            } else {
                var service = distribution.webService();
                var generic = Resource.GenericWebServiceConfigurationInput.newBuilder()
                        .setTemplateUrl(service.templateUrl()).setFormat(service.format())
                        .setMethod(Resource.WebServiceMethod.valueOf("WEB_SERVICE_METHOD_" + service.method()));
                int position = 0;
                for (var parameter : service.parameters()) {
                    var inputParameter = Resource.ParameterInput.newBuilder().setPosition(position++)
                            .setName(parameter.name()).setLabel(parameter.label()).setRequired(parameter.required())
                            .setReadOnly(parameter.readOnly()).setLocation(parameter.location() == DatasetMapping.ParameterLocation.PATH
                                    ? Resource.WebServiceParameterLocation.WEB_SERVICE_PARAMETER_LOCATION_PATH
                                    : Resource.WebServiceParameterLocation.WEB_SERVICE_PARAMETER_LOCATION_QUERY);
                    switch (parameter.schema()) {
                        case DatasetMapping.StringSchema schema -> {
                            var value = Resource.StringParameter.newBuilder().setMultipleValues(schema.multipleValues());
                            if (schema.defaultValue() != null) value.setDefault(schema.defaultValue());
                            if (schema.pattern() != null) value.setPattern(schema.pattern());
                            for (String option : schema.allowedValues()) {
                                value.addEnum(Resource.StringEnumOption.newBuilder().setValue(option));
                            }
                            inputParameter.setString(value);
                        }
                        case DatasetMapping.BooleanSchema schema -> {
                            var value = Resource.BooleanParameter.newBuilder();
                            if (schema.defaultValue() != null) value.setDefault(schema.defaultValue());
                            inputParameter.setBoolean(value);
                        }
                        case DatasetMapping.IntegerSchema schema -> {
                            var value = Resource.IntegerParameter.newBuilder().setMultipleValues(schema.multipleValues());
                            if (schema.minimum() != null) value.setMinimum(schema.minimum());
                            if (schema.maximum() != null) value.setMaximum(schema.maximum());
                            if (schema.defaultValue() != null) value.setDefault(schema.defaultValue());
                            for (Long option : schema.allowedValues()) {
                                value.addEnum(Resource.IntegerEnumOption.newBuilder().setValue(option));
                            }
                            inputParameter.setInteger(value);
                        }
                        case DatasetMapping.FloatSchema schema -> {
                            var value = Resource.FloatParameter.newBuilder().setMultipleValues(schema.multipleValues());
                            if (schema.minimum() != null) value.setMinimum(schema.minimum());
                            if (schema.maximum() != null) value.setMaximum(schema.maximum());
                            if (schema.defaultValue() != null) value.setDefault(schema.defaultValue());
                            for (Double option : schema.allowedValues()) {
                                value.addEnum(Resource.FloatEnumOption.newBuilder().setValue(option));
                            }
                            inputParameter.setFloat(value);
                        }
                        case DatasetMapping.DateTimeSchema schema -> {
                            var value = Resource.DateTimeParameter.newBuilder();
                            if (schema.minimum() != null) value.setMinimum(timestamp(schema.minimum()));
                            if (schema.maximum() != null) value.setMaximum(timestamp(schema.maximum()));
                            if (schema.defaultValue() != null) value.setDefault(timestamp(schema.defaultValue()));
                            if (schema.layout() != null) value.setLayout(schema.layout());
                            inputParameter.setDateTime(value);
                        }
                    }
                    if (parameter.property() != null) {
                        inputParameter.setProperty(parameter.property());
                    }
                    if (parameter.role() != null) {
                        inputParameter.setRole(Resource.WebServiceParameterRole.valueOf("WEB_SERVICE_PARAMETER_ROLE_" + parameter.role().name()));
                    }
                    generic.addParameters(inputParameter);
                }
                var webService = Resource.WebServiceInput.newBuilder().setName(service.name())
                        .setType(Resource.WebServiceType.WEB_SERVICE_TYPE_HTTP).setGeneric(generic);
                if (service.description() != null) {
                    webService.setDescription(service.description());
                }
                if (service.provider() != null) webService.setProviderId(bindings.resolve(service.provider()));
                webService.addAllContactPoints(bindings.resolveAll(service.contactPoints()));
                if (service.spatialCoverage() != null) webService.setSpatialCoverage(service.spatialCoverage());
                if (service.temporalCoverage() != null) {
                    var temporal = Resource.Temporal.newBuilder();
                    if (service.temporalCoverage().start() != null) temporal.setStart(timestamp(service.temporalCoverage().start()));
                    if (service.temporalCoverage().end() != null) temporal.setEnd(timestamp(service.temporalCoverage().end()));
                    webService.setTemporalCoverage(temporal);
                }
                if (service.documentation() != null) {
                    var documentation = Resource.Documentation.newBuilder().setUri(service.documentation().uri());
                    if (service.documentation().title() != null) documentation.setTitle(service.documentation().title());
                    if (service.documentation().description() != null) documentation.setDescription(service.documentation().description());
                    webService.setDocumentation(documentation);
                }
                input.setType("WEB_SERVICE").setWebService(webService);
            }
            if (distribution.license() != null) {
                input.setLicense(distribution.license());
            }
            if (distribution.issued() != null) {
                input.setIssued(timestamp(distribution.issued()));
            }
            if (distribution.modified() != null) {
                input.setModified(timestamp(distribution.modified()));
            }
            if (distribution.byteSize() != null) {
                input.setByteSize(distribution.byteSize());
            }
            dataset.addDistributions(input);
        }
        return Resource.CreateResourceRequest.newBuilder().setGroupId(groupId).setDataset(dataset).build();
    }

    private static Timestamp timestamp(Instant value) {
        return Timestamp.newBuilder().setSeconds(value.getEpochSecond()).setNanos(value.getNano()).build();
    }
}
