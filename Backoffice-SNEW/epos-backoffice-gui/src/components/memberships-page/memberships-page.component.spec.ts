import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { User } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { MembershipsPageComponent } from './memberships-page.component';

describe('MembershipsPageComponent', () => {
  const user: User = {
    id: 'user-id',
    email: 'user@example.org',
    given_name: 'Test',
    family_name: 'User',
    status: 'ACTIVE',
    global_roles: [],
    global_granted_permissions: [],
    global_role_permissions: [],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z',
    memberships: [{
      group: { id: '1375500a-fc04-5831-a7f5-39ea5fcb0445', name: 'Data Group' },
      status: 'PENDING',
      roles: [],
      granted_permissions: [],
      role_permissions: [],
      role_requests: [{ id: 'request-id', role: 'EDITOR', status: 'PENDING', requested_at: '2026-09-14T08:00:00Z' }],
      created_at: '2026-09-14T08:00:00Z',
      updated_at: '2026-09-14T08:00:00Z'
    }]
  };
  const open = vi.fn();
  const currentUser = signal<User | null>(user);
  const currentUserStore = {
    user: currentUser,
    loading: signal(false),
    error: signal<string | null>(null),
    refresh: vi.fn()
  };
  const apiService = {
    requestGroupRole: vi.fn(),
    withdrawGroupMembershipRequest: vi.fn(),
    leaveOwnGroupMembership: vi.fn()
  };
  const dialog = { confirm: vi.fn() };
  const toast = { success: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    currentUser.set(user);
    currentUserStore.refresh.mockResolvedValue(undefined);
    apiService.requestGroupRole.mockReturnValue(of({
      id: 'role-request-id',
      role: 'EDITOR',
      status: 'PENDING',
      requested_at: '2026-09-14T08:00:00Z'
    }));
    apiService.withdrawGroupMembershipRequest.mockReturnValue(of(undefined));
    apiService.leaveOwnGroupMembership.mockReturnValue(of(undefined));
    dialog.confirm.mockResolvedValue(true);
    TestBed.configureTestingModule({
      imports: [MembershipsPageComponent],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: DialogService, useValue: dialog },
        { provide: MembershipRequestDialogService, useValue: { open, groups: signal([{ id: 'group-id' }]), groupsLoading: signal(false) } },
        { provide: ToastService, useValue: toast }
      ]
    });
  });

  it('shows current memberships and opens a new membership request', () => {
    const fixture = TestBed.createComponent(MembershipsPageComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Data Group');
    expect(fixture.nativeElement.textContent).toContain('PENDING');
    expect(fixture.nativeElement.textContent).toContain('EDITOR');
    expect((fixture.nativeElement.querySelector('.membership-group-logo') as HTMLImageElement).src).toContain('1375500a-fc04-5831-a7f5-39ea5fcb0445.svg');

    (fixture.nativeElement.querySelector('.memberships-page-header button') as HTMLButtonElement).click();
    expect(open).toHaveBeenCalledOnce();
  });

  it('cancels a pending membership request after confirmation', async () => {
    const fixture = TestBed.createComponent(MembershipsPageComponent);
    fixture.detectChanges();

    const button = fixture.nativeElement.querySelector('.membership-remove-button') as HTMLButtonElement;
    expect(button.textContent).toContain('Cancel Request');
    button.click();

    await vi.waitFor(() => expect(apiService.withdrawGroupMembershipRequest).toHaveBeenCalled());
    expect(apiService.withdrawGroupMembershipRequest).toHaveBeenCalledWith('1375500a-fc04-5831-a7f5-39ea5fcb0445');
    expect(currentUserStore.refresh).toHaveBeenCalledOnce();
    expect(toast.success).toHaveBeenCalledWith('Membership request cancelled.');
  });

  it('leaves an active membership using the self endpoint for a regular user', async () => {
    currentUser.set({
      ...user,
      memberships: [{ ...user.memberships[0], status: 'ACTIVE', roles: ['EDITOR'], role_requests: [] }]
    });
    const fixture = TestBed.createComponent(MembershipsPageComponent);
    fixture.detectChanges();

    const button = fixture.nativeElement.querySelector('.membership-remove-button') as HTMLButtonElement;
    expect(button.textContent.trim()).toBe('Leave');
    button.click();

    await vi.waitFor(() => expect(apiService.leaveOwnGroupMembership).toHaveBeenCalled());
    expect(apiService.leaveOwnGroupMembership).toHaveBeenCalledWith('1375500a-fc04-5831-a7f5-39ea5fcb0445');
    expect(currentUserStore.refresh).toHaveBeenCalledOnce();
    expect(toast.success).toHaveBeenCalledWith('Group membership removed.');
  });

  it('offers Leave to global and group administrators for their own membership', () => {
    currentUser.set({
      ...user,
      global_roles: ['ADMIN'],
      memberships: [{ ...user.memberships[0], status: 'ACTIVE', roles: ['EDITOR'], role_requests: [] }]
    });
    const fixture = TestBed.createComponent(MembershipsPageComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.membership-remove-button')?.textContent.trim()).toBe('Leave');

    currentUser.set({
      ...user,
      memberships: [{ ...user.memberships[0], status: 'ACTIVE', roles: ['GROUP_ADMIN'], role_requests: [] }]
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.membership-remove-button')?.textContent.trim()).toBe('Leave');
  });

  it('shows current and pending roles as unavailable in the role request dialog', () => {
    currentUser.set({
      ...user,
      memberships: [{
        ...user.memberships[0],
        status: 'ACTIVE',
        roles: ['EDITOR'],
        role_requests: [{ id: 'request-id', role: 'REVIEWER', status: 'PENDING', requested_at: '2026-09-14T08:00:00Z' }]
      }]
    });
    const fixture = TestBed.createComponent(MembershipsPageComponent);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.membership-role-request-button') as HTMLButtonElement).click();
    fixture.detectChanges();

    const dialog = fixture.nativeElement.querySelector('.role-request-dialog') as HTMLElement;
    const options = dialog.querySelectorAll<HTMLInputElement>('.role-request-option input');
    expect(dialog.textContent).toContain('Current roles');
    expect(dialog.textContent).toContain('Current role');
    expect(dialog.textContent).toContain('Request pending');
    expect(options[0].disabled).toBe(true);
    expect(options[1].disabled).toBe(true);
  });

  it('submits one role request for each selected role', async () => {
    currentUser.set({
      ...user,
      memberships: [{ ...user.memberships[0], status: 'ACTIVE', roles: [], role_requests: [] }]
    });
    const fixture = TestBed.createComponent(MembershipsPageComponent);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.membership-role-request-button') as HTMLButtonElement).click();
    fixture.detectChanges();

    const options = fixture.nativeElement.querySelectorAll('.role-request-option input') as NodeListOf<HTMLInputElement>;
    options[0].click();
    options[1].click();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.role-request-dialog .app-button--primary') as HTMLButtonElement).click();

    await vi.waitFor(() => expect(apiService.requestGroupRole).toHaveBeenCalledTimes(2));
    expect(apiService.requestGroupRole).toHaveBeenCalledWith('1375500a-fc04-5831-a7f5-39ea5fcb0445', { role: 'EDITOR' });
    expect(apiService.requestGroupRole).toHaveBeenCalledWith('1375500a-fc04-5831-a7f5-39ea5fcb0445', { role: 'REVIEWER' });
    await vi.waitFor(() => expect(currentUserStore.refresh).toHaveBeenCalledOnce());
    expect(toast.success).toHaveBeenCalledWith('2 role requests submitted.');
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.role-request-dialog')).toBeNull();
  });

});
