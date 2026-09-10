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
  ParameterDraftMetadata,
  ReplaceDatasetDraftRequest,
  ResourceVersionReference,
  WebServiceDistribution,
  WebServiceDistributionDraft
} from '../../generated/backofficeSchemas';
import type {
  DatasetDetailForm,
  DatasetDistributionForm,
  DatasetSpatialForm,
  DatasetTemporalForm,
  DatasetWebServiceDocumentationForm,
  DatasetWebServiceForm,
  DatasetWebServiceParameterForm,
  DatasetWebServiceParameterMetadataForm,
  DatasetWebServiceProviderForm
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
      end: temporalCoverage?.end ? toDateTimeLocalValue(temporalCoverage.end) : ''
    },
    persistentIdentifiers: metadata.persistent_identifiers?.length
      ? metadata.persistent_identifiers.map((identifier) => ({
          id: identifier.id,
          type: identifier.type,
          value: identifier.value
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
      : []
  };
}

export function toDatasetSaveRequest(
  form: DatasetDetailForm,
  revision: number,
  preservedMetadata: DatasetResourceMetadata
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
        .filter((identifier) => hasIdentifierValue(identifier.type) && hasIdentifierValue(identifier.value))
        .map((identifier) => ({ type: identifier.type, value: identifier.value })),
      contact_points: form.contactPoints.filter(hasIdentifierValue),
      providers: form.providers.filter((provider) => hasIdentifierValue(provider.id)).map((provider) => provider.id),
      categories: form.categories.filter((category) => hasIdentifierValue(category.id)).map((category) => category.id),
      created: toIsoDateTimeOrNull(form.created),
      updated: toIsoDateTimeOrNull(form.updated)
    }
  };
}

export function hasRequiredParameterWithoutDefault(form: DatasetDetailForm): boolean {
  return form.distributions.some(
    (distribution) =>
      distribution.kind === 'web-service' &&
      distribution.webService.parameters.some(
        (parameter) =>
          parameter.metadata.required &&
          (parameter.defaultValue === null ||
            (typeof parameter.defaultValue === 'string' && !parameter.defaultValue.trim()))
      )
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
    type: distribution.type
  };

  if (distribution.type === 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE') {
    const webService = (distribution as WebServiceDistribution).web_service;

    return {
      ...base,
      kind: 'web-service',
      webService: {
        name: webService.name,
        description: webService.description,
        templateUrl: webService.template_url,
        format: webService.format,
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
          : [buildEmptyWebServiceParameterForm()]
      }
    };
  }

  const downloadableFile = (distribution as DownloadableFileDistribution).downloadable_file;

  return {
    ...base,
    kind: 'downloadable-file',
    downloadableFile: {
      url: downloadableFile.url,
      format: downloadableFile.format
    }
  };
}

function buildWebServiceDocumentationForm(
  documentation: Documentation
): DatasetWebServiceDocumentationForm {
  return {
    title: documentation.title ?? '',
    description: documentation.description ?? '',
    uri: documentation.uri
  };
}

function buildWebServiceProviderForm(
  provider: ResourceVersionReference
): DatasetWebServiceProviderForm {
  return { id: provider.id, name: provider.name };
}

function buildTemporalForm(
  temporalCoverage: WebServiceDistribution['web_service']['temporal_coverage']
): DatasetTemporalForm {
  return {
    start: temporalCoverage?.start ? toDateTimeLocalValue(temporalCoverage.start) : '',
    end: temporalCoverage?.end ? toDateTimeLocalValue(temporalCoverage.end) : ''
  };
}

function buildWebServiceParameterForm(parameter: Parameter): DatasetWebServiceParameterForm {
  const metadata: DatasetWebServiceParameterMetadataForm = {
    id: parameter.metadata.id,
    variable: parameter.metadata.variable,
    property: parameter.metadata.property,
    label: parameter.metadata.label,
    required: parameter.metadata.required,
    multiple_values: parameter.metadata.multiple_values,
    read_only: parameter.metadata.read_only
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        metadata,
        defaultValue: parameter.default,
        allowedValues: parameter.allowed_values,
        pattern: parameter.pattern
      };
    case 'BOOLEAN':
      return { kind: 'BOOLEAN', metadata, defaultValue: parameter.default };
    case 'INTEGER':
      return {
        kind: 'INTEGER',
        metadata,
        minimum: parameter.minimum == null ? '' : String(parameter.minimum),
        maximum: parameter.maximum == null ? '' : String(parameter.maximum),
        defaultValue: parameter.default == null ? '' : String(parameter.default),
        allowedValues: parameter.allowed_values.map((value) => String(value))
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        metadata,
        minimum: parameter.minimum == null ? '' : String(parameter.minimum),
        maximum: parameter.maximum == null ? '' : String(parameter.maximum),
        defaultValue: parameter.default == null ? '' : String(parameter.default),
        allowedValues: parameter.allowed_values.map((value) => String(value))
      };
    case 'DATETIME':
      return {
        kind: 'DATETIME',
        metadata,
        minimum: parameter.minimum ?? '',
        maximum: parameter.maximum ?? '',
        defaultValue: parameter.default ?? '',
        pattern: parameter.pattern
      };
  }
}

