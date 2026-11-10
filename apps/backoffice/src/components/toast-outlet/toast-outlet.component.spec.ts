import { TestBed } from '@angular/core/testing';

import { ToastService } from '../../services/toast.service';
import { ToastOutletComponent, toastTargetAnchor } from './toast-outlet.component';

describe('ToastOutletComponent', () => {
  it('maps backend validation targets to detail section anchors', () => {
    expect(toastTargetAnchor('metadata.spatial_coverage')).toBe('spatial-coverage');
    expect(toastTargetAnchor('metadata.distributions.0.web_service.parameters.0.name')).toBe('webservice-parameters');
    expect(toastTargetAnchor('metadata.distributions.0.web_service.documentation_uri')).toBe('webservice-documentation');
    expect(toastTargetAnchor('metadata.distributions.0.web_service.spatial_coverage')).toBe('web-service');
    expect(toastTargetAnchor('metadata.description')).toBe('general-information');
    expect(toastTargetAnchor('request')).toBeNull();
  });

  it('renders a linked error heading for a single navigable issue', async () => {
    await TestBed.configureTestingModule({ imports: [ToastOutletComponent] }).compileComponents();
    const toastService = TestBed.inject(ToastService);
    toastService.error('Failed to save resource version.', {
      duration: null,
      issues: [{ message: 'Value must be valid WKT geometry.', target: 'metadata.spatial_coverage' }]
    });

    const fixture = TestBed.createComponent(ToastOutletComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.toast-title-link')?.textContent.trim()).toBe('Something went wrong');
  });

  it('keeps the error heading as text when the issue has no target', async () => {
    await TestBed.configureTestingModule({ imports: [ToastOutletComponent] }).compileComponents();
    const toastService = TestBed.inject(ToastService);
    toastService.error('Failed to save resource version.', {
      duration: null,
      issues: [{ message: 'Invalid value.' }]
    });

    const fixture = TestBed.createComponent(ToastOutletComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.toast-title-link')).toBeNull();
    expect(fixture.nativeElement.querySelector('.toast-title')?.textContent.trim()).toBe('Something went wrong');
  });
});
