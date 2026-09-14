import type {
  DatasetDraftMetadata,
  DatasetResourceMetadata,
  Distribution,
  DistributionDraft,
  Documentation,
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
} from '../../generated/backofficeSchemas';
import type {
  DatasetDetailForm,
  DatasetDistributionForm,
  DatasetSpatialForm,
  DatasetTemporalForm,
  DatasetWebServiceDocumentationForm,
  DatasetWebServiceForm,
  DatasetWebServiceParameterForm,
  DatasetWebServiceProviderForm,
  DatasetWebServiceScalarParameterForm,
} from '../models/dataset-detail-form';

export function toDatasetDetailForm(metadata: DatasetResourceMetadata): DatasetDetailForm {
  const temporalCoverage = metadata.temporal_coverage;

  return {
    title: metadata.title ?? '',
    description: metadata.description ?? '',
    keywords: metadata.keywords.join(', '),
    accrualPeriodicity: metadata.accrual_periodicity ?? '',
    qualityAnnotation: metadata.quality_annotation ?? '',
    created: metadata.created ? toDateTimeLocalValue(metadata.created) : '',
    updated: metadata.updated ? toDateTimeLocalValue(metadata.updated) : '',
    spatial: metadata.spatial_coverage ?? null,
    temporal: {
      start: temporalCoverage?.start ? toDateTimeLocalValue(temporalCoverage.start) : '',
      end: temporalCoverage?.end ? toDateTimeLocalValue(temporalCoverage.end) : '',
    },
    persistentIdentifiers: metadata.persistent_identifiers?.length
      ? metadata.persistent_identifiers.map((identifier) => ({
          id: identifier.id,
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
      description: undefinedIfBlank(form.description),
      keywords: parseKeywords(form.keywords),
      accrual_periodicity: undefinedIfBlank(form.accrualPeriodicity),
      quality_annotation: undefinedIfBlank(form.qualityAnnotation),
      access_right: preservedMetadata.access_right ?? undefined,
      distributions: form.distributions.map(buildDistributionDraft),
      spatial_coverage: buildSpatialCoverageDraft(form.spatial),
      temporal_coverage: buildTemporalCoverageDraft(form.temporal),
      persistent_identifiers: form.persistentIdentifiers
        .filter(
          (identifier) =>
            hasIdentifierValue(identifier.type) && hasIdentifierValue(identifier.value),
        )
        .map((identifier) => ({ type: identifier.type, value: identifier.value })),
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
      distribution.webService.parameters.some(
        (parameter) => 'multipleValues' in parameter && parameter.multipleValues && !parameter.allowedValues.some(hasIdentifierValue),
      ),
  );
}

export function hasInvalidExplodeParameter(form: DatasetDetailForm): boolean {
  return form.distributions.some(
    (distribution) =>
      distribution.kind === 'web-service' &&
      distribution.webService.parameters.some(
        (parameter) =>
          parameter.explode &&
          (parameter.location !== 'QUERY' || !('multipleValues' in parameter) || !parameter.multipleValues),
      ),
  );
}

export function hasIncompleteWebServiceBody(form: DatasetDetailForm): boolean {
  return form.distributions.some((distribution) => {
    if (distribution.kind !== 'web-service') return false;

    const webService = distribution.webService;
    const hasContentType = Boolean(webService.bodyContentType.trim());
    const hasTemplate = Boolean(webService.bodyTemplate.trim());
    const hasBody = hasContentType || hasTemplate || webService.bodyParameters.length > 0;

    return hasBody && (!hasContentType || !hasTemplate);
  });
}

export function hasBodyOnUnsupportedMethod(form: DatasetDetailForm): boolean {
  return form.distributions.some((distribution) => {
    if (distribution.kind !== 'web-service') return false;

    const webService = distribution.webService;
    const hasBody = Boolean(
      webService.bodyContentType.trim() || webService.bodyTemplate.trim() || webService.bodyParameters.length,
    );

    return hasBody && (webService.method === 'GET' || webService.method === 'DELETE');
  });
}

export function hasParameterDefaultOutsideAllowedValues(form: DatasetDetailForm): boolean {
  return form.distributions.some(
    (distribution) =>
      distribution.kind === 'web-service' &&
      [...distribution.webService.parameters, ...distribution.webService.bodyParameters].some(
        hasScalarDefaultOutsideAllowedValues,
      ),
  );
}

export function temporalWarningKind(form: DatasetDetailForm): 'none' | 'warning' | 'error' {
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

function buildDistributionForm(distribution: Distribution): DatasetDistributionForm {
  const base = {
    title: distribution.title,
    description: distribution.description,
    issued: distribution.issued ? toDateTimeLocalValue(distribution.issued) : '',
    modified: distribution.modified ? toDateTimeLocalValue(distribution.modified) : '',
    license: distribution.license ?? '',
    type: distribution.type,
  };

  if (
    distribution.type ===
    'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE'
  ) {
    const webService = (distribution as WebServiceDistribution).web_service;

    return {
      ...base,
      kind: 'web-service',
      webService: {
        name: webService.name,
        description: webService.description ?? '',
        templateUrl: webService.template_url,
        method: webService.method,
        type: webService.type,
        visibleOn: webService.visible_on ?? [],
        format: webService.format,
        bodyContentType: webService.body_content_type ?? '',
        bodyTemplate: webService.body_template ?? '',
        bodyParameters: webService.body_parameters?.map(buildWebServiceScalarParameterForm) ?? [],
        documentation: webService.documentation
          ? buildWebServiceDocumentationForm(webService.documentation)
          : { title: '', description: '', uri: '' },
        provider: webService.provider
          ? buildWebServiceProviderForm(webService.provider)
          : { id: '', name: '' },
        contactPoints: webService.contact_points?.length
          ? webService.contact_points.map((contactPoint) => contactPoint)
          : [],
        spatialCoverage: webService.spatial_coverage ?? null,
        temporalCoverage: buildTemporalForm(webService.temporal_coverage),
        parameters: webService.parameters?.length
          ? webService.parameters.map(buildWebServiceParameterForm)
          : [buildEmptyWebServiceParameterForm()],
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

function buildWebServiceDocumentationForm(
  documentation: Documentation,
): DatasetWebServiceDocumentationForm {
  return {
    title: documentation.title ?? '',
    description: documentation.description ?? '',
    uri: documentation.uri,
  };
}

function buildWebServiceProviderForm(
  provider: ResourceVersionReference,
): DatasetWebServiceProviderForm {
  return { id: provider.id, name: provider.name };
}

function buildTemporalForm(
  temporalCoverage: WebServiceDistribution['web_service']['temporal_coverage'],
): DatasetTemporalForm {
  return {
    start: temporalCoverage?.start ? toDateTimeLocalValue(temporalCoverage.start) : '',
    end: temporalCoverage?.end ? toDateTimeLocalValue(temporalCoverage.end) : '',
  };
}

function buildWebServiceParameterForm(parameter: Parameter): DatasetWebServiceParameterForm {
  const common = {
    id: parameter.id,
    position: parameter.position,
    name: parameter.name,
    label: parameter.label,
    description: parameter.description ?? '',
    required: parameter.required,
    property: parameter.property ?? '',
    location: parameter.location,
    explode: parameter.explode,
    readOnly: parameter.read_only,
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        multipleValues: parameter.multiple_values,
        defaultValue: parameter.default ?? null,
        allowedValues: parameter.enum ?? [],
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
        defaultValue: parameter.default == null ? '' : String(parameter.default),
        allowedValues: (parameter.enum ?? []).map(String),
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        ...common,
        multipleValues: parameter.multiple_values,
        minimum: parameter.minimum == null ? '' : String(parameter.minimum),
        maximum: parameter.maximum == null ? '' : String(parameter.maximum),
        defaultValue: parameter.default == null ? '' : String(parameter.default),
        allowedValues: (parameter.enum ?? []).map(String),
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
    position: parameter.position,
    name: parameter.name,
    label: parameter.label,
    description: parameter.description ?? '',
    required: parameter.required,
    readOnly: parameter.read_only,
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        defaultValue: parameter.default ?? null,
        allowedValues: parameter.enum ?? [],
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
        defaultValue: parameter.default == null ? '' : String(parameter.default),
        allowedValues: (parameter.enum ?? []).map(String),
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
    explode: false,
    readOnly: false,
    multipleValues: false,
    defaultValue: null,
    allowedValues: [],
    pattern: '',
  };
}

function buildDistributionDraft(distribution: DatasetDistributionForm): DistributionDraft {
  const base = {
    title: distribution.title,
    description: undefinedIfBlank(distribution.description),
    license: undefinedIfBlank(distribution.license),
    issued: toIsoDateTimeOrNull(distribution.issued),
    modified: toIsoDateTimeOrNull(distribution.modified),
  };

  if (distribution.kind === 'web-service') {
    return {
      ...base,
      type: 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE',
      web_service: buildWebServiceDraft(distribution.webService),
    } satisfies WebServiceDistributionDraft;
  }

  return {
    ...base,
    type: 'http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE',
    downloadable_file: distribution.downloadableFile,
  } satisfies DownloadableFileDistributionDraft;
}

function buildWebServiceDraft(
  webService: DatasetWebServiceForm,
): WebServiceDistributionDraft['web_service'] {
  return {
    name: webService.name,
    description: undefinedIfBlank(webService.description),
    template_url: webService.templateUrl,
    method: webService.method,
    type: webService.type,
    visible_on: webService.visibleOn.length ? webService.visibleOn : undefined,
    body_content_type: undefinedIfBlank(webService.bodyContentType),
    body_template: undefinedIfBlank(webService.bodyTemplate),
    body_parameters: webService.bodyParameters.map(buildWebServiceScalarParameterDraft),
    format: webService.format,
    provider: hasIdentifierValue(webService.provider.id) ? webService.provider.id : undefined,
    documentation: {
      title: undefinedIfBlank(webService.documentation.title),
      description: undefinedIfBlank(webService.documentation.description),
      uri: webService.documentation.uri,
    },
    spatial_coverage: buildSpatialCoverageDraft(webService.spatialCoverage),
    temporal_coverage: buildTemporalCoverageDraft(webService.temporalCoverage),
    contact_points: webService.contactPoints.filter(hasIdentifierValue),
    parameters: webService.parameters
      .filter((parameter) => !isEmptyWebServiceParameter(parameter))
      .map(buildWebServiceParameterDraft),
  };
}

function buildWebServiceParameterDraft(
  parameter: DatasetWebServiceParameterForm,
  position: number,
): ParameterDraft {
  const common = {
    position,
    name: parameter.name,
    label: parameter.label,
    description: undefinedIfBlank(parameter.description),
    required: parameter.required,
    property: undefinedIfBlank(parameter.property),
    location: parameter.location,
    explode: parameter.explode,
    read_only: parameter.readOnly,
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        multiple_values: parameter.multipleValues,
        default: blankToNull(parameter.defaultValue),
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
        default: toNumberOrNull(parameter.defaultValue),
        enum: numericEnum(parameter.allowedValues, Number.isInteger),
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        ...common,
        multiple_values: parameter.multipleValues,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: toNumberOrNull(parameter.defaultValue),
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
    position,
    name: parameter.name,
    label: parameter.label,
    description: undefinedIfBlank(parameter.description),
    required: parameter.required,
    read_only: parameter.readOnly,
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        ...common,
        default: blankToNull(parameter.defaultValue),
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
        default: toNumberOrNull(parameter.defaultValue),
        enum: numericEnum(parameter.allowedValues, Number.isInteger),
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        ...common,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: toNumberOrNull(parameter.defaultValue),
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
        !hasIdentifierValue(parameter.defaultValue ?? '') &&
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
        !hasIdentifierValue(parameter.defaultValue) &&
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

  if (!('allowedValues' in parameter) || parameter.defaultValue === null) {
    return false;
  }

  const allowedValues = parameter.allowedValues.filter(hasIdentifierValue);
  if (!allowedValues.length) return false;

  if (parameter.kind === 'STRING') {
    return (
      !!parameter.defaultValue.trim() && !allowedValues.includes(parameter.defaultValue)
    );
  }

  const defaultValue = toNumberOrNull(parameter.defaultValue);
  return (
    defaultValue !== null &&
    !allowedValues.some((value) => toNumberOrNull(value) === defaultValue)
  );
}

function stringEnum(values: string[]): string[] | undefined {
  const result = values.filter(hasIdentifierValue);
  return result.length ? result : undefined;
}

function numericEnum(
  values: string[],
  predicate: (value: number) => boolean = Number.isFinite,
): number[] | undefined {
  const result = values
    .map(toNumberOrNull)
    .filter((value): value is number => value !== null && predicate(value));
  return result.length ? result : undefined;
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

function parseKeywords(value: string): string[] {
  return value
    .split(',')
    .map((keyword) => keyword.trim())
    .filter(Boolean);
}

function hasIdentifierValue(value: string | number | null | undefined): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}
