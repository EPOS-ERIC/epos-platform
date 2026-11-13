import { ApplicationConfig, inject, provideAppInitializer, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { OAuthStorage, provideOAuthClient } from 'angular-oauth2-oidc';

import { AuthService } from '../services/auth.service';
import { LocalStorageService } from '../services/local-storage.service';
import { CurrentUserStore } from '../stores/current-user.store';
import { GroupsStore } from '../stores/groups.store';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideHttpClient(),
    provideRouter(routes),
    provideOAuthClient(),
    { provide: OAuthStorage, useExisting: LocalStorageService },
    provideAppInitializer(() => inject(AuthService).initialize()),
    provideAppInitializer(() => inject(CurrentUserStore).initialize()),
    provideAppInitializer(() => inject(GroupsStore).initialize())
  ]
};
