import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { Observable, Subject, of, throwError } from 'rxjs';
import { vi } from 'vitest';

import type {
  DatasetResourceMetadata,
  DiscoveryResourceType,
  Group,
  ResourceVersion,
  ResourceVersionSummary,
  SoftwareApplicationResourceMetadata,
  User
} from '../../generated/backofficeSchemas';
import { ApiService, type ResourceVersionResponse } from '../services/api.service';
import { DialogService } from '../services/dialog.service';
import { ResourceActionsService } from '../services/resource-actions.service';
import { ResourceCreationDialogService } from '../services/resource-creation-dialog.service';
import { CurrentUserStore } from './current-user.store';
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
  const apiService = { getResourceVersion: vi.fn(), getUser: vi.fn(), listResourceVersions: vi.fn() };
  const dialogService = { dismiss: vi.fn(), prompt: vi.fn(), confirm: vi.fn() };
  const router = { navigate: vi.fn() };
  const actionError = signal('');
  const currentUserStore = { groupPermissions: vi.fn(() => ['group.resource.edit', 'group.resource.submit', 'group.resource.discard']) };
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
  const copyGroups = signal<Group[]>([]);
  const resourceCreationDialog = { groups: copyGroups, openCopy: vi.fn() };
  let store: ResourceDetailStore;

  beforeEach(() => {
    vi.clearAllMocks();
    actionError.set('');
    copyGroups.set([]);
    apiService.getResourceVersion.mockReturnValue(of(datasetVersion()));
    apiService.getUser.mockReturnValue(of(user()));
    apiService.listResourceVersions.mockReturnValue(of([]));
    TestBed.configureTestingModule({
      providers: [
        ResourceDetailStore,
        { provide: ApiService, useValue: apiService },
        { provide: DialogService, useValue: dialogService },
        { provide: ResourceActionsService, useValue: resourceActions },
        { provide: ResourceCreationDialogService, useValue: resourceCreationDialog },
        { provide: CurrentUserStore, useValue: currentUserStore },
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
    expect(store.datasetForm()?.keywords).toEqual(['earth', 'science']);
    expect(store.title()).toBe('Mapped dataset');
    expect(store.keywords()).toEqual(['earth', 'science']);
    expect(store.navigationItems()[1]?.anchor).toBe('spatial-coverage');
  });

  it('resolves a rejection author for dataset versions', () => {
    apiService.getResourceVersion.mockReturnValue(of(datasetVersion({
      status: 'REJECTED',
      rejected_by: 'reviewer-id',
      rejected_at: '2026-03-01T12:00:00Z',
      rejection_note: 'Please update the metadata.'
    })));

    store.load(context('dataset', 'dataset-id'));

    expect(apiService.getUser).toHaveBeenCalledWith('reviewer-id');
    expect(store.rejectionAuthorName()).toBe('Review User');
  });

  it('resolves a rejection author for software versions', () => {
    apiService.getResourceVersion.mockReturnValue(of(softwareVersion({
      status: 'REJECTED',
      rejected_by: 'reviewer-id',
      rejected_at: '2026-03-01T12:00:00Z',
      rejection_note: 'Please update the metadata.'
    })));

    store.load(context('software-application', 'software-id'));

    expect(apiService.getUser).toHaveBeenCalledWith('reviewer-id');
    expect(store.rejectionAuthorName()).toBe('Review User');
  });

  it('keeps a rejected detail available when its author cannot be resolved', () => {
    apiService.getResourceVersion.mockReturnValue(of(datasetVersion({
      status: 'REJECTED',
      rejected_by: 'missing-user',
      rejection_note: 'Please update the metadata.'
    })));
    apiService.getUser.mockReturnValue(throwError(() => new Error('Not found')));

    store.load(context('dataset', 'dataset-id'));

    expect(store.loadError()).toBe('');
    expect(store.datasetVersion()?.status).toBe('REJECTED');
    expect(store.rejectionAuthorName()).toBe('missing-user');
  });

  it('builds invalid distribution payloads for backend validation', () => {
    store.load(context('dataset', 'dataset-id'));
    store.updateDatasetDistributions([{
      kind: 'downloadable-file',
      title: '',
      description: '',
      issued: '',
      modified: '',
      license: '',
      type: 'DOWNLOADABLE_FILE',
      downloadableFile: { url: '', format: '' }
    }]);

    const saveContext = store.buildSaveContext();
    expect(saveContext).not.toBeNull();
    expect(saveContext?.payload).toMatchObject({
      metadata: {
        distributions: [{ title: '', description: undefined }]
      }
    });
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

  it('derives capabilities from effective group permissions', () => {
    apiService.getResourceVersion.mockReturnValue(of(datasetVersion({ status: 'SUBMITTED' })));
    currentUserStore.groupPermissions.mockReturnValue(['group.resource.publish', 'group.resource.reject']);

    store.load(context('dataset', 'dataset-id'));

    expect(store.capabilities().publish).toBe(true);
    expect(store.capabilities().reject).toBe(true);
    expect(store.canEdit()).toBe(false);
    expect(store.sidebarActions()).toEqual([]);
    expect(resourceActions.sidebarActions).toHaveBeenCalledWith('SUBMITTED', ['group.resource.publish', 'group.resource.reject']);
  });

  it('opens copy selection for a published resource when a writable target exists', () => {
    copyGroups.set([{ id: 'target-group', name: 'Target Group', position: 0, concept: 'DATA' }]);
    apiService.getResourceVersion.mockReturnValue(of(datasetVersion({
      status: 'PUBLISHED',
      resource_id: 'published-resource',
      group_id: 'source-group'
    })));
    store.load({ ...context('dataset', 'published-resource'), groupId: 'source-group', sectionPath: 'data' });

    expect(store.canCopy()).toBe(true);
    store.copy();

    expect(resourceCreationDialog.openCopy).toHaveBeenCalledWith(
      'dataset',
      'published-resource',
      'source-group',
      'data'
    );
  });

  it('enables version history for loaded draft and submitted versions', () => {
    expect(store.canViewHistory()).toBe(false);

    store.load(context('dataset', 'dataset-id'));
    expect(store.canViewHistory()).toBe(true);

    apiService.getResourceVersion.mockReturnValue(of(datasetVersion({ status: 'SUBMITTED' })));
    store.load(context('dataset', 'dataset-id'));
    expect(store.canViewHistory()).toBe(true);

    apiService.getResourceVersion.mockReturnValue(of(datasetVersion({ status: 'PUBLISHED' })));
    store.load(context('dataset', 'dataset-id'));
    expect(store.canViewHistory()).toBe(false);
  });

  it('loads and sorts version history using the loaded resource identity', () => {
    apiService.getResourceVersion.mockReturnValue(
      of(datasetVersion({ resource_id: 'loaded-resource', group_id: 'loaded-group' }))
    );
    apiService.listResourceVersions.mockReturnValue(
      of([
        versionSummary('older', { resource_id: 'loaded-resource', updated_at: '2026-01-01T00:00:00Z' }),
        versionSummary('newer', { resource_id: 'loaded-resource', updated_at: '2026-03-01T00:00:00Z' })
      ])
    );

    store.load({ ...context('dataset', 'route-resource'), groupId: 'route-group' });
    store.openVersionHistory();

    expect(apiService.listResourceVersions).toHaveBeenCalledWith('dataset', 'loaded-resource');
    expect(store.historyOpen()).toBe(true);
    expect(store.historyLoading()).toBe(false);
    expect(store.historyItems().map((item) => item.version_id)).toEqual(['newer', 'older']);
  });

  it('selects at most two versions and enables comparison for exactly two', () => {
    store.load(context('dataset', 'dataset-id'));
    store.toggleVersionSelection('first');
    expect(store.canCompareVersions()).toBe(false);
    store.toggleVersionSelection('second');

    expect(store.selectedVersionIds()).toEqual(['first', 'second']);
    expect(store.canCompareVersions()).toBe(true);
    expect(store.isVersionSelected('first')).toBe(true);
    expect(store.isVersionSelectionDisabled('third')).toBe(true);

    store.toggleVersionSelection('third');
    expect(store.selectedVersionIds()).toEqual(['first', 'second']);

    store.toggleVersionSelection('first');
    expect(store.selectedVersionIds()).toEqual(['second']);
    expect(store.canCompareVersions()).toBe(false);
    expect(store.isVersionSelectionDisabled('third')).toBe(false);
  });

  it('navigates with the oldest selected version on the left and closes history', () => {
    apiService.getResourceVersion.mockReturnValue(
      of(datasetVersion({ resource_id: 'loaded-resource', group_id: 'loaded-group' }))
    );
    apiService.listResourceVersions.mockReturnValue(of([
      versionSummary('version-left', {
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-03-01T00:00:00Z'
      }),
      versionSummary('version-right', {
        created_at: '2026-02-01T00:00:00Z',
        updated_at: '2026-02-01T00:00:00Z'
      })
    ]));
    store.load({ ...context('dataset', 'route-resource'), sectionPath: 'data-section' });
    store.openVersionHistory();
    store.toggleVersionSelection('version-right');
    store.toggleVersionSelection('version-left');

    store.compareSelectedVersions();

    expect(router.navigate).toHaveBeenCalledWith(['/', 'data-section', 'loaded-resource', 'compare'], {
      queryParams: {
        leftVersionId: 'version-left',
        rightVersionId: 'version-right',
        sourceVersionId: 'version-id',
        groupId: 'loaded-group',
        dataType: 'dataset',
        returnUrl: '/data?groupId=group-id'
      }
    });
    expect(store.historyOpen()).toBe(false);
    expect(store.selectedVersionIds()).toEqual([]);
  });

  it('reports history errors and cancels pending history on close and load', () => {
    const first = trackedVersionHistoryRequest();
    apiService.listResourceVersions.mockReturnValueOnce(first.observable);
    store.load(context('dataset', 'dataset-id'));
    store.openVersionHistory();

    store.closeVersionHistory();
    expect(first.unsubscribed()).toBe(true);
    expect(store.historyLoading()).toBe(false);

    apiService.listResourceVersions.mockReturnValueOnce(throwError(() => new Error('network')));
    store.openVersionHistory();
    expect(store.historyError()).toBe('Failed to load version history.');

    const second = trackedVersionHistoryRequest();
    apiService.listResourceVersions.mockReturnValueOnce(second.observable);
    store.openVersionHistory();
    store.load(context('dataset', 'replacement'));
    expect(second.unsubscribed()).toBe(true);
    expect(store.historyOpen()).toBe(false);
    expect(store.historyError()).toBe('');
  });

  function context(resourceType: DiscoveryResourceType, resourceId: string): ResourceDetailRouteContext {
    return {
      resourceType,
      resourceId,
      versionId: 'version-id',
      groupId: 'group-id',
      sectionPath: resourceType === 'dataset' ? 'data' : 'software',
      returnUrl: '/data?groupId=group-id'
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

  function softwareVersion(
    overrides: Partial<ResourceVersion> = {}
  ): ResourceVersionResponse<'software-application'> {
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
      metadata: softwareMetadata,
      ...overrides
    } as ResourceVersionResponse<'software-application'>;
  }

  function user(): User {
    return {
      id: 'reviewer-id',
      email: 'review.user@example.com',
      given_name: 'Review',
      family_name: 'User',
      status: 'ACTIVE',
      global_roles: [],
      global_granted_permissions: [],
      global_role_permissions: [],
      memberships: [],
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    };
  }

  function versionSummary(
    versionId: string,
    overrides: Partial<ResourceVersionSummary> = {}
  ): ResourceVersionSummary {
    return {
      resource_type: 'dataset',
      resource_id: 'dataset-id',
      version_id: versionId,
      revision: 1,
      name: `Version ${versionId}`,
      status: 'DRAFT',
      created_by: 'user',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-02-01T00:00:00Z',
      ...overrides
    } as ResourceVersionSummary;
  }

  function trackedVersionHistoryRequest(): {
    observable: Observable<ResourceVersionSummary[]>;
    unsubscribed: () => boolean;
  } {
    let wasUnsubscribed = false;
    return {
      observable: new Observable(() => () => {
        wasUnsubscribed = true;
      }),
      unsubscribed: () => wasUnsubscribed
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
