import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, catchError, defer, finalize, map, of, switchMap, tap } from 'rxjs';

import type {
  GroupPermission,
  Problem,
  RejectResourceRequest,
  ResourceVersion,
  VersioningStatus,
  WriteResult
} from '../../generated/backofficeSchemas';
import type { ResourceActionItem, ResourceActionKey, ResourceVersionContext, SaveResourceContext } from '../models/resource-actions';
import { resourceCapabilities } from '../models/resource-capabilities';
import { ApiService } from './api.service';
import { GroupsStore } from '../stores/groups.store';
import { ToastService, type ToastIssue } from './toast.service';

type RunOptions = {
  includeProblemDetail?: boolean;
  persistentError?: boolean;
};

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

  setValidationError(message: string, label: string): void {
    this.errorState.set(message);
    this.toastService.error('Resource could not be saved.', {
      duration: null,
      issues: [{ label, message }]
    });
  }

  sidebarActions(status: VersioningStatus | null, permissions: readonly GroupPermission[]): ResourceActionItem[] {
    const pending = this.pendingActionState();
    const capabilities = resourceCapabilities(status, permissions);

    if (status === 'PUBLISHED') {
      const actions: ResourceActionItem[] = [];
      if (capabilities.createDraft) {
        actions.push(this.action('create-new-draft', 'Create New Draft', 'Creating...', 'primary', pending));
      }
      if (capabilities.archive) {
        actions.push(this.action('archive', 'Archive', 'Archiving...', 'danger', pending));
      }
      return actions;
    }

    if (capabilities.createDraft) {
      return [this.action('create-new-draft', 'Create New Draft', 'Creating...', 'primary', pending)];
    }

    if (capabilities.reopen) {
      return [this.action('revert-to-draft', 'Revert to Draft', 'Reverting...', 'primary', pending)];
    }

    if (status === 'SUBMITTED') {
      const actions: ResourceActionItem[] = [];
      if (capabilities.publish) {
        actions.push(this.action('publish', 'Publish', 'Publishing...', 'primary', pending));
      }
      if (capabilities.reject) {
        actions.push(this.action('reject', 'Reject', undefined, 'danger', pending));
      }
      return actions;
    }

    if (status === 'DRAFT') {
      const actions: ResourceActionItem[] = [];
      if (capabilities.save) {
        actions.push(this.action('save', 'Save', 'Saving...', 'primary', pending));
      }
      if (capabilities.submit) {
        actions.push(this.action('submit', 'Submit', undefined, 'ghost', pending));
      }
      return actions;
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
    this.toastService.dismiss();
    return this.run(
      'save',
      () => this.apiService
        .saveResourceVersionDraft(context.resourceId, context.versionId, context.targetStatus, context.payload)
        .pipe(switchMap(() => this.apiService.getResourceVersion(context.resourceType, context.resourceId, context.versionId) as Observable<ResourceVersion>)),
      'Failed to save resource version.',
      'Failed to save resource version',
      'Resource changes saved successfully.',
      { includeProblemDetail: true, persistentError: true }
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
    successMessage?: string,
    options: RunOptions = {}
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
          if (options.persistentError) {
            const issue = options.includeProblemDetail ? this.problemIssue(error) : null;
            this.toastService.error(errorMessage, {
              duration: null,
              ...(issue ? { issues: [issue] } : {})
            });
          } else {
            this.toastService.error(errorMessage);
          }
          return of(null);
        }),
        finalize(() => this.pendingActionState.set(null))
      );
    });
  }

  private problemIssue(error: unknown): ToastIssue | null {
    const problem = this.problemFrom(error);
    if (!problem?.detail?.trim()) {
      return null;
    }

    return {
      message: problem.detail.trim(),
      ...(problem.instance ? { reference: problem.instance } : {}),
      ...(problem.target ? {
        label: this.problemTargetLabel(problem.target),
        target: problem.target
      } : { label: 'Validation issue' })
    };
  }

  private problemFrom(error: unknown): Problem | null {
    const body: unknown = error instanceof HttpErrorResponse ? error.error : null;
    if (!body || typeof body !== 'object') {
      return null;
    }

    const candidate = body as Record<string, unknown>;
    if (
      typeof candidate['type'] !== 'string'
      || typeof candidate['title'] !== 'string'
      || typeof candidate['status'] !== 'number'
    ) {
      return null;
    }

    return {
      type: candidate['type'],
      title: candidate['title'],
      status: candidate['status'],
      ...(typeof candidate['detail'] === 'string' ? { detail: candidate['detail'] } : {}),
      ...(typeof candidate['instance'] === 'string' ? { instance: candidate['instance'] } : {}),
      ...(typeof candidate['target'] === 'string' ? { target: candidate['target'] } : {})
    };
  }

  private problemTargetLabel(target: string): string {
    const field = target.split('.').at(-1)?.replaceAll('_', ' ').trim();
    return field ? `${field.charAt(0).toUpperCase()}${field.slice(1)}` : 'Validation issue';
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
