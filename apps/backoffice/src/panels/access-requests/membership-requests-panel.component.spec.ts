import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import type { MembershipRequestItem } from './membership-requests-panel.component';
import { MembershipRequestsPanelComponent } from './membership-requests-panel.component';

describe('MembershipRequestsPanelComponent', () => {
  const request: MembershipRequestItem = {
    key: 'group-id:user-id',
    groupId: 'group-id',
    groupName: 'Data Group',
    groupLogoPath: '/group.svg',
    userId: 'user-id',
    userEmail: 'user@example.org',
    userName: 'Test User',
    requestedAt: '2026-09-14T08:00:00Z',
    requestedRoles: ['EDITOR'],
    selectedRoles: ['EDITOR'],
    pendingRoleRequests: [{ id: 'role-request-id', role: 'EDITOR' }]
  };

  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [MembershipRequestsPanelComponent] });
  });

  it('renders request and group details and emits accept and reject decisions', () => {
    const fixture = TestBed.createComponent(MembershipRequestsPanelComponent);
    const decision = vi.fn();
    fixture.componentInstance.decision.subscribe(decision);
    fixture.componentRef.setInput('requests', [request]);
    fixture.componentRef.setInput('subtitle', 'Review membership requests.');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('user@example.org');
    expect(fixture.nativeElement.textContent).toContain('Data Group');
    expect(fixture.nativeElement.querySelector('.request-group-logo').getAttribute('src')).toBe('/group.svg');

    (fixture.nativeElement.querySelector('.app-button--primary') as HTMLButtonElement).click();
    (fixture.nativeElement.querySelector('.request-reject') as HTMLButtonElement).click();

    expect(decision).toHaveBeenNthCalledWith(1, { item: request, decision: 'accept' });
    expect(decision).toHaveBeenNthCalledWith(2, { item: request, decision: 'reject' });
  });

  it('shows and emits the optional role picker only when role selection is enabled', () => {
    const fixture = TestBed.createComponent(MembershipRequestsPanelComponent);
    const roleToggle = vi.fn();
    fixture.componentInstance.roleToggle.subscribe(roleToggle);
    fixture.componentRef.setInput('requests', [request]);
    fixture.componentRef.setInput('availableRoles', ['EDITOR', 'REVIEWER']);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.request-role-picker')).toBeNull();

    fixture.componentRef.setInput('canSelectRoles', true);
    fixture.detectChanges();
    const reviewer = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.request-role-picker label'))
      .find((label) => label.textContent?.includes('REVIEWER'));
    (reviewer?.querySelector('input') as HTMLInputElement).click();

    expect(roleToggle).toHaveBeenCalledWith({ item: request, role: 'REVIEWER' });
  });
});
