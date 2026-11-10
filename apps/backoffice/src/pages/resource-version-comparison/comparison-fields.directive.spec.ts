import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { ComparisonFieldsDirective } from './comparison-fields.directive';

@Component({
  imports: [ComparisonFieldsDirective],
  template: `
    <div [appComparisonFields]="1">
      <section class="version-section-pane--left">
        <input type="search" class="parameter-search-input" value="navigation only" />
        <input class="field-input" value="Old title" />
        <input class="field-input" value="Same value" />
        <label><span class="detail-label">ID</span><input class="field-input" value="old-id" /></label>
        <div class="provider-row" data-comparison-key="provider:a">Provider A</div>
        <div class="contact-point-row" data-comparison-key="contact-point:a">Contact A</div>
        <div class="category-row" data-comparison-key="category:a">Category A</div>
        <div class="category-row" data-comparison-key="category:b">Category B</div>
        <div data-comparison-resource-id="item-a"><input class="keyed-a" value="Same keyed value" /></div>
        <div data-comparison-resource-id="item-b"><input class="keyed-b" value="Old keyed value" /></div>
        <section class="distribution-body" data-comparison-key="resource:existing" data-comparison-status="unchanged">
          <input class="distribution-field" value="Existing distribution" />
        </section>
        <section class="parameter-editor" data-comparison-key="id:existing" data-comparison-status="unchanged">
          <input class="parameter-field" value="Existing parameter" />
        </section>
      </section>
      <section class="version-section-pane--right">
        <input class="field-input" value="New title" />
        <input class="field-input" value="Same value" />
        <label><span class="detail-label">ID</span><input class="field-input" value="new-id" /></label>
        <div class="provider-row" data-comparison-key="provider:b">Provider B</div>
        <div class="provider-row" data-comparison-key="provider:a">Provider A</div>
        <div class="contact-point-row" data-comparison-key="contact-point:b">Contact B</div>
        <div class="contact-point-row" data-comparison-key="contact-point:a">Contact A</div>
        <div class="category-row" data-comparison-key="category:b">Category B</div>
        <div class="category-row" data-comparison-key="category:a">Category A</div>
        <div class="category-row" data-comparison-key="category:c">Category C</div>
        <div data-comparison-resource-id="item-b"><input class="keyed-b" value="New keyed value" /></div>
        <div data-comparison-resource-id="item-a"><input class="keyed-a" value="Same keyed value" /></div>
        <section class="distribution-body" data-comparison-key="resource:added" data-comparison-status="added">
          <input class="distribution-field" value="Added distribution" />
        </section>
        <section class="parameter-editor" data-comparison-key="id:added" data-comparison-status="added">
          <input class="parameter-field" value="Added parameter" />
        </section>
      </section>
    </div>
  `
})
class TestHostComponent {}

@Component({
  imports: [ComparisonFieldsDirective],
  template: `
    <div [appComparisonFields]="1">
      <section class="version-section-pane--left">
        <section class="distribution-body" data-comparison-resource-id="distribution-a" data-comparison-key="resource:distribution-a" data-comparison-status="changed">
          <label><span class="detail-label">Title</span><input class="distribution-title" value="Old distribution title" /></label>
          <label><span class="detail-label">Title</span><input class="web-service-title" value="Same web service title" /></label>
        </section>
      </section>
      <section class="version-section-pane--right">
        <section class="distribution-body" data-comparison-resource-id="distribution-a" data-comparison-key="resource:distribution-a" data-comparison-status="changed">
          <label><span class="detail-label">Title</span><input class="distribution-title" value="New distribution title" /></label>
          <label><span class="detail-label">Title</span><input class="web-service-title" value="Same web service title" /></label>
        </section>
      </section>
    </div>
  `
})
class MatchedDistributionHostComponent {}

@Component({
  imports: [ComparisonFieldsDirective],
  template: `
    <div [appComparisonFields]="1">
      <section class="version-section-pane--left">
        <section class="parameter-editor" data-comparison-key="resource:first" data-comparison-status="unchanged">
          <input class="first-parameter-field" value="Same value" />
        </section>
        <section class="parameter-editor" data-comparison-key="resource:second" data-comparison-status="changed">
          <input class="second-parameter-field" value="Old value" />
        </section>
      </section>
      <section class="version-section-pane--right">
        <section class="parameter-editor" data-comparison-key="resource:first" data-comparison-status="unchanged">
          <input class="first-parameter-field" value="Same value" />
        </section>
        <section class="parameter-editor" data-comparison-key="resource:second" data-comparison-status="changed">
          <input class="second-parameter-field" value="New value" />
        </section>
      </section>
    </div>
  `
})
class MultipleParameterEditorsHostComponent {}

