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
    fixture.componentRef.setInput('showBackButton', true);
    fixture.componentRef.setInput('detailBackLink', ['/data']);
    fixture.componentRef.setInput('detailBackQueryParams', { groupId: 'group-id' });
    fixture.detectChanges();

    const link = fixture.nativeElement.querySelector('.membership-request-button') as HTMLAnchorElement;
    expect(link.textContent).toContain('My memberships');
    expect(link.getAttribute('href')).toBe('/memberships?groupId=group-id');
    expect((fixture.nativeElement.querySelector('.sidebar-back-button') as HTMLAnchorElement).getAttribute('href')).toBe('/data?groupId=group-id');
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
    fixture.detectChanges();

    const adminLinks = fixture.nativeElement.querySelectorAll('.admin-nav-section .nav-item') as NodeListOf<HTMLAnchorElement>;
    expect(adminLinks).toHaveLength(1);
    expect(adminLinks[0].textContent).toContain('Manage');
    expect(adminLinks[0].getAttribute('href')).toBe('/global-manage');
  });
});
