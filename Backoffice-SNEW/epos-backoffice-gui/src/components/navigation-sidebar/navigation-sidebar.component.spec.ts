import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { NavigationSidebarComponent } from './navigation-sidebar.component';

describe('NavigationSidebarComponent', () => {
  it('links to the current user memberships page', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('selectedGroupId', 'group-id');
    fixture.detectChanges();

    const link = fixture.nativeElement.querySelector('.membership-request-button') as HTMLAnchorElement;
    expect(link.textContent).toContain('My memberships');
    expect(link.getAttribute('href')).toBe('/memberships?groupId=group-id');
  });

  it('uses the supplied label for comparison navigation', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('detailSectionLabel', 'Compare');
    fixture.componentRef.setInput('detailRootLabel', 'Dataset');
    fixture.componentRef.setInput('detailItems', [
      { label: 'General Information', anchor: 'compare-general-information' }
    ]);
    fixture.detectChanges();

    expect((fixture.nativeElement.querySelector('.nav-section-title') as HTMLElement).textContent?.trim()).toBe('Compare');
    expect(fixture.nativeElement.querySelector('.nav-item--tree-link')?.getAttribute('href')).toBe('#compare-general-information');
  });

  it('shows a single global management entry when an admin capability is granted', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('canAccessGlobalManage', true);
    fixture.componentRef.setInput('globalManageReturnUrl', '/software?groupId=group-id');
    fixture.detectChanges();

    const adminLinks = fixture.nativeElement.querySelectorAll('.admin-nav-section .nav-item') as NodeListOf<HTMLAnchorElement>;
    expect(adminLinks).toHaveLength(1);
    expect(adminLinks[0].textContent).toContain('Manage');
    expect(adminLinks[0].getAttribute('href')).toBe('/global-manage?returnUrl=%2Fsoftware%3FgroupId%3Dgroup-id');
  });

  it('uses an accordion for the main navigation groups', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('currentSectionPath', 'data');
    fixture.detectChanges();

    const toggles = fixture.nativeElement.querySelectorAll('.main-tree-toggle-button') as NodeListOf<HTMLButtonElement>;
    expect(toggles).toHaveLength(3);
    expect(toggles[0].getAttribute('aria-expanded')).toBe('true');
    expect(toggles[1].getAttribute('aria-expanded')).toBe('false');

    toggles[1].click();
    fixture.detectChanges();

    expect(toggles[0].getAttribute('aria-expanded')).toBe('false');
    expect(toggles[1].getAttribute('aria-expanded')).toBe('true');

    toggles[1].click();
    fixture.detectChanges();

    expect(toggles[1].getAttribute('aria-expanded')).toBe('false');
  });
});
