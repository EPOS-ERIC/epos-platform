import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Group, GroupMembership, User } from '../../generated/backofficeSchemas';
import { ApiService } from './api.service';
import { CurrentUserStore } from '../stores/current-user.store';
import { GroupsStore } from '../stores/groups.store';
import { MembershipRequestDialogService } from './membership-request-dialog.service';
import { ToastService } from './toast.service';

describe('MembershipRequestDialogService', () => {
  const group: Group = { id: 'group-id', name: 'Data Group', position: 0, concept: 'DATA' };
  const membership: GroupMembership = {
    user: { id: 'user-id', given_name: 'Test', family_name: 'User', email: 'user@example.org' },
    status: 'PENDING',
    roles: [],
    granted_permissions: [],
    role_permissions: [],
    role_requests: [],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z'
  };
  const apiService = { requestGroupMembership: vi.fn() };
  const currentUser = signal<User | null>(null);
  const currentUserStore = { user: currentUser, refresh: vi.fn() };
  const toast = { success: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    currentUser.set(null);
    apiService.requestGroupMembership.mockReturnValue(of(membership));
    currentUserStore.refresh.mockResolvedValue(undefined);
    TestBed.configureTestingModule({
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: GroupsStore, useValue: { groups: signal([group]), loading: signal(false) } },
        { provide: ToastService, useValue: toast }
      ]
    });
  });

  it('submits the selected group and role, then refreshes the current user', async () => {
    const dialog = TestBed.inject(MembershipRequestDialogService);
    dialog.open();
    dialog.role.set('REVIEWER');

    dialog.submit();
    await vi.waitFor(() => expect(dialog.saving()).toBe(false));

    expect(apiService.requestGroupMembership).toHaveBeenCalledWith('group-id', { roles: ['REVIEWER'] });
    expect(currentUserStore.refresh).toHaveBeenCalledOnce();
    expect(dialog.openState()).toBe(false);
    expect(toast.success).toHaveBeenCalledWith('Membership request submitted.');
  });

  it('excludes groups with active or pending memberships from new requests', () => {
    currentUser.set({
      id: 'user-id',
      email: 'user@example.org',
      given_name: 'Test',
      family_name: 'User',
      status: 'ACTIVE',
      global_roles: [],
      global_granted_permissions: [],
      global_role_permissions: [],
      created_at: '2026-09-14T08:00:00Z',
      updated_at: '2026-09-14T08:00:00Z',
      memberships: [{
        group: { id: group.id, name: group.name },
        status: 'PENDING',
        roles: [],
        granted_permissions: [],
        role_permissions: [],
        role_requests: [],
        created_at: '2026-09-14T08:00:00Z',
        updated_at: '2026-09-14T08:00:00Z'
      }]
    });

    expect(TestBed.inject(MembershipRequestDialogService).groups()).toEqual([]);
  });
});
