import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { ComparisonFieldsDirective } from './comparison-fields.directive';

@Component({
  imports: [ComparisonFieldsDirective],
  template: `
    <div [appComparisonFields]="1">
      <section class="version-section-pane--left">
        <input value="Old title" />
        <input value="Same value" />
        <label><span class="detail-label">ID</span><input value="old-id" /></label>
        <div class="provider-row">Old provider</div>
      </section>
      <section class="version-section-pane--right">
        <input value="New title" />
        <input value="Same value" />
        <label><span class="detail-label">ID</span><input value="new-id" /></label>
        <div class="provider-row">New provider</div>
      </section>
    </div>
  `
})
class TestHostComponent {}

describe('ComparisonFieldsDirective', () => {
  it('highlights only rendered controls and read-only rows with different values', async () => {
    await TestBed.configureTestingModule({ imports: [TestHostComponent] }).compileComponents();
    const fixture = TestBed.createComponent(TestHostComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    const leftInputs = fixture.nativeElement.querySelectorAll('.version-section-pane--left input');
    const rightInputs = fixture.nativeElement.querySelectorAll('.version-section-pane--right input');
    expect(leftInputs[0].classList.contains('comparison-value--left')).toBe(true);
    expect(rightInputs[0].classList.contains('comparison-value--right')).toBe(true);
    expect(leftInputs[1].classList.contains('comparison-value--left')).toBe(false);
    expect(rightInputs[1].classList.contains('comparison-value--right')).toBe(false);
    expect(leftInputs[2].classList.contains('comparison-value--left')).toBe(false);
    expect(rightInputs[2].classList.contains('comparison-value--right')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left .provider-row').classList.contains('comparison-value--left')).toBe(true);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right .provider-row').classList.contains('comparison-value--right')).toBe(true);
  });
});
