import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { vi } from 'vitest';

import type { DiscoveryItem } from '../../../generated/backofficeSchemas';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';
import { ResultsTableComponent } from './results-table.component';

describe('ResultsTableComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [{ provide: MembershipRequestDialogService, useValue: { open: vi.fn() } }]
    });
  });

  it('renders an accessible table skeleton while discovery results are loading', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);

    fixture.componentRef.setInput('selectedGroupId', 'group-id');
    fixture.componentRef.setInput('discoveryLoading', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.table-scroll').getAttribute('aria-busy')).toBe('true');
    expect(fixture.nativeElement.querySelectorAll('.results-skeleton tbody tr')).toHaveLength(10);
    expect(fixture.nativeElement.querySelector('[role="status"]').textContent).toContain('Loading discovery results');
  });

  it('renders the API timestamp in a readable UTC format', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);
    const timestamp = '2026-09-10T07:34:26.486Z';
    const item: DiscoveryItem = {
      type: 'dataset',
      resource_id: 'resource-id',
      version_id: 'version-id',
      name: 'Dataset',
      status: 'PUBLISHED',
      updated_at: timestamp,
      concept: 'DATA',
      group: { id: 'group-id', name: 'Group' }
    };

    fixture.componentRef.setInput('selectedGroupId', 'group-id');
    fixture.componentRef.setInput('discoveryItems', [item]);
    fixture.componentRef.setInput('returnUrl', '/data?groupId=group-id');
    fixture.detectChanges();

    const cells = fixture.nativeElement.querySelectorAll('tbody td') as NodeListOf<HTMLTableCellElement>;
    expect(cells[4]?.textContent?.trim()).toBe('Sep 10, 2026, 07:34 UTC');
    expect((cells[4]?.querySelector('time') as HTMLTimeElement).dateTime).toBe(timestamp);
    expect((fixture.nativeElement.querySelector('.item-link') as HTMLAnchorElement).getAttribute('href')).toBe('/data/resource-id/version-id?groupId=group-id&dataType=dataset&returnUrl=%2Fdata%3FgroupId%3Dgroup-id');
  });

  it('shows active column filters and emits the requested filter section', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);
    const item: DiscoveryItem = {
      type: 'dataset',
      resource_id: 'resource-id',
      version_id: 'version-id',
      name: 'Dataset',
      status: 'DRAFT',
      updated_at: '2026-09-10T07:34:26.486Z',
      concept: 'DATA',
      group: { id: 'group-id', name: 'Group' },
      created_by: { id: 'author-id', name: 'Author' }
    };
    const editFilter = vi.fn();
    fixture.componentInstance.editFilter.subscribe(editFilter);
    fixture.componentRef.setInput('selectedGroupId', 'group-id');
    fixture.componentRef.setInput('discoveryItems', [item]);
    fixture.componentRef.setInput('filterByAuthor', 'author-id');
    fixture.componentRef.setInput('selectedDataTypes', ['dataset']);
    fixture.componentRef.setInput('selectedVersioningStatuses', ['DRAFT']);
    fixture.detectChanges();

    const triggers = fixture.nativeElement.querySelectorAll('.active-filter-trigger') as NodeListOf<HTMLButtonElement>;
    expect(triggers).toHaveLength(3);
    expect(Array.from(triggers).map((trigger) => trigger.getAttribute('aria-label'))).toEqual([
      'Edit active Type filter',
      'Edit active Author filter',
      'Edit active Status filter'
    ]);
    triggers[1].click();
    expect(editFilter).toHaveBeenCalledWith('author');
  });

  it('links group administrators to the group management route', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);

    fixture.componentRef.setInput('sectionPath', 'software');
    fixture.componentRef.setInput('selectedGroupId', 'group-id');
    fixture.componentRef.setInput('canManageGroup', true);
    fixture.componentRef.setInput('returnUrl', '/software?groupId=group-id&dataType=software-application');
    fixture.detectChanges();

    const link = fixture.nativeElement.querySelector('.toolbar a') as HTMLAnchorElement;
    expect(link.textContent?.trim()).toBe('Manage Group');
    expect(link.getAttribute('href')).toBe('/software/manage-group?groupId=group-id&returnUrl=%2Fsoftware%3FgroupId%3Dgroup-id%26dataType%3Dsoftware-application');
  });

  it('offers Copy only for published rows when a writable target group exists', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);
    const published: DiscoveryItem = {
      type: 'dataset',
      resource_id: 'published-resource',
      version_id: 'published-version',
      name: 'Published dataset',
      status: 'PUBLISHED',
      updated_at: '2026-09-10T07:34:26.486Z',
      concept: 'DATA',
      group: { id: 'source-group', name: 'Source Group' }
    };
    const draft: DiscoveryItem = {
      ...published,
      resource_id: 'draft-resource',
      version_id: 'draft-version',
      name: 'Draft dataset',
      status: 'DRAFT'
    };
    const copyResource = vi.fn();
    fixture.componentInstance.copyResource.subscribe(copyResource);
    fixture.componentRef.setInput('selectedGroupId', 'source-group');
    fixture.componentRef.setInput('canCopyResources', true);
    fixture.componentRef.setInput('discoveryItems', [published, draft]);
    fixture.detectChanges();

    const copyButtons = fixture.nativeElement.querySelectorAll('.copy-resource-button') as NodeListOf<HTMLButtonElement>;
    expect(copyButtons).toHaveLength(1);
    copyButtons[0].click();
    expect(copyResource).toHaveBeenCalledWith(published);
  });

  it('shows the selected group logo in the heading and group column', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);
    const groupId = '23b9b58a-9f1c-5d18-85f9-73842733835c';
    const item: DiscoveryItem = {
      type: 'dataset',
      resource_id: 'resource-id',
      version_id: 'version-id',
      name: 'Dataset',
      status: 'PUBLISHED',
      updated_at: '2026-09-10T07:34:26.486Z',
      concept: 'DATA',
      group: { id: groupId, name: 'Seismology' }
    };

    fixture.componentRef.setInput('selectedGroupId', groupId);
    fixture.componentRef.setInput('selectedGroupName', 'Seismology');
    fixture.componentRef.setInput('discoveryItems', [item]);
    fixture.detectChanges();

    expect((fixture.nativeElement.querySelector('.group-summary-logo') as HTMLImageElement).getAttribute('src')).toBe(`group-icons/${groupId}.png`);
    expect((fixture.nativeElement.querySelector('.group-cell-logo') as HTMLImageElement).getAttribute('src')).toBe(`group-icons/${groupId}.png`);
  });

  it('renders all-group results and links each item to its canonical section', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);
    const groupId = '23b9b58a-9f1c-5d18-85f9-73842733835c';
    const item: DiscoveryItem = {
      type: 'software-source-code',
      resource_id: 'resource-id',
      version_id: 'version-id',
      name: 'Facility software',
      status: 'PUBLISHED',
      updated_at: '2026-09-10T07:34:26.486Z',
      concept: 'FACILITIES',
      group: { id: groupId, name: 'Seismology' }
    };

    fixture.componentRef.setInput('sectionTitle', 'All Resources');
    fixture.componentRef.setInput('sectionPath', 'all');
    fixture.componentRef.setInput('allGroups', true);
    fixture.componentRef.setInput('returnUrl', '/all');
    fixture.componentRef.setInput('discoveryItems', [item]);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.subtitle')?.textContent?.trim()).toBe('Resources across all groups you belong to.');
    expect(fixture.nativeElement.querySelector('.group-summary-logo')).toBeNull();
    expect(fixture.nativeElement.querySelector('tbody tr')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('app-list-pagination')).toBeTruthy();
    expect((fixture.nativeElement.querySelector('.item-link') as HTMLAnchorElement).getAttribute('href')).toBe(`/facilities/resource-id/version-id?groupId=${groupId}&dataType=software-source-code&returnUrl=%2Fall`);
  });

  it('uses the shared pagination component and changes the page size', async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsTableComponent],
      providers: [provideRouter([])]
    }).compileComponents();
    const fixture = TestBed.createComponent(ResultsTableComponent);
    const items = Array.from({ length: 12 }, (_, index): DiscoveryItem => ({
      type: 'dataset',
      resource_id: `resource-${index + 1}`,
      version_id: `version-${index + 1}`,
      name: `Dataset ${index + 1}`,
      status: 'PUBLISHED',
      updated_at: '2026-09-10T07:34:26.486Z',
      concept: 'DATA',
      group: { id: 'group-id', name: 'Group' }
    }));

    fixture.componentRef.setInput('selectedGroupId', 'group-id');
    fixture.componentRef.setInput('discoveryItems', items);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('tbody tr')).toHaveLength(10);
    const pageSize = fixture.nativeElement.querySelector('.page-size-control select') as HTMLSelectElement;
    pageSize.value = '25';
    pageSize.dispatchEvent(new Event('change'));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('tbody tr')).toHaveLength(12);
    expect(fixture.nativeElement.textContent).toContain('Showing 1-12 of 12 results');
  });
});
