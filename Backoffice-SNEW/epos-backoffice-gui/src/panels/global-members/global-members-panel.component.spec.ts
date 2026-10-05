import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Group, User, UserListItem } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GlobalMembersPanelComponent } from './global-members-panel.component';

describe('GlobalMembersPanelComponent', () => {
  const group: Group = {
    id: 'group-id',
    name: 'Data Group',
    position: 0,
    concept: 'DATA'
  };
  const userSummary: UserListItem = {
    id: 'user-id',
    email: 'user@example.org',
    given_name: 'Test',
    family_name: 'User',
    status: 'ACTIVE',
    global_roles: [],
    global_granted_permissions: [],
    global_role_permissions: [],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z'
  };
  const user: User = { ...userSummary, memberships: [] };
  const currentAdmin = signal<User | null>({
    ...user,
    id: 'admin-id',
    email: 'admin@example.org',
    global_roles: ['ADMIN']
  });
  const apiService = {
    getUsers: vi.fn(),
    getUser: vi.fn(),
    listPermissions: vi.fn(),
    setGroupMembershipRole: vi.fn(),
    setGroupMembershipPermission: vi.fn(),
    removeGroupMember: vi.fn(),
    listNotificationTopics: vi.fn(),
    listUserNotificationSubscriptions: vi.fn(),
    setUserNotificationSubscription: vi.fn(),
    setGroupUserNotificationSubscription: vi.fn(),
    blockUser: vi.fn(),
    unblockUser: vi.fn()
  };
  const dialog = { confirm: vi.fn(), prompt: vi.fn() };
  const toast = { success: vi.fn() };
  const currentUserStore = { hasGlobalPermission: vi.fn(), user: currentAdmin };

  beforeEach(() => {
    vi.clearAllMocks();
    currentAdmin.set({
      ...user,
      id: 'admin-id',
      email: 'admin@example.org',
      global_roles: ['ADMIN']
    });
    currentUserStore.hasGlobalPermission.mockReturnValue(true);
    apiService.getUsers.mockReturnValue(of([]));
    apiService.getUser.mockReturnValue(of(user));
    apiService.listPermissions.mockReturnValue(of([
      { key: 'group.resource.create', label: 'Create resources', description: 'Create resources.', scope: 'GROUP' }
    ]));
    apiService.setGroupMembershipRole.mockReturnValue(of(undefined));
    apiService.setGroupMembershipPermission.mockReturnValue(of(undefined));
    apiService.removeGroupMember.mockReturnValue(of(undefined));
    apiService.listNotificationTopics.mockReturnValue(of([
      {
        key: 'global.resource.published',
        label: 'Global resource publications',
        description: 'Email this user when a resource is published.',
        scope: 'GLOBAL',
        required_permissions: []
      },
      {
        key: 'group.resource.published',
        label: 'Group resource publications',
        description: 'Email this user when a group resource is published.',
        scope: 'GROUP',
        required_permissions: []
      }
    ]));
    apiService.listUserNotificationSubscriptions.mockReturnValue(of([]));
    apiService.setUserNotificationSubscription.mockReturnValue(of(undefined));
    apiService.setGroupUserNotificationSubscription.mockReturnValue(of(undefined));
    apiService.blockUser.mockReturnValue(of(undefined));
    apiService.unblockUser.mockReturnValue(of(undefined));
    dialog.confirm.mockResolvedValue(true);
    dialog.prompt.mockResolvedValue('Policy violation');
    TestBed.configureTestingModule({
      imports: [GlobalMembersPanelComponent],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: DialogService, useValue: dialog },
        { provide: ToastService, useValue: toast },
        { provide: CurrentUserStore, useValue: currentUserStore }
      ]
    });
  });

  it('shows users to a membership-only manager', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.membership.manage'
    );
    apiService.getUsers.mockReturnValue(of([userSummary]));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.users-card')).not.toBeNull();
    expect(fixture.nativeElement.textContent).toContain('user@example.org');
    expect(apiService.getUsers).toHaveBeenCalled();
    expect(apiService.listPermissions).not.toHaveBeenCalled();
  });

  it('manages global and group notifications from the user dialog for a notification-only manager', () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.notification.manage'
    );
    apiService.getUsers.mockReturnValue(of([userSummary]));
    apiService.getUser.mockReturnValue(of({
      ...user,
      memberships: [{
        group: { id: group.id, name: group.name },
        status: 'ACTIVE',
        roles: [],
        granted_permissions: [],
        role_permissions: [],
        role_requests: [],
        created_at: '2026-09-14T08:00:00Z',
        updated_at: '2026-09-14T08:00:00Z'
      }]
    }));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.users-card tbody .app-button') as HTMLButtonElement).click();
    fixture.detectChanges();

    const membershipDialog = fixture.nativeElement.querySelector('.membership-dialog') as HTMLElement;
    expect(membershipDialog.textContent).toContain('Global Notifications');
    expect(membershipDialog.querySelector('.membership-list')).not.toBeNull();
    expect(membershipDialog.querySelector('.membership-notifications')?.textContent).toContain('Group resource publications');
    expect(apiService.listUserNotificationSubscriptions).toHaveBeenCalledWith('user-id');

    (membershipDialog.querySelector('.global-notifications .notification-option input') as HTMLInputElement).click();

    expect(apiService.setUserNotificationSubscription).toHaveBeenCalledWith(
      'user-id',
      'global.resource.published',
      true
    );

    (membershipDialog.querySelector('.membership-notifications .notification-option input') as HTMLInputElement).click();

    expect(apiService.setGroupUserNotificationSubscription).toHaveBeenCalledWith(
      'group-id',
      'user-id',
      'group.resource.published',
      true
    );
  });

  it('blocks an active user with a required reason as an active global administrator', async () => {
    apiService.getUsers.mockReturnValue(of([userSummary]));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    const blockButton = fixture.nativeElement.querySelector('.user-status-button--block') as HTMLButtonElement;
    expect(blockButton.textContent).toContain('Block');
    expect(blockButton.querySelector('app-icon')).not.toBeNull();
    blockButton.click();

    await vi.waitFor(() => expect(apiService.blockUser).toHaveBeenCalledWith('user-id', { reason: 'Policy violation' }));
    expect(dialog.prompt).toHaveBeenCalledWith(expect.objectContaining({ required: true, destructive: true }));
    expect(toast.success).toHaveBeenCalledWith('User blocked.');
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.status-pill')?.textContent).toContain('BLOCKED');
  });

  it('unblocks a blocked user after confirmation as an active global administrator', async () => {
    apiService.getUsers.mockReturnValue(of([{ ...userSummary, status: 'BLOCKED' }]));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    const unblockButton = fixture.nativeElement.querySelector('.user-status-button') as HTMLButtonElement;
    expect(unblockButton.textContent).toContain('Unblock');
    unblockButton.click();

    await vi.waitFor(() => expect(apiService.unblockUser).toHaveBeenCalledWith('user-id'));
    expect(dialog.confirm).toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalledWith('User unblocked.');
  });

  it('does not expose block actions to users without an active global administrator role', () => {
    currentAdmin.set({ ...user, id: 'manager-id', global_roles: [] });
    apiService.getUsers.mockReturnValue(of([userSummary]));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.user-status-button')).toBeNull();
  });

  it('removes an active group membership as a global membership manager', async () => {
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.membership.manage'
    );
    apiService.getUsers.mockReturnValue(of([userSummary]));
    apiService.getUser
      .mockReturnValueOnce(of({
        ...user,
        memberships: [{
          group: { id: group.id, name: group.name },
          status: 'ACTIVE',
          roles: ['EDITOR'],
          granted_permissions: [],
          role_permissions: [],
          role_requests: [],
          created_at: '2026-09-14T08:00:00Z',
          updated_at: '2026-09-14T08:00:00Z'
        }]
      }))
      .mockReturnValueOnce(of(user));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.users-card tbody .app-button') as HTMLButtonElement).click();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.membership-remove-button') as HTMLButtonElement).click();

    await vi.waitFor(() => expect(apiService.removeGroupMember).toHaveBeenCalledWith('group-id', 'user-id'));
    expect(toast.success).toHaveBeenCalledWith('Group membership removed.');
  });
});
