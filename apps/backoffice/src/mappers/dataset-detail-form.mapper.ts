import type {
  DatasetDraftMetadata,
  DatasetResourceMetadata,
  Distribution,
  DistributionDraft,
  DownloadableFileDistribution,
  DownloadableFileDistributionDraft,
  Parameter,
  ParameterDraft,
  ReplaceDatasetDraftRequest,
  ResourceVersionReference,
  ScalarParameter,
  ScalarParameterDraft,
  WebServiceDistribution,
  WebServiceDistributionDraft,
  WebServiceConfiguration,
  WebServiceConfigurationDraft,
} from '../../generated/backofficeSchemas';
import type {
  DatasetDetailForm,
  DatasetDistributionForm,
  DatasetEnumOptionForm,
  DatasetGeologicalTemporalForm,
  DatasetSpatialForm,
  DatasetTemporalForm,
  DatasetWebServiceConfigurationForm,
  DatasetWebServiceForm,
  DatasetWebServiceParameterForm,
  DatasetWebServiceProviderForm,
  DatasetWebServiceScalarParameterForm,
} from '../models/dataset-detail-form';

export function toDatasetDetailForm(metadata: DatasetResourceMetadata): DatasetDetailForm {
  const temporalCoverage = metadata.temporal_coverage;
  const geologicalTemporalCoverage = metadata.geological_temporal_coverage;

  return {
    title: metadata.title ?? '',
    description: metadata.description ?? '',
    keywords: [...metadata.keywords],
    accrualPeriodicity: metadata.accrual_periodicity ?? '',
    qualityAnnotation: metadata.quality_annotation ?? '',
    created: metadata.created ? toDateTimeLocalValue(metadata.created) : '',
    updated: metadata.updated ? toDateTimeLocalValue(metadata.updated) : '',
    spatial: metadata.spatial_coverage ?? null,
    temporalMode: geologicalTemporalCoverage ? 'geological' : 'calendar',
    temporal: {
      start: temporalCoverage?.start ? toDateTimeLocalValue(temporalCoverage.start) : '',
      end: temporalCoverage?.end ? toDateTimeLocalValue(temporalCoverage.end) : '',
    },
    geologicalTemporal: {
      startMya: geologicalTemporalCoverage?.start_mya == null ? '' : String(geologicalTemporalCoverage.start_mya),
      endMya: geologicalTemporalCoverage?.end_mya == null ? '' : String(geologicalTemporalCoverage.end_mya),
    },
    persistentIdentifiers: metadata.persistent_identifiers?.length
      ? metadata.persistent_identifiers.map((identifier) => ({
          id: identifier.id,
          resourceId: identifier.resource_id,
          type: identifier.type,
          value: identifier.value,
        }))
      : [{ type: '', value: '' }],
    contactPoints: metadata.contact_points?.length
      ? metadata.contact_points.map((contactPoint) => contactPoint)
      : [],
    providers: metadata.providers?.length
      ? metadata.providers.map((provider) => ({ id: provider.id, name: provider.name }))
      : [{ id: '', name: '' }],
    categories: metadata.categories?.length
      ? metadata.categories.map((category) => ({ id: category.id, name: category.name }))
      : [],
    distributions: metadata.distributions?.length
      ? metadata.distributions.map(buildDistributionForm)
      : [],
  };
}

export function toDatasetSaveRequest(
  form: DatasetDetailForm,
  revision: number,
  preservedMetadata: DatasetResourceMetadata,
): ReplaceDatasetDraftRequest {
  return {
    resource_type: 'dataset',
    revision,
    metadata: {
      title: form.title || 'Untitled dataset',
      description: form.description.trim(),
      keywords: [...form.keywords],
      accrual_periodicity: form.accrualPeriodicity || undefined,
      quality_annotation: undefinedIfBlank(form.qualityAnnotation),
      access_right: preservedMetadata.access_right ?? undefined,
      distributions: form.distributions.map(buildDistributionDraft),
      spatial_coverage: buildSpatialCoverageDraft(form.spatial),
      ...(form.temporalMode === 'geological'
        ? { geological_temporal_coverage: buildGeologicalTemporalCoverageDraft(form.geologicalTemporal) }
        : { temporal_coverage: buildTemporalCoverageDraft(form.temporal) }),
      persistent_identifiers: form.persistentIdentifiers
        .filter(
          (identifier) =>
            hasIdentifierValue(identifier.type) && hasIdentifierValue(identifier.value),
        )
        .map((identifier) => ({
          resource_id: identifier.resourceId,
          type: identifier.type,
          value: identifier.value,
        })),
      contact_points: form.contactPoints.filter(hasIdentifierValue),
      providers: form.providers
        .filter((provider) => hasIdentifierValue(provider.id))
        .map((provider) => provider.id),
      categories: form.categories
        .filter((category) => hasIdentifierValue(category.id))
        .map((category) => category.id),
      created: toIsoDateTimeOrNull(form.created),
      updated: toIsoDateTimeOrNull(form.updated),
    },
  };
}

