import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import { ApiService } from '../../../../services/api.service';
import { CategoriesComponent } from './categories.component';

describe('CategoriesComponent', () => {
  it('applies the neutral appearance used by comparison mode', () => {
    TestBed.configureTestingModule({
      imports: [CategoriesComponent],
      providers: [{ provide: ApiService, useValue: { getGroupCategories: vi.fn(() => of([])) } }]
    });
    const fixture = TestBed.createComponent(CategoriesComponent);
    fixture.componentRef.setInput('model', [{ id: 'category-id', name: 'Category' }]);
    fixture.componentRef.setInput('neutralAppearance', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.detail-section-card--neutral')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.identifier-item')?.textContent).toContain('Category');
  });

  it('shows only selected categories when disabled', () => {
    TestBed.configureTestingModule({
      imports: [CategoriesComponent],
      providers: [{ provide: ApiService, useValue: { getGroupCategories: vi.fn(() => of([])) } }]
    });
    const fixture = TestBed.createComponent(CategoriesComponent);
    fixture.componentRef.setInput('groupId', 'group-id');
    fixture.componentRef.setInput('model', [{ id: 'category-id', name: 'Selected Category' }]);
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.category-row--selected')?.textContent).toContain('Selected Category');
    expect(fixture.nativeElement.querySelector('.category-badge--available')).toBeNull();
    expect(fixture.nativeElement.querySelector('.category-remove')).toBeNull();
    expect(fixture.nativeElement.querySelector('.subtitle')).toBeNull();
  });

});
