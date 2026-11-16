import { TestBed } from '@angular/core/testing';
import { Subject } from 'rxjs';
import { OAuthService } from 'angular-oauth2-oidc';
import { vi } from 'vitest';

import { AuthService } from './auth.service';
import { CapabilitiesService } from './capabilities.service';
import { LocalStorageService } from './local-storage.service';

describe('AuthService', () => {
  const oauthService = {
    events: new Subject(),
    setStorage: vi.fn(),
    hasValidAccessToken: vi.fn(() => false),
    initImplicitFlow: vi.fn(),
    getIdentityClaims: vi.fn(() => null),
    configure: vi.fn(),
    logOut: vi.fn(),
    setupAutomaticSilentRefresh: vi.fn(),
    loadDiscoveryDocumentAndTryLogin: vi.fn()
  };
  const localStorageService = {
    get: vi.fn(),
    set: vi.fn(() => true),
    remove: vi.fn(),
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn()
  };
  const capabilitiesService = {
    initialize: vi.fn()
  };

  beforeEach(() => {
    vi.clearAllMocks();
    capabilitiesService.initialize.mockResolvedValue({
      authentication: { root_url: 'https://auth.example.org/' },
      notifications: { email_enabled: true },
      organizations: { management: 'local' }
    });
    oauthService.loadDiscoveryDocumentAndTryLogin.mockResolvedValue(undefined);
    TestBed.configureTestingModule({
      providers: [
        AuthService,
        { provide: OAuthService, useValue: oauthService },
        { provide: LocalStorageService, useValue: localStorageService },
        { provide: CapabilitiesService, useValue: capabilitiesService }
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

  it('loads capabilities before configuring the OAuth issuer', async () => {
    const service = TestBed.inject(AuthService);

    await service.initialize();

    expect(capabilitiesService.initialize).toHaveBeenCalledOnce();
    expect(oauthService.configure).toHaveBeenCalledWith(expect.objectContaining({
      issuer: 'https://auth.example.org/oauth2'
    }));
    expect(localStorageService.set).toHaveBeenCalledWith(
      'epos-backoffice-auth-issuer',
      'https://auth.example.org/oauth2'
    );
  });
});
