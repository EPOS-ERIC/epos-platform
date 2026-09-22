import type { DatasetResourceMetadata } from '../../generated/backofficeSchemas';
import { describe, expect, it } from 'vitest';
import type {
  DatasetDetailForm,
  DatasetWebServiceParameterForm,
  DatasetWebServiceScalarParameterForm,
} from '../models/dataset-detail-form';
import {
  distributionValidationError,
  geologicalTemporalValidationError,
  hasBodyOnUnsupportedMethod,
  hasIncompleteWebServiceBody,
  hasInvalidExplodeParameter,
  hasMultipleParameterWithoutAllowedValues,
  hasParameterDefaultOutsideAllowedValues,
  temporalWarningKind,
  temporalWarningMessage,
  toDatasetDetailForm,
  toDatasetSaveRequest,
} from './dataset-detail-form.mapper';

const WEB_SERVICE_TYPE = 'WEB_SERVICE' as const;

function datasetMetadata(): DatasetResourceMetadata {
  return {
    resource_type: 'dataset',
    id: 'dataset-id',
    title: 'Dataset title',
    description: 'Dataset description',
    keywords: ['seismic', 'catalogue'],
    accrual_periodicity: null,
    quality_annotation: 'reviewed',
    access_right: 'restricted',
    distributions: [
      {
        id: 'distribution-id',
        title: 'Service distribution',
        description: 'Distribution description',
        type: WEB_SERVICE_TYPE,
        issued: null,
        modified: 'invalid-date',
        web_service: {
          id: 'service-id',
          name: 'Service',
          description: null,
          template_url: 'https://example.test/{value}',
          method: 'POST',
          type: 'DYNAMIC',
          visible_on: ['GRAPH', 'MAP'],
          format: 'application/json',
          body_content_type: 'application/json',
          body_template: '{"when":"{when}"}',
          provider: { id: 'provider-id', name: 'Provider' },
          documentation: { title: null, description: null, uri: 'https://example.test/docs' },
          contact_points: ['contact-id'],
          spatial_coverage: 'POINT (12.5 41.9)',
          temporal_coverage: { start: 'invalid-start', end: null },
          body_parameters: [
            {
              kind: 'DATETIME',
              position: 12,
              name: 'when',
              label: 'When',
              description: null,
              required: true,
              read_only: false,
              minimum: '2025-01-01T00:00:00Z',
              maximum: null,
              default: '2026-01-01T00:00:00Z',
              layout: '2006-01-02',
            },
          ],
          parameters: [
            {
              id: 'parameter-id',
              kind: 'FLOAT',
              position: 8,
              name: 'value',
              label: 'Value',
              description: 'Numeric value',
              required: true,
              property: 'schema:value',
              location: 'QUERY',
              explode: true,
              multiple_values: true,
              read_only: false,
              minimum: 1.5,
              maximum: null,
              default: 2.5,
              enum: [{ value: 1.5, label: 'Low' }, { value: 2.5 }],
            },
            {
              id: 'template-id',
              kind: 'TEMPLATE',
              position: 9,
              name: 'filter',
              label: 'Filter',
              description: null,
              required: false,
              property: null,
              location: 'HEADER',
              explode: false,
              read_only: true,
              template: '{code}:{enabled}',
              parameters: [
                {
                  kind: 'STRING',
                  position: 4,
                  name: 'code',
                  label: 'Code',
                  description: 'Filter code',
                  required: true,
                  read_only: false,
                  default: 'A',
                  enum: [{ value: 'A', label: 'Alpha' }, { value: 'B' }],
                  pattern: '[A-Z]',
                },
                {
                  kind: 'BOOLEAN',
                  position: 5,
                  name: 'enabled',
                  label: 'Enabled',
                  required: false,
                  read_only: true,
                  default: false,
                },
              ],
            },
          ],
        },
      },
    ],
    spatial_coverage: null,
    temporal_coverage: { start: 'invalid-start', end: null },
    persistent_identifiers: [{ id: 'identifier-id', type: 'DOI', value: '10.123/example' }],
    contact_points: ['contact-id'],
    providers: [{ id: 'provider-id', name: 'Provider' }],
    categories: [{ id: 'category-id', name: 'Category' }],
    created: 'invalid-created',
    updated: null,
  };
}

