import { TestBed } from '@angular/core/testing';

import type { DatasetDistributionForm } from '../../../../models/dataset-detail-form';
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
      type: 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE',
      webService: { method: 'GET', type: 'DYNAMIC' }
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
      type: 'http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE',
      downloadableFile: { url: '', format: '' }
    });
  });
});