describe('ComparisonFieldsDirective', () => {
  it('highlights only rendered controls and read-only rows with different values', async () => {
    await TestBed.configureTestingModule({ imports: [TestHostComponent] }).compileComponents();
    const fixture = TestBed.createComponent(TestHostComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    const leftInputs = fixture.nativeElement.querySelectorAll('.version-section-pane--left .field-input');
    const rightInputs = fixture.nativeElement.querySelectorAll('.version-section-pane--right .field-input');
    expect(leftInputs[0].classList.contains('comparison-value--left')).toBe(true);
    expect(rightInputs[0].classList.contains('comparison-value--right')).toBe(true);
    expect(leftInputs[1].classList.contains('comparison-value--left')).toBe(false);
    expect(rightInputs[1].classList.contains('comparison-value--right')).toBe(false);
    expect(leftInputs[2].classList.contains('comparison-value--left')).toBe(false);
    expect(rightInputs[2].classList.contains('comparison-value--right')).toBe(false);
    expect(fixture.nativeElement.querySelector('.parameter-search-input').classList.contains('comparison-value--left')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left [data-comparison-key="provider:a"]').classList.contains('comparison-value--left')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right [data-comparison-key="provider:a"]').classList.contains('comparison-value--right')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right [data-comparison-key="provider:b"]').classList.contains('comparison-value--right')).toBe(true);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left [data-comparison-key="contact-point:a"]').classList.contains('comparison-value--left')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right [data-comparison-key="contact-point:a"]').classList.contains('comparison-value--right')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right [data-comparison-key="contact-point:b"]').classList.contains('comparison-value--right')).toBe(true);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left [data-comparison-key="category:a"]').classList.contains('comparison-value--left')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right [data-comparison-key="category:a"]').classList.contains('comparison-value--right')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right [data-comparison-key="category:c"]').classList.contains('comparison-value--right')).toBe(true);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left .keyed-a').classList.contains('comparison-value--left')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right .keyed-a').classList.contains('comparison-value--right')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left .keyed-b').classList.contains('comparison-value--left')).toBe(true);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right .keyed-b').classList.contains('comparison-value--right')).toBe(true);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left .distribution-field').classList.contains('comparison-value--left')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right .distribution-field').classList.contains('comparison-value--right')).toBe(true);
    expect(fixture.nativeElement.querySelector('.version-section-pane--left .parameter-field').classList.contains('comparison-value--left')).toBe(false);
    expect(fixture.nativeElement.querySelector('.version-section-pane--right .parameter-field').classList.contains('comparison-value--right')).toBe(true);
  });

  it('compares repeated field labels independently within a matched distribution', async () => {
    await TestBed.configureTestingModule({ imports: [MatchedDistributionHostComponent] }).compileComponents();
    const fixture = TestBed.createComponent(MatchedDistributionHostComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    const leftDistributionTitle = fixture.nativeElement.querySelector('.version-section-pane--left .distribution-title');
    const rightDistributionTitle = fixture.nativeElement.querySelector('.version-section-pane--right .distribution-title');
    const leftWebServiceTitle = fixture.nativeElement.querySelector('.version-section-pane--left .web-service-title');
    const rightWebServiceTitle = fixture.nativeElement.querySelector('.version-section-pane--right .web-service-title');
    expect(leftDistributionTitle.classList.contains('comparison-value--left')).toBe(true);
    expect(rightDistributionTitle.classList.contains('comparison-value--right')).toBe(true);
    expect(leftWebServiceTitle.classList.contains('comparison-value--left')).toBe(false);
    expect(rightWebServiceTitle.classList.contains('comparison-value--right')).toBe(false);
  });

  it('compares parameter editors from every web service parameter collection', async () => {
    await TestBed.configureTestingModule({ imports: [MultipleParameterEditorsHostComponent] }).compileComponents();
    const fixture = TestBed.createComponent(MultipleParameterEditorsHostComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    const leftPane = fixture.nativeElement.querySelector('.version-section-pane--left');
    const rightPane = fixture.nativeElement.querySelector('.version-section-pane--right');
    expect(leftPane.querySelector('.first-parameter-field').classList.contains('comparison-value--left')).toBe(false);
    expect(rightPane.querySelector('.first-parameter-field').classList.contains('comparison-value--right')).toBe(false);
    expect(leftPane.querySelector('.second-parameter-field').classList.contains('comparison-value--left')).toBe(true);
    expect(rightPane.querySelector('.second-parameter-field').classList.contains('comparison-value--right')).toBe(true);
  });
});