export function hasMultipleParameterWithoutAllowedValues(form: DatasetDetailForm): boolean {
  return form.distributions.some(
    (distribution) =>
      distribution.kind === 'web-service' &&
      webServiceParameters(distribution.webService.configuration).some(
        (parameter) => 'multipleValues' in parameter && parameter.multipleValues && !parameter.allowedValues.some((option) => hasIdentifierValue(option.value)),
      ),
  );
}

export function hasInvalidExplodeParameter(form: DatasetDetailForm): boolean {
  return form.distributions.some(
    (distribution) =>
      distribution.kind === 'web-service' &&
      webServiceParameters(distribution.webService.configuration).some(
        (parameter) =>
          parameter.explode &&
          (parameter.location !== 'QUERY' || !('multipleValues' in parameter) || !parameter.multipleValues),
      ),
  );
}

export function hasIncompleteWebServiceBody(form: DatasetDetailForm): boolean {
  return form.distributions.some((distribution) => {
    if (distribution.kind !== 'web-service') return false;

    const configuration = distribution.webService.configuration;
    if (configuration.type !== 'HTTP' && configuration.type !== 'DYNAMIC') return false;

    const hasContentType = Boolean(configuration.bodyContentType.trim());
    const hasTemplate = Boolean(configuration.bodyTemplate.trim());
    const hasBody = hasContentType || hasTemplate || configuration.bodyParameters.length > 0;

    return hasBody && (!hasContentType || !hasTemplate);
  });
}

export function hasBodyOnUnsupportedMethod(form: DatasetDetailForm): boolean {
  return form.distributions.some((distribution) => {
    if (distribution.kind !== 'web-service') return false;

    const configuration = distribution.webService.configuration;
    if (configuration.type !== 'HTTP' && configuration.type !== 'DYNAMIC') return false;
    const hasBody = Boolean(
      configuration.bodyContentType.trim() || configuration.bodyTemplate.trim() || configuration.bodyParameters.length,
    );

    return hasBody && (configuration.method === 'GET' || configuration.method === 'DELETE');
  });
}

export function distributionValidationError(form: DatasetDetailForm): string | null {
  for (const [index, distribution] of form.distributions.entries()) {
    const label = `Distribution ${index + 1}`;

    if (!hasIdentifierValue(distribution.title)) {
      return `${label}: title is required.`;
    }
    if (!hasIdentifierValue(distribution.description)) {
      return `${label}: description is required.`;
    }

    if (distribution.kind === 'downloadable-file') {
      if (!hasIdentifierValue(distribution.downloadableFile.url)) {
        return `${label}: downloadable file URL is required.`;
      }
      if (!hasIdentifierValue(distribution.downloadableFile.format)) {
        return `${label}: downloadable file format is required.`;
      }
      continue;
    }

    const configurationError = webServiceConfigurationValidationError(
      distribution.webService.configuration,
    );
    if (configurationError) {
      return `${label}: ${configurationError}`;
    }

  }

  return null;
}

export function hasParameterDefaultOutsideAllowedValues(form: DatasetDetailForm): boolean {
  return form.distributions.some(
    (distribution) =>
      distribution.kind === 'web-service' &&
      webServiceAllParameters(distribution.webService.configuration).some(
        hasScalarDefaultOutsideAllowedValues,
      ),
  );
}

function webServiceParameters(
  configuration: DatasetWebServiceConfigurationForm,
): DatasetWebServiceParameterForm[] {
  switch (configuration.type) {
    case 'HTTP':
    case 'DYNAMIC':
    case 'WFS':
      return configuration.parameters;
    case 'WMS':
      return [...configuration.getMapParameters, ...configuration.getFeatureInfoParameters];
    case 'WMTS':
      return [...configuration.getTileParameters, ...configuration.getFeatureInfoParameters];
  }
}

