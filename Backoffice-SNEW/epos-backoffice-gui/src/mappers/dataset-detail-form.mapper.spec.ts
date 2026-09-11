import type { DatasetResourceMetadata } from '../../generated/backofficeSchemas';
import { describe, expect, it } from 'vitest';
import type {
  DatasetDetailForm,
  DatasetWebServiceParameterForm
} from '../models/dataset-detail-form';
import {
  hasParameterDefaultOutsideAllowedValues,
  hasRequiredParameterWithoutDefault,
  temporalWarningKind,
  temporalWarningMessage,
  toDatasetDetailForm,
  toDatasetSaveRequest
} from './dataset-detail-form.mapper';

const WEB_SERVICE_TYPE =
  'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE' as const;

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
          description: 'Service description',
          template_url: 'https://example.test/{value}',
          format: 'application/json',
          provider: { id: 'provider-id', name: 'Provider' },
          documentation: { title: null, description: null, uri: 'https://example.test/docs' },
          contact_points: ['contact-id'],
          spatial_coverage: 'POINT (12.5 41.9)',
          temporal_coverage: { start: 'invalid-start', end: null },
          parameters: [
            {
              kind: 'FLOAT',
              metadata: {
                id: 'parameter-id',
                variable: 'value',
                property: 'value',
                label: 'Value',
                required: true,
                multiple_values: false,
                read_only: false
              },
              minimum: 1.5,
              maximum: null,
              default: 2.5,
              allowed_values: [1.5, 2.5]
            }
          ]
        }
      }
    ],
    spatial_coverage: null,
    temporal_coverage: { start: 'invalid-start', end: null },
    persistent_identifiers: [{ id: 'identifier-id', type: 'DOI', value: '10.123/example' }],
    contact_points: ['contact-id'],
    providers: [{ id: 'provider-id', name: 'Provider' }],
    categories: [{ id: 'category-id', name: 'Category' }],
    created: 'invalid-created',
    updated: null
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
    temporal: { start: '', end: '' },
    persistentIdentifiers: [],
    contactPoints: [],
    providers: [],
    categories: [],
    distributions: []
  };
}

function parameter(
  defaultValue: string | null,
  required = true
): Extract<DatasetWebServiceParameterForm, { kind: 'STRING' }> {
  return {
    kind: 'STRING',
    metadata: {
      id: '',
      variable: 'value',
      property: '',
      label: '',
      required,
      multiple_values: false,
      read_only: false
    },
    defaultValue,
    allowedValues: [],
    pattern: ''
  };
}

function formWithParameters(parameters: DatasetWebServiceParameterForm[]): DatasetDetailForm {
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
          name: '',
          description: '',
          templateUrl: '',
          format: '',
          documentation: { title: '', description: '', uri: '' },
          provider: { id: '', name: '' },
          contactPoints: [],
          spatialCoverage: null,
          temporalCoverage: { start: '', end: '' },
          parameters
        }
      }
    ]
  };
}

describe('toDatasetDetailForm', () => {
  it('maps resource metadata and nested web service values into editable form values', () => {
    const form = toDatasetDetailForm(datasetMetadata());

    expect(form).toMatchObject({
      title: 'Dataset title',
      description: 'Dataset description',
      keywords: 'seismic, catalogue',
      accrualPeriodicity: '',
      qualityAnnotation: 'reviewed',
      created: 'invalid-created',
      updated: '',
      spatial: null,
      temporal: { start: 'invalid-start', end: '' },
      persistentIdentifiers: [{ id: 'identifier-id', type: 'DOI', value: '10.123/example' }],
      providers: [{ id: 'provider-id', name: 'Provider' }]
    });
    expect(form.distributions[0]).toMatchObject({
      kind: 'web-service',
      issued: '',
      modified: 'invalid-date',
      webService: {
        documentation: { title: '', description: '', uri: 'https://example.test/docs' },
        provider: { id: 'provider-id', name: 'Provider' },
        temporalCoverage: { start: 'invalid-start', end: '' },
        parameters: [
          {
            kind: 'FLOAT',
            minimum: '1.5',
            maximum: '',
            defaultValue: '2.5',
            allowedValues: ['1.5', '2.5']
          }
        ]
      }
    });
  });

  it('supplies the current placeholder rows when identifiers, providers, or parameters are absent', () => {
    const metadata = datasetMetadata();
    metadata.persistent_identifiers = [];
    metadata.providers = [];
    const distribution = metadata.distributions[0];
    if (distribution.type === WEB_SERVICE_TYPE) {
      distribution.web_service.parameters = [];
    }

    const form = toDatasetDetailForm(metadata);

    expect(form.persistentIdentifiers).toEqual([{ type: '', value: '' }]);
    expect(form.providers).toEqual([{ id: '', name: '' }]);
    expect(form.distributions[0]).toMatchObject({
      webService: {
        parameters: [{ kind: 'STRING', defaultValue: null, allowedValues: [], pattern: '' }]
      }
    });
  });
});

