import { describe, expect, it } from 'vitest';

import { enumLabel } from './enum-label';

describe('enumLabel', () => {
  it('formats enum values as readable labels', () => {
    expect(enumLabel('THREE_TIMES_A_YEAR')).toBe('Three times a year');
    expect(enumLabel('START_DATE')).toBe('Start date');
    expect(enumLabel('')).toBe('None');
  });

  it('expands geographic abbreviations', () => {
    expect(enumLabel('MIN_LON')).toBe('Minimum longitude');
    expect(enumLabel('MAX_LAT')).toBe('Maximum latitude');
  });
});