function webServiceAllParameters(
  configuration: DatasetWebServiceConfigurationForm,
): Array<DatasetWebServiceParameterForm | DatasetWebServiceScalarParameterForm> {
  return configuration.type === 'HTTP' || configuration.type === 'DYNAMIC'
    ? [...configuration.parameters, ...configuration.bodyParameters]
    : webServiceParameters(configuration);
}

function webServiceConfigurationValidationError(
  configuration: DatasetWebServiceConfigurationForm,
): string | null {
  switch (configuration.type) {
    case 'HTTP':
    case 'DYNAMIC':
      if (!hasIdentifierValue(configuration.templateUrl)) {
        return 'web service template URL is required.';
      }
      if (!hasIdentifierValue(configuration.format)) return 'web service format is required.';
      return null;
    case 'WFS':
      if (!hasIdentifierValue(configuration.getFeatureTemplateUrl)) {
        return 'WFS GetFeature URL is required.';
      }
      if (!configuration.featureTypes.some(hasIdentifierValue)) {
        return 'at least one WFS feature type is required.';
      }
      if (!hasIdentifierValue(configuration.defaultFeatureType)) {
        return 'WFS default feature type is required.';
      }
      if (!configuration.featureTypes.includes(configuration.defaultFeatureType)) {
        return 'WFS default feature type must be one of the configured feature types.';
      }
      if (!configuration.outputFormats.some(hasIdentifierValue)) {
        return 'at least one WFS output format is required.';
      }
      if (!hasIdentifierValue(configuration.defaultOutputFormat)) {
        return 'WFS default output format is required.';
      }
      if (!configuration.outputFormats.includes(configuration.defaultOutputFormat)) {
        return 'WFS default output format must be one of the configured output formats.';
      }
      if (
        hasIdentifierValue(configuration.resultLimit) &&
        (!Number.isInteger(toNumberOrNull(configuration.resultLimit)) ||
          Number(configuration.resultLimit) < 1)
      ) {
        return 'WFS result limit must be a positive integer.';
      }
      return null;
    case 'WMS':
    case 'WMTS': {
      if (!hasIdentifierValue(configuration.capabilitiesUrl)) {
        return `${configuration.type} capabilities URL is required.`;
      }
      if (!configuration.layers.some((layer) => hasIdentifierValue(layer.name))) {
        return `at least one ${configuration.type} layer is required.`;
      }
      if (!hasIdentifierValue(configuration.defaultLayer)) {
        return `${configuration.type} default layer is required.`;
      }
      if (!configuration.layers.some((layer) => layer.name === configuration.defaultLayer)) {
        return `${configuration.type} default layer must be one of the configured layers.`;
      }
      const invalidStyle = configuration.layers.some(
        (layer) => hasIdentifierValue(layer.defaultStyle) && !layer.styles.includes(layer.defaultStyle),
      );
      if (invalidStyle) {
        return `${configuration.type} default styles must be included in their layer styles.`;
      }
      return null;
    }
  }
}

export function temporalWarningKind(form: DatasetDetailForm): 'none' | 'warning' | 'error' {
  if (form.temporalMode !== 'calendar') {
    return 'none';
  }

  const temporal = form.temporal;

  if (!temporal?.start) {
    return 'none';
  }

  if (!temporal.end) {
    return 'warning';
  }

  const start = new Date(temporal.start).getTime();
  const end = new Date(temporal.end).getTime();

  if (!Number.isFinite(start) || !Number.isFinite(end)) {
    return 'none';
  }

  return end <= start ? 'error' : 'none';
}

export function temporalWarningMessage(form: DatasetDetailForm): string {
  const warningKind = temporalWarningKind(form);

  if (warningKind === 'warning') {
    return 'If no end date is provided, it is assumed that this dataset is acquired continuously.';
  }

  if (warningKind === 'error') {
    return 'End date must be after Start Date';
  }

  return '';
}

export function geologicalTemporalValidationError(form: DatasetDetailForm): string | null {
  if (form.temporalMode !== 'geological') {
    return null;
  }

  const start = toNumberOrNull(form.geologicalTemporal.startMya);
  const end = toNumberOrNull(form.geologicalTemporal.endMya);
  if ((start !== null && start < 0) || (end !== null && end < 0)) {
    return 'Geological ages must be zero or greater.';
  }
  if (start !== null && end !== null && start < end) {
    return 'Geological start age must be older than or equal to the end age.';
  }
  return null;
}

