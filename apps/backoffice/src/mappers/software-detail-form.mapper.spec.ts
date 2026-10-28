import type {
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata
} from '../../generated/backofficeSchemas';
import { describe, expect, it } from 'vitest';
import {
  isSoftwareDescriptionValid,
  toSoftwareApplicationSaveRequest,
  toSoftwareSourceCodeSaveRequest
} from './software-detail-form.mapper';

function applicationForm(): SoftwareApplicationResourceMetadata {
  return {
    resource_type: 'software-application',
    id: 'application-id',
    name: '',
    description: 'Application description',
    keywords: ['catalogue'],
    license: '   ',
    version: '1.0',
    download_url: null,
    requirements: '',
    operating_system: 'Linux',
    main_entity_of_page: ' ',
    creators: [
      { id: 'creator-id', name: 'Creator' },
      { id: '', name: 'Ignored' }
    ],
    identifiers: [
      { id: 'identifier-id', type: 'DOI', value: '10.123/example' },
      { id: 'ignored-id', type: ' ', value: 'ignored' }
    ],
    contact_points: ['contact-id', ' '],
    categories: [
      { id: 'category-id', name: 'Category' },
      { id: ' ', name: 'Ignored' }
    ]
  };
}

function sourceCodeForm(): SoftwareSourceCodeResourceMetadata {
  return {
    resource_type: 'software-source-code',
    id: 'source-id',
    name: '',
    description: 'Source description',
    programming_language: 'TypeScript',
    keywords: ['source'],
    repository: ' ',
    license: null,
    software_requirements: '',
    runtime_platform: 'Node.js',
    main_entity_of_page: null,
    version: '2.0',
    creators: [{ id: 'creator-id', name: 'Creator' }],
    identifiers: [{ id: 'ignored-id', type: 'URL', value: ' ' }],
    contact_points: null,
    categories: [{ id: '', name: 'Ignored' }]
  };
}

describe('software save request mappers', () => {
  it('applies the application name fallback and current blank/reference filtering', () => {
    expect(toSoftwareApplicationSaveRequest(applicationForm(), 4)).toEqual({
      resource_type: 'software-application',
      revision: 4,
      metadata: {
        name: 'Untitled software application',
        description: 'Application description',
        keywords: ['catalogue'],
        license: undefined,
        version: '1.0',
        download_url: undefined,
        requirements: undefined,
        operating_system: 'Linux',
        main_entity_of_page: undefined,
        creators: ['creator-id'],
        identifiers: [{ type: 'DOI', value: '10.123/example' }],
        categories: ['category-id'],
        contact_points: ['contact-id']
      }
    });
  });

  it('applies the source-code name fallback and current blank/reference filtering', () => {
    expect(toSoftwareSourceCodeSaveRequest(sourceCodeForm(), 5)).toEqual({
      resource_type: 'software-source-code',
      revision: 5,
      metadata: {
        name: 'Untitled software source code',
        description: 'Source description',
        keywords: ['source'],
        repository: undefined,
        programming_language: 'TypeScript',
        license: undefined,
        software_requirements: undefined,
        runtime_platform: 'Node.js',
        main_entity_of_page: undefined,
        version: '2.0',
        creators: ['creator-id'],
        identifiers: [],
        categories: [],
        contact_points: []
      }
    });
  });
});

describe('isSoftwareDescriptionValid', () => {
  it('requires at least one non-whitespace description character for both software forms', () => {
    const application = applicationForm();
    const sourceCode = sourceCodeForm();

    expect(isSoftwareDescriptionValid(application)).toBe(true);
    expect(isSoftwareDescriptionValid(sourceCode)).toBe(true);

    application.description = '   ';
    sourceCode.description = '';
    expect(isSoftwareDescriptionValid(application)).toBe(false);
    expect(isSoftwareDescriptionValid(sourceCode)).toBe(false);
  });
});
