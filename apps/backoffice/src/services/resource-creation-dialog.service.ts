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
  private sourceResourceId = '';

  readonly openState = signal(false);
  readonly mode = signal<'create' | 'copy'>('create');
  readonly isCopy = computed(() => this.mode() === 'copy');
  readonly title = signal('');
  readonly description = signal('');
  readonly resourceType = signal<DiscoveryResourceType>('dataset');
  readonly groupId = signal('');
  readonly saving = signal(false);
  readonly error = signal('');
  readonly groups = computed(() => this.groupsStore.groups().filter((group) =>
    this.currentUserStore.hasGroupPermission(group.id, 'group.resource.create')
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
    this.mode.set('create');
    this.sourceResourceId = '';
    this.title.set('');
    this.description.set('');
    this.resourceType.set('dataset');
    this.groupId.set(groups.some((group) => group.id === preferredGroupId) ? preferredGroupId : groups[0].id);
    this.error.set('');
    this.openState.set(true);
  }

  openCopy(
    resourceType: DiscoveryResourceType,
    resourceId: string,
    preferredGroupId = '',
    fallbackSectionPath = 'data'
  ): void {
    const groups = this.groups();
    if (!groups.length || !resourceId) return;

    this.fallbackSectionPath = fallbackSectionPath;
    this.mode.set('copy');
    this.sourceResourceId = resourceId;
    this.title.set('');
    this.description.set('');
    this.resourceType.set(resourceType);
    this.groupId.set(groups.some((group) => group.id === preferredGroupId) ? preferredGroupId : groups[0].id);
    this.error.set('');
    this.openState.set(true);
  }

  close(): void {
    if (this.saving()) {
      return;
    }

    this.openState.set(false);
    this.mode.set('create');
    this.sourceResourceId = '';
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
    const isCopy = this.isCopy();

    if (
      this.saving()
      || !groupId
      || !this.currentUserStore.hasGroupPermission(groupId, 'group.resource.create')
      || (!isCopy && (!title || !description))
      || (isCopy && !this.sourceResourceId)
    ) {
      return;
    }

    this.saving.set(true);
    this.error.set('');
    const request = isCopy
      ? this.apiService.copyResource(groupId, resourceType, this.sourceResourceId)
      : this.apiService.createResourceDraft(groupId, this.buildRequest(resourceType, title, description));
    request
      .pipe(
        switchMap((result) => this.resourceActions.resolveCreatedResourceRoute(result, groupId, this.fallbackSectionPath)),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.saving.set(false)),
        catchError((error) => {
          console.error(isCopy ? 'Failed to copy resource' : 'Failed to create resource draft', error);
          this.error.set(isCopy ? 'Failed to copy resource.' : 'Failed to create resource draft.');
          return of(null);
        })
      )
      .subscribe((route) => {
        if (!route) {
          this.error.set(isCopy
            ? 'Resource copied, but its detail page could not be opened.'
            : 'Created resource, but could not open its detail page.'
          );
          return;
        }

        this.openState.set(false);
        void this.router.navigate(route, { replaceUrl: true, queryParams: { groupId, dataType: resourceType } });
      });
  }

  private buildRequest(resourceType: DiscoveryResourceType, title: string, description: string): CreateResourceDraftRequest {
    if (resourceType === 'dataset') {
      return { resource_type: 'dataset', metadata: { title, description } };
    }

    if (resourceType === 'software-application') {
      return { resource_type: 'software-application', metadata: { name: title, description } };
    }

    return { resource_type: 'software-source-code', metadata: { name: title, description } };
  }
}
