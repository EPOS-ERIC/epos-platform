package eu.epos.semanticaggregator.upstream;

import com.google.protobuf.Timestamp;
import eu.epos.semanticaggregator.ingest.DatasetMapping;
import java.time.Instant;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.UUID;
import metadata_lifecycle.v1.Resource;

/** Maps validated RDF extraction to the Metadata Lifecycle transport contract. */
public final class DatasetRequestMapper {
    private static final Set<String> WFS_PARAMETERS = Set.of("service", "request", "version", "typename", "typenames",
            "outputformat", "count", "maxfeatures", "bbox", "srsname");
    private static final Set<String> WMS_PARAMETERS = Set.of("service", "version", "request", "layers", "styles", "srs",
            "crs", "bbox", "width", "height", "format", "transparent", "bgcolor", "exceptions", "query_layers",
            "info_format", "feature_count", "x", "y", "i", "j");
    private static final Set<String> WMTS_PARAMETERS = Set.of("service", "request", "version", "layer", "style", "format",
            "tilematrixset", "tilematrix", "tilerow", "tilecol", "infoformat", "i", "j");
    public Resource.CreateResourceRequest toRequest(DatasetMapping mapping, String groupId) {
        return toRequest(mapping, groupId, new ReferenceBindings());
    }

    public Resource.CreateResourceRequest toRequest(DatasetMapping mapping, String groupId, ReferenceBindings bindings) {
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
            dataset.setAccrualPeriodicity(enumValue(Resource.DatasetAccrualPeriodicity.class,
                    "DATASET_ACCRUAL_PERIODICITY_", mapping.accrualPeriodicity()));
        }
        if (mapping.accessRight() != null) {
            dataset.setAccessRight(enumValue(Resource.DatasetAccessRight.class, "DATASET_ACCESS_RIGHT_", mapping.accessRight()));
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
            dataset.addPersistentIdentifiers(Resource.VersionedIdentifierInput.newBuilder()
                    .setType(identifier.type()).setValue(identifier.value()));
        }
        for (var distribution : mapping.distributions()) {
            var input = Resource.DistributionInput.newBuilder().setTitle(distribution.title())
                    .setDescription(distribution.description());
            if (distribution.webService() == null) {
                input.setType(Resource.DistributionType.DISTRIBUTION_TYPE_DOWNLOADABLE_FILE).setDownloadableFile(Resource.DownloadableFile.newBuilder()
                        .setUrl(distribution.url()).setFormat(distribution.format()));
            } else {
                var service = distribution.webService();
                var webService = webService(service);
                if (service.provider() != null) webService.setProviderId(bindings.resolve(service.provider()));
                webService.addAllContactPoints(bindings.resolveAll(service.contactPoints()));
                if (service.documentation() != null) {
                    webService.setDocumentationUri(service.documentation().uri());
                }
                input.setType(Resource.DistributionType.DISTRIBUTION_TYPE_WEB_SERVICE).setWebService(webService);
            }
            if (distribution.license() != null) {
                input.setLicense(distribution.license());
            }
            if (distribution.byteSize() != null) {
                input.setByteSize(distribution.byteSize());
            }
            dataset.addDistributions(input);
        }
        return Resource.CreateResourceRequest.newBuilder().setGroupId(groupId).setDataset(dataset).build();
    }

    private static Resource.WebServiceInput.Builder webService(DatasetMapping.WebService service) {
        String protocol = parameterDefault(service.parameters(), "service");
        var result = Resource.WebServiceInput.newBuilder();
        switch (protocol == null ? "" : protocol.toUpperCase(Locale.ROOT)) {
            case "WMS" -> {
                var layer = requiredParameterDefault(service.parameters(), "layers");
                var layers = parameterValues(service.parameters(), "layers");
                var configuration = Resource.WMSWebServiceConfigurationInput.newBuilder()
                        .setCapabilitiesUrl(capabilitiesUrl(service.templateUrl(), "WMS")).setDefaultLayer(layer)
                        .addAllLayers(layers.stream().map(value -> Resource.WMSLayerInput.newBuilder()
                                .setName(value).setDefaultStyle("").addStyles("").build()).toList())
                        .setTransparencyEditable(false).setDefaultTransparency(false)
                        .addAllGetMapParameters(parametersExcept(service.parameters(), WMS_PARAMETERS));
                result.setType(Resource.WebServiceType.WEB_SERVICE_TYPE_WMS)
                        .addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP).setWms(configuration);
            }
            case "WMTS" -> {
                var layer = requiredParameterDefault(service.parameters(), "layer");
                var layers = parameterValues(service.parameters(), "layer");
                var configuration = Resource.WMTSWebServiceConfigurationInput.newBuilder()
                        .setCapabilitiesUrl(capabilitiesUrl(service.templateUrl(), "WMTS")).setDefaultLayer(layer)
                        .addAllLayers(layers.stream().map(value -> Resource.WMTSLayerInput.newBuilder().setName(value).build()).toList())
                        .addAllGetTileParameters(parametersExcept(service.parameters(), WMTS_PARAMETERS));
                result.setType(Resource.WebServiceType.WEB_SERVICE_TYPE_WMTS)
                        .addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP)
                        .addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE).setWmts(configuration);
            }
            case "WFS" -> {
                String version = requiredParameterDefault(service.parameters(), "version");
                if (version.equals("2.0.2")) version = "2.0.0";
                var featureTypes = parameterValues(service.parameters(), "typenames");
                if (featureTypes.isEmpty()) featureTypes = parameterValues(service.parameters(), "typename");
                var formats = parameterValues(service.parameters(), "outputformat");
                var configuration = Resource.WFSWebServiceConfigurationInput.newBuilder()
                        .setGetFeatureTemplateUrl(service.templateUrl()).setVersion(version)
                        .addAllFeatureTypes(featureTypes).setDefaultFeatureType(featureTypes.getFirst())
                        .addAllOutputFormats(formats).setDefaultOutputFormat(formats.getFirst())
                        .setResultLimitEditable(false).addAllParameters(parametersExcept(service.parameters(), WFS_PARAMETERS));
                String count = parameterDefault(service.parameters(), "count");
                if (count != null) configuration.setResultLimit(Long.parseLong(count));
                result.setType(Resource.WebServiceType.WEB_SERVICE_TYPE_WFS)
                        .addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP)
                        .addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE).setWfs(configuration);
            }
            default -> {
                var generic = Resource.GenericWebServiceConfigurationInput.newBuilder()
                        .setTemplateUrl(service.templateUrl()).setFormat(service.format())
                        .setMethod(Resource.WebServiceMethod.valueOf("WEB_SERVICE_METHOD_" + service.method()))
                        .addAllParameters(parameters(service.parameters()));
                result.setType(Resource.WebServiceType.WEB_SERVICE_TYPE_HTTP).setGeneric(generic);
                if (isGeoJsonFormat(service.format())) {
                    result.addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP)
                            .addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_TABLE);
                } else if (isCovJsonFormat(service.format())) {
                    result.addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_MAP)
                            .addVisibleOn(Resource.WebServiceVisibleOn.WEB_SERVICE_VISIBLE_ON_GRAPH);
                }
            }
        }
        return result;
    }

    private static boolean isGeoJsonFormat(String value) {
        String format = normalizedFormat(value);
        return format.contains("geojson") || format.contains("geo+json") || format.contains("geo-json");
    }

    private static boolean isCovJsonFormat(String value) {
        String format = normalizedFormat(value);
        return format.contains("covjson") || format.contains("cov+json") || format.contains("cov-json");
    }

    private static String normalizedFormat(String value) {
        return value.toLowerCase(Locale.ROOT).replace("%2b", "+");
    }

    private static String capabilitiesUrl(String endpoint, String service) {
        return endpoint + (endpoint.contains("?") ? "&" : "?") + "service=" + service + "&request=GetCapabilities";
    }

    private static String requiredParameterDefault(List<DatasetMapping.Parameter> parameters, String name) {
        String value = parameterDefault(parameters, name);
        if (value == null) throw new IllegalArgumentException("missing default value for OGC parameter " + name);
        return value;
    }

    private static String parameterDefault(List<DatasetMapping.Parameter> parameters, String name) {
        for (var parameter : parameters) {
            if (parameter.name().equalsIgnoreCase(name) && parameter.schema() instanceof DatasetMapping.StringSchema schema
                    && !schema.defaultValues().isEmpty()) return schema.defaultValues().getFirst();
        }
        return null;
    }

    private static List<String> parameterValues(List<DatasetMapping.Parameter> parameters, String name) {
        for (var parameter : parameters) {
            if (parameter.name().equalsIgnoreCase(name) && parameter.schema() instanceof DatasetMapping.StringSchema schema) {
                return schema.allowedValues().isEmpty() ? schema.defaultValues() : schema.allowedValues();
            }
        }
        throw new IllegalArgumentException("missing OGC parameter " + name);
    }

    private static List<Resource.ParameterInput> parametersExcept(List<DatasetMapping.Parameter> source, Set<String> reserved) {
        return parameters(source.stream().filter(parameter -> !reserved.contains(parameter.name().toLowerCase(Locale.ROOT))).toList());
    }

    private static List<Resource.ParameterInput> parameters(List<DatasetMapping.Parameter> source) {
        var result = new java.util.ArrayList<Resource.ParameterInput>();
        for (int position = 0; position < source.size(); position++) result.add(parameter(source.get(position), position));
        return result;
    }

    private static Resource.ParameterInput parameter(DatasetMapping.Parameter parameter, int position) {
        var input = Resource.ParameterInput.newBuilder().setPosition(position).setName(parameter.name()).setLabel(parameter.label())
                .setRequired(parameter.required()).setReadOnly(parameter.readOnly()).setLocation(parameter.location() == DatasetMapping.ParameterLocation.PATH
                        ? Resource.WebServiceParameterLocation.WEB_SERVICE_PARAMETER_LOCATION_PATH
                        : Resource.WebServiceParameterLocation.WEB_SERVICE_PARAMETER_LOCATION_QUERY);
        switch (parameter.schema()) {
            case DatasetMapping.StringSchema schema -> {
                var value = Resource.StringParameter.newBuilder().setMultipleValues(schema.multipleValues()).addAllDefault(schema.defaultValues());
                if (schema.pattern() != null) value.setPattern(schema.pattern());
                for (String option : schema.allowedValues()) value.addEnum(Resource.StringEnumOption.newBuilder().setValue(option));
                input.setString(value);
            }
            case DatasetMapping.BooleanSchema schema -> { var value = Resource.BooleanParameter.newBuilder(); if (schema.defaultValue() != null) value.setDefault(schema.defaultValue()); input.setBoolean(value); }
            case DatasetMapping.IntegerSchema schema -> { var value = Resource.IntegerParameter.newBuilder().setMultipleValues(schema.multipleValues()).addAllDefault(schema.defaultValues()); if (schema.minimum() != null) value.setMinimum(schema.minimum()); if (schema.maximum() != null) value.setMaximum(schema.maximum()); for (Long option : schema.allowedValues()) value.addEnum(Resource.IntegerEnumOption.newBuilder().setValue(option)); input.setInteger(value); }
            case DatasetMapping.FloatSchema schema -> { var value = Resource.FloatParameter.newBuilder().setMultipleValues(schema.multipleValues()).addAllDefault(schema.defaultValues()); if (schema.minimum() != null) value.setMinimum(schema.minimum()); if (schema.maximum() != null) value.setMaximum(schema.maximum()); for (Double option : schema.allowedValues()) value.addEnum(Resource.FloatEnumOption.newBuilder().setValue(option)); input.setFloat(value); }
            case DatasetMapping.DateTimeSchema schema -> { var value = Resource.DateTimeParameter.newBuilder(); if (schema.minimum() != null) value.setMinimum(timestamp(schema.minimum())); if (schema.maximum() != null) value.setMaximum(timestamp(schema.maximum())); if (schema.defaultValue() != null) value.setDefault(timestamp(schema.defaultValue())); if (schema.layout() != null) value.setLayout(schema.layout()); input.setDateTime(value); }
            case DatasetMapping.TemplateSchema schema -> input.setTemplate(Resource.TemplateParameterInput.newBuilder()
                    .setTemplate(schema.template()).addAllParameters(schema.parameters().stream()
                            .map(DatasetRequestMapper::scalarParameter).toList()));
        }
        if (parameter.property() != null) input.setProperty(parameter.property());
        if (parameter.role() != null) input.setRole(Resource.WebServiceParameterRole.valueOf("WEB_SERVICE_PARAMETER_ROLE_" + parameter.role().name()));
        return input.build();
    }

    private static Resource.ScalarParameterInput scalarParameter(DatasetMapping.Parameter parameter) {
        var input = Resource.ScalarParameterInput.newBuilder().setName(parameter.name()).setLabel(parameter.label())
                .setRequired(parameter.required()).setReadOnly(parameter.readOnly());
        switch (parameter.schema()) {
            case DatasetMapping.StringSchema schema -> {
                var value = Resource.ScalarStringParameter.newBuilder().addAllDefault(schema.defaultValues());
                if (schema.pattern() != null) value.setPattern(schema.pattern());
                for (String option : schema.allowedValues()) value.addEnum(Resource.StringEnumOption.newBuilder().setValue(option));
                input.setString(value);
            }
            case DatasetMapping.BooleanSchema schema -> { var value = Resource.ScalarBooleanParameter.newBuilder(); if (schema.defaultValue() != null) value.setDefault(schema.defaultValue()); input.setBoolean(value); }
            case DatasetMapping.IntegerSchema schema -> { var value = Resource.ScalarIntegerParameter.newBuilder(); if (schema.minimum() != null) value.setMinimum(schema.minimum()); if (schema.maximum() != null) value.setMaximum(schema.maximum()); value.addAllDefault(schema.defaultValues()); for (Long option : schema.allowedValues()) value.addEnum(Resource.IntegerEnumOption.newBuilder().setValue(option)); input.setInteger(value); }
            case DatasetMapping.FloatSchema schema -> { var value = Resource.ScalarFloatParameter.newBuilder(); if (schema.minimum() != null) value.setMinimum(schema.minimum()); if (schema.maximum() != null) value.setMaximum(schema.maximum()); value.addAllDefault(schema.defaultValues()); for (Double option : schema.allowedValues()) value.addEnum(Resource.FloatEnumOption.newBuilder().setValue(option)); input.setFloat(value); }
            case DatasetMapping.DateTimeSchema schema -> { var value = Resource.ScalarDateTimeParameter.newBuilder(); if (schema.minimum() != null) value.setMinimum(timestamp(schema.minimum())); if (schema.maximum() != null) value.setMaximum(timestamp(schema.maximum())); if (schema.defaultValue() != null) value.setDefault(timestamp(schema.defaultValue())); if (schema.layout() != null) value.setLayout(schema.layout()); input.setDateTime(value); }
            case DatasetMapping.TemplateSchema ignored -> throw new IllegalArgumentException("template parameters cannot be nested");
        }
        if (parameter.property() != null) input.setProperty(parameter.property());
        if (parameter.role() != null) input.setRole(Resource.WebServiceParameterRole.valueOf("WEB_SERVICE_PARAMETER_ROLE_" + parameter.role().name()));
        return input.build();
    }

    private static Timestamp timestamp(Instant value) {
        return Timestamp.newBuilder().setSeconds(value.getEpochSecond()).setNanos(value.getNano()).build();
    }

    private static <T extends Enum<T>> T enumValue(Class<T> type, String prefix, String value) {
        String normalized = value.substring(Math.max(value.lastIndexOf('/'), value.lastIndexOf('#')) + 1)
                .toUpperCase(java.util.Locale.ROOT).replaceAll("[^A-Z0-9]+", "_");
        try {
            return Enum.valueOf(type, prefix + normalized);
        } catch (IllegalArgumentException exception) {
            throw new IllegalArgumentException("unsupported " + prefix.toLowerCase(java.util.Locale.ROOT) + value, exception);
        }
    }
}
