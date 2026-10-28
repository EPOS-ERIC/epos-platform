import { Injectable, signal } from '@angular/core';
import { AuthConfig, OAuthService } from 'angular-oauth2-oidc';

import { LocalStorageService } from './local-storage.service';

const LIVE_AUTH_ISSUER = 'https://aaai.epos-eu.org/oauth2';
const LOCAL_AUTH_ISSUER = 'http://localhost:35000/oauth2';
const AUTH_ISSUER_KEY = 'epos-backoffice-auth-issuer';
const RETURN_URL_KEY = 'epos-backoffice-return-url';

const authRootUrl = '__AUTH_ROOT_URL__';

@Injectable({ providedIn: 'root' })
export class AuthService {
  readonly authenticated = signal(false);
  readonly initializationError = signal<string | null>(null);

  private initialization?: Promise<void>;
  private readonly isLocalDevelopment = ['localhost', '127.0.0.1'].includes(window.location.hostname);
  private readonly issuer = authRootUrl && !authRootUrl.startsWith('__AUTH_')
    ? `${authRootUrl.replace(/\/$/, '')}/oauth2`
    : (this.isLocalDevelopment ? LOCAL_AUTH_ISSUER : LIVE_AUTH_ISSUER);

  constructor(
    private readonly oauthService: OAuthService,
    private readonly localStorage: LocalStorageService
  ) {
    this.oauthService.events.subscribe(() => this.authenticated.set(this.oauthService.hasValidAccessToken()));
  }

  initialize(): Promise<void> {
    this.initialization ??= this.initializeOAuth();
    return this.initialization;
  }

  login(returnUrl = '/data'): void {
    sessionStorage.setItem(RETURN_URL_KEY, this.safeReturnUrl(returnUrl));
    this.oauthService.initImplicitFlow();
  }

  logout(): void {
    sessionStorage.removeItem(RETURN_URL_KEY);
    this.oauthService.logOut(true);
    this.authenticated.set(false);
    window.location.assign(new URL('login', document.baseURI).toString());
  }

  consumeReturnUrl(): string {
    const returnUrl = this.safeReturnUrl(sessionStorage.getItem(RETURN_URL_KEY) ?? '/data');
    sessionStorage.removeItem(RETURN_URL_KEY);
    return returnUrl;
  }

  get accessToken(): string {
    return this.oauthService.hasValidAccessToken() ? this.oauthService.getAccessToken() : '';
  }

  private async initializeOAuth(): Promise<void> {
    const previousIssuer = this.localStorage.get(AUTH_ISSUER_KEY);
    const tokenIssuer = (this.oauthService.getIdentityClaims() as { iss?: string } | null)?.iss;

    this.oauthService.configure(this.makeAuthConfig());
    if ((previousIssuer && previousIssuer !== this.issuer) || (tokenIssuer && tokenIssuer !== this.issuer)) {
      this.oauthService.logOut(true);
    }
    this.localStorage.set(AUTH_ISSUER_KEY, this.issuer);

    const { JwksValidationHandler } = await import('angular-oauth2-oidc-jwks');
    this.oauthService.tokenValidationHandler = new JwksValidationHandler();
    this.oauthService.setupAutomaticSilentRefresh();

    try {
      await this.oauthService.loadDiscoveryDocumentAndTryLogin();
      this.authenticated.set(this.oauthService.hasValidAccessToken());
    } catch (error) {
      console.error('Failed to initialize OAuth', error);
      this.initializationError.set('Authentication service is currently unavailable.');
    }
  }

  private makeAuthConfig(): AuthConfig {
    const callbackBase = new URL(document.baseURI);

    return {
      issuer: this.issuer,
      redirectUri: new URL('last-page-redirect', callbackBase).toString(),
      silentRefreshRedirectUri: new URL('silent-token-refresh.html', callbackBase).toString(),
      clientId: 'eposICS',
      responseType: 'id_token token',
      timeoutFactor: 1,
      scope: ['openid', 'profile', 'single-logout'].join(' '),
      disableAtHashCheck: true
    };
  }

  private safeReturnUrl(returnUrl: string): string {
    return returnUrl.startsWith('/') && !returnUrl.startsWith('//') ? returnUrl : '/data';
  }
}
