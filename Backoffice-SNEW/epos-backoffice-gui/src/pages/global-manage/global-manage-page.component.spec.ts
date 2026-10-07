import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { vi } from 'vitest';
import { of } from 'rxjs';

import { ApiService } from '../../services/api.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GlobalManagePageComponent } from './global-manage-page.component';

describe('GlobalManagePageComponent', () => {
  const currentUserStore = { hasGlobalPermission: vi.fn() };
  const apiService = {
    getContactPoints: vi.fn(() => of([])),
    getGroups: vi.fn(() => of([]))
  };

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      imports: [GlobalManagePageComponent],
      providers: [
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: ApiService, useValue: apiService },
        provideRouter([])
      ]
    });
  });

  it('shows requests and members to a membership manager', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.membership.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.detectChanges();

    expect(destinationTitles(fixture.nativeElement)).toEqual(['Requests', 'Members']);
  });

  it('shows members to an activity reader', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.activity.read'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.detectChanges();

    expect(destinationTitles(fixture.nativeElement)).toEqual(['Members']);
  });

  it('shows the categories destination to a category scheme manager', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.category_scheme.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.detectChanges();

    expect(destinationTitles(fixture.nativeElement)).toEqual(['Groups & Categories']);
  });

  it('shows the categories destination only to a category manager', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.category.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.detectChanges();

    expect(destinationTitles(fixture.nativeElement)).toEqual(['Groups & Categories']);
  });

  it('opens groups management within categories from a deep link', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.category_scheme.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.componentRef.setInput('panel', 'categories');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.management-landing')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Create and manage platform groups');
    expect(apiService.getGroups).toHaveBeenCalled();
  });

  it('shows both groups and categories panels when both permissions are present', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.category_scheme.manage' || permission === 'global.category.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.componentRef.setInput('panel', 'categories');
    fixture.componentRef.setInput('groups', []);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('app-global-groups-panel')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('app-categories-panel')).not.toBeNull();
  });

  it('opens contact point management from a deep link', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.contact_point.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.componentRef.setInput('panel', 'contact-points');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.management-landing')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Create and manage reusable contact points');
    expect(apiService.getContactPoints).toHaveBeenCalled();
  });

  function destinationTitles(element: HTMLElement): string[] {
    return Array.from<HTMLElement>(element.querySelectorAll('.management-destination h3'))
      .map((title) => title.textContent?.trim() ?? '');
  }
});
