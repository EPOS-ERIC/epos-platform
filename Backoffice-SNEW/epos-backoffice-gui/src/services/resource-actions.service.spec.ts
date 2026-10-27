import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of, Subject, throwError } from 'rxjs';
import { vi } from 'vitest';

import type { ResourceVersion, WriteResult } from '../../generated/backofficeSchemas';
import type { ResourceVersionContext, SaveResourceContext } from '../models/resource-actions';
import { ApiService } from './api.service';
import { GroupsStore } from '../stores/groups.store';
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
    getResourceVersion: vi.fn(),
    saveResourceVersionDraft: vi.fn(),
    submitResourceVersion: vi.fn()
  };
  const toastService = {
    dismiss: vi.fn(),
    error: vi.fn(),
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

  it('returns only lifecycle actions allowed by their individual permissions', () => {
    expect(service.sidebarActions('DRAFT', ['group.resource.edit']).map((action) => action.key)).toEqual(['save']);
    expect(service.sidebarActions('DRAFT', ['group.resource.submit']).map((action) => action.key)).toEqual(['submit']);
    expect(service.sidebarActions('SUBMITTED', ['group.resource.publish']).map((action) => action.key)).toEqual(['publish']);
    expect(service.sidebarActions('SUBMITTED', ['group.resource.reject']).map((action) => action.key)).toEqual(['reject']);
    expect(service.sidebarActions('PUBLISHED', ['group.resource.create']).map((action) => action.key)).toEqual(['create-new-draft']);
    expect(service.sidebarActions('PUBLISHED', ['group.resource.unpublish']).map((action) => action.key)).toEqual(['archive']);
  });

  it('tracks a pending action and disables the other lifecycle actions', () => {
    const request = new Subject<WriteResult>();
    apiService.submitResourceVersion.mockReturnValue(request);

    service.submit(context, 'Resource title').subscribe();

    expect(service.saving()).toBe(true);
    expect(service.sidebarActions('DRAFT', ['group.resource.edit', 'group.resource.submit']).every((action) => action.disabled)).toBe(true);

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
    expect(toastService.error).toHaveBeenCalledWith('Failed to submit resource version.');
    expect(toastService.success).not.toHaveBeenCalled();
  });

  it('shows a success notification after saving resource changes', () => {
    const saveContext: SaveResourceContext = {
      resourceType: 'software-application',
      resourceId: context.resourceId,
      versionId: context.versionId,
      targetStatus: 'draft',
      payload: {
        resource_type: 'software-application',
        revision: 1,
        metadata: { name: 'Application', description: 'Description' }
      }
    };
    const savedVersion: ResourceVersion = {
      resource_id: context.resourceId,
      version_id: context.versionId,
      revision: 1,
      status: 'DRAFT',
      created_by: 'user-id',
      group_id: context.groupId,
      group_name: 'Group',
      created_at: '2026-10-20T08:00:00Z',
      updated_at: '2026-10-20T08:00:00Z',
      metadata: {
        resource_type: 'software-application',
        id: context.resourceId,
        name: 'Application',
        description: 'Description',
        keywords: [],
        creators: [],
        identifiers: [],
        contact_points: [],
        categories: []
      }
    };
    apiService.saveResourceVersionDraft.mockReturnValue(of(undefined));
    apiService.getResourceVersion.mockReturnValue(of(savedVersion));

    service.save(saveContext).subscribe();

    expect(toastService.success).toHaveBeenCalledWith('Resource changes saved successfully.');
  });

  it('keeps save validation errors visible and includes backend problem details', () => {
    const saveContext: SaveResourceContext = {
      resourceType: 'dataset',
      resourceId: context.resourceId,
      versionId: context.versionId,
      targetStatus: 'draft',
      payload: {
        resource_type: 'dataset',
        revision: 1,
        metadata: { title: 'Dataset', description: 'Description' }
      }
    };
    apiService.saveResourceVersionDraft.mockReturnValue(throwError(() => new HttpErrorResponse({
      status: 400,
      error: {
        type: 'validation_failed',
        title: 'Bad Request',
        status: 400,
        detail: 'Value must be valid WKT geometry.',
        instance: 'trace-id',
        target: 'metadata.spatial_coverage'
      }
    })));
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    service.save(saveContext).subscribe();

    expect(toastService.error).toHaveBeenCalledWith('Failed to save resource version.', {
      duration: null,
      issues: [{
        label: 'Spatial coverage',
        message: 'Value must be valid WKT geometry.',
        reference: 'trace-id',
        target: 'metadata.spatial_coverage'
      }]
    });
  });

  it('shows frontend validation failures as persistent issues', () => {
    service.setValidationError('Distribution 1: title is required.', 'Distributions');

    expect(service.error()).toBe('Distribution 1: title is required.');
    expect(toastService.error).toHaveBeenCalledWith('Resource could not be saved.', {
      duration: null,
      issues: [{
        label: 'Distributions',
        message: 'Distribution 1: title is required.'
      }]
    });
  });
});
