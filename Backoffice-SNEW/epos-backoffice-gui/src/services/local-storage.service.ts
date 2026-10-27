import { DOCUMENT } from '@angular/common';
import { Injectable, inject } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class LocalStorageService {
  private readonly document = inject(DOCUMENT);

  get(key: string): string | null {
    try {
      return this.document.defaultView?.localStorage.getItem(key) ?? null;
    } catch {
      return null;
    }
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
}
