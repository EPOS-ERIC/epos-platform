import type { DatasetResourceMetadata, DatasetWebService } from '../../generated/backofficeSchemas';
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
    title: 'Dataset title',
    description: 'Dataset description',
    keywords: ['seismic', 'catalogue'],
    accrual_periodicity: null,
    quality_annotation: 'reviewed',
    conforms_to: 'EPOS-DCAT-AP',
    provenance: 'Imported and reviewed',
    type: 'Seismic catalogue',
    version: '2.1',
    spatial_resolution_in_meters: 250,
    temporal_resolution: 60,
    access_right: 'RESTRICTED',
    distributions: [
      {
        id: 'distribution-id',
        resource_id: 'distribution-resource-id',
        title: 'Service distribution',
        description: 'Distribution description',
        type: WEB_SERVICE_TYPE,
        web_service: {
          id: 'service-id',
          resource_id: 'service-resource-id',
          visible_on: ['GRAPH', 'MAP'],
          provider: { id: 'provider-id', name: 'Provider' },
          documentation_uri: 'https://example.test/docs',
          contact_points: ['contact-id'],
          keywords: ['web service'],
          access_rights: 'RESTRICTED',
          license: 'CC-BY-4.0',
          categories: [{ id: 'service-category-id', name: 'Service category' }],
          type: 'HTTP',
          template_url: 'https://example.test/{value}',
          method: 'POST',
          format: 'application/json',
          available_formats: [{ kind: 'ORIGINAL', output_format: 'application/json', label: 'JSON' }],
          body_required: false,
          body_content_type: 'application/json',
          body_template: '{"when":"{when}"}',
          body_parameters: [
              {
                id: 'body-parameter-id',
                resource_id: 'body-parameter-resource-id',
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
              resource_id: 'parameter-resource-id',
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
              default: [2.5],
              enum: [{ value: 1.5, label: 'Low' }, { value: 2.5 }],
            },
            {
              id: 'template-id',
              resource_id: 'template-resource-id',
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
                  resource_id: 'code-parameter-resource-id',
                  kind: 'STRING',
                  position: 4,
                  name: 'code',
                  label: 'Code',
                  description: 'Filter code',
                  required: true,
                  read_only: false,
                  default: ['A'],
                  enum: [{ value: 'A', label: 'Alpha' }, { value: 'B' }],
                  pattern: '[A-Z]',
                },
                {
                  id: 'enabled-parameter-id',
                  resource_id: 'enabled-parameter-resource-id',
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
    persistent_identifiers: [{
      id: 'identifier-id',
      resource_id: 'identifier-resource-id',
      type: 'DOI',
      value: '10.123/example',
    }],
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
    conformsTo: '',
    provenance: '',
    type: '',
    version: '',
    spatialResolutionInMeters: '',
    temporalResolution: '',
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
  defaultValue: string[],
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
  defaultValue: string[],
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

function coordinateParameter(
  role: 'LONGITUDE' | 'LATITUDE',
): Extract<DatasetWebServiceParameterForm, { kind: 'FLOAT' }> {
  return {
    kind: 'FLOAT',
    id: '',
    position: 0,
    name: role.toLowerCase(),
    label: role === 'LONGITUDE' ? 'Longitude' : 'Latitude',
    description: '',
    required: true,
    property: '',
    location: 'QUERY',
    role,
    explode: false,
    multipleValues: false,
    readOnly: false,
    minimum: '',
    maximum: '',
    defaultValue: [],
    allowedValues: [],
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
        license: '',
        byteSize: '',
        type: WEB_SERVICE_TYPE,
        webService: {
          visibleOn: [],
          documentationUri: 'https://example.test/docs',
          provider: { id: '', name: '' },
          contactPoints: [],
          keywords: [],
          accessRights: '',
          license: '',
          categories: [],
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
      conformsTo: 'EPOS-DCAT-AP',
      provenance: 'Imported and reviewed',
      type: 'Seismic catalogue',
      version: '2.1',
      spatialResolutionInMeters: '250',
      temporalResolution: '60',
      created: 'invalid-created',
      updated: '',
      temporal: { start: 'invalid-start', end: '' },
    });
    expect(form.distributions[0]).toMatchObject({
      id: 'distribution-id',
      resourceId: 'distribution-resource-id',
      nativeFormats: ['application/json'],
      kind: 'web-service',
      byteSize: '',
      webService: {
        resourceId: 'service-resource-id',
        visibleOn: ['GRAPH', 'MAP'],
        documentationUri: 'https://example.test/docs',
        keywords: ['web service'],
        accessRights: 'RESTRICTED',
        license: 'CC-BY-4.0',
        categories: [{ id: 'service-category-id', name: 'Service category' }],
        configuration: {
          type: 'HTTP',
          method: 'POST',
          bodyRequired: false,
          bodyContentType: 'application/json',
          bodyTemplate: '{"when":"{when}"}',
          bodyParameters: [
          {
            kind: 'DATETIME',
            resourceId: 'body-parameter-resource-id',
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
            resourceId: 'parameter-resource-id',
            position: 8,
            name: 'value',
            property: 'schema:value',
            location: 'QUERY',
            explode: true,
            multipleValues: true,
            minimum: '1.5',
            maximum: '',
            defaultValue: ['2.5'],
            allowedValues: [{ value: '1.5', label: 'Low' }, { value: '2.5', label: '' }],
          },
          {
            kind: 'TEMPLATE',
            resourceId: 'template-resource-id',
            template: '{code}:{enabled}',
            parameters: [
              { kind: 'STRING', resourceId: 'code-parameter-resource-id', name: 'code', allowedValues: [{ value: 'A', label: 'Alpha' }, { value: 'B', label: '' }], pattern: '[A-Z]' },
              { kind: 'BOOLEAN', resourceId: 'enabled-parameter-resource-id', name: 'enabled', defaultValue: false },
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

  it('preserves zero resolutions and clears blank resolution fields', () => {
    const metadata = datasetMetadata();
    metadata.spatial_resolution_in_meters = 0;
    metadata.temporal_resolution = 0;

    const form = toDatasetDetailForm(metadata);
    expect(form.spatialResolutionInMeters).toBe('0');
    expect(form.temporalResolution).toBe('0');

    let request = toDatasetSaveRequest(form, 1, metadata);
    expect(request.metadata.spatial_resolution_in_meters).toBe(0);
    expect(request.metadata.temporal_resolution).toBe(0);

    form.spatialResolutionInMeters = '';
    form.temporalResolution = '';
    request = toDatasetSaveRequest(form, 1, metadata);
    expect(request.metadata.spatial_resolution_in_meters).toBeNull();
    expect(request.metadata.temporal_resolution).toBeNull();
  });

  it('supplies current placeholder rows and empty optional arrays', () => {
    const metadata = datasetMetadata();
    metadata.persistent_identifiers = [];
    metadata.providers = [];
    const distribution = metadata.distributions[0];
    if (distribution.type === WEB_SERVICE_TYPE) {
      distribution.web_service.visible_on = null;
      if (distribution.web_service.type === 'HTTP') {
        distribution.web_service.parameters = [];
        distribution.web_service.body_parameters = undefined;
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
              defaultValue: [],
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
    if (!distribution || distribution.type !== WEB_SERVICE_TYPE) {
      throw new Error('Expected web service draft');
    }

    expect(distribution).toMatchObject({
      resource_id: 'distribution-resource-id',
      web_service: {
        resource_id: 'service-resource-id',
        visible_on: ['GRAPH', 'MAP'],
        keywords: ['web service'],
        access_rights: 'RESTRICTED',
        license: 'CC-BY-4.0',
        categories: ['service-category-id'],
        method: 'POST',
        type: 'HTTP',
        body_required: false,
        body_content_type: 'application/json',
        body_template: '{"when":"{when}"}',
        body_parameters: [
          {
            kind: 'DATETIME',
            resource_id: 'body-parameter-resource-id',
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
            resource_id: 'parameter-resource-id',
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
            resource_id: 'template-resource-id',
            position: 1,
            property: undefined,
            template: '{code}:{enabled}',
            parameters: [
              { kind: 'STRING', resource_id: 'code-parameter-resource-id', position: 0, enum: [{ value: 'A', label: 'Alpha' }, { value: 'B', label: undefined }], pattern: '[A-Z]' },
              { kind: 'BOOLEAN', resource_id: 'enabled-parameter-resource-id', position: 1, default: false },
            ],
          },
          ],
      },
    });
    expect(request.metadata.persistent_identifiers).toEqual([{
      resource_id: 'identifier-resource-id',
      type: 'DOI',
      value: '10.123/example',
    }]);
    expect(request.metadata).toMatchObject({
      conforms_to: 'EPOS-DCAT-AP',
      provenance: 'Imported and reviewed',
      type: 'Seismic catalogue',
      version: '2.1',
      spatial_resolution_in_meters: 250,
      temporal_resolution: 60,
    });
    expect(distribution).not.toHaveProperty('id');
    expect(distribution.web_service).not.toHaveProperty('configuration');
    expect(distribution.web_service).not.toHaveProperty('available_formats');
  });

  it('roundtrips WFS, WMS, and WMTS configurations', () => {
    const common = {
      id: 'service-id',
      resource_id: 'service-resource-id',
      keywords: [],
      categories: [],
    };
    const webServices: DatasetWebService[] = [
      {
        ...common,
        type: 'WFS',
        available_formats: [{ kind: 'ORIGINAL', output_format: 'application/geo+json', label: 'GeoJSON' }],
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
        ...common,
        type: 'WMS',
        capabilities_url: 'https://example.test/wms?service=WMS&request=GetCapabilities',
        default_layer: 'stations',
        layers: [{
          id: 'wms-layer-id',
          resource_id: 'wms-layer-resource-id',
          name: 'stations',
          default_style: '',
          styles: [''],
        }],
        transparency_editable: true,
        default_transparency: false,
        get_map_parameters: [],
        get_feature_info_parameters: [],
      },
      {
        ...common,
        type: 'WMTS',
        capabilities_url: 'https://example.test/wmts/capabilities.xml',
        default_layer: 'stations',
        layers: [{
          id: 'wmts-layer-id',
          resource_id: 'wmts-layer-resource-id',
          name: 'stations',
          default_style: 'standard',
          styles: ['standard'],
        }],
        get_tile_parameters: [],
        get_feature_info_parameters: [],
      },
    ];

    for (const webService of webServices) {
      const metadata = datasetMetadata();
      const distribution = metadata.distributions[0];
      if (distribution.type !== WEB_SERVICE_TYPE) throw new Error('Expected web service');
      distribution.web_service = webService;

      const form = toDatasetDetailForm(metadata);
      expect(form.distributions[0].nativeFormats).toEqual(
        'available_formats' in webService ? webService.available_formats.map(({ output_format }) => output_format) : [],
      );
      const requestDistribution = toDatasetSaveRequest(form, 1, metadata).metadata.distributions?.[0];
      if (!requestDistribution || requestDistribution.type !== WEB_SERVICE_TYPE) {
        throw new Error('Expected web service draft');
      }

      expect(requestDistribution.web_service.type).toBe(webService.type);
      expect(requestDistribution.web_service).not.toHaveProperty('configuration');
      expect(requestDistribution.web_service).not.toHaveProperty('available_formats');
      if (requestDistribution.web_service.type === 'WFS') {
        expect(requestDistribution.web_service.default_feature_type).toBe('stations');
      } else if (requestDistribution.web_service.type === 'WMS' || requestDistribution.web_service.type === 'WMTS') {
        expect(requestDistribution.web_service.layers[0]).toMatchObject({
          resource_id: `${requestDistribution.web_service.type.toLowerCase()}-layer-resource-id`,
          name: 'stations',
        });
      }
    }
  });

  it('serializes blank optional strings and empty visibleOn as undefined', () => {
    const form = formWithParameters([parameter([])]);
    form.description = '   ';
    form.distributions[0].description = ' ';
    const webService =
      form.distributions[0].kind === 'web-service' ? form.distributions[0].webService : undefined;
    if (!webService) throw new Error('Expected a web service');
    if (webService.configuration.type !== 'HTTP') throw new Error('Expected HTTP configuration');
    webService.configuration.bodyContentType = '';
    webService.configuration.bodyTemplate = '   ';

    const request = toDatasetSaveRequest(form, 1, datasetMetadata());
    const distribution = request.metadata.distributions?.[0];

    expect(request.metadata.description).toBe('');
    expect(distribution?.description).toBeUndefined();
    if (!distribution || distribution.type !== WEB_SERVICE_TYPE)
      throw new Error('Expected web service draft');
    expect(distribution.web_service.visible_on).toBeUndefined();
    if (distribution.web_service.type !== 'HTTP')
      throw new Error('Expected HTTP configuration');
    expect(distribution.web_service.body_content_type).toBeUndefined();
    expect(distribution.web_service.body_template).toBeUndefined();
    const stringDraft = distribution.web_service.parameters?.[0];
    if (!stringDraft || stringDraft.kind !== 'STRING') throw new Error('Expected string parameter');
    expect(stringDraft.enum).toBeUndefined();
  });

  it('omits an empty optional web service documentation URI', () => {
    const form = formWithParameters([]);
    const webService = form.distributions[0].kind === 'web-service'
      ? form.distributions[0].webService
      : undefined;
    if (!webService) throw new Error('Expected a web service');
    webService.documentationUri = '';

    const distribution = toDatasetSaveRequest(form, 1, datasetMetadata()).metadata.distributions?.[0];

    if (!distribution || distribution.type !== WEB_SERVICE_TYPE)
      throw new Error('Expected web service draft');
    expect(distribution.web_service.documentation_uri).toBeUndefined();
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
      defaultValue: ['2,75'],
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
            default: [2.75],
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
      license: '',
      byteSize: '',
      type: 'DOWNLOADABLE_FILE',
      downloadableFile: { url: '', format: '' },
    }];

    expect(distributionValidationError(form)).toBe('Distribution 1: downloadable file URL is required.');
    const distribution = form.distributions[0];
    if (distribution.kind !== 'downloadable-file') throw new Error('Expected a downloadable file');
    distribution.downloadableFile.url = 'https://example.test/file';
    expect(distributionValidationError(form)).toBe('Distribution 1: downloadable file format is required.');
  });

  it('accepts an optional documentation URI', () => {
    const form = formWithParameters([]);
    form.distributions[0].title = 'API distribution';
    form.distributions[0].description = 'API distribution description';
    if (form.distributions[0].kind !== 'web-service') throw new Error('Expected a web service');
    form.distributions[0].webService.documentationUri = '';
    expect(distributionValidationError(form)).toBeNull();
  });

  it('lets the backend validate DYNAMIC coordinate parameters and preserves their roles', () => {
    const form = formWithParameters([]);
    const distribution = form.distributions[0];
    distribution.title = 'Dynamic distribution';
    distribution.description = 'Dynamic service';
    if (distribution.kind !== 'web-service' || distribution.webService.configuration.type !== 'HTTP') {
      throw new Error('Expected HTTP web service configuration');
    }
    distribution.webService.configuration = {
      ...distribution.webService.configuration,
      type: 'DYNAMIC',
    };

    expect(distributionValidationError(form)).toBeNull();

    distribution.webService.configuration.parameters = [
      coordinateParameter('LONGITUDE'),
      coordinateParameter('LATITUDE'),
    ];
    expect(distributionValidationError(form)).toBeNull();

    const requestDistribution = toDatasetSaveRequest(form, 1, datasetMetadata()).metadata.distributions?.[0];
    if (!requestDistribution || requestDistribution.type !== WEB_SERVICE_TYPE) {
      throw new Error('Expected web service draft');
    }
    expect(requestDistribution.web_service).toMatchObject({
      type: 'DYNAMIC',
      parameters: [{ role: 'LONGITUDE' }, { role: 'LATITUDE' }],
    });
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
    const full = parameter(['fallback']);
    full.allowedValues = [{ value: 'first', label: '' }, { value: 'second', label: '' }];
    const nested = scalarParameter(['3']);
    nested.allowedValues = [{ value: '1', label: '' }, { value: '2', label: '' }];
    const template: DatasetWebServiceParameterForm = {
      ...parameter([], false),
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
    const multipleParameter = parameter([]);
    multipleParameter.multipleValues = true;

    expect(hasMultipleParameterWithoutAllowedValues(formWithParameters([multipleParameter]))).toBe(true);
    multipleParameter.allowedValues.push({ value: 'value', label: '' });
    expect(hasMultipleParameterWithoutAllowedValues(formWithParameters([multipleParameter]))).toBe(false);
  });

  it('requires explode parameters to be multiple-value query parameters', () => {
    const explodedParameter = parameter([]);
    explodedParameter.explode = true;

    expect(hasInvalidExplodeParameter(formWithParameters([explodedParameter]))).toBe(true);

    explodedParameter.location = 'QUERY';
    explodedParameter.multipleValues = true;
    expect(hasInvalidExplodeParameter(formWithParameters([explodedParameter]))).toBe(false);

    const templateParameter: DatasetWebServiceParameterForm = {
      ...parameter([], false),
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
