import { TestBed } from '@angular/core/testing';
import { Subject, of, throwError } from 'rxjs';

import type { DistributionConversionPlugin } from '../../../../../generated/backofficeSchemas';
import { ApiService } from '../../../../services/api.service';
import { DistributionConversionsComponent } from './distribution-conversions.component';

describe('DistributionConversionsComponent', () => {
  const plugins: DistributionConversionPlugin[] = [{
    id: 'plugin-id',
    name: 'geo-converter',
    description: 'Converts JSON into GeoJSON.',
    repository: 'https://example.test/converter',
    enabled: true,
    installed: false,
    branch: 'main',
    runtime: 'python',
    arguments: ['--verbose'],
    relations: [{
      id: 'relation-id',
      input_format: 'application/json',
      output_format: 'application/geo+json',
      distribution_id: 'distribution-id'
    }]
  }];

  it('loads and renders the conversion plugins for a distribution', () => {
    const listConversions = vi.fn(() => of(plugins));
    TestBed.overrideProvider(ApiService, {
      useValue: { listDistributionConversionRelations: listConversions }
    });
    const fixture = TestBed.createComponent(DistributionConversionsComponent);
    fixture.componentRef.setInput('distributionId', 'distribution-id');
    fixture.detectChanges();

    expect(listConversions).toHaveBeenCalledWith('distribution-id');
    expect(fixture.nativeElement.textContent).toContain('geo-converter');
    expect(fixture.nativeElement.textContent).toContain('application/json');
    expect(fixture.nativeElement.textContent).toContain('application/geo+json');
    expect(fixture.nativeElement.textContent).toContain('python');
    expect(fixture.nativeElement.textContent).toContain('Enabled');
    expect(fixture.nativeElement.textContent).toContain('Not installed');
  });

  it('shows loading and then an empty state', () => {
    const response = new Subject<DistributionConversionPlugin[]>();
    TestBed.overrideProvider(ApiService, {
      useValue: { listDistributionConversionRelations: () => response }
    });
    const fixture = TestBed.createComponent(DistributionConversionsComponent);
    fixture.componentRef.setInput('distributionId', 'distribution-id');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Loading conversions...');
    response.next([]);
    response.complete();
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('No conversions are associated with this distribution.');
  });

  it('shows an error and retries the request', () => {
    const listConversions = vi.fn()
      .mockReturnValueOnce(throwError(() => new Error('unavailable')))
      .mockReturnValueOnce(of(plugins));
    TestBed.overrideProvider(ApiService, {
      useValue: { listDistributionConversionRelations: listConversions }
    });
    const fixture = TestBed.createComponent(DistributionConversionsComponent);
    fixture.componentRef.setInput('distributionId', 'distribution-id');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Failed to load conversions for this distribution.');
    (fixture.nativeElement.querySelector('.conversion-error button') as HTMLButtonElement).click();
    fixture.detectChanges();
    expect(listConversions).toHaveBeenCalledTimes(2);
    expect(fixture.nativeElement.textContent).toContain('geo-converter');
  });
});
