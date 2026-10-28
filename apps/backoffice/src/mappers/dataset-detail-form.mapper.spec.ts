import type { DatasetResourceMetadata, WebServiceConfiguration } from '../../generated/backofficeSchemas';
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
        available_formats: [],
        web_service: {
          id: 'service-id',
          name: 'Service',
          description: null,
          visible_on: ['GRAPH', 'MAP'],
          provider: { id: 'provider-id', name: 'Provider' },
          documentation: { title: null, description: null, uri: 'https://example.test/docs' },
          contact_points: ['contact-id'],
          spatial_coverage: 'POINT (12.5 41.9)',
          temporal_coverage: { start: 'invalid-start', end: null },
          configuration: {
            type: 'HTTP',
            template_url: 'https://example.test/{value}',
            method: 'POST',
            format: 'application/json',
            body_required: false,
            body_content_type: 'application/json',
            body_template: '{"when":"{when}"}',
            body_parameters: [
              {
                id: 'body-parameter-id',
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
                  id: 'code-parameter-id',
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
                  id: 'enabled-parameter-id',
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
    keywords: [],
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
          visibleOn: [],
          documentation: { title: '', description: '', uri: 'https://example.test/docs' },
          provider: { id: '', name: '' },
          contactPoints: [],
          spatialCoverage: null,
          temporalCoverage: { start: '', end: '' },
          configuration: {
            type: 'HTTP',
            templateUrl: 'https://example.test/{value}',
            method: 'GET',
            format: 'application/json',
            bodyContentType: '',
            bodyTemplate: '',
            bodyRequired: true,
            bodyParameters,
            parameters,
          },
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
      keywords: ['seismic', 'catalogue'],
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
        visibleOn: ['GRAPH', 'MAP'],
        documentation: { title: '', description: '', uri: 'https://example.test/docs' },
        configuration: {
          type: 'HTTP',
          method: 'POST',
          bodyRequired: false,
          bodyContentType: 'application/json',
          bodyTemplate: '{"when":"{when}"}',
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
      distribution.web_service.visible_on = null;
      if (distribution.web_service.configuration.type === 'HTTP') {
        distribution.web_service.configuration.parameters = [];
        distribution.web_service.configuration.body_parameters = undefined;
      }
    }

    const form = toDatasetDetailForm(metadata);

    expect(form.persistentIdentifiers).toEqual([{ type: '', value: '' }]);
    expect(form.providers).toEqual([{ id: '', name: '' }]);
    expect(form.distributions[0]).toMatchObject({
      webService: {
        visibleOn: [],
        configuration: {
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
        visible_on: ['GRAPH', 'MAP'],
        configuration: {
          method: 'POST',
          type: 'HTTP',
          body_required: false,
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
      },
    });
  });

  it('roundtrips WFS, WMS, and WMTS configurations', () => {
    const configurations: WebServiceConfiguration[] = [
      {
        type: 'WFS',
        get_feature_template_url: 'https://example.test/wfs/{featureType}',
        version: '2.0.0',
        feature_types: ['stations'],
        default_feature_type: 'stations',
        output_formats: ['application/geo+json'],
        default_output_format: 'application/geo+json',
        result_limit: 500,
        result_limit_editable: true,
        parameters: [],
      },
      {
        type: 'WMS',
        capabilities_url: 'https://example.test/wms?service=WMS&request=GetCapabilities',
        default_layer: 'stations',
        layers: [{ name: 'stations', default_style: '', styles: [''] }],
        transparency_editable: true,
        default_transparency: false,
        get_map_parameters: [],
        get_feature_info_parameters: [],
      },
      {
        type: 'WMTS',
        capabilities_url: 'https://example.test/wmts/capabilities.xml',
        default_layer: 'stations',
        layers: [{ name: 'stations', default_style: 'standard', styles: ['standard'] }],
        get_tile_parameters: [],
        get_feature_info_parameters: [],
      },
    ];

    for (const configuration of configurations) {
      const metadata = datasetMetadata();
      const distribution = metadata.distributions[0];
      if (distribution.type !== WEB_SERVICE_TYPE) throw new Error('Expected web service');
      distribution.web_service.configuration = configuration;

      const form = toDatasetDetailForm(metadata);
      const requestDistribution = toDatasetSaveRequest(form, 1, metadata).metadata.distributions?.[0];
      if (!requestDistribution || requestDistribution.type !== WEB_SERVICE_TYPE) {
        throw new Error('Expected web service draft');
      }

      expect(requestDistribution.web_service.configuration).toEqual(configuration);
    }
  });

  it('serializes blank optional strings and empty visibleOn as undefined', () => {
    const form = formWithParameters([parameter(null)]);
    form.description = '   ';
    form.distributions[0].description = ' ';
    const webService =
      form.distributions[0].kind === 'web-service' ? form.distributions[0].webService : undefined;
    if (!webService) throw new Error('Expected a web service');
    webService.description = ' ';
    if (webService.configuration.type !== 'HTTP') throw new Error('Expected HTTP configuration');
    webService.configuration.bodyContentType = '';
    webService.configuration.bodyTemplate = '   ';

    const request = toDatasetSaveRequest(form, 1, datasetMetadata());
    const distribution = request.metadata.distributions?.[0];

    expect(request.metadata.description).toBe('');
    expect(distribution?.description).toBeUndefined();
    if (!distribution || distribution.type !== WEB_SERVICE_TYPE)
      throw new Error('Expected web service draft');
    expect(distribution.web_service.description).toBeUndefined();
    expect(distribution.web_service.visible_on).toBeUndefined();
    if (distribution.web_service.configuration.type !== 'HTTP')
      throw new Error('Expected HTTP configuration');
    expect(distribution.web_service.configuration.body_content_type).toBeUndefined();
    expect(distribution.web_service.configuration.body_template).toBeUndefined();
    const stringDraft = distribution.web_service.configuration.parameters?.[0];
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
        configuration: {
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
    const configuration = form.distributions[0].webService.configuration;
    if (configuration.type !== 'HTTP') throw new Error('Expected HTTP configuration');
    configuration.templateUrl = '';
    expect(distributionValidationError(form)).toBe('Distribution 1: web service template URL is required.');
    configuration.templateUrl = 'https://example.test';
    configuration.format = '';
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

    const configuration = distribution.webService.configuration;
    if (configuration.type !== 'HTTP') throw new Error('Expected HTTP configuration');
    configuration.bodyContentType = 'application/json';
    expect(hasIncompleteWebServiceBody(form)).toBe(true);
    expect(hasBodyOnUnsupportedMethod(form)).toBe(true);

    configuration.method = 'POST';
    configuration.bodyTemplate = '{}';
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
