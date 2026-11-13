import { TestBed } from '@angular/core/testing';
import { Subject } from 'rxjs';
import { OAuthService } from 'angular-oauth2-oidc';
import { vi } from 'vitest';

import { AuthService } from './auth.service';
import { LocalStorageService } from './local-storage.service';

describe('AuthService', () => {
  const oauthService = {
    events: new Subject(),
    setStorage: vi.fn(),
    hasValidAccessToken: vi.fn(() => false),
    initImplicitFlow: vi.fn()
  };
  const localStorageService = {
    get: vi.fn(),
    set: vi.fn(() => true),
    remove: vi.fn(),
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn()
  };

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      providers: [
        AuthService,
        { provide: OAuthService, useValue: oauthService },
        { provide: LocalStorageService, useValue: localStorageService }
      ]
    });
  });

  it('configures OAuth and return URL persistence with local storage', () => {
    const service = TestBed.inject(AuthService);

    expect(oauthService.setStorage).toHaveBeenCalledWith(localStorageService);

    service.login('/software');
    expect(localStorageService.set).toHaveBeenCalledWith('epos-backoffice-return-url', '/software');
    expect(oauthService.initImplicitFlow).toHaveBeenCalled();
  });
});
