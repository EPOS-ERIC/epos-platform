import { DestroyRef, Injectable, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { catchError, finalize, of, switchMap } from 'rxjs';

import type { CreateResourceDraftRequest, DiscoveryResourceType } from '../../generated/backofficeSchemas';
import { ApiService } from './api.service';
import { GroupsStore } from '../stores/groups.store';
import { ResourceActionsService } from './resource-actions.service';
import { CurrentUserStore } from '../stores/current-user.store';

@Injectable({ providedIn: 'root' })
export class ResourceCreationDialogService {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly groupsStore = inject(GroupsStore);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly resourceActions = inject(ResourceActionsService);
  private readonly router = inject(Router);
  private fallbackSectionPath = 'data';

  readonly openState = signal(false);
  readonly title = signal('');
  readonly description = signal('');
  readonly resourceType = signal<DiscoveryResourceType>('dataset');
  readonly groupId = signal('');
  readonly saving = signal(false);
  readonly error = signal('');
  readonly groups = computed(() => this.groupsStore.groups().filter((group) =>
    this.currentUserStore.hasGroupPermission(group.id, 'resource.create')
  ));
  readonly groupsLoading = this.groupsStore.loading;
  readonly resourceTypes: Array<{ label: string; value: DiscoveryResourceType }> = [
    { label: 'dataset', value: 'dataset' },
    { label: 'softwareapplication', value: 'software-application' },
    { label: 'softwaresourcecode', value: 'software-source-code' }
  ];

  open(preferredGroupId = '', fallbackSectionPath = 'data'): void {
    const groups = this.groups();
    if (!groups.length) return;

    this.fallbackSectionPath = fallbackSectionPath;
    this.title.set('');
    this.description.set('');
    this.resourceType.set('dataset');
    this.groupId.set(groups.some((group) => group.id === preferredGroupId) ? preferredGroupId : groups[0].id);
    this.error.set('');
    this.openState.set(true);
  }

  close(): void {
    if (this.saving()) {
      return;
    }

    this.openState.set(false);
    this.title.set('');
    this.description.set('');
    this.resourceType.set('dataset');
    this.error.set('');
  }

  submit(): void {
    const title = this.title().trim();
    const description = this.description().trim();
    const resourceType = this.resourceType();
    const groupId = this.groupId();

    if (
      this.saving()
      || !groupId
      || !this.currentUserStore.hasGroupPermission(groupId, 'resource.create')
      || !title
      || (resourceType !== 'dataset' && !description)
    ) {
      return;
    }

    const payload = this.buildRequest(resourceType, title, description);
    this.saving.set(true);
    this.error.set('');
    this.apiService
      .createResourceDraft(groupId, payload)
      .pipe(
        switchMap((result) => this.resourceActions.resolveCreatedResourceRoute(result, groupId, this.fallbackSectionPath)),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.saving.set(false)),
        catchError((error) => {
          console.error('Failed to create resource draft', error);
          this.error.set('Failed to create resource draft.');
          return of(null);
        })
      )
      .subscribe((route) => {
        if (!route) {
          this.error.set('Created resource, but could not open its detail page.');
          return;
        }

        this.openState.set(false);
        void this.router.navigate(route, { replaceUrl: true, queryParams: { groupId, dataType: resourceType } });
      });
  }

  private buildRequest(resourceType: DiscoveryResourceType, title: string, description: string): CreateResourceDraftRequest {
    if (resourceType === 'dataset') {
      return { resource_type: 'dataset', metadata: { title } };
    }

    if (resourceType === 'software-application') {
      return { resource_type: 'software-application', metadata: { name: title, description } };
    }

    return { resource_type: 'software-source-code', metadata: { name: title, description } };
  }
}
