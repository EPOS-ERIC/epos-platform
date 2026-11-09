import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';

import type { DatasetDistributionForm } from '../../../../models/dataset-detail-form';
import { ApiService } from '../../../../services/api.service';
import { DistributionComponent } from './distribution.component';

describe('DistributionComponent', () => {
  it('adds the first distribution from an empty dataset', () => {
    const fixture = TestBed.createComponent(DistributionComponent);
    const emissions: DatasetDistributionForm[][] = [];
    fixture.componentInstance.modelChange.subscribe((model) => emissions.push(model));
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.distribution-actions button') as HTMLButtonElement).click();
    fixture.detectChanges();

    expect(emissions).toHaveLength(1);
    expect(emissions[0]).toHaveLength(1);
    expect(emissions[0][0]).toMatchObject({
      kind: 'web-service',
      type: 'WEB_SERVICE',
      webService: { configuration: { method: 'GET', type: 'HTTP' } }
    });
  });

  it('does not add distributions in read-only mode', () => {
    const fixture = TestBed.createComponent(DistributionComponent);
    const emissions: DatasetDistributionForm[][] = [];
    fixture.componentInstance.modelChange.subscribe((model) => emissions.push(model));
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.distribution-actions')).toBeNull();
    expect(emissions).toHaveLength(0);
  });

  it('creates a downloadable file distribution when selected', () => {
    const fixture = TestBed.createComponent(DistributionComponent);
    const emissions: DatasetDistributionForm[][] = [];
    fixture.componentInstance.modelChange.subscribe((model) => emissions.push(model));
    fixture.detectChanges();

    const select = fixture.nativeElement.querySelector('.distribution-actions select') as HTMLSelectElement;
    select.value = 'downloadable-file';
    select.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.distribution-actions button') as HTMLButtonElement).click();

    expect(emissions[0][0]).toMatchObject({
      kind: 'downloadable-file',
      type: 'DOWNLOADABLE_FILE',
      downloadableFile: { url: '', format: '' }
    });
  });

  it('deletes the selected distribution and selects the adjacent one', () => {
    const fixture = TestBed.createComponent(DistributionComponent);
    const emissions: DatasetDistributionForm[][] = [];
    const model: DatasetDistributionForm[] = [
      {
        kind: 'downloadable-file',
        type: 'DOWNLOADABLE_FILE',
        title: 'First',
        description: 'First distribution',
        license: '',
        byteSize: '',
        downloadableFile: { url: 'https://example.test/first', format: 'text/csv' }
      },
      {
        kind: 'downloadable-file',
        type: 'DOWNLOADABLE_FILE',
        title: 'Second',
        description: 'Second distribution',
        license: '',
        byteSize: '',
        downloadableFile: { url: 'https://example.test/second', format: 'application/json' }
      }
    ];
    fixture.componentInstance.modelChange.subscribe((value) => emissions.push(value));
    fixture.componentRef.setInput('model', model);
    fixture.detectChanges();

    const tabs = fixture.nativeElement.querySelectorAll('.tab-button') as NodeListOf<HTMLButtonElement>;
    tabs[1].click();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.distribution-delete-button') as HTMLButtonElement).click();

    expect(emissions).toEqual([[model[0]]]);
    fixture.componentRef.setInput('model', emissions[0]);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.tab-button.active')?.textContent?.trim()).toBe('First');
  });

  it('keeps distribution and parameter navigation enabled in read-only mode', () => {
    TestBed.overrideProvider(ApiService, {
      useValue: {
        getContactPoints: () => of([]),
        getOrganizations: () => of([])
      }
    });
    const fixture = TestBed.createComponent(DistributionComponent);
    const emissions: DatasetDistributionForm[][] = [];
    fixture.componentInstance.modelChange.subscribe((model) => emissions.push(model));
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.distribution-actions button') as HTMLButtonElement).click();

    const model = emissions[0];
    const distribution = model[0];
    if (distribution.kind !== 'web-service') {
      throw new Error('Expected a web service distribution');
    }
    if (distribution.webService.configuration.type !== 'HTTP') {
      throw new Error('Expected an HTTP web service configuration');
    }
    distribution.webService.configuration.parameters = [
      {
        kind: 'BOOLEAN',
        id: 'first-id',
        position: 0,
        label: 'First parameter',
        name: 'first',
        description: '',
        property: '',
        location: 'QUERY',
        explode: false,
        required: false,
        readOnly: false,
        defaultValue: null
      },
      {
        kind: 'BOOLEAN',
        id: 'second-id',
        position: 1,
        label: 'Second parameter',
        name: 'second',
        description: '',
        property: 'searchable-property',
        location: 'QUERY',
        explode: false,
        required: false,
        readOnly: false,
        defaultValue: null
      }
    ];

    fixture.componentRef.setInput('model', model);
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    const distributionTab = fixture.nativeElement.querySelector('.tab-button') as HTMLButtonElement;
    const parameterItems = fixture.nativeElement.querySelectorAll('.parameter-list-item') as NodeListOf<HTMLButtonElement>;
    const searchInput = fixture.nativeElement.querySelector('.parameter-search-input') as HTMLInputElement;
    expect(distributionTab.disabled).toBe(false);
    expect(parameterItems[1].disabled).toBe(false);
    expect(searchInput.disabled).toBe(false);

    parameterItems[1].click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.parameter-editor h3')?.textContent?.trim()).toBe('Second parameter');
    expect((fixture.nativeElement.querySelector('.parameter-editor-fields') as HTMLFieldSetElement).disabled).toBe(true);
  });

  it('propagates mirrored comparison state and matches parameters through resource ids', () => {
    TestBed.overrideProvider(ApiService, {
      useValue: {
        getContactPoints: () => of([]),
        getOrganizations: () => of([])
      }
    });
    const fixture = TestBed.createComponent(DistributionComponent);
    const emissions: DatasetDistributionForm[][] = [];
    const selections: string[] = [];
    fixture.componentInstance.modelChange.subscribe((model) => emissions.push(model));
    fixture.componentInstance.comparisonSelectedParameterResourceIdChange.subscribe((resourceId) => selections.push(resourceId));
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.distribution-actions button') as HTMLButtonElement).click();

    const current = emissions[0];
    const currentDistribution = current[0];
    if (currentDistribution.kind !== 'web-service' || currentDistribution.webService.configuration.type !== 'HTTP') {
      throw new Error('Expected an HTTP web service distribution');
    }
    currentDistribution.resourceId = 'distribution-resource-id';
    currentDistribution.webService.configuration.parameters = [{
      kind: 'BOOLEAN',
      id: 'current-parameter-id',
      resourceId: 'parameter-resource-id',
      position: 0,
      label: 'Renamed parameter',
      name: 'renamed',
      description: '',
      property: '',
      location: 'QUERY',
      explode: false,
      required: false,
      readOnly: false,
      defaultValue: null
    }];
    const comparison = structuredClone(current);
    const comparisonDistribution = comparison[0];
    if (comparisonDistribution.kind !== 'web-service' || comparisonDistribution.webService.configuration.type !== 'HTTP') {
      throw new Error('Expected an HTTP comparison web service distribution');
    }
    comparisonDistribution.webService.configuration.parameters[0].id = 'previous-parameter-id';
    comparisonDistribution.webService.configuration.parameters[0].label = 'Original parameter';
    comparisonDistribution.webService.configuration.parameters[0].name = 'original';

    fixture.componentRef.setInput('model', current);
    fixture.componentRef.setInput('comparisonModel', comparison);
    fixture.componentRef.setInput('disabled', true);
    fixture.componentRef.setInput('comparisonMode', true);
    fixture.componentRef.setInput('comparisonSide', 'right');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.parameter-workspace--reversed')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.parameter-list-item')?.textContent).toContain('Changed');
    expect((fixture.nativeElement.querySelector('.parameter-editor') as HTMLElement).dataset['comparisonKey'])
      .toBe('resource:parameter-resource-id');
    (fixture.nativeElement.querySelector('.parameter-list-item') as HTMLButtonElement).click();
    expect(selections).toEqual(['parameter-resource-id']);
  });

  it('aligns right-side distribution tabs by resource id', () => {
    const fixture = TestBed.createComponent(DistributionComponent);
    const first: DatasetDistributionForm = {
      kind: 'downloadable-file',
      type: 'DOWNLOADABLE_FILE',
      resourceId: 'first-resource-id',
      title: 'First',
      description: '',
      license: '',
      byteSize: '',
      downloadableFile: { url: 'https://example.test/first', format: 'text/csv' }
    };
    const second: DatasetDistributionForm = {
      ...first,
      resourceId: 'second-resource-id',
      title: 'Second',
      downloadableFile: { url: 'https://example.test/second', format: 'application/json' }
    };

    fixture.componentRef.setInput('model', [second, first]);
    fixture.componentRef.setInput('comparisonModel', [first, second]);
    fixture.componentRef.setInput('disabled', true);
    fixture.componentRef.setInput('comparisonMode', true);
    fixture.componentRef.setInput('comparisonSide', 'right');
    fixture.detectChanges();

    const tabs = fixture.nativeElement.querySelectorAll('.tab-button') as NodeListOf<HTMLButtonElement>;
    expect(tabs[0].textContent?.trim()).toBe('First');
    expect(tabs[1].textContent?.trim()).toBe('Second');
    expect(fixture.nativeElement.querySelector('.distribution-level-title')?.textContent?.trim()).toBe('First');
  });

  it('marks added distributions and emits their resource id when selected', () => {
    const fixture = TestBed.createComponent(DistributionComponent);
    const selections: string[] = [];
    const existing: DatasetDistributionForm = {
      kind: 'downloadable-file',
      type: 'DOWNLOADABLE_FILE',
      resourceId: 'existing-resource-id',
      title: 'Existing',
      description: '',
      license: '',
      byteSize: '',
      downloadableFile: { url: 'https://example.test/existing', format: 'text/csv' }
    };
    const added: DatasetDistributionForm = {
      ...existing,
      resourceId: 'added-resource-id',
      title: 'Added distribution',
      downloadableFile: { url: 'https://example.test/added', format: 'application/json' }
    };
    fixture.componentInstance.comparisonSelectedDistributionResourceIdChange.subscribe(
      (resourceId) => selections.push(resourceId)
    );
    fixture.componentRef.setInput('model', [existing, added]);
    fixture.componentRef.setInput('comparisonModel', [structuredClone(existing)]);
    fixture.componentRef.setInput('disabled', true);
    fixture.componentRef.setInput('comparisonMode', true);
    fixture.componentRef.setInput('comparisonSide', 'right');
    fixture.detectChanges();

    const tabs = fixture.nativeElement.querySelectorAll('.tab-button') as NodeListOf<HTMLButtonElement>;
    expect(tabs[0].textContent).not.toContain('Changed');
    expect(tabs[1].textContent).toContain('Added');
    expect(tabs[1].classList.contains('tab-button--positive')).toBe(true);

    tabs[1].click();
    expect(selections).toEqual(['added-resource-id']);
  });

  it('keeps its current distribution when the selected counterpart is absent', () => {
    const fixture = TestBed.createComponent(DistributionComponent);
    const existing: DatasetDistributionForm = {
      kind: 'downloadable-file',
      type: 'DOWNLOADABLE_FILE',
      resourceId: 'existing-resource-id',
      title: 'Existing',
      description: '',
      license: '',
      byteSize: '',
      downloadableFile: { url: 'https://example.test/existing', format: 'text/csv' }
    };
    fixture.componentRef.setInput('model', [existing]);
    fixture.componentRef.setInput('comparisonModel', []);
    fixture.componentRef.setInput('disabled', true);
    fixture.componentRef.setInput('comparisonMode', true);
    fixture.componentRef.setInput('comparisonSide', 'left');
    fixture.detectChanges();

    fixture.componentRef.setInput('comparisonSelectedDistributionResourceId', 'missing-resource-id');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.tab-button.active')?.textContent).toContain('Existing');
    expect(fixture.nativeElement.querySelector('.distribution-level-title')?.textContent).toContain('Existing');
  });

  it('does not load current conversions in comparison mode', () => {
    const listConversions = vi.fn(() => of([]));
    TestBed.overrideProvider(ApiService, {
      useValue: {
        getContactPoints: () => of([]),
        getOrganizations: () => of([]),
        listDistributionConversionRelations: listConversions
      }
    });
    const fixture = TestBed.createComponent(DistributionComponent);
    const emissions: DatasetDistributionForm[][] = [];
    fixture.componentInstance.modelChange.subscribe((model) => emissions.push(model));
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.distribution-actions button') as HTMLButtonElement).click();

    const model = emissions[0];
    model[0].id = 'distribution-id';
    fixture.componentRef.setInput('model', model);
    fixture.componentRef.setInput('comparisonMode', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('app-distribution-conversions')).toBeNull();
    expect(listConversions).not.toHaveBeenCalled();
  });
});
