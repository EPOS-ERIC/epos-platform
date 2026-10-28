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

    const button = fixture.nativeElement.querySelector('.distribution-actions button') as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    button.click();

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
});
