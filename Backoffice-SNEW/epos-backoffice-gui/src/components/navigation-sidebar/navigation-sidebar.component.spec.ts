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
    fixture.detectChanges();

    const link = fixture.nativeElement.querySelector('.membership-request-button') as HTMLAnchorElement;
    expect(link.textContent).toContain('My memberships');
    expect(link.getAttribute('href')).toBe('/memberships');
  });
});
