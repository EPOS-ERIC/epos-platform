import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import type { DiscoveryItem } from '../../../generated/backofficeSchemas';
import { ResultsTableComponent } from './results-table.component';

describe('ResultsTableComponent', () => {
  it('renders the API timestamp without converting it to local time', async () => {
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
    expect(cells[4]?.textContent?.trim()).toBe(timestamp);
    expect((fixture.nativeElement.querySelector('.item-link') as HTMLAnchorElement).getAttribute('href')).toBe('/data/resource-id/version-id?groupId=group-id&dataType=dataset&returnUrl=%2Fdata%3FgroupId%3Dgroup-id');
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
});