describe('toDatasetSaveRequest', () => {
  it('maps outbound fields, filters blank references, and preserves access_right', () => {
    const form = toDatasetDetailForm(datasetMetadata());
    form.title = '';
    form.description = '   ';
    form.keywords = ' first, , second ';
    form.persistentIdentifiers.push({ type: ' ', value: 'ignored' });
    form.contactPoints.push('  ');
    form.providers.push({ id: '', name: 'Ignored' });
    form.categories.push({ id: ' ', name: 'Ignored' });

    const request = toDatasetSaveRequest(form, 7, datasetMetadata());

    expect(request).toMatchObject({
      resource_type: 'dataset',
      revision: 7,
      metadata: {
        title: 'Untitled dataset',
        description: undefined,
        keywords: ['first', 'second'],
        access_right: 'restricted',
        persistent_identifiers: [{ type: 'DOI', value: '10.123/example' }],
        contact_points: ['contact-id'],
        providers: ['provider-id'],
        categories: ['category-id']
      }
    });
  });

  it('normalizes comma decimals and maps blank numeric values to null', () => {
    const numericParameter: DatasetWebServiceParameterForm = {
      kind: 'FLOAT',
      metadata: {
        id: 'parameter-id',
        variable: 'value',
        property: '',
        label: '',
        required: false,
        multiple_values: false,
        read_only: false
      },
      minimum: '1,5',
      maximum: '   ',
      defaultValue: '2,75',
      allowedValues: ['3.5', '4,5', 'bad']
    };
    const request = toDatasetSaveRequest(
      formWithParameters([numericParameter]),
      2,
      datasetMetadata()
    );
    const distribution = request.metadata.distributions?.[0];

    expect(distribution).toMatchObject({
      web_service: {
        parameters: [
          {
            kind: 'FLOAT',
            minimum: 1.5,
            maximum: null,
            default: 2.75,
            allowed_values: [3.5]
          }
        ]
      }
    });
  });
});

describe('dataset validation helpers', () => {
  it('detects parameter defaults that are missing from allowed values', () => {
    const stringParameter = parameter('fallback');
    stringParameter.allowedValues = ['first', 'second'];

    expect(hasParameterDefaultOutsideAllowedValues(formWithParameters([stringParameter]))).toBe(true);

    stringParameter.allowedValues.push('fallback');
    expect(hasParameterDefaultOutsideAllowedValues(formWithParameters([stringParameter]))).toBe(false);
  });

  it('detects required null and blank string defaults but accepts non-required or false defaults', () => {
    expect(hasRequiredParameterWithoutDefault(formWithParameters([parameter(null)]))).toBe(true);
    expect(hasRequiredParameterWithoutDefault(formWithParameters([parameter('   ')]))).toBe(true);
    expect(hasRequiredParameterWithoutDefault(formWithParameters([parameter('', false)]))).toBe(false);

    const booleanParameter: DatasetWebServiceParameterForm = {
      kind: 'BOOLEAN',
      metadata: { ...parameter(null).metadata, required: true },
      defaultValue: false
    };
    expect(hasRequiredParameterWithoutDefault(formWithParameters([booleanParameter]))).toBe(false);
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