function buildDistributionForm(distribution: Distribution): DatasetDistributionForm {
  const base = {
    id: distribution.id,
    resourceId: distribution.resource_id,
    nativeFormats: distribution.type === 'WEB_SERVICE'
      ? distribution.available_formats
          .filter((format) => format.kind === 'ORIGINAL')
          .map((format) => format.output_format)
      : [],
    title: distribution.title,
    description: distribution.description,
    license: distribution.license ?? '',
    byteSize: distribution.byte_size == null ? '' : String(distribution.byte_size),
    type: distribution.type,
  };

  if (distribution.type === 'WEB_SERVICE') {
    const webService = (distribution as WebServiceDistribution).web_service;

    return {
      ...base,
      kind: 'web-service',
      webService: {
        resourceId: webService.resource_id,
        visibleOn: webService.visible_on ?? [],
        documentationUri: webService.documentation_uri ?? '',
        provider: webService.provider
          ? buildWebServiceProviderForm(webService.provider)
          : { id: '', name: '' },
        contactPoints: webService.contact_points?.length
          ? webService.contact_points.map((contactPoint) => contactPoint)
          : [],
        configuration: buildWebServiceConfigurationForm(webService.configuration),
      },
    };
  }

  const downloadableFile = (distribution as DownloadableFileDistribution).downloadable_file;

  return {
    ...base,
    kind: 'downloadable-file',
    downloadableFile: {
      url: downloadableFile.url,
      format: downloadableFile.format,
    },
  };
}

function buildWebServiceProviderForm(
  provider: ResourceVersionReference,
): DatasetWebServiceProviderForm {
  return { id: provider.id, name: provider.name };
}

function buildWebServiceConfigurationForm(
  configuration: WebServiceConfiguration,
): DatasetWebServiceConfigurationForm {
  switch (configuration.type) {
    case 'HTTP':
    case 'DYNAMIC':
      return {
        type: configuration.type,
        templateUrl: configuration.template_url,
        method: configuration.method,
        format: configuration.format,
        bodyContentType: configuration.body_content_type ?? '',
        bodyTemplate: configuration.body_template ?? '',
        bodyRequired: configuration.body_required ?? true,
        bodyParameters:
          configuration.body_parameters?.map(buildWebServiceScalarParameterForm) ?? [],
        parameters: parameterForms(configuration.parameters),
      };
    case 'WFS':
      return {
        type: 'WFS',
        getFeatureTemplateUrl: configuration.get_feature_template_url,
        version: configuration.version,
        featureTypes: [...configuration.feature_types],
        defaultFeatureType: configuration.default_feature_type,
        outputFormats: [...configuration.output_formats],
        defaultOutputFormat: configuration.default_output_format,
        resultLimit: configuration.result_limit == null ? '' : String(configuration.result_limit),
        resultLimitEditable: configuration.result_limit_editable,
        parameters: parameterForms(configuration.parameters),
      };
    case 'WMS':
      return {
        type: 'WMS',
        capabilitiesUrl: configuration.capabilities_url,
        defaultLayer: configuration.default_layer,
        layers: configuration.layers.map((layer) => ({
          resourceId: layer.resource_id,
          name: layer.name,
          defaultStyle: layer.default_style,
          styles: [...layer.styles],
        })),
        transparencyEditable: configuration.transparency_editable,
        defaultTransparency: configuration.default_transparency,
        getMapParameters: parameterForms(configuration.get_map_parameters),
        getFeatureInfoParameters: parameterForms(configuration.get_feature_info_parameters),
      };
    case 'WMTS':
      return {
        type: 'WMTS',
        capabilitiesUrl: configuration.capabilities_url,
        defaultLayer: configuration.default_layer,
        layers: configuration.layers.map((layer) => ({
          resourceId: layer.resource_id,
          name: layer.name,
          defaultStyle: layer.default_style ?? '',
          styles: [...(layer.styles ?? [])],
        })),
        getTileParameters: parameterForms(configuration.get_tile_parameters),
        getFeatureInfoParameters: parameterForms(configuration.get_feature_info_parameters),
      };
  }
}

