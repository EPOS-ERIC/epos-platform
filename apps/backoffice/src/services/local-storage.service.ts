import { DOCUMENT } from '@angular/common';
import { Injectable, inject } from '@angular/core';
import type { OAuthStorage } from 'angular-oauth2-oidc';

@Injectable({ providedIn: 'root' })
export class LocalStorageService implements OAuthStorage {
  private readonly document = inject(DOCUMENT);

  get(key: string): string | null {
    try {
      return this.document.defaultView?.localStorage.getItem(key) ?? null;
    } catch {
      return null;
    }
  }

  getItem(key: string): string | null {
    return this.get(key);
  }

  getJson(key: string): unknown {
    const value = this.get(key);

    if (value === null) {
      return null;
    }

    try {
      return JSON.parse(value) as unknown;
    } catch {
      return null;
    }
  }

  set(key: string, value: string): boolean {
    try {
      this.document.defaultView?.localStorage.setItem(key, value);
      return Boolean(this.document.defaultView);
    } catch {
      return false;
    }
  }

  setItem(key: string, value: string): void {
    this.set(key, value);
  }

  setJson(key: string, value: unknown): boolean {
    try {
      return this.set(key, JSON.stringify(value));
    } catch {
      return false;
    }
  }

  remove(key: string): void {
    try {
      this.document.defaultView?.localStorage.removeItem(key);
    } catch {
      // Storage can be unavailable in restricted browser contexts.
    }
  }

  removeItem(key: string): void {
    this.remove(key);
  }
}
