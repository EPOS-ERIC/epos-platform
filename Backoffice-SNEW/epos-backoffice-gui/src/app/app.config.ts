import { ApplicationConfig, inject, provideAppInitializer, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { provideOAuthClient } from 'angular-oauth2-oidc';

import { AuthService } from '../services/auth.service';
import { GroupsStore } from '../services/groups.store';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideHttpClient(),
    provideRouter(routes),
    provideOAuthClient(),
    provideAppInitializer(() => inject(AuthService).initialize()),
    provideAppInitializer(() => inject(GroupsStore).initialize())
  ]
};
