import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { GroupMembership } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { ToastService } from '../../services/toast.service';
import { ManageGroupPageComponent } from './manage-group-page.component';

describe('ManageGroupPageComponent', () => {
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
  const apiService = {
    getGroupMemberships: vi.fn(),
    listPermissions: vi.fn(),
    removeGroupMember: vi.fn(),
    setGroupMembershipRole: vi.fn(),
    setGroupMembershipPermission: vi.fn()
  };
  const dialog = { confirm: vi.fn() };
  const toast = { success: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    apiService.getGroupMemberships.mockReturnValue(of([groupAdmin, member]));
    apiService.listPermissions.mockReturnValue(of([
      { key: 'group.resource.create', label: 'Create resources', description: 'Create resources.', scope: 'GROUP' },
      { key: 'group.resource.edit', label: 'Edit resources', description: 'Edit resources.', scope: 'GROUP' },
      { key: 'group.resource.submit', label: 'Submit resources', description: 'Submit resources.', scope: 'GROUP' }
    ]));
    apiService.removeGroupMember.mockReturnValue(of(undefined));
    apiService.setGroupMembershipRole.mockReturnValue(of(undefined));
    apiService.setGroupMembershipPermission.mockReturnValue(of(undefined));
    dialog.confirm.mockResolvedValue(true);
    TestBed.configureTestingModule({
      imports: [ManageGroupPageComponent],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: DialogService, useValue: dialog },
        { provide: ToastService, useValue: toast }
      ]
    });
  });

  it('offers only EDITOR and REVIEWER when managing a member', () => {
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('groupName', 'Data Group');
    fixture.detectChanges();

    const manageButtons = fixture.nativeElement.querySelectorAll('tbody tr .member-actions .app-button:first-child') as NodeListOf<HTMLButtonElement>;
    manageButtons[1].click();
    fixture.detectChanges();

    const dialogText = (fixture.nativeElement.querySelector('.membership-dialog') as HTMLElement).textContent;
    expect(dialogText).toContain('EDITOR');
    expect(dialogText).toContain('REVIEWER');
    expect(dialogText).not.toContain('GROUP_ADMIN');
  });

  it('disables permissions already granted by a role while keeping other direct permissions editable', () => {
    const fixture = TestBed.createComponent(ManageGroupPageComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
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
    fixture.detectChanges();

    const removeButton = fixture.nativeElement.querySelectorAll('tbody tr')[1].querySelector('.remove-button') as HTMLButtonElement;
    removeButton.click();

    await vi.waitFor(() => expect(apiService.removeGroupMember).toHaveBeenCalled());
    expect(apiService.removeGroupMember).toHaveBeenCalledWith('group-id', 'member-id');
    expect(toast.success).toHaveBeenCalledWith('Group member removed.');
  });
});
