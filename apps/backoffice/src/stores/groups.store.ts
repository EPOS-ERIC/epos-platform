import { Injectable, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import type { Concept, Group } from '../../generated/backofficeSchemas';
import { ApiService } from '../services/api.service';

@Injectable({ providedIn: 'root' })
export class GroupsStore {
  private readonly apiService = inject(ApiService);
  private initialization?: Promise<void>;

  readonly groups = signal<Group[]>([]);
  readonly concepts = signal<Concept[]>([]);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  initialize(): Promise<void> {
    this.initialization ??= this.loadGroups();
    return this.initialization;
  }

  replaceAll(groups: Group[]): void {
    this.groups.set(groups);
  }

  replaceConcepts(concepts: Concept[]): void {
    this.concepts.set(concepts);
  }

  conceptName(group: Group | undefined): string | undefined {
    return group ? this.concepts().find(({ id }) => id === group.concept_id)?.name.toUpperCase() : undefined;
  }
  // Updates or insert the Group
  upsert(group: Group): void {
    this.groups.update((groups) => {
      const existing = groups.some(({ id }) => id === group.id);
      return existing ? groups.map((item) => item.id === group.id ? group : item) : [...groups, group];
    });
  }

  remove(groupId: string): void {
    this.groups.update((groups) => groups.filter(({ id }) => id !== groupId));
  }

  private async loadGroups(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const [groups, concepts] = await Promise.all([
        firstValueFrom(this.apiService.getGroups()),
        firstValueFrom(this.apiService.getConcepts())
      ]);
      this.groups.set(groups);
      this.concepts.set(concepts);
    } catch (error) {
      console.error('Failed to load groups', error);
      this.error.set('Failed to load groups.');
    } finally {
      this.loading.set(false);
    }
  }
}
