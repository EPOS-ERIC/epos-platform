import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { Observable, Subject, of } from 'rxjs';
import { vi } from 'vitest';

import type {
  DatasetResourceMetadata,
  DiscoveryResourceType,
  ResourceVersion,
  SoftwareApplicationResourceMetadata
} from '../../generated/backofficeSchemas';
import { ApiService, type ResourceVersionResponse } from '../services/api.service';
import { DialogService } from '../services/dialog.service';
import { ResourceActionsService } from '../services/resource-actions.service';
import { ResourceDetailStore, type ResourceDetailRouteContext } from './resource-detail.store';

describe('ResourceDetailStore', () => {
  const datasetMetadata: DatasetResourceMetadata = {
    resource_type: 'dataset',
    id: 'dataset-id',
    title: 'Mapped dataset',
    description: 'Dataset description',
    keywords: ['earth', 'science'],
    distributions: [],
    persistent_identifiers: [],
    contact_points: [],
    providers: [],
    categories: []
  };
  const softwareMetadata: SoftwareApplicationResourceMetadata = {
    resource_type: 'software-application',
    id: 'software-id',
    name: 'Software name',
    description: 'Software description',
    keywords: ['tool'],
    creators: [],
    identifiers: [],
    contact_points: [],
    categories: []
  };
  const apiService = { getResourceVersion: vi.fn() };
  const dialogService = { dismiss: vi.fn(), prompt: vi.fn(), confirm: vi.fn() };
  const router = { navigate: vi.fn() };
  const actionError = signal('');
  const resourceActions = {
    error: actionError.asReadonly(),
    saving: signal(false),
    publishing: signal(false),
    creatingDraft: signal(false),
    archiving: signal(false),
    rejecting: signal(false),
    reopening: signal(false),
    deleting: signal(false),
    clearError: vi.fn(() => actionError.set('')),
    setError: vi.fn((message: string) => actionError.set(message)),
    sidebarActions: vi.fn(() => []),
    save: vi.fn(),
    submit: vi.fn(),
    publish: vi.fn(),
    reject: vi.fn(),
    reopen: vi.fn(),
    archive: vi.fn(),
    createDraft: vi.fn(),
    delete: vi.fn()
  };
  let store: ResourceDetailStore;

  beforeEach(() => {
    vi.clearAllMocks();
    actionError.set('');
    apiService.getResourceVersion.mockReturnValue(of(datasetVersion()));
    TestBed.configureTestingModule({
      providers: [
        ResourceDetailStore,
        { provide: ApiService, useValue: apiService },
        { provide: DialogService, useValue: dialogService },
        { provide: ResourceActionsService, useValue: resourceActions },
        { provide: Router, useValue: router }
      ]
    });
    store = TestBed.inject(ResourceDetailStore);
  });

  it('maps dataset state and derives presentation values', () => {
    store.load(context('dataset', 'dataset-id'));

    expect(store.loading()).toBe(false);
    expect(store.datasetVersion()?.status).toBe('DRAFT');
    expect(store.resourceVersion()).toBeNull();
    expect(store.datasetForm()?.title).toBe('Mapped dataset');
    expect(store.datasetForm()?.keywords).toBe('earth, science');
    expect(store.title()).toBe('Mapped dataset');
    expect(store.navigationItems()[1]?.anchor).toBe('spatial-coverage');
  });

  it('blocks invalid distributions before calling the save endpoint', () => {
    store.load(context('dataset', 'dataset-id'));
    store.updateDatasetDistributions([{
      kind: 'downloadable-file',
      title: '',
      description: '',
      issued: '',
      modified: '',
      license: '',
      type: 'http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE',
      downloadableFile: { url: '', format: '' }
    }]);

    expect(store.buildSaveContext()).toBeNull();
    expect(resourceActions.setError).toHaveBeenCalledWith('Distribution 1: title is required.');
  });

  it('cancels the first request when a second load starts', () => {
    const first = trackedRequest();
    const second = new Subject<ResourceVersionResponse<DiscoveryResourceType>>();
    apiService.getResourceVersion.mockReturnValueOnce(first.observable).mockReturnValueOnce(second);

    store.load(context('dataset', 'first'));
    store.load(context('dataset', 'second'));

    expect(first.unsubscribed()).toBe(true);
    first.subject.next(datasetVersion({ resource_id: 'first', metadata: { ...datasetMetadata, title: 'Stale' } }));
    second.next(datasetVersion({ resource_id: 'second', metadata: { ...datasetMetadata, title: 'Current' } }));

    expect(store.datasetVersion()?.resource_id).toBe('second');
    expect(store.title()).toBe('Current');
  });

  it('clear cancels a pending request and prevents stale state', () => {
    const request = trackedRequest();
    apiService.getResourceVersion.mockReturnValue(request.observable);

    store.load(context('dataset', 'dataset-id'));
    expect(store.loading()).toBe(true);
    store.clear();
    request.subject.next(datasetVersion());

    expect(request.unsubscribed()).toBe(true);
    expect(store.loading()).toBe(false);
    expect(store.context()).toBeNull();
    expect(store.datasetVersion()).toBeNull();
  });

  it('builds software application state with an isolated plain-object form', () => {
    const version = softwareVersion();
    apiService.getResourceVersion.mockReturnValue(of(version));

    store.load(context('software-application', 'software-id'));

    expect(store.datasetVersion()).toBeNull();
    expect(store.resourceVersion()).toBe(version);
    expect(store.softwareApplicationForm()).toEqual(softwareMetadata);
    expect(store.softwareApplicationForm()).not.toBe(softwareMetadata);
    expect(store.softwareSourceCodeForm()).toBeNull();
    expect(store.rootLabel()).toBe('Software Application');
  });

  it('derives capabilities from status and route role', () => {
    apiService.getResourceVersion.mockReturnValue(of(datasetVersion({ status: 'SUBMITTED' })));

    store.load({ ...context('dataset', 'dataset-id'), isAdmin: true });

    expect(store.capabilities().publish).toBe(true);
    expect(store.capabilities().reject).toBe(true);
    expect(store.canEdit()).toBe(false);
    expect(store.sidebarActions()).toEqual([]);
    expect(resourceActions.sidebarActions).toHaveBeenCalledWith('SUBMITTED', true);
  });

  function context(resourceType: DiscoveryResourceType, resourceId: string): ResourceDetailRouteContext {
    return {
      resourceType,
      resourceId,
      versionId: 'version-id',
      groupId: 'group-id',
      sectionPath: resourceType === 'dataset' ? 'data' : 'software',
      isAdmin: false
    };
  }

  function datasetVersion(overrides: Partial<ResourceVersion> = {}): ResourceVersionResponse<'dataset'> {
    return {
      resource_id: 'dataset-id',
      version_id: 'version-id',
      revision: 3,
      status: 'DRAFT',
      created_by: 'user',
      group_id: 'group-id',
      group_name: 'Group',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      metadata: datasetMetadata,
      ...overrides
    } as ResourceVersionResponse<'dataset'>;
  }

  function softwareVersion(): ResourceVersionResponse<'software-application'> {
    return {
      resource_id: 'software-id',
      version_id: 'version-id',
      revision: 1,
      status: 'DRAFT',
      created_by: 'user',
      group_id: 'group-id',
      group_name: 'Group',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      metadata: softwareMetadata
    };
  }

  function trackedRequest(): {
    observable: Observable<ResourceVersionResponse<DiscoveryResourceType>>;
    subject: Subject<ResourceVersionResponse<DiscoveryResourceType>>;
    unsubscribed: () => boolean;
  } {
    const subject = new Subject<ResourceVersionResponse<DiscoveryResourceType>>();
    let wasUnsubscribed = false;
    return {
      observable: new Observable((subscriber) => {
        const subscription = subject.subscribe(subscriber);
        return () => {
          wasUnsubscribed = true;
          subscription.unsubscribe();
        };
      }),
      subject,
      unsubscribed: () => wasUnsubscribed
    };
  }
});
