import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { vi } from 'vitest';

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
    expect(link.getAttribute('href')).toBe('/memberships?groupId=group-id&returnUrl=%2Fdata');
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
    const root = fixture.nativeElement.querySelector('.nav-tree-item--root') as HTMLElement;
    expect(root.querySelector('.nav-root-label')?.textContent?.trim()).toBe('Dataset');
    expect(root.querySelector('.nav-item--tree-link')?.getAttribute('href')).toBe('#compare-general-information');

    const rootToggle = root.querySelector('.detail-root-toggle-button') as HTMLButtonElement;
    expect(rootToggle.getAttribute('aria-expanded')).toBe('true');
    rootToggle.click();
    fixture.detectChanges();

    expect(rootToggle.getAttribute('aria-expanded')).toBe('false');
    expect(root.querySelector('.nav-item--tree-link')).toBeNull();
  });

  it('renders distributions alongside the resource root', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('detailRootLabel', 'Dataset');
    fixture.componentRef.setInput('detailItems', [
      { label: 'General Information', anchor: 'general-information' },
      { label: 'Distributions', anchor: 'distributions' }
    ]);
    fixture.detectChanges();

    const detailSection = fixture.nativeElement.querySelector('.nav-section--detail') as HTMLElement;
    const root = detailSection.querySelector('.nav-tree-item--root') as HTMLElement;
    const distributions = detailSection.querySelector('[href="#distributions"]') as HTMLAnchorElement;
    expect(root.querySelector('[href="#general-information"]')).not.toBeNull();
    expect(root.querySelector('[href="#distributions"]')).toBeNull();
    expect(distributions.closest('.nav-tree-item')?.parentElement).toBe(detailSection);
  });

  it('keeps the active detail item in sync with manual scrolling', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const general = document.createElement('section');
    general.id = 'general-information';
    general.getBoundingClientRect = () => ({ top: -200 }) as DOMRect;
    const spatial = document.createElement('section');
    spatial.id = 'spatial-coverage';
    spatial.getBoundingClientRect = () => ({ top: 20 }) as DOMRect;
    document.body.append(general, spatial);

    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('detailRootLabel', 'Dataset');
    fixture.componentRef.setInput('detailItems', [
      { label: 'General Information', anchor: 'general-information' },
      { label: 'Spatial Coverage', anchor: 'spatial-coverage' }
    ]);
    fixture.detectChanges();
    (fixture.componentInstance as unknown as { syncActiveDetailAnchor(): void }).syncActiveDetailAnchor();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.nav-item--tree-link.active')?.getAttribute('href')).toBe('#spatial-coverage');
    general.remove();
    spatial.remove();
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
    const navigateByUrl = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);

    const toggles = fixture.nativeElement.querySelectorAll('.main-tree-toggle-button') as NodeListOf<HTMLButtonElement>;
    const mainLinks = fixture.nativeElement.querySelectorAll('.nav-item--main') as NodeListOf<HTMLAnchorElement>;
    expect(mainLinks).toHaveLength(4);
    expect(toggles).toHaveLength(3);
    expect(toggles[0].getAttribute('aria-expanded')).toBe('true');
    expect(toggles[1].getAttribute('aria-expanded')).toBe('false');

    toggles[1].click();
    fixture.detectChanges();

    expect(toggles[0].getAttribute('aria-expanded')).toBe('false');
    expect(toggles[1].getAttribute('aria-expanded')).toBe('true');
    expect(navigateByUrl).toHaveBeenCalledOnce();
    expect(navigateByUrl).toHaveBeenCalledWith('/software');

    toggles[1].click();
    fixture.detectChanges();

    expect(toggles[1].getAttribute('aria-expanded')).toBe('false');
    expect(navigateByUrl).toHaveBeenCalledOnce();

    toggles[0].click();
    fixture.detectChanges();

    expect(toggles[0].getAttribute('aria-expanded')).toBe('true');
    expect(navigateByUrl).toHaveBeenCalledOnce();
  });

  it('renders ALL as a main link without a group accordion', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('currentSectionPath', 'all');
    fixture.detectChanges();

    const mainLinks = Array.from(fixture.nativeElement.querySelectorAll('.nav-item--main')) as HTMLAnchorElement[];
    const allLink = mainLinks.find((link) => link.getAttribute('href') === '/all')!;
    const allTreeItem = allLink.closest('.nav-tree-item') as HTMLElement;
    expect(mainLinks.at(-1)).toBe(allLink);
    expect(allLink.textContent?.trim()).toBe('ALL');
    expect(allLink.getAttribute('href')).toBe('/all');
    expect(allLink.classList.contains('active')).toBe(true);
    expect(allTreeItem.querySelector('.main-tree-toggle-button')).toBeNull();
    expect(allTreeItem.querySelector('.main-tree-panel')).toBeNull();
  });

  it('marks the URL-selected group as active', async () => {
    await TestBed.configureTestingModule({
      imports: [NavigationSidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(NavigationSidebarComponent);
    fixture.componentRef.setInput('groups', [
      { id: 'group-1', name: 'First Group', position: 1, concept: 'DATA' },
      { id: 'group-2', name: 'Second Group', position: 2, concept: 'DATA' }
    ]);
    fixture.componentRef.setInput('currentSectionPath', 'data');
    fixture.componentRef.setInput('selectedGroupId', 'group-2');
    fixture.detectChanges();

    const selected = fixture.nativeElement.querySelector('.nav-item--sub.active') as HTMLAnchorElement;
    expect(selected.textContent).toContain('Second Group');
    expect(selected.getAttribute('href')).toBe('/data?groupId=group-2');
  });
});