function parameterForms(parameters: Parameter[] | undefined): DatasetWebServiceParameterForm[] {
  return parameters?.length
    ? parameters.map(buildWebServiceParameterForm)
    : [buildEmptyWebServiceParameterForm()];
}

function buildWebServiceParameterForm(parameter: Parameter): DatasetWebServiceParameterForm {
  const common = {
    id: parameter.id,
    resourceId: parameter.resource_id,
    position: parameter.position,
    name: parameter.name,
    label: parameter.label,
    description: parameter.description ?? '',
    required: parameter.required,
    property: parameter.property ?? '',
    location: parameter.location,
    role: parameter.role ?? ('' as const),
    explode: parameter.explode,
    readOnly: parameter.read_only,
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        multipleValues: parameter.multiple_values,
        defaultValue: [...(parameter.default ?? [])],
        allowedValues: buildEnumOptionForms(parameter.enum),
        pattern: parameter.pattern ?? '',
      };
    case 'BOOLEAN':
      return { kind: 'BOOLEAN', ...common, defaultValue: parameter.default ?? null };
    case 'INTEGER':
      return {
        kind: 'INTEGER',
        ...common,
        multipleValues: parameter.multiple_values,
        minimum: parameter.minimum == null ? '' : String(parameter.minimum),
        maximum: parameter.maximum == null ? '' : String(parameter.maximum),
        defaultValue: (parameter.default ?? []).map(String),
        allowedValues: buildEnumOptionForms(parameter.enum),
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        ...common,
        multipleValues: parameter.multiple_values,
        minimum: parameter.minimum == null ? '' : String(parameter.minimum),
        maximum: parameter.maximum == null ? '' : String(parameter.maximum),
        defaultValue: (parameter.default ?? []).map(String),
        allowedValues: buildEnumOptionForms(parameter.enum),
      };
    case 'DATETIME':
      return {
        kind: 'DATETIME',
        ...common,
        minimum: parameter.minimum ?? '',
        maximum: parameter.maximum ?? '',
        defaultValue: parameter.default ?? '',
        layout: parameter.layout ?? '',
      };
    case 'TEMPLATE':
      return {
        kind: 'TEMPLATE',
        ...common,
        template: parameter.template,
        parameters: parameter.parameters.map(buildWebServiceScalarParameterForm),
      };
  }
}

function buildWebServiceScalarParameterForm(
  parameter: ScalarParameter,
): DatasetWebServiceScalarParameterForm {
  const common = {
    resourceId: parameter.resource_id,
    position: parameter.position,
    name: parameter.name,
    label: parameter.label,
    description: parameter.description ?? '',
    required: parameter.required,
    readOnly: parameter.read_only,
    role: parameter.role ?? ('' as const),
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        defaultValue: [...(parameter.default ?? [])],
        allowedValues: buildEnumOptionForms(parameter.enum),
        pattern: parameter.pattern ?? '',
      };
    case 'BOOLEAN':
      return { kind: 'BOOLEAN', ...common, defaultValue: parameter.default ?? null };
    case 'INTEGER':
    case 'FLOAT':
      return {
        kind: parameter.kind,
        ...common,
        minimum: parameter.minimum == null ? '' : String(parameter.minimum),
        maximum: parameter.maximum == null ? '' : String(parameter.maximum),
        defaultValue: (parameter.default ?? []).map(String),
        allowedValues: buildEnumOptionForms(parameter.enum),
      };
    case 'DATETIME':
      return {
        kind: 'DATETIME',
        ...common,
        minimum: parameter.minimum ?? '',
        maximum: parameter.maximum ?? '',
        defaultValue: parameter.default ?? '',
        layout: parameter.layout ?? '',
      };
  }
}

function buildEmptyWebServiceParameterForm(): DatasetWebServiceParameterForm {
  return {
    kind: 'STRING',
    id: '',
    position: 0,
    name: '',
    label: '',
    description: '',
    required: false,
    property: '',
    location: 'QUERY',
    role: '',
    explode: false,
    readOnly: false,
    multipleValues: false,
    defaultValue: [],
    allowedValues: [],
    pattern: '',
  };
}

