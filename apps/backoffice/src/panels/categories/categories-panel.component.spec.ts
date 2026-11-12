import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { Category, Group } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { CategoriesPanelComponent } from './categories-panel.component';

describe('CategoriesPanelComponent', () => {
  const group: Group = {
    id: 'group-1',
    name: 'Data Group',
    position: 1,
    concept_id: 'data-concept'
  };
  const categories = Array.from({ length: 12 }, (_, index): Category => ({
    id: `category-${index + 1}`,
    group_id: group.id,
    parent_category_id: null,
    name: `Category ${String(index + 1).padStart(2, '0')}`
  }));
  const apiService = { getGroupCategories: vi.fn(() => of(categories)) };

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      imports: [CategoriesPanelComponent],
      providers: [{ provide: ApiService, useValue: apiService }]
    });
  });

  it('paginates categories in groups of ten', () => {
    const fixture = TestBed.createComponent(CategoriesPanelComponent);
    fixture.componentRef.setInput('groups', [group]);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.querySelectorAll('.category-tree-item')).toHaveLength(10);
    expect(element.textContent).not.toContain('Category 11');

    clickButton(element, 'Next');
    fixture.detectChanges();

    expect(element.querySelectorAll('.category-tree-item')).toHaveLength(2);
    expect(element.textContent).toContain('Category 11');
    expect(element.textContent).toContain('Page 2 of 2');

    const pageSize = element.querySelector<HTMLSelectElement>('.page-size-control select')!;
    pageSize.value = '25';
    pageSize.dispatchEvent(new Event('change'));
    fixture.detectChanges();

    expect(element.querySelectorAll('.category-tree-item')).toHaveLength(12);
    expect(element.textContent).toContain('Page 1 of 1');
  });

  function clickButton(element: HTMLElement, label: string): void {
    const button = Array.from(element.querySelectorAll<HTMLButtonElement>('button'))
      .find((candidate) => candidate.textContent?.trim() === label);
    expect(button).toBeDefined();
    button!.click();
  }
});
