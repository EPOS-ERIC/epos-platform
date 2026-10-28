import { TestBed } from '@angular/core/testing';

import { LocalStorageService } from './local-storage.service';

describe('LocalStorageService', () => {
  let service: LocalStorageService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    localStorage.clear();
    service = TestBed.inject(LocalStorageService);
  });

  it('writes and reads JSON values', () => {
    const value = { searchQuery: 'volcano', statuses: ['PUBLISHED'] };

    expect(service.setJson('filters', value)).toBe(true);
    expect(service.getJson('filters')).toEqual(value);
  });

  it('returns null for invalid JSON', () => {
    localStorage.setItem('invalid', '{');

    expect(service.getJson('invalid')).toBeNull();
  });
});
