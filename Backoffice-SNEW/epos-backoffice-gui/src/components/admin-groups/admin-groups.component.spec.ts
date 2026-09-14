import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Group, GroupMembership } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { GroupsStore } from '../../stores/groups.store';
import { AdminGroupsComponent } from './admin-groups.component';

describe('AdminGroupsComponent', () => {
  const group: Group = {
    id: 'group-id',
    name: 'Data Group',
    position: 0,
    concept: 'DATA'
  };
  const pendingMembership: GroupMembership = {
    user: {
      id: 'user-id',
      email: 'user@example.org',
      given_name: 'Test',
      family_name: 'User'
    },
    status: 'PENDING',
    roles: [],
    permissions: [],
    role_requests: [{
      id: 'request-id',
      role: 'EDITOR',
      status: 'PENDING',
      requested_at: '2026-09-14T08:00:00Z'
    }],
    created_at: '2026-09-14T08:00:00Z',
    updated_at: '2026-09-14T08:00:00Z'
  };
  const activeMembership: GroupMembership = {
    ...pendingMembership,
    status: 'ACTIVE'
  };
  const apiService = {
    getUsers: vi.fn(),
    getGroupMemberships: vi.fn(),
    decideGroupMembership: vi.fn(),
    decideGroupRoleRequest: vi.fn(),
    setGroupMembershipRole: vi.fn()
  };

  beforeEach(() => {
    vi.clearAllMocks();
    apiService.getUsers.mockReturnValue(of([]));
    apiService.getGroupMemberships.mockReturnValue(of([pendingMembership]));
    apiService.decideGroupMembership.mockReturnValue(of(activeMembership));
    apiService.decideGroupRoleRequest.mockReturnValue(of(activeMembership));
    apiService.setGroupMembershipRole.mockReturnValue(of(undefined));
    TestBed.configureTestingModule({
      imports: [AdminGroupsComponent],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: GroupsStore, useValue: { groups: signal([group]) } }
      ]
    });
  });

  it('accepts a membership with an admin-selected role instead of the requested role', async () => {
    const fixture = TestBed.createComponent(AdminGroupsComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('user@example.org');
    expect(fixture.nativeElement.textContent).toContain('Data Group');

    const roleLabels = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.request-role-picker label'));
    const editor = roleLabels.find((label) => label.textContent?.includes('EDITOR'));
    (editor?.querySelector('input') as HTMLInputElement).click();
    fixture.detectChanges();
    const groupAdmin = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.request-role-picker label'))
      .find((label) => label.textContent?.includes('GROUP_ADMIN'));
    (groupAdmin?.querySelector('input') as HTMLInputElement).click();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.request-actions .app-button--primary') as HTMLButtonElement).click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(apiService.decideGroupMembership).toHaveBeenCalledWith('group-id', 'user-id', { status: 'ACTIVE' });
    expect(apiService.decideGroupRoleRequest).toHaveBeenCalledWith('group-id', 'request-id', { status: 'REJECTED' });
    expect(apiService.setGroupMembershipRole).toHaveBeenCalledWith('group-id', 'user-id', 'GROUP_ADMIN', true);
    expect(fixture.nativeElement.textContent).toContain('No pending membership requests.');
  });
});
