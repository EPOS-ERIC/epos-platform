import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { User } from '../../generated/backofficeSchemas';
import { ApiService } from '../services/api.service';
import { AuthService } from '../services/auth.service';
import { CurrentUserStore } from './current-user.store';

describe('CurrentUserStore', () => {
  const currentUser: User = {
    id: 'user-id',
    email: 'admin@example.org',
    given_name: 'Admin',
    family_name: 'User',
    status: 'ACTIVE',
    global_roles: ['ADMIN'],
    global_granted_permissions: [],
    global_role_permissions: [],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z',
    memberships: []
  };
  const authenticated = signal(true);
  const authService = {
    authenticated,
    initialize: vi.fn().mockResolvedValue(undefined)
  };
  const apiService = {
    getUser: vi.fn()
  };

  beforeEach(() => {
    vi.clearAllMocks();
    authenticated.set(true);
    apiService.getUser.mockReturnValue(of(currentUser));
    TestBed.configureTestingModule({
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: AuthService, useValue: authService }
      ]
    });
  });

  it('loads the authenticated user through the self alias after OAuth initialization', async () => {
    const store = TestBed.inject(CurrentUserStore);

    await store.initialize();

    expect(authService.initialize).toHaveBeenCalledOnce();
    expect(apiService.getUser).toHaveBeenCalledWith('self');
    expect(store.user()).toEqual(currentUser);
    expect(store.isAdmin()).toBe(true);
    expect(store.loading()).toBe(false);
    expect(store.error()).toBeNull();
  });

  it('does not request a user when authentication is unavailable', async () => {
    authenticated.set(false);
    const store = TestBed.inject(CurrentUserStore);

    await store.initialize();

    expect(apiService.getUser).not.toHaveBeenCalled();
    expect(store.user()).toBeNull();
    expect(store.isAdmin()).toBe(false);
  });

  it('combines direct and role permissions for an active group membership', () => {
    const store = TestBed.inject(CurrentUserStore);
    store.user.set({
      ...currentUser,
      global_roles: [],
      memberships: [{
        group: { id: 'group-id', name: 'Data Group' },
        status: 'ACTIVE',
        roles: ['EDITOR'],
        granted_permissions: ['group.resource.submit'],
        role_permissions: ['group.resource.create', 'group.resource.edit'],
        role_requests: [],
        created_at: '2026-09-14T08:00:00Z',
        updated_at: '2026-09-14T08:00:00Z'
      }]
    });

    expect(store.groupPermissions('group-id')).toEqual([
      'group.resource.submit',
      'group.resource.create',
      'group.resource.edit'
    ]);
    expect(store.hasGroupPermission('group-id', 'group.resource.edit')).toBe(true);
    expect(store.hasGroupPermission('another-group', 'group.resource.edit')).toBe(false);
  });

  it('grants every resource permission to global administrators', () => {
    const store = TestBed.inject(CurrentUserStore);
    store.user.set(currentUser);

    expect(store.hasGroupPermission('group-id', 'group.resource.publish')).toBe(true);
    expect(store.hasGroupPermission('group-id', 'group.resource.unpublish')).toBe(true);
    expect(store.hasGroupPermission('group-id', 'group.category.manage')).toBe(true);
  });

  it('applies matching global permissions to every group', () => {
    const store = TestBed.inject(CurrentUserStore);
    store.user.set({
      ...currentUser,
      global_roles: [],
      global_granted_permissions: ['global.resource.publish'],
      global_role_permissions: ['global.category.manage']
    });

    expect(store.groupPermissions('group-id')).toEqual([
      'group.resource.publish',
      'group.category.manage'
    ]);
  });

  it('ignores permissions from inactive memberships', () => {
    const store = TestBed.inject(CurrentUserStore);
    store.user.set({
      ...currentUser,
      global_roles: [],
      memberships: [{
        group: { id: 'group-id', name: 'Data Group' },
        status: 'PENDING',
        roles: [],
        granted_permissions: ['group.resource.create'],
        role_permissions: ['group.resource.edit'],
        role_requests: [],
        created_at: '2026-09-14T08:00:00Z',
        updated_at: '2026-09-14T08:00:00Z'
      }]
    });

    expect(store.groupPermissions('group-id')).toEqual([]);
  });
});
