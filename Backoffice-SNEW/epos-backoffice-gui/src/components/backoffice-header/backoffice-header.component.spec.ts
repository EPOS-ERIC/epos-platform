import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import type { User } from '../../../generated/backofficeSchemas';
import { AuthService } from '../../services/auth.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { BackofficeHeaderComponent } from './backoffice-header.component';

describe('BackofficeHeaderComponent', () => {
  const user: User = {
    id: 'user-id',
    email: 'editor@example.org',
    given_name: 'Test',
    family_name: 'Editor',
    status: 'ACTIVE',
    global_roles: [],
    global_granted_permissions: [],
    global_role_permissions: [],
    created_at: '2026-09-16T08:00:00Z',
    updated_at: '2026-09-16T08:00:00Z',
    memberships: [{
      group: { id: 'group-id', name: 'Data Group' },
      status: 'ACTIVE',
      roles: ['EDITOR'],
      granted_permissions: [],
      role_permissions: [],
      role_requests: [],
      created_at: '2026-09-16T08:00:00Z',
      updated_at: '2026-09-16T08:00:00Z'
    }]
  };
  const currentUser = signal<User | null>(user);

  beforeEach(async () => {
    currentUser.set(user);
    await TestBed.configureTestingModule({
      imports: [BackofficeHeaderComponent],
      providers: [
        { provide: AuthService, useValue: { logout: () => undefined } },
        { provide: CurrentUserStore, useValue: { user: currentUser } },
        provideRouter([])
      ]
    }).compileComponents();
  });

  it('shows the current user name without exposing group roles', () => {
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.admin-copy strong')?.textContent?.trim()).toBe('Test Editor');
    expect(fixture.nativeElement.querySelector('.admin-subtitle')?.textContent?.trim()).toBe('Backoffice User');
    expect(fixture.nativeElement.querySelector('.global-admin-badge')).toBeNull();
  });

  it('shows the ADMIN badge only for a global administrator', () => {
    currentUser.set({ ...user, global_roles: ['ADMIN'] });
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.global-admin-badge')?.textContent?.trim()).toBe('ADMIN');
    expect(fixture.nativeElement.querySelector('.admin-subtitle')).toBeNull();
  });

  it('shows resource creation only when allowed for the selected group', () => {
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    fixture.componentRef.setInput('showSearchActions', true);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).not.toContain('Add New Resource');

    fixture.componentRef.setInput('canAddNewResource', true);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Add New Resource');
  });

  it('searches available authors and emits the selected author filter', async () => {
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    fixture.componentRef.setInput('availableAuthors', [
      { id: 'alice-id', name: 'Alice Editor' },
      { id: 'bob-id', name: 'Bob Reviewer' }
    ]);
    fixture.componentRef.setInput('selectedVersioningStatuses', ['PUBLISHED']);
    const filtersChange = vi.fn();
    fixture.componentInstance.filtersChange.subscribe(filtersChange);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.filter-menu > button') as HTMLButtonElement).click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.author-filter-options')).toBeNull();
    const authorSearch = fixture.nativeElement.querySelector('.author-filter-search') as HTMLInputElement;
    authorSearch.value = 'bob';
    authorSearch.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();

    const authorOptions = Array.from<HTMLButtonElement>(fixture.nativeElement.querySelectorAll('.author-filter-option'));
    expect(authorOptions.map((option) => option.textContent?.trim())).toEqual(['Bob Reviewer']);
    authorOptions[0].click();

    expect(filtersChange).toHaveBeenCalledWith({
      searchQuery: '',
      filterByAuthor: 'bob-id',
      dataTypes: [],
      versioningStatuses: ['PUBLISHED']
    });
  });

  it('opens the main filter panel on the requested section', async () => {
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    fixture.detectChanges();

    fixture.componentInstance.openFilterSection('status');
    fixture.detectChanges();
    await new Promise((resolve) => setTimeout(resolve));

    expect(fixture.nativeElement.querySelector('.filter-panel')).not.toBeNull();
    const statusSection = fixture.nativeElement.querySelector('[data-filter-section="status"]');
    const statusInput = statusSection.querySelector('input');
    expect(document.activeElement).toBe(statusInput);
    expect(statusSection.querySelector('.filter-group-content')?.classList.contains('filter-group-content--highlighted')).toBe(true);
  });

  it('shows concise copy for each main resource section', () => {
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    const cases = [
      ['Data', 'Manage resources and related metadata.'],
      ['Software', 'Manage resources and related metadata.'],
      ['Facilities', 'Manage resources and related metadata.']
    ];

    for (const [title, subtitle] of cases) {
      fixture.componentRef.setInput('sectionTitle', title);
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('.section-subtitle')?.textContent?.trim()).toBe(subtitle);
    }
  });

  it('renders the management context and complete back route in a single header', () => {
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    fixture.componentRef.setInput('managementContext', 'global');
    fixture.componentRef.setInput('managementEyebrow', 'Platform Administration');
    fixture.componentRef.setInput('managementTitle', 'Global Administration');
    fixture.componentRef.setInput('managementSubtitle', 'Choose an area to manage.');
    fixture.componentRef.setInput('managementBackRoute', '/software?groupId=group-id');
    fixture.componentRef.setInput('headerBackRoute', '/software?groupId=group-id');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('h1')?.textContent?.trim()).toBe('Global Administration');
    expect(fixture.nativeElement.querySelector('.management-eyebrow')?.textContent?.trim()).toBe('Platform Administration');
    const backLink = fixture.nativeElement.querySelector('.header-manage-back a') as HTMLAnchorElement;
    expect(backLink.getAttribute('href')).toBe('/software?groupId=group-id');
    expect(backLink.classList.contains('app-button--dark')).toBe(true);
    expect((fixture.nativeElement.querySelector('.header-copy') as HTMLAnchorElement).getAttribute('href')).toBe('/software?groupId=group-id');
    expect(fixture.nativeElement.querySelector('.header-back-affordance')).toBeTruthy();
  });
});
