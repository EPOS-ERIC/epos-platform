import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { CanonicalOrganization } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { CapabilitiesService } from '../../services/capabilities.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GlobalOrganizationsComponent } from './global-organizations.component';

describe('GlobalOrganizationsComponent', () => {
  const organizations = Array.from({ length: 12 }, (_, index): CanonicalOrganization => ({
    id: `organization-${index + 1}`,
    name: `Organization ${String(index + 1).padStart(2, '0')}`,
    address: { street: null, locality: null, postal_code: null, country: null },
    identifiers: [],
    parents: []
  }));
  const apiService = {
    getCanonicalOrganizations: vi.fn(() => of(organizations)),
    triggerMetadataSync: vi.fn(() => of(undefined))
  };
  const organizationManagement = signal<'local' | 'remote_sync'>('local');
  const toastService = { info: vi.fn(), error: vi.fn() };
  const currentUserStore = { hasGlobalPermission: vi.fn(() => true) };

  beforeEach(() => {
    vi.clearAllMocks();
    organizationManagement.set('local');
    TestBed.configureTestingModule({
      imports: [GlobalOrganizationsComponent],
      providers: [
        { provide: ApiService, useValue: apiService },
        { provide: CapabilitiesService, useValue: { organizationManagement } },
        { provide: CurrentUserStore, useValue: currentUserStore },
        { provide: ToastService, useValue: toastService }
      ]
    });
  });

  it('disables local management and schedules remote synchronization', () => {
    organizationManagement.set('remote_sync');
    const fixture = TestBed.createComponent(GlobalOrganizationsComponent);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.querySelector<HTMLButtonElement>('.organization-actions button')?.disabled).toBe(true);
    clickButton(element, 'Sync metadata');

    expect(apiService.triggerMetadataSync).toHaveBeenCalledOnce();
    expect(toastService.info).toHaveBeenCalledWith(expect.stringContaining('accepted'));
  });

  it('paginates organizations in groups of ten', () => {
    const fixture = TestBed.createComponent(GlobalOrganizationsComponent);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.querySelectorAll('.organization-item')).toHaveLength(10);
    expect(element.textContent).not.toContain('Organization 11');

    clickButton(element, 'Next');
    fixture.detectChanges();

    expect(element.querySelectorAll('.organization-item')).toHaveLength(2);
    expect(element.textContent).toContain('Organization 11');
    expect(element.textContent).toContain('Page 2 of 2');

    const pageSize = element.querySelector<HTMLSelectElement>('.page-size-control select')!;
    pageSize.value = '25';
    pageSize.dispatchEvent(new Event('change'));
    fixture.detectChanges();

    expect(element.querySelectorAll('.organization-item')).toHaveLength(12);
    expect(element.textContent).toContain('Page 1 of 1');
  });

  function clickButton(element: HTMLElement, label: string): void {
    const button = Array.from(element.querySelectorAll<HTMLButtonElement>('button'))
      .find((candidate) => candidate.textContent?.trim() === label);
    expect(button).toBeDefined();
    button!.click();
  }
});
