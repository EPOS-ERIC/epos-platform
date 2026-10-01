import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import { ApiService } from '../../../../services/api.service';
import { CategoriesComponent } from './categories.component';

describe('CategoriesComponent', () => {
  it('applies the neutral appearance used by comparison mode', () => {
    TestBed.configureTestingModule({
      imports: [CategoriesComponent],
      providers: [{ provide: ApiService, useValue: { getGroupCategories: vi.fn() } }]
    });
    const fixture = TestBed.createComponent(CategoriesComponent);
    fixture.componentRef.setInput('model', [{ id: 'category-id', name: 'Category' }]);
    fixture.componentRef.setInput('neutralAppearance', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.detail-section-card--neutral')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.identifier-item')?.textContent).toContain('Category');
  });
});
