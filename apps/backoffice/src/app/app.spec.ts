import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { provideOAuthClient } from 'angular-oauth2-oidc';
import { App } from './app';
import { routes } from './app.routes';

describe('App', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [provideHttpClient(), provideOAuthClient(), provideRouter([])],
    }).compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });

  it('should render global outlets', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('router-outlet')).toBeTruthy();
    expect(compiled.querySelector('app-dialog-outlet')).toBeTruthy();
    expect(compiled.querySelector('app-resource-creation-dialog')).toBeTruthy();
    expect(compiled.querySelector('app-toast-outlet')).toBeTruthy();
  });

  it('defines the authenticated all-resources route', () => {
    const route = routes.find((candidate) => candidate.path === 'all');
    expect(route?.data).toMatchObject({ sectionTitle: 'All Resources', allGroups: true });
    expect(route?.canActivate?.length).toBeGreaterThan(0);
  });
});
