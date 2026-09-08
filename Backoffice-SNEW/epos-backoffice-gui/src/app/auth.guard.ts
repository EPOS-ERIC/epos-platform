import { inject } from '@angular/core';
import { Router, type CanActivateFn } from '@angular/router';

import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = async (_route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  await authService.initialize();

  return authService.authenticated() ? true : router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
};