function emptyDatasetForm(): DatasetDetailForm {
  return {
    title: '',
    description: '',
    keywords: '',
    accrualPeriodicity: '',
    qualityAnnotation: '',
    created: '',
    updated: '',
    spatial: null,
    temporalMode: 'calendar',
    temporal: { start: '', end: '' },
    geologicalTemporal: { startMya: '', endMya: '' },
    persistentIdentifiers: [],
    contactPoints: [],
    providers: [],
    categories: [],
    distributions: [],
  };
}

function parameter(
  defaultValue: string | null,
  required = true,
): Extract<DatasetWebServiceParameterForm, { kind: 'STRING' }> {
  return {
    kind: 'STRING',
    id: '',
    position: 0,
    name: 'value',
    label: 'Value',
    description: '',
    required,
    property: '',
    location: 'QUERY',
    explode: false,
    multipleValues: false,
    readOnly: false,
    defaultValue,
    allowedValues: [],
    pattern: '',
  };
}

function scalarParameter(
  defaultValue: string | null,
  required = true,
): Extract<DatasetWebServiceScalarParameterForm, { kind: 'STRING' }> {
  return {
    kind: 'STRING',
    position: 0,
    name: 'nested',
    label: 'Nested',
    description: '',
    required,
    readOnly: false,
    defaultValue,
    allowedValues: [],
    pattern: '',
  };
}

function formWithParameters(
  parameters: DatasetWebServiceParameterForm[],
  bodyParameters: DatasetWebServiceScalarParameterForm[] = [],
): DatasetDetailForm {
  return {
    ...emptyDatasetForm(),
    distributions: [
      {
        kind: 'web-service',
        title: '',
        description: '',
        issued: '',
        modified: '',
        license: '',
        type: WEB_SERVICE_TYPE,
        webService: {
          name: 'Service',
          description: '',
          templateUrl: 'https://example.test/{value}',
          method: 'GET',
          type: 'DYNAMIC',
          visibleOn: [],
          format: 'application/json',
          bodyContentType: '',
          bodyTemplate: '',
          bodyParameters,
          documentation: { title: '', description: '', uri: 'https://example.test/docs' },
          provider: { id: '', name: '' },
          contactPoints: [],
          spatialCoverage: null,
          temporalCoverage: { start: '', end: '' },
          parameters,
        },
      },
    ],
  };
}

describe('toDatasetDetailForm', () => {
  it('maps the complete web service contract and recursive parameter forms', () => {
    const form = toDatasetDetailForm(datasetMetadata());

    expect(form).toMatchObject({
      title: 'Dataset title',
      description: 'Dataset description',
      keywords: 'seismic, catalogue',
      accrualPeriodicity: '',
      qualityAnnotation: 'reviewed',
      created: 'invalid-created',
      updated: '',
      temporal: { start: 'invalid-start', end: '' },
    });
    expect(form.distributions[0]).toMatchObject({
      kind: 'web-service',
      issued: '',
      modified: 'invalid-date',
      webService: {
        description: '',
        method: 'POST',
        type: 'DYNAMIC',
        visibleOn: ['GRAPH', 'MAP'],
        bodyContentType: 'application/json',
        bodyTemplate: '{"when":"{when}"}',
        documentation: { title: '', description: '', uri: 'https://example.test/docs' },
        bodyParameters: [
          {
            kind: 'DATETIME',
            position: 12,
            name: 'when',
            description: '',
            defaultValue: '2026-01-01T00:00:00Z',
            layout: '2006-01-02',
          },
        ],
        parameters: [
          {
            kind: 'FLOAT',
            id: 'parameter-id',
            position: 8,
            name: 'value',
            property: 'schema:value',
            location: 'QUERY',
            explode: true,
            multipleValues: true,
            minimum: '1.5',
            maximum: '',
            defaultValue: '2.5',
            allowedValues: [{ value: '1.5', label: 'Low' }, { value: '2.5', label: '' }],
          },
          {
            kind: 'TEMPLATE',
            template: '{code}:{enabled}',
            parameters: [
              { kind: 'STRING', name: 'code', allowedValues: [{ value: 'A', label: 'Alpha' }, { value: 'B', label: '' }], pattern: '[A-Z]' },
              { kind: 'BOOLEAN', name: 'enabled', defaultValue: false },
            ],
          },
        ],
      },
    });
  });

  it('roundtrips geological temporal coverage without emitting calendar coverage', () => {
    const metadata = datasetMetadata();
    metadata.temporal_coverage = undefined;
    metadata.geological_temporal_coverage = { start_mya: 145.5, end_mya: 0 };

    const form = toDatasetDetailForm(metadata);
    expect(form.temporalMode).toBe('geological');
    expect(form.geologicalTemporal).toEqual({ startMya: '145.5', endMya: '0' });

    const request = toDatasetSaveRequest(form, 3, metadata);
    expect(request.metadata.geological_temporal_coverage).toEqual({ start_mya: 145.5, end_mya: 0 });
    expect(request.metadata.temporal_coverage).toBeUndefined();
  });

  it('supplies current placeholder rows and empty optional arrays', () => {
    const metadata = datasetMetadata();
    metadata.persistent_identifiers = [];
    metadata.providers = [];
    const distribution = metadata.distributions[0];
    if (distribution.type === WEB_SERVICE_TYPE) {
      distribution.web_service.parameters = [];
      distribution.web_service.visible_on = null;
      distribution.web_service.body_parameters = undefined;
    }

    const form = toDatasetDetailForm(metadata);

    expect(form.persistentIdentifiers).toEqual([{ type: '', value: '' }]);
    expect(form.providers).toEqual([{ id: '', name: '' }]);
    expect(form.distributions[0]).toMatchObject({
      webService: {
        visibleOn: [],
        bodyParameters: [],
        parameters: [
          {
            kind: 'STRING',
            name: '',
            label: '',
            location: 'QUERY',
            multipleValues: false,
            defaultValue: null,
          },
        ],
      },
    });
  });
});