function buildDistributionDraft(distribution: DatasetDistributionForm): DistributionDraft {
  const base = {
    resource_id: distribution.resourceId,
    title: distribution.title,
    description: undefinedIfBlank(distribution.description),
    license: undefinedIfBlank(distribution.license),
    byte_size: toNumberOrNull(distribution.byteSize),
  };

  if (distribution.kind === 'web-service') {
    return {
      ...base,
      type: 'WEB_SERVICE',
      web_service: buildWebServiceDraft(distribution.webService),
    } satisfies WebServiceDistributionDraft;
  }

  return {
    ...base,
    type: 'DOWNLOADABLE_FILE',
    downloadable_file: distribution.downloadableFile,
  } satisfies DownloadableFileDistributionDraft;
}

function buildWebServiceDraft(
  webService: DatasetWebServiceForm,
): WebServiceDistributionDraft['web_service'] {
  return {
    resource_id: webService.resourceId,
    visible_on: webService.visibleOn.length ? webService.visibleOn : undefined,
    provider: hasIdentifierValue(webService.provider.id) ? webService.provider.id : undefined,
    documentation_uri: undefinedIfBlank(webService.documentationUri),
    contact_points: webService.contactPoints.filter(hasIdentifierValue),
    configuration: buildWebServiceConfigurationDraft(webService.configuration),
  };
}

function buildWebServiceConfigurationDraft(
  configuration: DatasetWebServiceConfigurationForm,
): WebServiceConfigurationDraft {
  switch (configuration.type) {
    case 'HTTP':
    case 'DYNAMIC': {
      const hasBody = Boolean(
        configuration.bodyContentType.trim() ||
          configuration.bodyTemplate.trim() ||
          configuration.bodyParameters.length,
      );
      return {
        type: configuration.type,
        template_url: configuration.templateUrl,
        method: configuration.method,
        format: configuration.format,
        parameters: parameterDrafts(configuration.parameters),
        body_content_type: undefinedIfBlank(configuration.bodyContentType),
        body_template: undefinedIfBlank(configuration.bodyTemplate),
        body_required: hasBody ? configuration.bodyRequired : undefined,
        body_parameters: configuration.bodyParameters.map(buildWebServiceScalarParameterDraft),
      };
    }
    case 'WFS':
      return {
        type: 'WFS',
        get_feature_template_url: configuration.getFeatureTemplateUrl,
        version: configuration.version,
        feature_types: configuration.featureTypes.filter(hasIdentifierValue),
        default_feature_type: configuration.defaultFeatureType,
        output_formats: configuration.outputFormats.filter(hasIdentifierValue),
        default_output_format: configuration.defaultOutputFormat,
        result_limit: toNumberOrNull(configuration.resultLimit),
        result_limit_editable: configuration.resultLimitEditable,
        parameters: parameterDrafts(configuration.parameters),
      };
    case 'WMS':
      return {
        type: 'WMS',
        capabilities_url: configuration.capabilitiesUrl,
        default_layer: configuration.defaultLayer,
        layers: configuration.layers.map((layer) => {
          const styles = layer.styles.includes(layer.defaultStyle)
            ? layer.styles
            : [layer.defaultStyle, ...layer.styles];
          return {
            resource_id: layer.resourceId,
            name: layer.name,
            default_style: layer.defaultStyle,
            styles,
          };
        }),
        transparency_editable: configuration.transparencyEditable,
        default_transparency: configuration.defaultTransparency,
        get_map_parameters: parameterDrafts(configuration.getMapParameters),
        get_feature_info_parameters: parameterDrafts(configuration.getFeatureInfoParameters),
      };
    case 'WMTS':
      return {
        type: 'WMTS',
        capabilities_url: configuration.capabilitiesUrl,
        default_layer: configuration.defaultLayer,
        layers: configuration.layers.map((layer) => ({
          resource_id: layer.resourceId,
          name: layer.name,
          default_style: undefinedIfBlank(layer.defaultStyle),
          styles: layer.styles.length ? layer.styles : undefined,
        })),
        get_tile_parameters: parameterDrafts(configuration.getTileParameters),
        get_feature_info_parameters: parameterDrafts(configuration.getFeatureInfoParameters),
      };
  }
}

function parameterDrafts(parameters: DatasetWebServiceParameterForm[]): ParameterDraft[] {
  return parameters
    .filter((parameter) => !isEmptyWebServiceParameter(parameter))
    .map(buildWebServiceParameterDraft);
}