function buildEmptyWebServiceParameterForm(): DatasetWebServiceParameterForm {
  return {
    kind: 'STRING',
    metadata: {
      id: '',
      variable: '',
      property: '',
      label: '',
      required: false,
      multiple_values: false,
      read_only: false
    },
    defaultValue: null,
    allowedValues: [],
    pattern: ''
  };
}

function buildDistributionDraft(distribution: DatasetDistributionForm): DistributionDraft {
  const base = {
    title: distribution.title,
    description: distribution.description,
    license: undefinedIfBlank(distribution.license),
    issued: toIsoDateTimeOrNull(distribution.issued),
    modified: toIsoDateTimeOrNull(distribution.modified)
  };

  if (distribution.kind === 'web-service') {
    return {
      ...base,
      type: 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE',
      web_service: buildWebServiceDraft(distribution.webService)
    } satisfies WebServiceDistributionDraft;
  }

  return {
    ...base,
    type: 'http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE',
    downloadable_file: distribution.downloadableFile
  } satisfies DownloadableFileDistributionDraft;
}

function buildWebServiceDraft(
  webService: DatasetWebServiceForm
): WebServiceDistributionDraft['web_service'] {
  return {
    name: webService.name,
    description: webService.description,
    template_url: webService.templateUrl,
    format: webService.format,
    provider: hasIdentifierValue(webService.provider.id) ? webService.provider.id : undefined,
    documentation: webService.documentation,
    spatial_coverage: buildSpatialCoverageDraft(webService.spatialCoverage),
    temporal_coverage: buildTemporalCoverageDraft(webService.temporalCoverage),
    contact_points: webService.contactPoints.filter(hasIdentifierValue),
    parameters: webService.parameters
      .filter((parameter) => !isEmptyWebServiceParameter(parameter))
      .map(buildWebServiceParameterDraft)
  };
}

function buildWebServiceParameterDraft(parameter: DatasetWebServiceParameterForm): ParameterDraft {
  const metadata: ParameterDraftMetadata = {
    kind: parameter.kind,
    variable: parameter.metadata.variable,
    property: parameter.metadata.property,
    label: parameter.metadata.label,
    required: parameter.metadata.required,
    multiple_values: parameter.metadata.multiple_values,
    read_only: parameter.metadata.read_only
  };

  switch (parameter.kind) {
    case 'STRING':
      return {
        kind: 'STRING',
        metadata,
        default: blankToNull(parameter.defaultValue),
        allowed_values: parameter.allowedValues.filter(hasIdentifierValue),
        pattern: parameter.pattern
      };
    case 'BOOLEAN':
      return { kind: 'BOOLEAN', metadata, default: parameter.defaultValue };
    case 'INTEGER':
      return {
        kind: 'INTEGER',
        metadata,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: toNumberOrNull(parameter.defaultValue),
        allowed_values: parameter.allowedValues
          .map((value) => Number(value))
          .filter((value) => Number.isFinite(value))
      };
    case 'FLOAT':
      return {
        kind: 'FLOAT',
        metadata,
        minimum: toNumberOrNull(parameter.minimum),
        maximum: toNumberOrNull(parameter.maximum),
        default: toNumberOrNull(parameter.defaultValue),
        allowed_values: parameter.allowedValues
          .map((value) => Number(value))
          .filter((value) => Number.isFinite(value))
      };
    case 'DATETIME':
      return {
        kind: 'DATETIME',
        metadata,
        minimum: toIsoDateTimeOrNull(parameter.minimum),
        maximum: toIsoDateTimeOrNull(parameter.maximum),
        default: toIsoDateTimeOrNull(parameter.defaultValue),
        pattern: parameter.pattern
      };
  }
}

function buildSpatialCoverageDraft(spatial: DatasetSpatialForm): DatasetDraftMetadata['spatial_coverage'] {
  return blankToNull(spatial);
}

function buildTemporalCoverageDraft(
  temporal: DatasetTemporalForm
): DatasetDraftMetadata['temporal_coverage'] {
  return {
    start: toIsoDateTimeOrNull(temporal.start),
    end: toIsoDateTimeOrNull(temporal.end)
  };
}

function isEmptyWebServiceParameter(parameter: DatasetWebServiceParameterForm): boolean {
  const isMetadataEmpty = [
    parameter.metadata.variable,
    parameter.metadata.property,
    parameter.metadata.label
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
        !hasIdentifierValue(parameter.pattern)
      );
  }
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
