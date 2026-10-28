import { TestBed } from '@angular/core/testing';
import { Observable, Subject, of } from 'rxjs';
import { vi } from 'vitest';

import type { DiscoveryResourceType, ResourceVersion } from '../../generated/backofficeSchemas';
import type { ResourceVersionComparisonContext } from '../models/resource-version-comparison';
import { ApiService, type ResourceVersionResponse } from '../services/api.service';
import { ResourceVersionComparisonStore } from './resource-version-comparison.store';

describe('ResourceVersionComparisonStore', () => {
  const apiService = { getResourceVersion: vi.fn() };
  let store: ResourceVersionComparisonStore;

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      providers: [
        ResourceVersionComparisonStore,
        { provide: ApiService, useValue: apiService }
      ]
    });
    store = TestBed.inject(ResourceVersionComparisonStore);
  });

  it('loads both versions and exposes their side-by-side diff', () => {
    const leftVersion = version('left', { revision: 1 });
    const rightVersion = version('right', { revision: 2 });
    rightVersion.metadata = { ...rightVersion.metadata, title: 'Updated dataset' };
    apiService.getResourceVersion
      .mockReturnValueOnce(of(leftVersion))
      .mockReturnValueOnce(of(rightVersion));

    store.load(context());

    expect(apiService.getResourceVersion).toHaveBeenNthCalledWith(1, 'dataset', 'resource-id', 'left');
    expect(apiService.getResourceVersion).toHaveBeenNthCalledWith(2, 'dataset', 'resource-id', 'right');
    expect(store.loading()).toBe(false);
    expect(store.error()).toBe('');
    expect(store.leftVersion()?.version_id).toBe('left');
    expect(store.rightVersion()?.version_id).toBe('right');
    expect(store.diffRows().find((row) => row.path === 'Title')?.changed).toBe(true);
    expect(store.diffRows().some((row) => row.path === 'Revision')).toBe(false);
    expect(store.diffSections()[0]).toMatchObject({
      title: 'General Information',
      changedCount: 1
    });
    expect(store.diffSections().map((section) => section.title)).toEqual([
      'General Information',
      'Spatial Coverage',
      'Temporal Coverage',
      'Persistent Identifiers',
      'Contact Points',
      'Providers',
      'Categories',
      'Distributions'
    ]);
    expect(store.changedCount()).toBeGreaterThan(0);
  });

  it('places the oldest version on the left regardless of query parameter order', () => {
    const newerVersion = version('newer', {
      created_at: '2026-02-01T00:00:00Z',
      updated_at: '2026-02-01T00:00:00Z'
    });
    const olderVersion = version('older', {
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-03-01T00:00:00Z'
    });
    apiService.getResourceVersion
      .mockReturnValueOnce(of(newerVersion))
      .mockReturnValueOnce(of(olderVersion));

    store.load(context());

    expect(store.leftVersion()?.version_id).toBe('older');
    expect(store.rightVersion()?.version_id).toBe('newer');
  });

  it('cancels both requests from an earlier comparison and ignores stale emissions', () => {
    const firstLeft = trackedRequest();
    const firstRight = trackedRequest();
    const currentLeft = new Subject<ResourceVersionResponse<DiscoveryResourceType>>();
    const currentRight = new Subject<ResourceVersionResponse<DiscoveryResourceType>>();
    apiService.getResourceVersion
      .mockReturnValueOnce(firstLeft.observable)
      .mockReturnValueOnce(firstRight.observable)
      .mockReturnValueOnce(currentLeft)
      .mockReturnValueOnce(currentRight);

    store.load(context());
    store.load({ ...context(), leftVersionId: 'current-left', rightVersionId: 'current-right' });

    expect(firstLeft.unsubscribed()).toBe(true);
    expect(firstRight.unsubscribed()).toBe(true);
    firstLeft.subject.next(version('stale-left'));
    firstRight.subject.next(version('stale-right'));
    currentLeft.next(version('current-left'));
    currentLeft.complete();
    currentRight.next(version('current-right'));
    currentRight.complete();

    expect(store.leftVersion()?.version_id).toBe('current-left');
    expect(store.rightVersion()?.version_id).toBe('current-right');
  });

  function context(): ResourceVersionComparisonContext {
    return {
      resourceType: 'dataset',
      resourceId: 'resource-id',
      leftVersionId: 'left',
      rightVersionId: 'right',
      sourceVersionId: 'source',
      groupId: 'group-id',
      sectionPath: 'data'
    };
  }

  function version(versionId: string, overrides: Partial<ResourceVersion> = {}): ResourceVersionResponse<'dataset'> {
    return {
      resource_id: 'resource-id',
      version_id: versionId,
      revision: 1,
      status: 'DRAFT',
      created_by: 'user',
      group_id: 'group-id',
      group_name: 'Group',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-02T00:00:00Z',
      metadata: {
        resource_type: 'dataset',
        id: 'resource-id',
        title: 'Dataset',
        description: 'Description',
        keywords: [],
        distributions: [],
        persistent_identifiers: [],
        contact_points: [],
        providers: [],
        categories: []
      },
      ...overrides
    } as ResourceVersionResponse<'dataset'>;
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
