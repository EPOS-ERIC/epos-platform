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
    fixture.detectChanges();

    const cells = fixture.nativeElement.querySelectorAll('tbody td') as NodeListOf<HTMLTableCellElement>;
    expect(cells[4]?.textContent?.trim()).toBe(timestamp);
  });
});
