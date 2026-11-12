import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Group } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { GlobalGroupsPanelComponent } from './global-groups-panel.component';

describe('GlobalGroupsPanelComponent', () => {
  const group: Group = {
    id: 'group-1',
    name: 'Group 01',
    label: 'First group',
    position: 1,
    concept_id: 'data-concept'
  };
  const concepts = [
    { id: 'data-concept', name: 'Data' },
    { id: 'software-concept', name: 'Software' }
  ];
  const apiService = {
    getGroups: vi.fn(),
    getConcepts: vi.fn(),
    createGroup: vi.fn(),
    replaceGroup: vi.fn(),
    deleteGroup: vi.fn()
  };

  beforeEach(() => {
    vi.clearAllMocks();
    apiService.getGroups.mockReturnValue(of([group]));
    apiService.getConcepts.mockReturnValue(of(concepts));
    TestBed.configureTestingModule({
      imports: [GlobalGroupsPanelComponent],
      providers: [{ provide: ApiService, useValue: apiService }]
    });
  });

  it('searches and paginates groups', async () => {
    const groups = Array.from({ length: 12 }, (_, index): Group => ({
      id: `group-${index + 1}`,
      name: `Group ${String(index + 1).padStart(2, '0')}`,
      position: index + 1,
      concept_id: index % 2 ? 'software-concept' : 'data-concept'
    }));
    apiService.getGroups.mockReturnValue(of(groups));
    const fixture = TestBed.createComponent(GlobalGroupsPanelComponent);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.querySelectorAll('.group-item')).toHaveLength(10);
    clickButton(element, 'Next');
    fixture.detectChanges();
    expect(element.querySelectorAll('.group-item')).toHaveLength(2);
    expect(element.textContent).toContain('Group 11');

    const search = element.querySelector<HTMLInputElement>('input[type="search"]')!;
    search.value = 'Group 01';
    search.dispatchEvent(new Event('input', { bubbles: true }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(element.querySelectorAll('.group-item')).toHaveLength(1);
    expect(element.textContent).toContain('Page 1 of 1');
  });

  it('creates a group', async () => {
    const created: Group = { id: 'group-2', name: 'New Group', position: 0, concept_id: 'data-concept' };
    apiService.createGroup.mockReturnValue(of(created));
    const fixture = TestBed.createComponent(GlobalGroupsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Add New Group');
    fixture.detectChanges();
    await fixture.whenStable();
    setInput(fixture.nativeElement, 'name', created.name);
    await fixture.whenStable();
    submitForm(fixture.nativeElement);
    fixture.detectChanges();

    expect(apiService.createGroup).toHaveBeenCalledWith({
      name: created.name,
      label: null,
      concept_id: 'data-concept',
      position: undefined,
      description: null,
      logo: null,
      page: null,
      color: null
    });
    expect(fixture.nativeElement.textContent).toContain(created.name);
  });

  it('replaces an existing group', async () => {
    const updated: Group = { ...group, label: 'Updated label' };
    apiService.replaceGroup.mockReturnValue(of(updated));
    const fixture = TestBed.createComponent(GlobalGroupsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Edit');
    fixture.detectChanges();
    await fixture.whenStable();
    setInput(fixture.nativeElement, 'label', updated.label!);
    await fixture.whenStable();
    submitForm(fixture.nativeElement);
    fixture.detectChanges();

    expect(apiService.replaceGroup).toHaveBeenCalledWith(group.id, {
      name: group.name,
      label: updated.label,
      concept_id: group.concept_id,
      position: group.position,
      description: null,
      logo: null,
      page: null,
      color: null
    });
    expect(fixture.nativeElement.textContent).toContain(updated.label);
  });

  it('deletes a group after confirmation', () => {
    apiService.deleteGroup.mockReturnValue(of(undefined));
    const fixture = TestBed.createComponent(GlobalGroupsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Remove');
    fixture.detectChanges();
    clickButton(fixture.nativeElement, 'Remove group');
    fixture.detectChanges();

    expect(apiService.deleteGroup).toHaveBeenCalledWith(group.id);
    expect(fixture.nativeElement.textContent).not.toContain(group.name);
  });

  function clickButton(element: HTMLElement, label: string): void {
    const button = Array.from(element.querySelectorAll<HTMLButtonElement>('button'))
      .find((candidate) => candidate.textContent?.replace(/\s+/g, ' ').trim().includes(label));
    expect(button).toBeDefined();
    button!.click();
  }

  function setInput(element: HTMLElement, name: string, value: string): void {
    const input = element.querySelector<HTMLInputElement>(`[name="${name}"]`)!;
    input.value = value;
    input.dispatchEvent(new Event('input', { bubbles: true }));
  }

  function submitForm(element: HTMLElement): void {
    element.querySelector<HTMLFormElement>('form')!.dispatchEvent(new Event('submit'));
  }
});
