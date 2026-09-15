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
    global_permissions: [],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z',
    memberships: [{
      group: { id: '1375500a-fc04-5831-a7f5-39ea5fcb0445', name: 'Data Group' },
      status: 'PENDING',
      roles: [],
      permissions: [],
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
    withdrawGroupMembershipRequest: vi.fn(),
    leaveOwnGroupMembership: vi.fn()
  };
  const dialog = { confirm: vi.fn() };
  const toast = { success: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    currentUser.set(user);
    currentUserStore.refresh.mockResolvedValue(undefined);
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

  it('does not offer Leave to global or group administrators', () => {
    currentUser.set({
      ...user,
      global_roles: ['ADMIN'],
      memberships: [{ ...user.memberships[0], status: 'ACTIVE', roles: ['EDITOR'], role_requests: [] }]
    });
    const fixture = TestBed.createComponent(MembershipsPageComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.membership-remove-button')).toBeNull();

    currentUser.set({
      ...user,
      memberships: [{ ...user.memberships[0], status: 'ACTIVE', roles: ['GROUP_ADMIN'], role_requests: [] }]
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.membership-remove-button')).toBeNull();
  });

});