describe('toDatasetSaveRequest', () => {
  it('roundtrips all new web service fields with deterministic parameter positions', () => {
    const form = toDatasetDetailForm(datasetMetadata());
    const request = toDatasetSaveRequest(form, 7, datasetMetadata());
    const distribution = request.metadata.distributions?.[0];

    expect(distribution).toMatchObject({
      web_service: {
        name: 'Service',
        description: undefined,
        method: 'POST',
        type: 'DYNAMIC',
        visible_on: ['GRAPH', 'MAP'],
        body_content_type: 'application/json',
        body_template: '{"when":"{when}"}',
        body_parameters: [
          {
            kind: 'DATETIME',
            position: 0,
            name: 'when',
            description: undefined,
            default: '2026-01-01T00:00:00.000Z',
            layout: '2006-01-02',
          },
        ],
        parameters: [
          {
            kind: 'FLOAT',
            position: 0,
            name: 'value',
            description: 'Numeric value',
            property: 'schema:value',
            location: 'QUERY',
            explode: true,
            multiple_values: true,
            enum: [{ value: 1.5, label: 'Low' }, { value: 2.5, label: undefined }],
          },
          {
            kind: 'TEMPLATE',
            position: 1,
            property: undefined,
            template: '{code}:{enabled}',
            parameters: [
              { kind: 'STRING', position: 0, enum: [{ value: 'A', label: 'Alpha' }, { value: 'B', label: undefined }], pattern: '[A-Z]' },
              { kind: 'BOOLEAN', position: 1, default: false },
            ],
          },
        ],
      },
    });
  });

  it('serializes blank optional strings and empty visibleOn as undefined', () => {
    const form = formWithParameters([parameter(null)]);
    form.description = '   ';
    form.distributions[0].description = ' ';
    const webService =
      form.distributions[0].kind === 'web-service' ? form.distributions[0].webService : undefined;
    if (!webService) throw new Error('Expected a web service');
    webService.description = ' ';
    webService.bodyContentType = '';
    webService.bodyTemplate = '   ';

    const request = toDatasetSaveRequest(form, 1, datasetMetadata());
    const distribution = request.metadata.distributions?.[0];

    expect(request.metadata.description).toBe('');
    expect(distribution?.description).toBeUndefined();
    if (!distribution || distribution.type !== WEB_SERVICE_TYPE)
      throw new Error('Expected web service draft');
    expect(distribution.web_service.description).toBeUndefined();
    expect(distribution.web_service.visible_on).toBeUndefined();
    expect(distribution.web_service.body_content_type).toBeUndefined();
    expect(distribution.web_service.body_template).toBeUndefined();
    const stringDraft = distribution.web_service.parameters?.[0];
    if (!stringDraft || stringDraft.kind !== 'STRING') throw new Error('Expected string parameter');
    expect(stringDraft.enum).toBeUndefined();
  });

  it('omits empty optional web service documentation', () => {
    const form = formWithParameters([]);
    const webService = form.distributions[0].kind === 'web-service'
      ? form.distributions[0].webService
      : undefined;
    if (!webService) throw new Error('Expected a web service');
    webService.documentation = { title: '', description: '', uri: '' };

    const distribution = toDatasetSaveRequest(form, 1, datasetMetadata()).metadata.distributions?.[0];

    if (!distribution || distribution.type !== WEB_SERVICE_TYPE)
      throw new Error('Expected web service draft');
    expect(distribution.web_service.documentation).toBeUndefined();
  });

  it('normalizes comma decimals and maps numeric enum values by kind', () => {
    const numericParameter: DatasetWebServiceParameterForm = {
      kind: 'FLOAT',
      id: '',
      position: 99,
      name: 'value',
      label: 'Value',
      description: '',
      required: false,
      property: '',
      location: 'QUERY',
      explode: false,
      multipleValues: true,
      readOnly: false,
      minimum: '1,5',
      maximum: '   ',
      defaultValue: '2,75',
      allowedValues: [
        { value: '3.5', label: 'Three and a half' },
        { value: '4,5', label: '' },
        { value: 'bad', label: '' },
      ],
    };
    const request = toDatasetSaveRequest(
      formWithParameters([numericParameter]),
      2,
      datasetMetadata(),
    );

    expect(request.metadata.distributions?.[0]).toMatchObject({
      web_service: {
        parameters: [
          {
            kind: 'FLOAT',
            position: 0,
            minimum: 1.5,
            maximum: null,
            default: 2.75,
            enum: [{ value: 3.5, label: 'Three and a half' }, { value: 4.5, label: undefined }],
          },
        ],
      },
    });
  });
});

