import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Group, GroupMembership } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { ManageGroupPageComponent } from './manage-group-page.component';

describe('ManageGroupPageComponent', () => {
  const group: Group = { id: 'group-id', name: 'Data Group', position: 0, concept: 'DATA' };
  const groupAdmin: GroupMembership = {
    user: { id: 'admin-id', email: 'admin@example.org', given_name: 'Group', family_name: 'Admin' },
    status: 'ACTIVE',
    roles: ['GROUP_ADMIN'],
    granted_permissions: ['group.resource.edit'],
    role_permissions: ['group.membership.manage'],
    role_requests: [],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z'
  };
  const member: GroupMembership = {
    user: { id: 'member-id', email: 'member@example.org', given_name: 'Regular', family_name: 'Member' },
    status: 'ACTIVE',
    roles: ['EDITOR'],
    granted_permissions: [],
    role_permissions: ['group.resource.create', 'group.resource.edit'],
    role_requests: [],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z'
  };
  const pendingMember: GroupMembership = {
    ...member,
    user: { id: 'pending-id', email: 'pending@example.org', given_name: 'Pending', family_name: 'Member' },
    status: 'PENDING',
    roles: [],
    role_requests: [{
      id: 'pending-role-request-id',
      role: 'EDITOR',
      status: 'PENDING',
      requested_at: '2026-09-15T08:00:00Z'
    }]
  };
  const memberWithRoleRequest: GroupMembership = {
    ...member,
    role_requests: [{
      id: 'role-request-id',
      role: 'REVIEWER',
      status: 'PENDING',
      requested_at: '2026-09-15T08:00:00Z'
    }]
  };
  const apiService = {
    getGroupMemberships: vi.fn(),
    listPermissions: vi.fn(),
    removeGroupMember: vi.fn(),
    setGroupMembershipRole: vi.fn(),
    setGroupMembershipPermission: vi.fn(),
    decideGroupMembership: vi.fn(),
    decideGroupRoleRequest: vi.fn(),
    getGroupCategories: vi.fn(),
    listNotificationTopics: vi.fn(),
    listUserNotificationSubscriptions: vi.fn(),
    setGroupUserNotificationSubscription: vi.fn()
  };
  const dialog = { confirm: vi.fn() };
  const toast = { success: vi.fn() };
  const currentUserStore = { hasGroupPermission: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission !== 'group.category.manage'
    );
    apiService.getGroupMemberships.mockReturnValue(of([groupAdmin, member]));
    apiService.listPermissions.mockReturnValue(of([
      { key: 'group.resource.create', label: 'Create resources', description: 'Create resources.', scope: 'GROUP' },
      { key: 'group.resource.edit', label: 'Edit resources', description: 'Edit resources.', scope: 'GROUP' },
      { key: 'group.resource.submit', label: 'Submit resources', description: 'Submit resources.', scope: 'GROUP' }
    ]));
    apiService.removeGroupMember.mockReturnValue(of(undefined));
    apiService.setGroupMembershipRole.mockReturnValue(of(undefined));
    apiService.setGroupMembershipPermission.mockReturnValue(of(undefined));
    apiService.decideGroupMembership.mockReturnValue(of(member));
    apiService.decideGroupRoleRequest.mockReturnValue(of(memberWithRoleRequest));
    apiService.getGroupCategories.mockReturnValue(of([]));
    apiService.listNotificationTopics.mockReturnValue(of([{
      key: 'group.resource.published',
      label: 'Group resource publications',
      description: 'Email this member when a resource is published.',
      scope: 'GROUP',
      required_permissions: []
    }]));
    apiService.listUserNotificationSubscriptions.mockReturnValue(of([]));
    apiService.setGroupUserNotificationSubscription.mockReturnValue(of(undefined));
    dialog.confirm.mockResolvedValue(true);
    TestBed.configureTestingModule({
      imports: [ManageGroupPageComponent],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: DialogService, useValue: dialog },
        { provide: ToastService, useValue: toast },
        { provide: CurrentUserStore, useValue: currentUserStore },
        provideRouter([])
      ]
    });
  });

  it('offers GROUP_ADMIN when managing a member with role management permission', () => {
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('groupName', 'Data Group');
    fixture.componentRef.setInput('panel', 'member-access');
    fixture.detectChanges();

    const manageButtons = fixture.nativeElement.querySelectorAll('tbody tr .member-actions .app-button:first-child') as NodeListOf<HTMLButtonElement>;
    manageButtons[1].click();
    fixture.detectChanges();

    const dialogText = (fixture.nativeElement.querySelector('.membership-dialog') as HTMLElement).textContent;
    expect(dialogText).toContain('EDITOR');
    expect(dialogText).toContain('REVIEWER');
    expect(dialogText).toContain('GROUP_ADMIN');

    const groupAdminOption = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.membership-dialog label.membership-option'))
      .find((option) => option.textContent?.includes('GROUP_ADMIN'))!;
    (groupAdminOption.querySelector('input') as HTMLInputElement).click();

    expect(apiService.setGroupMembershipRole).toHaveBeenCalledWith('group-id', 'member-id', 'GROUP_ADMIN', true);
  });

  it('shows requested role options and applies the selected roles when role management is granted', async () => {
    apiService.getGroupMemberships
      .mockReturnValueOnce(of([pendingMember]))
      .mockReturnValueOnce(of([member]));
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('groupName', 'Data Group');
    fixture.componentRef.setInput('panel', 'membership-requests');
    fixture.detectChanges();

    const roleLabels = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.request-role-picker label'));
    const editor = roleLabels.find((label) => label.textContent?.includes('EDITOR'))!;
    expect(editor.textContent).toContain('Requested');
    expect(roleLabels.some((label) => label.textContent?.includes('GROUP_ADMIN'))).toBe(false);

    (editor.querySelector('input') as HTMLInputElement).click();
    fixture.detectChanges();
    const reviewer = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.request-role-picker label'))
      .find((label) => label.textContent?.includes('REVIEWER'))!;
    (reviewer.querySelector('input') as HTMLInputElement).click();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('app-membership-requests-panel .app-button--primary') as HTMLButtonElement).click();
    await fixture.whenStable();

    expect(apiService.decideGroupMembership).toHaveBeenCalledWith('group-id', 'pending-id', 'accept');
    expect(apiService.decideGroupRoleRequest).toHaveBeenCalledWith('group-id', 'pending-role-request-id', 'reject');
    expect(apiService.setGroupMembershipRole).toHaveBeenCalledWith('group-id', 'pending-id', 'REVIEWER', true);
  });

  it('keeps requested roles read-only without role management permission', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.membership.manage'
    );
    apiService.getGroupMemberships.mockReturnValue(of([pendingMember]));
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('panel', 'membership-requests');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('EDITOR');
    expect(fixture.nativeElement.querySelector('.request-role-picker')).toBeNull();
  });

  it('disables permissions already granted by a role while keeping other direct permissions editable', () => {
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('panel', 'member-access');
    fixture.detectChanges();

    const manageButtons = fixture.nativeElement.querySelectorAll('tbody tr .member-actions .app-button:first-child') as NodeListOf<HTMLButtonElement>;
    manageButtons[1].click();
    fixture.detectChanges();

    const permissionOptions = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.membership-dialog label.membership-option'));
    const inheritedPermissionOption = permissionOptions
      .find((option) => option.textContent?.includes('Edit resources'));
    const inheritedPermissionInput = inheritedPermissionOption?.querySelector('input') as HTMLInputElement;

    expect(inheritedPermissionInput.checked).toBe(true);
    expect(inheritedPermissionInput.disabled).toBe(true);
    expect(inheritedPermissionOption?.textContent).toContain('Already granted by role');
    expect(fixture.nativeElement.querySelector('.membership-dialog')?.textContent).toContain('2 inherited');
    expect(fixture.nativeElement.querySelector('.membership-dialog')?.textContent).toContain('Create resources.');

    const directPermissionOption = permissionOptions
      .find((option) => option.textContent?.includes('Submit resources'));
    const directPermissionInput = directPermissionOption?.querySelector('input') as HTMLInputElement;
    expect(directPermissionInput.disabled).toBe(false);

    directPermissionInput.click();

    expect(apiService.setGroupMembershipPermission).toHaveBeenCalledWith('group-id', 'member-id', 'group.resource.submit', true);
  });

  it('removes a regular member after confirmation', async () => {
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('groupName', 'Data Group');
    fixture.componentRef.setInput('panel', 'member-access');
    fixture.detectChanges();

    const removeButton = fixture.nativeElement.querySelectorAll('tbody tr')[1].querySelector('.remove-button') as HTMLButtonElement;
    removeButton.click();

    await vi.waitFor(() => expect(apiService.removeGroupMember).toHaveBeenCalled());
    expect(apiService.removeGroupMember).toHaveBeenCalledWith('group-id', 'member-id');
    expect(toast.success).toHaveBeenCalledWith('Group member removed.');
  });

  it('shows only the category destination when that is the only granted permission', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.category.manage'
    );
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groups', [group]);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.members-card')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Group Categories');
    expect(apiService.getGroupMemberships).not.toHaveBeenCalled();
    expect(apiService.getGroupCategories).not.toHaveBeenCalled();
  });

  it('loads category management only after navigating to its panel', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.category.manage'
    );
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groups', [group]);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('panel', 'categories');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Group Categories');
    expect(apiService.getGroupCategories).toHaveBeenCalledWith('group-id');
  });

  it('does not expose member removal to a role-only manager', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.role.manage'
    );
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('panel', 'member-access');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.members-card')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.member-actions .app-button--ghost')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.remove-button')).toBeNull();
  });

  it('allows a membership manager to accept a group membership request', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.membership.manage'
    );
    apiService.getGroupMemberships.mockReturnValue(of([pendingMember]));
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('panel', 'membership-requests');
    fixture.detectChanges();

    const requestsCard = fixture.nativeElement.querySelector('.requests-card') as HTMLElement;
    expect(requestsCard.textContent).toContain('pending@example.org');
    expect(requestsCard.textContent).toContain('EDITOR');
    (requestsCard.querySelector('.app-button--primary') as HTMLButtonElement).click();

    expect(apiService.decideGroupMembership).toHaveBeenCalledWith('group-id', 'pending-id', 'accept');
    expect(toast.success).toHaveBeenCalledWith('Membership request accepted.');
  });

  it('allows a role manager to reject a group role request', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.role.manage'
    );
    apiService.getGroupMemberships.mockReturnValue(of([memberWithRoleRequest]));
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('panel', 'membership-requests');
    fixture.detectChanges();

    const requestsCard = fixture.nativeElement.querySelector('.requests-card') as HTMLElement;
    expect(requestsCard.textContent).toContain('member@example.org');
    expect(requestsCard.textContent).toContain('REVIEWER');
    (requestsCard.querySelector('.request-reject') as HTMLButtonElement).click();

    expect(apiService.decideGroupRoleRequest).toHaveBeenCalledWith('group-id', 'role-request-id', 'reject');
    expect(toast.success).toHaveBeenCalledWith('Role request rejected.');
  });

  it('shows only destinations allowed by the current permissions', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.role.manage'
    );
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.detectChanges();

    const destinationTitles = Array.from<HTMLElement>(
      fixture.nativeElement.querySelectorAll('.management-destination h3')
    ).map((element) => element.textContent?.trim());
    expect(destinationTitles).toEqual(['Requests', 'Members']);
  });

  it('routes notification managers through the members destination', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.notification.manage'
    );
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.detectChanges();

    const destinationTitles = Array.from<HTMLElement>(
      fixture.nativeElement.querySelectorAll('.management-destination h3')
    ).map((element) => element.textContent?.trim());
    expect(destinationTitles).toEqual(['Members']);
    expect(apiService.getGroupMemberships).not.toHaveBeenCalled();
  });

  it('manages group notifications from the member dialog for a notification-only manager', () => {
    currentUserStore.hasGroupPermission.mockImplementation((_groupId: string, permission: string) =>
      permission === 'group.notification.manage'
    );
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('panel', 'member-access');
    fixture.detectChanges();

    const manageButtons = fixture.nativeElement.querySelectorAll('tbody tr .member-actions .app-button') as NodeListOf<HTMLButtonElement>;
    manageButtons[1].click();
    fixture.detectChanges();

    const membershipDialog = fixture.nativeElement.querySelector('.membership-dialog') as HTMLElement;
    expect(membershipDialog.textContent).toContain('Notifications');
    expect(membershipDialog.textContent).toContain('Group resource publications');
    expect(membershipDialog.textContent).not.toContain('Direct permissions');
    expect(apiService.listUserNotificationSubscriptions).toHaveBeenCalledWith('member-id');

    (membershipDialog.querySelector('.notification-option input') as HTMLInputElement).click();

    expect(apiService.setGroupUserNotificationSubscription).toHaveBeenCalledWith(
      'group-id',
      'member-id',
      'group.resource.published',
      true
    );
  });
});
