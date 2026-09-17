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
});
