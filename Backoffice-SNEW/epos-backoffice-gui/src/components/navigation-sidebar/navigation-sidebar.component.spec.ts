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
});
