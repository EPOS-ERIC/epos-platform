import { DOCUMENT } from '@angular/common';
import { TestBed } from '@angular/core/testing';
import { OAuthStorage } from 'angular-oauth2-oidc';

import { LocalStorageService } from './local-storage.service';

describe('LocalStorageService', () => {
  let service: LocalStorageService;
  let storage: Storage;

  beforeEach(() => {
    const values = new Map<string, string>();
    storage = {
      get length() {
        return values.size;
      },
      clear: () => values.clear(),
      getItem: (key) => values.get(key) ?? null,
      key: (index) => Array.from(values.keys())[index] ?? null,
      removeItem: (key) => values.delete(key),
      setItem: (key, value) => values.set(key, value)
    };
    TestBed.configureTestingModule({
      providers: [
        { provide: DOCUMENT, useValue: { defaultView: { localStorage: storage } } },
        { provide: OAuthStorage, useExisting: LocalStorageService }
      ]
    });
    storage.clear();
    service = TestBed.inject(LocalStorageService);
  });

  it('writes and reads JSON values', () => {
    const value = { searchQuery: 'volcano', statuses: ['PUBLISHED'] };

    expect(service.setJson('filters', value)).toBe(true);
    expect(service.getJson('filters')).toEqual(value);
  });

  it('returns null for invalid JSON', () => {
    storage.setItem('invalid', '{');

    expect(service.getJson('invalid')).toBeNull();
  });

  it('provides the OAuth storage interface using local storage', () => {
    const oauthStorage = TestBed.inject(OAuthStorage);

    expect(oauthStorage).toBe(service);
    oauthStorage.setItem('access_token', 'token');
    expect(storage.getItem('access_token')).toBe('token');
    expect(oauthStorage.getItem('access_token')).toBe('token');

    oauthStorage.removeItem('access_token');
    expect(storage.getItem('access_token')).toBeNull();
  });
});
