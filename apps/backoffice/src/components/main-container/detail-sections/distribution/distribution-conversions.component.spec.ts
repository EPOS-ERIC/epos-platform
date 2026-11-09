import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Subject, of, throwError } from 'rxjs';

import type { DistributionConversionPlugin } from '../../../../../generated/backofficeSchemas';
import { ApiService } from '../../../../services/api.service';
import { ToastService } from '../../../../services/toast.service';
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

  it('submits a valid conversion request and clears the form', async () => {
    const response = new Subject<void>();
    const requestConversion = vi.fn(() => response);
    const toast = { success: vi.fn(), error: vi.fn() };
    TestBed.overrideProvider(ApiService, {
      useValue: {
        listDistributionConversionRelations: () => of(plugins),
        requestDistributionConversion: requestConversion
      }
    });
    TestBed.overrideProvider(ToastService, { useValue: toast });
    const fixture = TestBed.createComponent(DistributionConversionsComponent);
    fixture.componentRef.setInput('distributionId', 'distribution-id');
    fixture.componentRef.setInput('nativeFormats', ['application/json']);
    fixture.detectChanges();

    const inputFormat = fixture.nativeElement.querySelector('.conversion-request select') as HTMLSelectElement;
    inputFormat.value = 'application/json';
    inputFormat.dispatchEvent(new Event('change'));
    const visualization = fixture.nativeElement.querySelector('.conversion-visualizations input') as HTMLInputElement;
    visualization.click();
    const message = fixture.nativeElement.querySelector('.conversion-request textarea') as HTMLTextAreaElement;
    message.value = ' Add a map conversion. ';
    message.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    const submit = fixture.nativeElement.querySelector('.conversion-request-submit') as HTMLButtonElement;
    expect(submit.disabled).toBe(false);
    submit.click();
    fixture.detectChanges();

    expect(requestConversion).toHaveBeenCalledWith('distribution-id', {
      input_format: 'application/json',
      visualizations: ['GRAPH'],
      message: 'Add a map conversion.'
    });
    expect(submit.textContent?.trim()).toBe('Sending...');
    expect(submit.disabled).toBe(true);

    response.next();
    response.complete();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(toast.success).toHaveBeenCalledWith('Conversion request submitted.');
    expect(inputFormat.value).toBe('');
    expect(message.value).toBe('');
    expect(visualization.checked).toBe(false);
  });

  it('shows the backend problem detail when a conversion request fails', () => {
    const toast = { success: vi.fn(), error: vi.fn() };
    TestBed.overrideProvider(ApiService, {
      useValue: {
        listDistributionConversionRelations: () => of(plugins),
        requestDistributionConversion: () => throwError(() => new HttpErrorResponse({
          status: 400,
          error: {
            type: 'validation_failed',
            title: 'Bad Request',
            status: 400,
            detail: 'Select a native format for this distribution.',
            target: 'input_format'
          }
        }))
      }
    });
    TestBed.overrideProvider(ToastService, { useValue: toast });
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const fixture = TestBed.createComponent(DistributionConversionsComponent);
    fixture.componentRef.setInput('distributionId', 'distribution-id');
    fixture.componentRef.setInput('nativeFormats', ['application/json']);
    fixture.detectChanges();

    const inputFormat = fixture.nativeElement.querySelector('.conversion-request select') as HTMLSelectElement;
    inputFormat.value = 'application/json';
    inputFormat.dispatchEvent(new Event('change'));
    (fixture.nativeElement.querySelector('.conversion-visualizations input') as HTMLInputElement).click();
    const message = fixture.nativeElement.querySelector('.conversion-request textarea') as HTMLTextAreaElement;
    message.value = 'Add a map conversion.';
    message.dispatchEvent(new Event('input'));
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.conversion-request-submit') as HTMLButtonElement).click();

    expect(toast.error).toHaveBeenCalledWith('Select a native format for this distribution.');
  });
});
