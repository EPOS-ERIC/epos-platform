import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { vi } from 'vitest';

import { CurrentUserStore } from '../../stores/current-user.store';
import { GlobalManagePageComponent } from './global-manage-page.component';

describe('GlobalManagePageComponent', () => {
  const currentUserStore = { hasGlobalPermission: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      imports: [GlobalManagePageComponent],
      providers: [
        { provide: CurrentUserStore, useValue: currentUserStore },
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

  it('shows the categories destination to a category scheme manager', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.category_scheme.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.detectChanges();

    expect(destinationTitles(fixture.nativeElement)).toEqual(['Categories']);
  });

  it('opens the contact point placeholder from a deep link', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.contact_point.manage'
    );
    const fixture = TestBed.createComponent(GlobalManagePageComponent);
    fixture.componentRef.setInput('panel', 'contact-points');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.management-landing')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Contact point management will be defined');
  });

  function destinationTitles(element: HTMLElement): string[] {
    return Array.from<HTMLElement>(element.querySelectorAll('.management-destination h3'))
      .map((title) => title.textContent?.trim() ?? '');
  }
});
