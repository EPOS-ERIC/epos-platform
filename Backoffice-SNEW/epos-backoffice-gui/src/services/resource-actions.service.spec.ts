import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Subject, throwError } from 'rxjs';
import { vi } from 'vitest';

import type { WriteResult } from '../../generated/backofficeSchemas';
import type { ResourceVersionContext } from '../models/resource-actions';
import { ApiService } from './api.service';
import { GroupsStore } from './groups.store';
import { ResourceActionsService } from './resource-actions.service';
import { ToastService } from './toast.service';

describe('ResourceActionsService', () => {
  const context: ResourceVersionContext = {
    resourceType: 'dataset',
    resourceId: 'resource-id',
    versionId: 'version-id',
    groupId: 'group-id'
  };
  const apiService = {
    submitResourceVersion: vi.fn()
  };
  const toastService = {
    success: vi.fn()
  };
  let service: ResourceActionsService;

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: GroupsStore, useValue: { groups: signal([]) } },
        { provide: ToastService, useValue: toastService }
      ]
    });
    service = TestBed.inject(ResourceActionsService);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns lifecycle actions allowed by status and role', () => {
    expect(service.sidebarActions('DRAFT', false).map((action) => action.key)).toEqual(['save', 'submit']);
    expect(service.sidebarActions('SUBMITTED', false)).toEqual([]);
    expect(service.sidebarActions('SUBMITTED', true).map((action) => action.key)).toEqual(['publish', 'reject']);
    expect(service.sidebarActions('PUBLISHED', true).map((action) => action.key)).toEqual(['create-new-draft', 'archive']);
  });

  it('tracks a pending action and disables the other lifecycle actions', () => {
    const request = new Subject<WriteResult>();
    apiService.submitResourceVersion.mockReturnValue(request);

    service.submit(context, 'Resource title').subscribe();

    expect(service.saving()).toBe(true);
    expect(service.sidebarActions('DRAFT', false).every((action) => action.disabled)).toBe(true);

    request.next({ id: context.resourceId, resourceType: context.resourceType, status: 'SUBMITTED' });
    request.complete();

    expect(service.saving()).toBe(false);
    expect(toastService.success).toHaveBeenCalledWith('Successfully Submitted Resource title');
  });

  it('exposes action failures and clears the pending state', () => {
    apiService.submitResourceVersion.mockReturnValue(throwError(() => new Error('request failed')));
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    service.submit(context, 'Resource title').subscribe();

    expect(service.error()).toBe('Failed to submit resource version.');
    expect(service.saving()).toBe(false);
    expect(toastService.success).not.toHaveBeenCalled();
  });
});
