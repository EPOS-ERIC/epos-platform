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
    listActivity: vi.fn(),
    listPermissions: vi.fn(),
    setGlobalRole: vi.fn(),
    setGlobalPermission: vi.fn(),
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
    apiService.listActivity.mockReturnValue(of([]));
    apiService.listPermissions.mockReturnValue(of([
      { key: 'global.resource.create', label: 'Create global resources', description: 'Create resources platform-wide.', scope: 'GLOBAL' },
      { key: 'group.resource.create', label: 'Create resources', description: 'Create resources.', scope: 'GROUP' }
    ]));
    apiService.setGlobalRole.mockReturnValue(of(undefined));
    apiService.setGlobalPermission.mockReturnValue(of(undefined));
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

    (fixture.nativeElement.querySelector('.users-card tbody .app-button') as HTMLButtonElement).click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.activity-history')).toBeNull();
  });

  it('loads, filters, and searches only the selected actor activity', async () => {
    const activityGroupId = '23b9b58a-9f1c-5d18-85f9-73842733835c';
    const targetUser = {
      ...userSummary,
      id: 'target-user-id',
      email: 'target@example.org',
      given_name: 'Target',
      family_name: 'User'
    };
    currentUserStore.hasGlobalPermission.mockImplementation((permission: string) =>
      permission === 'global.activity.read'
    );
    apiService.getUsers.mockReturnValue(of([userSummary, targetUser]));
    apiService.listActivity.mockReturnValue(of([
      {
        action: 'RESOURCE_CREATED',
        occurred_at: '2026-09-14T08:00:00Z',
        actor_id: 'user-id',
        group_id: activityGroupId,
        resource_type: 'dataset',
        resource_id: 'resource-id'
      },
      {
        action: 'GROUP_CREATED',
        occurred_at: '2026-09-16T08:00:00Z',
        actor_id: 'another-user-id',
        group_id: activityGroupId
      },
      {
        action: 'RESOURCE_PUBLISHED',
        occurred_at: '2026-09-15T08:00:00Z',
        actor_id: 'user-id',
        target_user_id: targetUser.id,
        note: 'Published after review'
      }
    ]));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.componentRef.setInput('groups', [{ ...group, id: activityGroupId }]);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.users-card tbody .app-button') as HTMLButtonElement).click();
    fixture.detectChanges();

    const activityHistory = fixture.nativeElement.querySelector('.activity-history') as HTMLDetailsElement;
    expect(activityHistory).not.toBeNull();
    expect(apiService.listActivity).not.toHaveBeenCalled();

    activityHistory.open = true;
    activityHistory.dispatchEvent(new Event('toggle'));
    fixture.detectChanges();

    expect(apiService.listActivity).toHaveBeenCalledOnce();
    const actions = Array.from<HTMLElement>(activityHistory.querySelectorAll('.activity-history-item strong'))
      .map((action) => action.textContent?.trim());
    expect(actions).toEqual(['Resource Published', 'Resource Created']);
    expect(activityHistory.textContent).toContain('Published after review');
    expect(activityHistory.textContent).toContain('Data Group');
    expect(activityHistory.textContent).toContain('Target User');
    expect(activityHistory.textContent).not.toContain('target-user-id');
    expect(activityHistory.textContent).not.toContain(activityGroupId);
    expect((activityHistory.querySelector('.activity-history-group img') as HTMLImageElement).getAttribute('src'))
      .toBe(`/group-icons/${activityGroupId}.png`);
    expect(activityHistory.textContent).not.toContain('Group Created');

    const searchInput = activityHistory.querySelector('.activity-history-search input') as HTMLInputElement;
    searchInput.value = 'created';
    searchInput.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(activityHistory.querySelectorAll('.activity-history-item').length).toBe(1);
    expect(activityHistory.textContent).toContain('Resource Created');
    expect(activityHistory.textContent).not.toContain('Resource Published');
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

  it('assigns and removes the global administrator role', () => {
    const permission = 'global.resource.publish' as const;
    const topic = {
      key: 'global.resource.submitted' as const,
      label: 'Global resource submissions',
      description: 'Email this user when a resource is submitted.',
      scope: 'GLOBAL' as const,
      required_permissions: [permission]
    };
    const subscription = {
      user_id: user.id,
      topic_key: topic.key,
      channel: 'EMAIL' as const
    };
    const administrator = {
      ...user,
      global_roles: ['ADMIN'],
      global_role_permissions: [permission]
    };
    apiService.getUsers.mockReturnValue(of([userSummary]));
    apiService.listNotificationTopics.mockReturnValue(of([topic]));
    apiService.listUserNotificationSubscriptions
      .mockReturnValueOnce(of([]))
      .mockReturnValueOnce(of([subscription]))
      .mockReturnValueOnce(of([subscription]));
    apiService.getUser
      .mockReturnValueOnce(of(user))
      .mockReturnValueOnce(of(administrator))
      .mockReturnValueOnce(of(user));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.users-card tbody .app-button') as HTMLButtonElement).click();
    fixture.detectChanges();

    let roleInput = fixture.nativeElement.querySelector('.global-access-group .global-access-option input') as HTMLInputElement;
    roleInput.click();
    expect(apiService.setGlobalRole).toHaveBeenNthCalledWith(1, 'user-id', 'ADMIN', true);

    fixture.detectChanges();
    let notificationInput = fixture.nativeElement.querySelector('.global-notifications input') as HTMLInputElement;
    expect(notificationInput.checked).toBe(true);
    expect(notificationInput.disabled).toBe(false);

    roleInput = fixture.nativeElement.querySelector('.global-access-group .global-access-option input') as HTMLInputElement;
    roleInput.click();
    expect(apiService.setGlobalRole).toHaveBeenNthCalledWith(2, 'user-id', 'ADMIN', false);

    fixture.detectChanges();
    notificationInput = fixture.nativeElement.querySelector('.global-notifications input') as HTMLInputElement;
    expect(notificationInput.checked).toBe(false);
    expect(notificationInput.disabled).toBe(true);
    expect(apiService.listUserNotificationSubscriptions).toHaveBeenCalledTimes(3);
  });

  it('grants and revokes a direct global permission', () => {
    const grantedUser = { ...user, global_granted_permissions: ['global.resource.create' as const] };
    apiService.getUsers.mockReturnValue(of([userSummary]));
    apiService.getUser
      .mockReturnValueOnce(of(user))
      .mockReturnValueOnce(of(grantedUser))
      .mockReturnValueOnce(of(user));
    const fixture = TestBed.createComponent(GlobalMembersPanelComponent);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.users-card tbody .app-button') as HTMLButtonElement).click();
    fixture.detectChanges();

    let permissionInput = fixture.nativeElement.querySelector('.global-access-permissions input') as HTMLInputElement;
    permissionInput.click();
    expect(apiService.setGlobalPermission).toHaveBeenNthCalledWith(1, 'user-id', 'global.resource.create', true);

    fixture.detectChanges();
    permissionInput = fixture.nativeElement.querySelector('.global-access-permissions input') as HTMLInputElement;
    permissionInput.click();
    expect(apiService.setGlobalPermission).toHaveBeenNthCalledWith(2, 'user-id', 'global.resource.create', false);
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
