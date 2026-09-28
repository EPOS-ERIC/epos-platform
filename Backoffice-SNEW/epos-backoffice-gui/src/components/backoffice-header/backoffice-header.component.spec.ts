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

  it('renders the management context and complete back route in a single header', () => {
    const fixture = TestBed.createComponent(BackofficeHeaderComponent);
    fixture.componentRef.setInput('managementContext', 'global');
    fixture.componentRef.setInput('managementEyebrow', 'Platform Administration');
    fixture.componentRef.setInput('managementTitle', 'Global Administration');
    fixture.componentRef.setInput('managementSubtitle', 'Choose an area to manage.');
    fixture.componentRef.setInput('managementBackRoute', '/software?groupId=group-id');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('h1')?.textContent?.trim()).toBe('Global Administration');
    expect(fixture.nativeElement.querySelector('.management-eyebrow')?.textContent?.trim()).toBe('Platform Administration');
    expect((fixture.nativeElement.querySelector('.header-manage-back a') as HTMLAnchorElement).getAttribute('href')).toBe('/software?groupId=group-id');
  });
});
