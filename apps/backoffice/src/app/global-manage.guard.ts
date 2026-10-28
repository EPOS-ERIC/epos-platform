import { inject } from '@angular/core';
import { Router, type CanActivateFn } from '@angular/router';

import { CurrentUserStore } from '../stores/current-user.store';

export const globalManageGuard: CanActivateFn = async () => {
  const currentUserStore = inject(CurrentUserStore);
  const router = inject(Router);
  await currentUserStore.initialize();

  return currentUserStore.canAccessGlobalManage()
    ? true
    : router.createUrlTree(['/data']);
};
