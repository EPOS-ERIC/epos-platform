import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import type { Group } from '../../../generated/backofficeSchemas';
import { GroupSelectionComponent } from './group-selection.component';

describe('GroupSelectionComponent', () => {
  it('shows sorted groups for the current concept and links them to the results route', () => {
    const groups: Group[] = [
      { id: 'data-2', name: 'Second Data Group', position: 2, concept: 'DATA' },
      { id: 'software-1', name: 'Software Group', position: 1, concept: 'SOFTWARE' },
      { id: 'data-1', name: 'First Data Group', label: 'First Group', description: 'Group description', logo: 'https://example.org/logo.png', position: 1, concept: 'DATA' }
    ];
    TestBed.configureTestingModule({
      imports: [GroupSelectionComponent],
      providers: [provideRouter([])]
    });
    const fixture = TestBed.createComponent(GroupSelectionComponent);
    fixture.componentRef.setInput('groups', groups);
    fixture.componentRef.setInput('concept', 'DATA');
    fixture.componentRef.setInput('sectionPath', 'data');
    fixture.componentRef.setInput('sectionTitle', 'Data');
    fixture.detectChanges();
    const cards = fixture.nativeElement.querySelectorAll('.group-selection-card') as NodeListOf<HTMLAnchorElement>;

    expect(cards).toHaveLength(2);
    expect(cards[0].textContent).toContain('First Group');
    expect(cards[0].querySelector('.group-selection-description')?.textContent).toContain('Group description');
    expect(cards[0].getAttribute('href')).toBe('/data?groupId=data-1');
    expect(cards[1].getAttribute('href')).toBe('/data?groupId=data-2');
    expect((cards[0].querySelector('img') as HTMLImageElement).getAttribute('src')).toBe('https://example.org/logo.png');
    expect(fixture.nativeElement.textContent).not.toContain('Software Group');
  });
});
