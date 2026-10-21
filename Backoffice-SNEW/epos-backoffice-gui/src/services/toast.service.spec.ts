import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import { ToastService } from './toast.service';

describe('ToastService', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    TestBed.configureTestingModule({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('keeps the toast mounted while the exit transition runs', () => {
    const service = TestBed.inject(ToastService);

    service.success('Saved');
    expect(service.state()).toEqual({ message: 'Saved', tone: 'success', visible: true });

    vi.advanceTimersByTime(3500);
    expect(service.state()?.visible).toBe(false);

    vi.advanceTimersByTime(180);
    expect(service.state()).toBeNull();
  });

  it('keeps persistent error details visible until they are dismissed', () => {
    const service = TestBed.inject(ToastService);
    const issues = [{
      label: 'Spatial coverage',
      message: 'Value must be valid WKT geometry.',
      target: 'metadata.spatial_coverage'
    }];

    service.error('Failed to save resource version.', { duration: null, issues });
    vi.advanceTimersByTime(10_000);

    expect(service.state()).toEqual({
      message: 'Failed to save resource version.',
      tone: 'error',
      visible: true,
      issues
    });

    service.dismiss();
    vi.advanceTimersByTime(180);
    expect(service.state()).toBeNull();
  });
});
