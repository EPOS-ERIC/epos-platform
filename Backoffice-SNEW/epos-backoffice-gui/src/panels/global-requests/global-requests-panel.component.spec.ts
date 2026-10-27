import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Group, GroupMembership } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GroupsStore } from '../../stores/groups.store';
import { GlobalRequestsPanelComponent } from './global-requests-panel.component';

describe('GlobalRequestsPanelComponent', () => {
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
    granted_permissions: [],
    role_permissions: [],
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
  const activeMembershipWithRoleRequest: GroupMembership = {
    ...activeMembership,
    role_requests: [{
      id: 'active-role-request-id',
      role: 'REVIEWER',
      status: 'PENDING',
      requested_at: '2026-09-15T08:00:00Z'
    }]
  };
  const apiService = {
    getGroupMemberships: vi.fn(),
    decideGroupMembership: vi.fn(),
    decideGroupRoleRequest: vi.fn(),
    setGroupMembershipRole: vi.fn()
  };
  const currentUserStore = { hasGlobalPermission: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    currentUserStore.hasGlobalPermission.mockReturnValue(true);
    apiService.getGroupMemberships.mockReturnValue(of([pendingMembership, activeMembershipWithRoleRequest]));
    apiService.decideGroupMembership.mockReturnValue(of(activeMembership));
    apiService.decideGroupRoleRequest.mockReturnValue(of(activeMembership));
    apiService.setGroupMembershipRole.mockReturnValue(of(undefined));
    TestBed.configureTestingModule({
      imports: [GlobalRequestsPanelComponent],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: GroupsStore, useValue: { groups: signal([group]) } }
      ]
    });
  });

  it('accepts a membership with an admin-selected role instead of the requested role', async () => {
    const fixture = TestBed.createComponent(GlobalRequestsPanelComponent);
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
    (fixture.nativeElement.querySelector('.requests-card .request-actions .app-button--primary') as HTMLButtonElement).click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(apiService.decideGroupMembership).toHaveBeenCalledWith('group-id', 'user-id', 'accept');
    expect(apiService.decideGroupRoleRequest).toHaveBeenCalledWith('group-id', 'request-id', 'reject');
    expect(apiService.setGroupMembershipRole).toHaveBeenCalledWith('group-id', 'user-id', 'GROUP_ADMIN', true);
    expect(fixture.nativeElement.textContent).toContain('No pending membership requests.');
  });

  it('shows and accepts role requests from active memberships', async () => {
    const fixture = TestBed.createComponent(GlobalRequestsPanelComponent);
    fixture.detectChanges();

    const roleRequestsCard = fixture.nativeElement.querySelector('.role-requests-card') as HTMLElement;
    expect(roleRequestsCard.textContent).toContain('user@example.org');
    expect(roleRequestsCard.textContent).toContain('REVIEWER');

    (roleRequestsCard.querySelector('.app-button--primary') as HTMLButtonElement).click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(apiService.decideGroupRoleRequest).toHaveBeenCalledWith('group-id', 'active-role-request-id', 'accept');
    expect(roleRequestsCard.textContent).toContain('No pending role requests.');
  });
});
