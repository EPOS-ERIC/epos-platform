import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router, convertToParamMap, type ActivatedRouteSnapshot, type RouterStateSnapshot } from '@angular/router';
import { vi } from 'vitest';

import type { User } from '../../generated/backofficeSchemas';
import { CurrentUserStore } from '../stores/current-user.store';
import { GroupsStore } from '../stores/groups.store';
import { groupAdminGuard } from './group-admin.guard';

describe('groupAdminGuard', () => {
  const user: User = {
    id: 'user-id',
    email: 'user@example.org',
    given_name: 'Group',
    family_name: 'Admin',
    status: 'ACTIVE',
    global_roles: [],
    global_granted_permissions: [],
    global_role_permissions: [],
    memberships: [{
      group: { id: 'group-id', name: 'Data Group' },
      status: 'ACTIVE',
      roles: ['GROUP_ADMIN'],
      granted_permissions: [],
      role_permissions: [],
      role_requests: [],
      created_at: '2026-09-14T08:00:00Z',
      updated_at: '2026-09-14T08:00:00Z'
    }],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z'
  };
  const currentUser = signal<User | null>(user);
  const currentUserStore = { user: currentUser, initialize: vi.fn() };
  const groupsStore = {
    groups: signal([{ id: 'group-id', name: 'Data Group', position: 0, concept: 'DATA' as const }]),
    initialize: vi.fn()
  };
  const router = { createUrlTree: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    currentUser.set(user);
    currentUserStore.initialize.mockResolvedValue(undefined);
    groupsStore.initialize.mockResolvedValue(undefined);
    router.createUrlTree.mockReturnValue('redirect');
    TestBed.configureTestingModule({
      providers: [
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: GroupsStore, useValue: groupsStore },
        { provide: Router, useValue: router }
      ]
    });
  });

  it('allows an active GROUP_ADMIN membership for the selected group', async () => {
    const result = await TestBed.runInInjectionContext(() => groupAdminGuard(route('group-id'), state()));

    expect(result).toBe(true);
  });

  it('redirects users without GROUP_ADMIN in the selected group', async () => {
    const result = await TestBed.runInInjectionContext(() => groupAdminGuard(route('another-group'), state()));

    expect(result).toBe('redirect');
    expect(router.createUrlTree).toHaveBeenCalledWith(['/', 'data'], { queryParams: { groupId: 'another-group' } });
  });

  function route(groupId: string): ActivatedRouteSnapshot {
    return {
      data: { sectionPath: 'data' },
      queryParamMap: convertToParamMap({ groupId })
    } as unknown as ActivatedRouteSnapshot;
  }

  function state(): RouterStateSnapshot {
    return { url: `/data/manage-group?groupId=group-id` } as RouterStateSnapshot;
  }
});
