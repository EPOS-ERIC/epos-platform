import { TestBed } from '@angular/core/testing';
import { Router, type ActivatedRouteSnapshot, type RouterStateSnapshot } from '@angular/router';
import { vi } from 'vitest';

import { CurrentUserStore } from '../stores/current-user.store';
import { globalManageGuard } from './global-manage.guard';

describe('globalManageGuard', () => {
  const currentUserStore = {
    initialize: vi.fn(),
    canAccessGlobalManage: vi.fn()
  };
  const router = { createUrlTree: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    currentUserStore.initialize.mockResolvedValue(undefined);
    router.createUrlTree.mockReturnValue('redirect');
    TestBed.configureTestingModule({
      providers: [
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: Router, useValue: router }
      ]
    });
  });

  it('redirects when the global management capability is missing', async () => {
    currentUserStore.canAccessGlobalManage.mockReturnValue(false);

    const result = await TestBed.runInInjectionContext(() =>
      globalManageGuard(route(), state())
    );

    expect(result).toBe('redirect');
    expect(router.createUrlTree).toHaveBeenCalledWith(['/data']);
  });

  it('allows the global management landing when any global capability is granted', async () => {
    currentUserStore.canAccessGlobalManage.mockReturnValue(true);

    const result = await TestBed.runInInjectionContext(() =>
      globalManageGuard(route(), state())
    );

    expect(result).toBe(true);
    expect(currentUserStore.canAccessGlobalManage).toHaveBeenCalledOnce();
  });

  function route(): ActivatedRouteSnapshot {
    return { data: { globalManage: true } } as unknown as ActivatedRouteSnapshot;
  }

  function state(): RouterStateSnapshot {
    return { url: '/global-manage' } as RouterStateSnapshot;
  }
});
