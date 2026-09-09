import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, defer, finalize, map, of, switchMap, tap } from 'rxjs';

import type {
  RejectResourceRequest,
  ResourceVersion,
  VersioningStatus,
  WriteResult
} from '../../generated/backofficeSchemas';
import type { ResourceActionItem, ResourceActionKey, ResourceVersionContext, SaveResourceContext } from '../models/resource-actions';
import { ApiService } from './api.service';
import { GroupsStore } from './groups.store';
import { ToastService } from './toast.service';

@Injectable({ providedIn: 'root' })
export class ResourceActionsService {
  private readonly apiService = inject(ApiService);
  private readonly groupsStore = inject(GroupsStore);
  private readonly toastService = inject(ToastService);
  private readonly pendingActionState = signal<ResourceActionKey | null>(null);
  private readonly errorState = signal('');

  readonly pendingAction = this.pendingActionState.asReadonly();
  readonly error = this.errorState.asReadonly();
  readonly saving = computed(() => this.pendingActionState() === 'save' || this.pendingActionState() === 'submit');
  readonly publishing = computed(() => this.pendingActionState() === 'publish');
  readonly creatingDraft = computed(() => this.pendingActionState() === 'create-new-draft');
  readonly archiving = computed(() => this.pendingActionState() === 'archive');
  readonly rejecting = computed(() => this.pendingActionState() === 'reject');
  readonly reopening = computed(() => this.pendingActionState() === 'revert-to-draft');
  readonly deleting = computed(() => this.pendingActionState() === 'delete');

  clearError(): void {
    this.errorState.set('');
  }

  setError(message: string): void {
    this.errorState.set(message);
  }

  sidebarActions(status: VersioningStatus | null, isAdmin: boolean): ResourceActionItem[] {
    const pending = this.pendingActionState();

    if (status === 'PUBLISHED') {
      return [
        this.action('create-new-draft', 'Create New Draft', 'Creating...', 'primary', pending),
        this.action('archive', 'Archive', 'Archiving...', 'danger', pending)
      ];
    }

    if (status === 'ARCHIVED') {
      return [this.action('create-new-draft', 'Create New Draft', 'Creating...', 'primary', pending)];
    }

    if (status === 'REJECTED') {
      return [this.action('revert-to-draft', 'Revert to Draft', 'Reverting...', 'primary', pending)];
    }

    if (status === 'SUBMITTED') {
      return isAdmin
        ? [
            this.action('publish', 'Publish', 'Publishing...', 'primary', pending),
            this.action('reject', 'Reject', undefined, 'danger', pending)
          ]
        : [];
    }

    if (status === 'DRAFT') {
      return [
        this.action('save', 'Save', 'Saving...', 'primary', pending),
        this.action('submit', 'Submit', undefined, 'ghost', pending)
      ];
    }

    return [];
  }

  createDraft(
    context: ResourceVersionContext,
    status: 'PUBLISHED' | 'ARCHIVED',
    fallbackSectionPath: string
  ): Observable<readonly [string, string, string, string] | null> {
    return this.run(
      'create-new-draft',
      () => {
        const request = status === 'PUBLISHED'
          ? this.apiService.createDraftFromPublication(context.resourceType, context.resourceId)
          : this.apiService.createDraftFromArchive(context.resourceType, context.resourceId);
        return request.pipe(switchMap((result) => this.resolveCreatedResourceRoute(result, context.groupId, fallbackSectionPath)));
      },
      `Failed to create a new draft from this ${status.toLowerCase()} resource.`,
      `Failed to create draft from ${status.toLowerCase()}`
    );
  }

  archive(context: ResourceVersionContext, resourceName: string): Observable<WriteResult | null> {
    return this.run(
      'archive',
      () => this.apiService.unpublishResourceVersion(context.resourceType, context.resourceId, context.versionId),
      'Failed to archive this resource version.',
      'Failed to archive published resource version',
      `Successfully Archived ${resourceName}`
    );
  }

