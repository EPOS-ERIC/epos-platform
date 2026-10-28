import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Group, WriteResult } from '../../generated/backofficeSchemas';
import { GroupsStore } from '../stores/groups.store';
import { CurrentUserStore } from '../stores/current-user.store';
import { ApiService } from './api.service';
import { ResourceActionsService } from './resource-actions.service';
import { ResourceCreationDialogService } from './resource-creation-dialog.service';

describe('ResourceCreationDialogService', () => {
  const targetGroup: Group = { id: 'target-group', name: 'Target Group', position: 0, concept: 'DATA' };
  const copyResult: WriteResult = { id: 'copied-resource', resourceType: 'dataset', status: 'DRAFT' };
  const apiService = { createResourceDraft: vi.fn(), copyResource: vi.fn() };
  const resourceActions = { resolveCreatedResourceRoute: vi.fn() };
  const currentUserStore = { hasGroupPermission: vi.fn() };
  const router = { navigate: vi.fn() };
  let service: ResourceCreationDialogService;

  beforeEach(() => {
    vi.clearAllMocks();
    apiService.copyResource.mockReturnValue(of(copyResult));
    resourceActions.resolveCreatedResourceRoute.mockReturnValue(of(['/', 'data', 'copied-resource', 'version-id']));
    currentUserStore.hasGroupPermission.mockReturnValue(true);
    TestBed.configureTestingModule({
      providers: [
        ResourceCreationDialogService,
        { provide: ApiService, useValue: apiService },
        { provide: GroupsStore, useValue: { groups: signal([targetGroup]), loading: signal(false) } },
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: ResourceActionsService, useValue: resourceActions },
        { provide: Router, useValue: router }
      ]
    });
    service = TestBed.inject(ResourceCreationDialogService);
  });

  it('copies the published resource into the selected target group and opens its draft', () => {
    service.openCopy('dataset', 'source-resource', 'target-group', 'data');

    expect(service.openState()).toBe(true);
    expect(service.isCopy()).toBe(true);
    expect(service.groupId()).toBe('target-group');

    service.submit();

    expect(apiService.copyResource).toHaveBeenCalledWith('target-group', 'dataset', 'source-resource');
    expect(resourceActions.resolveCreatedResourceRoute).toHaveBeenCalledWith(copyResult, 'target-group', 'data');
    expect(router.navigate).toHaveBeenCalledWith(
      ['/', 'data', 'copied-resource', 'version-id'],
      { replaceUrl: true, queryParams: { groupId: 'target-group', dataType: 'dataset' } }
    );
    expect(service.openState()).toBe(false);
  });
});