function buildWebServiceParameterDraft(
  parameter: DatasetWebServiceParameterForm,
  position: number,
): ParameterDraft {
  const common = {
    resource_id: parameter.resourceId,
    position,
    name: parameter.name,
    label: parameter.label,
    description: undefinedIfBlank(parameter.description),
    required: parameter.required,
    property: undefinedIfBlank(parameter.property),
    location: parameter.location,
    role: parameter.role || undefined,
    explode: parameter.explode,
    read_only: parameter.readOnly,
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        multiple_values: parameter.multipleValues,
        default: stringDefaults(parameter.defaultValue),
        enum: stringEnum(parameter.allowedValues),
        pattern: undefinedIfBlank(parameter.pattern),
      };
    case 'BOOLEAN':
      return { kind: 'BOOLEAN', ...common, default: parameter.defaultValue };
    case 'INTEGER':
      return {
        kind: 'INTEGER',
        ...common,
        multiple_values: parameter.multipleValues,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: numericDefaults(parameter.defaultValue, Number.isInteger),
        enum: numericEnum(parameter.allowedValues, Number.isInteger),
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        ...common,
        multiple_values: parameter.multipleValues,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: numericDefaults(parameter.defaultValue),
        enum: numericEnum(parameter.allowedValues),
      };
    case 'DATETIME':
      return {
        kind: 'DATETIME',
        ...common,
        minimum: toIsoDateTimeOrNull(parameter.minimum),
        maximum: toIsoDateTimeOrNull(parameter.maximum),
        default: toIsoDateTimeOrNull(parameter.defaultValue),
        layout: undefinedIfBlank(parameter.layout),
      };
    case 'TEMPLATE':
      return {
        kind: 'TEMPLATE',
        ...common,
        template: parameter.template,
        parameters: parameter.parameters.map(buildWebServiceScalarParameterDraft),
      };
  }
}

function buildWebServiceScalarParameterDraft(
  parameter: DatasetWebServiceScalarParameterForm,
  position: number,
): ScalarParameterDraft {
  const common = {
    resource_id: parameter.resourceId,
    position,
    name: parameter.name,
    label: parameter.label,
    description: undefinedIfBlank(parameter.description),
    required: parameter.required,
    read_only: parameter.readOnly,
    role: parameter.role || undefined,
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        default: stringDefaults(parameter.defaultValue),
        enum: stringEnum(parameter.allowedValues),
        pattern: undefinedIfBlank(parameter.pattern),
      };
    case 'BOOLEAN':
      return { kind: 'BOOLEAN', ...common, default: parameter.defaultValue };
    case 'INTEGER':
      return {
        kind: 'INTEGER',
        ...common,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: numericDefaults(parameter.defaultValue, Number.isInteger),
        enum: numericEnum(parameter.allowedValues, Number.isInteger),
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        ...common,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: numericDefaults(parameter.defaultValue),
        enum: numericEnum(parameter.allowedValues),
      };
    case 'DATETIME':
      return {
        kind: 'DATETIME',
        ...common,
        minimum: toIsoDateTimeOrNull(parameter.minimum),
        maximum: toIsoDateTimeOrNull(parameter.maximum),
        default: toIsoDateTimeOrNull(parameter.defaultValue),
        layout: undefinedIfBlank(parameter.layout),
      };
  }
}

function buildSpatialCoverageDraft(
  spatial: DatasetSpatialForm,
): DatasetDraftMetadata['spatial_coverage'] {
  return blankToNull(spatial);
}

function buildTemporalCoverageDraft(
  temporal: DatasetTemporalForm,
): DatasetDraftMetadata['temporal_coverage'] {
  return {
    start: toIsoDateTimeOrNull(temporal.start),
    end: toIsoDateTimeOrNull(temporal.end),
  };
}

function buildGeologicalTemporalCoverageDraft(
  temporal: DatasetGeologicalTemporalForm,
): DatasetDraftMetadata['geological_temporal_coverage'] {
  return {
    start_mya: toNumberOrNull(temporal.startMya),
    end_mya: toNumberOrNull(temporal.endMya),
  };
}

