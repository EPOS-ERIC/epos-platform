import { TestBed } from '@angular/core/testing';

import { CapabilitiesService } from '../../services/capabilities.service';
import { ToastService } from '../../services/toast.service';
import { ToastOutletComponent, toastTargetAnchor } from './toast-outlet.component';

describe('ToastOutletComponent', () => {
  let issueTracker: { url: string; open_new: string } | null;
  const capabilitiesService = {
    issueTracker: () => issueTracker
  };

  beforeEach(() => {
    issueTracker = {
      url: 'https://issues.example.org',
      open_new: 'https://issues.example.org/new'
    };
    TestBed.configureTestingModule({
      imports: [ToastOutletComponent],
      providers: [{ provide: CapabilitiesService, useValue: capabilitiesService }]
    });
  });

  it('maps backend validation targets to detail section anchors', () => {
    expect(toastTargetAnchor('metadata.spatial_coverage')).toBe('spatial-coverage');
    expect(toastTargetAnchor('metadata.distributions.0.web_service.parameters.0.name')).toBe('webservice-parameters');
    expect(toastTargetAnchor('metadata.distributions.0.web_service.documentation_uri')).toBe('webservice-documentation');
    expect(toastTargetAnchor('metadata.distributions.0.web_service.spatial_coverage')).toBe('web-service');
    expect(toastTargetAnchor('metadata.description')).toBe('general-information');
    expect(toastTargetAnchor('request')).toBeNull();
  });

  it('renders a linked error heading for a single navigable issue', async () => {
    await TestBed.compileComponents();
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
    await TestBed.compileComponents();
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

  it('uses the issue creation URL from capabilities', async () => {
    await TestBed.compileComponents();
    const toastService = TestBed.inject(ToastService);
    toastService.error('Failed to save resource version.', {
      duration: null,
      issues: [{ message: 'Unexpected error.', reference: 'error-id' }]
    });

    const fixture = TestBed.createComponent(ToastOutletComponent);
    fixture.detectChanges();

    const element = fixture.nativeElement as HTMLElement;
    const link = element.querySelector<HTMLAnchorElement>('.toast-reference-help a');
    expect(link?.href).toBe('https://issues.example.org/new');
  });

  it('keeps the reporting guidance without a link when the capability is absent', async () => {
    issueTracker = null;
    await TestBed.compileComponents();
    const toastService = TestBed.inject(ToastService);
    toastService.error('Failed to save resource version.', {
      duration: null,
      issues: [{ message: 'Unexpected error.', reference: 'error-id' }]
    });

    const fixture = TestBed.createComponent(ToastOutletComponent);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.querySelector('.toast-reference-help a')).toBeNull();
    expect(element.querySelector('.toast-reference-help')?.textContent).toContain('Include this error ID in your report.');
  });
});