describe('distributionValidationError', () => {
  it('reports required web service fields before saving', () => {
    const form = formWithParameters([]);

    expect(distributionValidationError(form)).toBe('Distribution 1: title is required.');
    form.distributions[0].title = 'API distribution';
    expect(distributionValidationError(form)).toBe('Distribution 1: description is required.');
    form.distributions[0].description = 'API distribution description';
    if (form.distributions[0].kind !== 'web-service') throw new Error('Expected a web service');
    form.distributions[0].webService.name = '';
    expect(distributionValidationError(form)).toBe('Distribution 1: web service name is required.');

    form.distributions[0].webService.name = 'API';
    form.distributions[0].webService.templateUrl = '';
    expect(distributionValidationError(form)).toBe('Distribution 1: web service template URL is required.');
    form.distributions[0].webService.templateUrl = 'https://example.test';
    form.distributions[0].webService.format = '';
    expect(distributionValidationError(form)).toBe('Distribution 1: web service format is required.');
  });

  it('requires downloadable file URL and format', () => {
    const form = emptyDatasetForm();
    form.distributions = [{
      kind: 'downloadable-file',
      title: 'Download',
      description: 'Downloadable distribution',
      issued: '',
      modified: '',
      license: '',
      type: 'DOWNLOADABLE_FILE',
      downloadableFile: { url: '', format: '' },
    }];

    expect(distributionValidationError(form)).toBe('Distribution 1: downloadable file URL is required.');
    const distribution = form.distributions[0];
    if (distribution.kind !== 'downloadable-file') throw new Error('Expected a downloadable file');
    distribution.downloadableFile.url = 'https://example.test/file';
    expect(distributionValidationError(form)).toBe('Distribution 1: downloadable file format is required.');
  });

  it('requires a URI only when documentation content is provided', () => {
    const form = formWithParameters([]);
    form.distributions[0].title = 'API distribution';
    form.distributions[0].description = 'API distribution description';
    if (form.distributions[0].kind !== 'web-service') throw new Error('Expected a web service');
    form.distributions[0].webService.documentation = { title: '', description: '', uri: '' };
    expect(distributionValidationError(form)).toBeNull();

    form.distributions[0].webService.documentation.title = 'Guide';
    expect(distributionValidationError(form)).toBe(
      'Distribution 1: documentation URI is required when documentation is provided.',
    );
  });
});

