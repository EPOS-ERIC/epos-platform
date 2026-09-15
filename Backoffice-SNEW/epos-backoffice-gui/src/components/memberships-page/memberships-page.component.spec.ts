import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import type { User } from '../../../generated/backofficeSchemas';
import { CurrentUserStore } from '../../stores/current-user.store';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';
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

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      imports: [MembershipsPageComponent],
      providers: [
        { provide: CurrentUserStore, useValue: { user: signal(user), loading: signal(false), error: signal(null) } },
        { provide: MembershipRequestDialogService, useValue: { open, groups: signal([{ id: 'group-id' }]), groupsLoading: signal(false) } }
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
});
