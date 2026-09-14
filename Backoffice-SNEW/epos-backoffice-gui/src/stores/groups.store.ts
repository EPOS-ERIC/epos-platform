import { Injectable, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import type { Group } from '../../generated/backofficeSchemas';
import { ApiService } from '../services/api.service';

@Injectable({ providedIn: 'root' })
export class GroupsStore {
  private readonly apiService = inject(ApiService);
  private initialization?: Promise<void>;

  readonly groups = signal<Group[]>([]);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  initialize(): Promise<void> {
    this.initialization ??= this.loadGroups();
    return this.initialization;
  }

  private async loadGroups(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      this.groups.set(await firstValueFrom(this.apiService.getGroups()));
    } catch (error) {
      console.error('Failed to load groups', error);
      this.error.set('Failed to load groups.');
    } finally {
      this.loading.set(false);
    }
  }
}