describe('dataset validation helpers', () => {
  it('validates geological temporal ages', () => {
    const form = emptyDatasetForm();
    form.temporalMode = 'geological';
    form.geologicalTemporal = { startMya: '10', endMya: '20' };
    expect(geologicalTemporalValidationError(form)).toContain('older');

    form.geologicalTemporal = { startMya: '20', endMya: '10' };
    expect(geologicalTemporalValidationError(form)).toBeNull();
  });

  it('checks enum defaults on full, template scalar, and body scalar parameters', () => {
    const full = parameter('fallback');
    full.allowedValues = [{ value: 'first', label: '' }, { value: 'second', label: '' }];
    const nested = scalarParameter('3');
    nested.allowedValues = [{ value: '1', label: '' }, { value: '2', label: '' }];
    const template: DatasetWebServiceParameterForm = {
      ...parameter(null, false),
      kind: 'TEMPLATE',
      template: '{nested}',
      parameters: [nested],
    };

    expect(hasParameterDefaultOutsideAllowedValues(formWithParameters([full]))).toBe(true);
    expect(hasParameterDefaultOutsideAllowedValues(formWithParameters([template]))).toBe(true);
    expect(hasParameterDefaultOutsideAllowedValues(formWithParameters([], [nested]))).toBe(true);
    nested.allowedValues.push({ value: '3', label: '' });
    expect(hasParameterDefaultOutsideAllowedValues(formWithParameters([template]))).toBe(false);
  });

  it('requires a non-empty enum when multiple values are enabled', () => {
    const multipleParameter = parameter(null);
    multipleParameter.multipleValues = true;

    expect(hasMultipleParameterWithoutAllowedValues(formWithParameters([multipleParameter]))).toBe(true);
    multipleParameter.allowedValues.push({ value: 'value', label: '' });
    expect(hasMultipleParameterWithoutAllowedValues(formWithParameters([multipleParameter]))).toBe(false);
  });

  it('requires explode parameters to be multiple-value query parameters', () => {
    const explodedParameter = parameter(null);
    explodedParameter.explode = true;

    expect(hasInvalidExplodeParameter(formWithParameters([explodedParameter]))).toBe(true);

    explodedParameter.location = 'QUERY';
    explodedParameter.multipleValues = true;
    expect(hasInvalidExplodeParameter(formWithParameters([explodedParameter]))).toBe(false);

    const templateParameter: DatasetWebServiceParameterForm = {
      ...parameter(null, false),
      kind: 'TEMPLATE',
      explode: true,
      template: '{}',
      parameters: [],
    };
    expect(hasInvalidExplodeParameter(formWithParameters([templateParameter]))).toBe(true);
  });

  it('rejects incomplete request bodies and methods that do not support them', () => {
    const form = formWithParameters([]);
    const distribution = form.distributions[0];
    if (distribution.kind !== 'web-service') throw new Error('Expected a web service distribution');

    distribution.webService.bodyContentType = 'application/json';
    expect(hasIncompleteWebServiceBody(form)).toBe(true);
    expect(hasBodyOnUnsupportedMethod(form)).toBe(true);

    distribution.webService.method = 'POST';
    distribution.webService.bodyTemplate = '{}';
    expect(hasIncompleteWebServiceBody(form)).toBe(false);
    expect(hasBodyOnUnsupportedMethod(form)).toBe(false);
  });

  it('returns the current warning kinds and messages for temporal coverage', () => {
    const form = emptyDatasetForm();
    expect(temporalWarningKind(form)).toBe('none');
    expect(temporalWarningMessage(form)).toBe('');

    form.temporal = { start: '2026-01-02T00:00', end: '' };
    expect(temporalWarningKind(form)).toBe('warning');
    expect(temporalWarningMessage(form)).toContain('acquired continuously');

    form.temporal = { start: '2026-01-02T00:00', end: '2026-01-01T00:00' };
    expect(temporalWarningKind(form)).toBe('error');
    expect(temporalWarningMessage(form)).toBe('End date must be after Start Date');

    form.temporal = { start: 'invalid', end: 'also-invalid' };
    expect(temporalWarningKind(form)).toBe('none');
  });
});
