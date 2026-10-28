import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import type { RoleRequestItem } from './role-requests-panel.component';
import { RoleRequestsPanelComponent } from './role-requests-panel.component';

describe('RoleRequestsPanelComponent', () => {
  const request: RoleRequestItem = {
    key: 'group-id:request-id',
    groupId: 'group-id',
    groupName: 'Data Group',
    groupLogoPath: '/group.svg',
    userId: 'user-id',
    userEmail: 'user@example.org',
    userName: 'Test User',
    requestId: 'request-id',
    role: 'REVIEWER',
    requestedAt: '2026-09-14T08:00:00Z'
  };

  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [RoleRequestsPanelComponent] });
  });

  it('renders request and group details and emits accept and reject decisions', () => {
    const fixture = TestBed.createComponent(RoleRequestsPanelComponent);
    const decision = vi.fn();
    fixture.componentInstance.decision.subscribe(decision);
    fixture.componentRef.setInput('requests', [request]);
    fixture.componentRef.setInput('subtitle', 'Review role requests.');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('user@example.org');
    expect(fixture.nativeElement.textContent).toContain('Data Group');
    expect(fixture.nativeElement.textContent).toContain('REVIEWER');
    expect(fixture.nativeElement.querySelector('.request-group-logo').getAttribute('src')).toBe('/group.svg');

    (fixture.nativeElement.querySelector('.app-button--primary') as HTMLButtonElement).click();
    (fixture.nativeElement.querySelector('.request-reject') as HTMLButtonElement).click();

    expect(decision).toHaveBeenNthCalledWith(1, { item: request, decision: 'accept' });
    expect(decision).toHaveBeenNthCalledWith(2, { item: request, decision: 'reject' });
  });
});