function isEmptyWebServiceParameter(parameter: DatasetWebServiceParameterForm): boolean {
  const isMetadataEmpty = [
    parameter.name,
    parameter.label,
    parameter.description,
    parameter.property,
  ].every((value) => !hasIdentifierValue(value));

  switch (parameter.kind) {
    case 'STRING':
      return (
        isMetadataEmpty &&
        !parameter.defaultValue.some(hasIdentifierValue) &&
        !parameter.allowedValues.length &&
        !hasIdentifierValue(parameter.pattern)
      );
    case 'BOOLEAN':
      return isMetadataEmpty && parameter.defaultValue === null;
    case 'INTEGER':
    case 'FLOAT':
      return (
        isMetadataEmpty &&
        !hasIdentifierValue(parameter.minimum) &&
        !hasIdentifierValue(parameter.maximum) &&
        !parameter.defaultValue.some(hasIdentifierValue) &&
        !parameter.allowedValues.length
      );
    case 'DATETIME':
      return (
        isMetadataEmpty &&
        !hasIdentifierValue(parameter.minimum) &&
        !hasIdentifierValue(parameter.maximum) &&
        !hasIdentifierValue(parameter.defaultValue) &&
        !hasIdentifierValue(parameter.layout)
      );
    case 'TEMPLATE':
      return (
        isMetadataEmpty && !hasIdentifierValue(parameter.template) && !parameter.parameters.length
      );
  }
}

function hasScalarDefaultOutsideAllowedValues(
  parameter: DatasetWebServiceParameterForm | DatasetWebServiceScalarParameterForm,
): boolean {
  if (parameter.kind === 'TEMPLATE') {
    return parameter.parameters.some(hasScalarDefaultOutsideAllowedValues);
  }

  if (!('allowedValues' in parameter) || !parameter.defaultValue.length) {
    return false;
  }

  const allowedValues = parameter.allowedValues.filter((option) => hasIdentifierValue(option.value));
  if (!allowedValues.length) return false;

  if (parameter.kind === 'STRING') {
    return parameter.defaultValue.some(
      (value) => hasIdentifierValue(value) && !allowedValues.some((option) => option.value === value)
    );
  }

  return parameter.defaultValue.some((value) => {
    const defaultValue = toNumberOrNull(value);
    return defaultValue !== null
      && !allowedValues.some((option) => toNumberOrNull(option.value) === defaultValue);
  });
}

function stringDefaults(values: string[]): string[] | null {
  const defaults = values.filter(hasIdentifierValue);
  return defaults.length ? defaults : null;
}

function numericDefaults(
  values: string[],
  predicate: (value: number) => boolean = Number.isFinite,
): number[] | null {
  const defaults = values
    .map(toNumberOrNull)
    .filter((value): value is number => value !== null && predicate(value));
  return defaults.length ? defaults : null;
}

function stringEnum(values: DatasetEnumOptionForm[]): Array<{ value: string; label?: string }> | undefined {
  const result = values
    .filter((option) => hasIdentifierValue(option.value))
    .map((option) => ({ value: option.value, label: undefinedIfBlank(option.label) }));
  return result.length ? result : undefined;
}

function numericEnum(
  values: DatasetEnumOptionForm[],
  predicate: (value: number) => boolean = Number.isFinite,
): Array<{ value: number; label?: string }> | undefined {
  const result = values
    .map((option) => ({ value: toNumberOrNull(option.value), label: undefinedIfBlank(option.label) }))
    .filter((option): option is { value: number; label: string | undefined } => option.value !== null && predicate(option.value));
  return result.length ? result : undefined;
}

function buildEnumOptionForms(
  options: Array<{ value: string | number; label?: string }> | undefined,
): DatasetEnumOptionForm[] {
  return (options ?? []).map((option) => ({
    value: String(option.value),
    label: option.label ?? '',
  }));
}

function toDateTimeLocalValue(value: string): string {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const pad = (input: number): string => String(input).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function toIsoDateTimeOrNull(value: string | null | undefined): string | null {
  if (!hasIdentifierValue(value)) {
    return null;
  }

  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}

function toNumberOrNull(value: string | number | null | undefined): number | null {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : null;
  }

  if (!hasIdentifierValue(value)) {
    return null;
  }

  const normalizedValue = value.trim().replace(',', '.');
  const numberValue = Number(normalizedValue);
  return Number.isFinite(numberValue) ? numberValue : null;
}

function blankToNull(value: string | null | undefined): string | null {
  if (value == null) {
    return null;
  }

  return hasIdentifierValue(value) ? value : null;
}

function undefinedIfBlank(value: string | null | undefined): string | undefined {
  if (value == null) {
    return undefined;
  }

  return hasIdentifierValue(value) ? value : undefined;
}

function hasIdentifierValue(value: string | number | null | undefined): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}