  submit(context: ResourceVersionContext, resourceName: string): Observable<WriteResult | null> {
    return this.run(
      'submit',
      () => this.apiService.submitResourceVersion(context.resourceType, context.resourceId, context.versionId),
      'Failed to submit resource version.',
      'Failed to submit resource version',
      `Successfully Submitted ${resourceName}`
    );
  }

  publish(context: ResourceVersionContext, resourceName: string): Observable<WriteResult | null> {
    return this.run(
      'publish',
      () => this.apiService.publishResourceVersion(context.resourceType, context.resourceId, context.versionId),
      'Failed to publish resource version.',
      'Failed to publish resource version',
      `Successfully Published ${resourceName}`
    );
  }

  reject(context: ResourceVersionContext, note: string, resourceName: string): Observable<WriteResult | null> {
    const payload: RejectResourceRequest = { note };
    return this.run(
      'reject',
      () => this.apiService.rejectResourceVersion(context.resourceType, context.resourceId, context.versionId, payload),
      'Failed to reject resource version.',
      'Failed to reject resource version',
      `Successfully Rejected ${resourceName}`
    );
  }

  reopen(context: ResourceVersionContext, resourceName: string): Observable<WriteResult | null> {
    return this.run(
      'revert-to-draft',
      () => this.apiService.reopenResourceVersion(context.resourceType, context.resourceId, context.versionId),
      'Failed to revert resource to draft.',
      'Failed to reopen resource version',
      `Successfully Reverted ${resourceName} to Draft`
    );
  }

  save(context: SaveResourceContext): Observable<ResourceVersion | null> {
    return this.run(
      'save',
      () => this.apiService
        .saveResourceVersionDraft(context.resourceId, context.versionId, context.targetStatus, context.payload)
        .pipe(switchMap(() => this.apiService.getResourceVersion(context.resourceType, context.resourceId, context.versionId) as Observable<ResourceVersion>)),
      'Failed to save resource version.',
      'Failed to save resource version'
    );
  }

  delete(context: ResourceVersionContext): Observable<boolean | null> {
    return this.run(
      'delete',
      () => this.apiService
        .deleteResourceVersionDraft(context.resourceType, context.resourceId, context.versionId)
        .pipe(map(() => true)),
      'Failed to delete resource draft.',
      'Failed to delete resource draft'
    );
  }

  resolveCreatedResourceRoute(
    result: WriteResult,
    groupId: string,
    fallbackSectionPath: string
  ): Observable<readonly [string, string, string, string] | null> {
    return this.apiService
      .getDiscovery({ groupId, type: [result.resourceType], versioningStatus: ['DRAFT'] })
      .pipe(
        map((items) => items.find((item) => item.resource_id === result.id && item.type === result.resourceType)),
        map((item) => item ? ['/', this.sectionPathForGroup(groupId, fallbackSectionPath), item.resource_id, item.version_id] as const : null)
      );
  }

  private action(
    key: ResourceActionKey,
    label: string,
    loadingLabel: string | undefined,
    variant: ResourceActionItem['variant'],
    pending: ResourceActionKey | null
  ): ResourceActionItem {
    return { key, label, loadingLabel, variant, disabled: pending !== null, loading: pending === key };
  }

  private run<T>(
    action: ResourceActionKey,
    request: () => Observable<T>,
    errorMessage: string,
    logMessage: string,
    successMessage?: string
  ): Observable<T | null> {
    return defer(() => {
      if (this.pendingActionState()) {
        return of(null);
      }

      this.pendingActionState.set(action);
      this.errorState.set('');
      return request().pipe(
        tap(() => {
          if (successMessage) {
            this.toastService.success(successMessage);
          }
        }),
        catchError((error) => {
          console.error(logMessage, error);
          this.errorState.set(errorMessage);
          return of(null);
        }),
        finalize(() => this.pendingActionState.set(null))
      );
    });
  }

  private sectionPathForGroup(groupId: string, fallbackSectionPath: string): string {
    switch (this.groupsStore.groups().find((group) => group.id === groupId)?.concept) {
      case 'DATA':
        return 'data';
      case 'SOFTWARE':
        return 'software';
      case 'FACILITIES':
        return 'facilities';
      default:
        return fallbackSectionPath;
    }
  }
}
