import { DestroyRef, Injectable, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Router } from '@angular/router';
import { EMPTY, Subject, Subscription, catchError, map, of, switchMap } from 'rxjs';

import type {
  DiscoveryResourceType,
  ResourceVersion,
  ResourceVersionSummary,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata,
  VersioningStatus
} from '../../generated/backofficeSchemas';
import {
  distributionValidationError,
  geologicalTemporalValidationError,
  hasBodyOnUnsupportedMethod,
  hasIncompleteWebServiceBody,
  hasInvalidExplodeParameter,
  hasParameterDefaultOutsideAllowedValues,
  hasMultipleParameterWithoutAllowedValues,
  temporalWarningKind,
  temporalWarningMessage,
  toDatasetDetailForm,
  toDatasetSaveRequest
} from '../mappers/dataset-detail-form.mapper';
import {
  isSoftwareDescriptionValid,
  toSoftwareApplicationSaveRequest,
  toSoftwareSourceCodeSaveRequest
} from '../mappers/software-detail-form.mapper';
import type {
  DatasetCategoryForm,
  DatasetDetailForm,
  DatasetDistributionForm,
  DatasetGeologicalTemporalForm,
  DatasetProviderForm,
  DatasetSpatialForm,
  DatasetTemporalForm,
  DatasetWebServiceContactPointForm
} from '../models/dataset-detail-form';
import type {
  ResourceActionItem,
  ResourceActionKey,
  ResourceVersionContext,
  SaveResourceContext
} from '../models/resource-actions';
import { resourceCapabilities } from '../models/resource-capabilities';
import { orderVersionsByCreatedAt } from '../models/resource-version-comparison';
import { ApiService, type ResourceVersionResponse } from '../services/api.service';
import { DialogService } from '../services/dialog.service';
import { ResourceActionsService } from '../services/resource-actions.service';
import { CurrentUserStore } from './current-user.store';

export type ResourceDetailRouteContext = {
  resourceType: DiscoveryResourceType;
  resourceId: string;
  versionId: string;
  groupId: string;
  sectionPath: string;
  returnUrl: string;
};

export type ResourceDetailNavigationItem = {
  label: string;
  anchor: string;
  children?: ResourceDetailNavigationItem[];
};

type SoftwareResourceType = 'software-application' | 'software-source-code';
type SoftwareResourceVersion = ResourceVersionResponse<SoftwareResourceType>;

type LoadedDatasetState = {
  kind: 'loaded-dataset';
  context: ResourceDetailRouteContext;
  version: ResourceVersionResponse<'dataset'>;
  form: DatasetDetailForm;
};

type LoadedSoftwareState = {
  kind: 'loaded-software';
  context: ResourceDetailRouteContext;
  version: SoftwareResourceVersion;
  applicationForm: SoftwareApplicationResourceMetadata | null;
  sourceCodeForm: SoftwareSourceCodeResourceMetadata | null;
};

type ResourceDetailState =
  | { kind: 'idle' }
  | { kind: 'loading'; context: ResourceDetailRouteContext }
  | { kind: 'error'; context: ResourceDetailRouteContext; message: string }
  | LoadedDatasetState
  | LoadedSoftwareState;

type LoadCommand =
  | { kind: 'load'; context: ResourceDetailRouteContext }
  | { kind: 'clear' };

const DATASET_NAVIGATION_ITEMS: ResourceDetailNavigationItem[] = [
  { label: 'General Information', anchor: 'general-information' },
  { label: 'Spatial Coverage', anchor: 'spatial-coverage' },
  { label: 'Temporal Coverage', anchor: 'temporal-coverage' },
  { label: 'Persistent Identifiers', anchor: 'persistent-identifiers' },
  { label: 'Contact Points', anchor: 'contact-points' },
  { label: 'Providers', anchor: 'providers' },
  { label: 'Categories', anchor: 'categories' },
  {
    label: 'Distributions',
    anchor: 'distributions',
    children: [
      {
        label: 'Web Service',
        anchor: 'web-service',
        children: [
          { label: 'Documentation', anchor: 'webservice-documentation' },
          { label: 'Provider', anchor: 'webservice-provider' },
          { label: 'Contact Points', anchor: 'webservice-contact-points' },
          { label: 'Spatial Coverage', anchor: 'webservice-spatial-coverage' },
          { label: 'Temporal Coverage', anchor: 'webservice-temporal-coverage' },
          { label: 'Parameters', anchor: 'webservice-parameters' }
        ]
      }
    ]
  }
];

const SOFTWARE_NAVIGATION_ITEMS: ResourceDetailNavigationItem[] = [
  { label: 'General Information', anchor: 'general-information' },
  { label: 'Persistent Identifiers', anchor: 'persistent-identifiers' },
  { label: 'Contact Points', anchor: 'contact-points' },
  { label: 'Categories', anchor: 'categories' }
];

export function resourceDetailNavigationItems(
  resourceType: DiscoveryResourceType
): ResourceDetailNavigationItem[] {
  return resourceType === 'dataset' ? DATASET_NAVIGATION_ITEMS : SOFTWARE_NAVIGATION_ITEMS;
}

export function resourceDetailRootLabel(resourceType: DiscoveryResourceType): string {
  switch (resourceType) {
    case 'dataset':
      return 'Dataset';
    case 'software-application':
      return 'Software Application';
    case 'software-source-code':
      return 'Software Source Code';
  }
}

@Injectable()
export class ResourceDetailStore {
  private readonly apiService = inject(ApiService);
  private readonly dialogService = inject(DialogService);
  private readonly resourceActions = inject(ResourceActionsService);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  private readonly state = signal<ResourceDetailState>({ kind: 'idle' });
  private readonly loadCommands = new Subject<LoadCommand>();
  private readonly historyOpenState = signal(false);
  private readonly historyLoadingState = signal(false);
  private readonly historyItemsState = signal<ResourceVersionSummary[]>([]);
  private readonly historyErrorState = signal('');
  private readonly selectedVersionIdsState = signal<readonly string[]>([]);
  private historySubscription: Subscription | null = null;

  readonly loading = computed(() => this.state().kind === 'loading');
  readonly context = computed(() => {
    const state = this.state();
    return state.kind === 'idle' ? null : state.context;
  });
  readonly loadError = computed(() => {
    const state = this.state();
    return state.kind === 'error' ? state.message : '';
  });
  readonly datasetVersion = computed(() => {
    const state = this.state();
    return state.kind === 'loaded-dataset' ? state.version : null;
  });
  readonly resourceVersion = computed(() => {
    const state = this.state();
    return state.kind === 'loaded-software' ? state.version : null;
  });
  readonly datasetForm = computed(() => {
    const state = this.state();
    return state.kind === 'loaded-dataset' ? state.form : null;
  });
  readonly softwareApplicationForm = computed(() => {
    const state = this.state();
    return state.kind === 'loaded-software' ? state.applicationForm : null;
  });
  readonly softwareSourceCodeForm = computed(() => {
    const state = this.state();
    return state.kind === 'loaded-software' ? state.sourceCodeForm : null;
  });
  readonly status = computed<VersioningStatus | null>(() =>
    this.datasetVersion()?.status ?? this.resourceVersion()?.status ?? null
  );
  readonly name = computed(() => {
    const metadata = (this.datasetVersion() ?? this.resourceVersion())?.metadata as
      | { title?: string; name?: string }
      | undefined;
    return metadata?.title ?? metadata?.name ?? 'Resource detail';
  });
  readonly title = computed(() =>
    this.datasetForm()?.title ||
    this.softwareApplicationForm()?.name ||
    this.softwareSourceCodeForm()?.name ||
    this.name()
  );
  readonly description = computed(() => {
    const metadata = (this.datasetVersion() ?? this.resourceVersion())?.metadata as
      | { description?: string }
      | undefined;
    return metadata?.description ?? 'No description available.';
  });
  readonly keywords = computed(() => {
    const keywords = (this.datasetVersion() ?? this.resourceVersion())?.metadata.keywords;
    return keywords?.length ? keywords.join(', ') : 'None';
  });
  readonly distributionCount = computed(() => {
    const metadata = (this.datasetVersion() ?? this.resourceVersion())?.metadata as
      | { distributions?: unknown[] }
      | undefined;
    return metadata?.distributions?.length ?? 0;
  });
  readonly navigationItems = computed(() => {
    const resourceType = this.context()?.resourceType;
    return resourceType ? resourceDetailNavigationItems(resourceType) : [];
  });
  readonly rootLabel = computed(() => {
    const resourceType = this.context()?.resourceType;
    return resourceType ? resourceDetailRootLabel(resourceType) : 'Data';
  });
  readonly groupPermissions = computed(() => {
    const version = this.datasetVersion() ?? this.resourceVersion();
    const groupId = version?.group_id ?? this.context()?.groupId ?? '';
    return this.currentUserStore.groupPermissions(groupId);
  });
  readonly capabilities = computed(() => resourceCapabilities(this.status(), this.groupPermissions()));
  readonly canEdit = computed(() => this.capabilities().edit);
  readonly canDelete = computed(() => this.capabilities().delete && Boolean(this.versionContext()));
  readonly canViewHistory = computed(() => this.capabilities().viewHistory);
  readonly historyOpen = this.historyOpenState.asReadonly();
  readonly historyLoading = this.historyLoadingState.asReadonly();
  readonly historyItems = this.historyItemsState.asReadonly();
  readonly historyError = this.historyErrorState.asReadonly();
  readonly selectedVersionIds = this.selectedVersionIdsState.asReadonly();
  readonly canCompareVersions = computed(() => this.selectedVersionIds().length === 2);
  readonly sidebarActions = computed<ResourceActionItem[]>(() => {
    const context = this.context();
    return context ? this.resourceActions.sidebarActions(this.status(), this.groupPermissions()) : [];
  });
  readonly temporalWarningKind = computed(() => {
    const form = this.datasetForm();
    return form ? temporalWarningKind(form) : 'none';
  });
  readonly temporalWarningMessage = computed(() => {
    const form = this.datasetForm();
    return form ? temporalWarningMessage(form) : '';
  });

  readonly actionError = this.resourceActions.error;
  readonly saving = this.resourceActions.saving;
  readonly publishing = this.resourceActions.publishing;
  readonly creatingDraft = this.resourceActions.creatingDraft;
  readonly archiving = this.resourceActions.archiving;
  readonly rejecting = this.resourceActions.rejecting;
  readonly reopening = this.resourceActions.reopening;
  readonly deleting = this.resourceActions.deleting;
  constructor() {
    this.loadCommands
      .pipe(
        switchMap((command) => {
          if (command.kind === 'clear') {
            this.state.set({ kind: 'idle' });
            return EMPTY;
          }

          const context = command.context;
          this.state.set({ kind: 'loading', context });
          return this.apiService.getResourceVersion(context.resourceType, context.resourceId, context.versionId).pipe(
            map((version) => this.loadedState(context, version)),
            catchError((error) => {
              console.error(`Failed to load ${context.resourceType} resource version`, error);
              return of({
                kind: 'error',
                context,
                message: 'Failed to load resource version.'
              } satisfies ResourceDetailState);
            })
          );
        }),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe((state) => this.state.set(state));
  }

  load(context: ResourceDetailRouteContext): void {
    this.closeVersionHistory();
    this.resourceActions.clearError();
    this.dialogService.dismiss();
    this.loadCommands.next({ kind: 'load', context });
  }

  clear(): void {
    this.closeVersionHistory();
    this.resourceActions.clearError();
    this.dialogService.dismiss();
    this.loadCommands.next({ kind: 'clear' });
  }

  openVersionHistory(): void {
    const context = this.context();
    const version = this.datasetVersion() ?? this.resourceVersion();
    if (!this.canViewHistory() || !context || !version) {
      return;
    }

    this.historySubscription?.unsubscribe();
    this.historyOpenState.set(true);
    this.historyLoadingState.set(true);
    this.historyItemsState.set([]);
    this.historyErrorState.set('');
    this.selectedVersionIdsState.set([]);

    this.historySubscription = this.apiService
      .listResourceVersions(context.resourceType, version.resource_id)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (items) => {
          this.historyItemsState.set(
            [...items].sort((left, right) => Date.parse(right.updated_at) - Date.parse(left.updated_at))
          );
          this.historyLoadingState.set(false);
        },
        error: (error) => {
          console.error('Failed to load resource version history', error);
          this.historyItemsState.set([]);
          this.historyErrorState.set('Failed to load version history.');
          this.historyLoadingState.set(false);
        }
      });
  }

  closeVersionHistory(): void {
    this.historySubscription?.unsubscribe();
    this.historySubscription = null;
    this.historyOpenState.set(false);
    this.historyLoadingState.set(false);
    this.historyItemsState.set([]);
    this.historyErrorState.set('');
    this.selectedVersionIdsState.set([]);
  }

  toggleVersionSelection(versionId: string): void {
    if (!this.canViewHistory()) {
      return;
    }

    const selectedIds = this.selectedVersionIds();
    if (selectedIds.includes(versionId)) {
      this.selectedVersionIdsState.set(selectedIds.filter((id) => id !== versionId));
    } else if (selectedIds.length < 2) {
      this.selectedVersionIdsState.set([...selectedIds, versionId]);
    }
  }

  isVersionSelected(versionId: string): boolean {
    return this.selectedVersionIds().includes(versionId);
  }

  isVersionSelectionDisabled(versionId: string): boolean {
    return !this.isVersionSelected(versionId) && this.selectedVersionIds().length >= 2;
  }

  compareSelectedVersions(): void {
    const [firstVersionId, secondVersionId] = this.selectedVersionIds();
    const context = this.context();
    const version = this.datasetVersion() ?? this.resourceVersion();
    if (!this.canViewHistory() || !firstVersionId || !secondVersionId || !context || !version) {
      return;
    }

    const firstVersion = this.historyItems().find((item) => item.version_id === firstVersionId);
    const secondVersion = this.historyItems().find((item) => item.version_id === secondVersionId);
    const [leftVersionId, rightVersionId] = firstVersion && secondVersion
      ? orderVersionsByCreatedAt(firstVersion, secondVersion).map((item) => item.version_id)
      : [firstVersionId, secondVersionId];

    void this.router.navigate(['/', context.sectionPath, version.resource_id, 'compare'], {
      queryParams: {
        leftVersionId,
        rightVersionId,
        sourceVersionId: version.version_id,
        groupId: version.group_id,
        dataType: context.resourceType,
        returnUrl: context.returnUrl
      }
    });
    this.closeVersionHistory();
  }

  setDatasetForm(form: DatasetDetailForm): void {
    const state = this.state();
    if (state.kind === 'loaded-dataset') {
      this.state.set({ ...state, form });
    }
  }

  updateSoftwareApplicationForm(metadata: SoftwareApplicationResourceMetadata | SoftwareSourceCodeResourceMetadata): void {
    const state = this.state();
    if (state.kind === 'loaded-software' && metadata.resource_type === 'software-application') {
      this.state.set({ ...state, applicationForm: metadata, sourceCodeForm: null });
    }
  }

  updateSoftwareSourceCodeForm(metadata: SoftwareApplicationResourceMetadata | SoftwareSourceCodeResourceMetadata): void {
    const state = this.state();
    if (state.kind === 'loaded-software' && metadata.resource_type === 'software-source-code') {
      this.state.set({ ...state, applicationForm: null, sourceCodeForm: metadata });
    }
  }

  updateDatasetSpatial(spatial: DatasetSpatialForm | null): void {
    this.updateDatasetForm((form) => ({ ...form, spatial }));
  }

  updateDatasetTemporal(temporal: DatasetTemporalForm): void {
    this.updateDatasetForm((form) => ({ ...form, temporal }));
  }

  updateDatasetTemporalMode(temporalMode: 'calendar' | 'geological'): void {
    this.updateDatasetForm((form) => ({ ...form, temporalMode }));
  }

  updateDatasetGeologicalTemporal(geologicalTemporal: DatasetGeologicalTemporalForm): void {
    this.updateDatasetForm((form) => ({ ...form, geologicalTemporal }));
  }

  updateDatasetContactPoints(contactPoints: DatasetWebServiceContactPointForm[]): void {
    this.updateDatasetForm((form) => ({ ...form, contactPoints }));
  }

  updateDatasetProviders(providers: DatasetProviderForm[]): void {
    this.updateDatasetForm((form) => ({ ...form, providers }));
  }

  updateDatasetCategories(categories: DatasetCategoryForm[]): void {
    this.updateDatasetForm((form) => ({ ...form, categories }));
  }

  updateDatasetDistributions(distributions: DatasetDistributionForm[]): void {
    this.updateDatasetForm((form) => ({ ...form, distributions }));
  }

  buildSaveContext(): SaveResourceContext | null {
    const targetStatus = this.status()?.toLowerCase() === 'submitted' ? 'submitted' : 'draft';
    const datasetVersion = this.datasetVersion();
    const datasetForm = this.datasetForm();

    if (datasetVersion && datasetForm) {
      if (!datasetForm.description.trim()) {
        this.resourceActions.setError('Dataset description is required.');
        return null;
      }

      const geologicalTemporalError = geologicalTemporalValidationError(datasetForm);
      if (geologicalTemporalError) {
        this.resourceActions.setError(geologicalTemporalError);
        return null;
      }

      const distributionError = distributionValidationError(datasetForm);
      if (distributionError) {
        this.resourceActions.setError(distributionError);
        return null;
      }

      if (hasInvalidExplodeParameter(datasetForm)) {
        this.resourceActions.setError('Explode requires a QUERY parameter with Multiple values enabled.');
        return null;
      }

      if (hasMultipleParameterWithoutAllowedValues(datasetForm)) {
        this.resourceActions.setError('Every multiple-value web service parameter must define at least one allowed value.');
        return null;
      }

      if (hasParameterDefaultOutsideAllowedValues(datasetForm)) {
        this.resourceActions.setError('A web service parameter default value must be included in its allowed values.');
        return null;
      }

      if (hasIncompleteWebServiceBody(datasetForm)) {
        this.resourceActions.setError('Request body content type and template are both required.');
        return null;
      }

      if (hasBodyOnUnsupportedMethod(datasetForm)) {
        this.resourceActions.setError('Request bodies are supported only for POST and PUT web services.');
        return null;
      }

      return {
        resourceType: 'dataset',
        resourceId: datasetVersion.resource_id,
        versionId: datasetVersion.version_id,
        targetStatus,
        payload: toDatasetSaveRequest(datasetForm, datasetVersion.revision, datasetVersion.metadata)
      };
    }

    const resourceVersion = this.resourceVersion();
    const applicationForm = this.softwareApplicationForm();
    if (resourceVersion && applicationForm) {
      if (!isSoftwareDescriptionValid(applicationForm)) {
        this.resourceActions.setError('Software description is required.');
        return null;
      }

      return {
        resourceType: 'software-application',
        resourceId: resourceVersion.resource_id,
        versionId: resourceVersion.version_id,
        targetStatus,
        payload: toSoftwareApplicationSaveRequest(applicationForm, resourceVersion.revision)
      };
    }

    const sourceCodeForm = this.softwareSourceCodeForm();
    if (resourceVersion && sourceCodeForm) {
      if (!isSoftwareDescriptionValid(sourceCodeForm)) {
        this.resourceActions.setError('Software description is required.');
        return null;
      }

      return {
        resourceType: 'software-source-code',
        resourceId: resourceVersion.resource_id,
        versionId: resourceVersion.version_id,
        targetStatus,
        payload: toSoftwareSourceCodeSaveRequest(sourceCodeForm, resourceVersion.revision)
      };
    }

    return null;
  }

  handleSidebarAction(actionKey: ResourceActionKey): void {
    switch (actionKey) {
      case 'save':
        this.save();
        break;
      case 'submit':
        this.submit();
        break;
      case 'publish':
        this.publish();
        break;
      case 'create-new-draft':
        this.createDraft();
        break;
      case 'archive':
        this.archive();
        break;
      case 'reject':
        void this.openRejectDialog();
        break;
      case 'revert-to-draft':
        this.reopen();
        break;
      case 'delete':
        void this.openDeleteDialog();
        break;
    }
  }

  save(): void {
    if (this.saving() || !this.canEdit()) {
      return;
    }

    this.resourceActions.clearError();
    const saveContext = this.buildSaveContext();
    if (!saveContext) {
      return;
    }

    this.resourceActions.save(saveContext).pipe(takeUntilDestroyed(this.destroyRef)).subscribe((savedVersion) => {
      if (savedVersion) {
        this.applySavedVersion(saveContext.resourceType, savedVersion);
      }
    });
  }

  submit(): void {
    if (this.saving() || this.deleting() || !this.capabilities().submit) {
      return;
    }
    this.runAndReload((context) => this.resourceActions.submit(context, this.title()));
  }

  publish(): void {
    if (this.publishing() || this.rejecting() || !this.capabilities().publish) {
      return;
    }
    this.runAndReload((context) => this.resourceActions.publish(context, this.title()));
  }

  archive(): void {
    if (this.archiving() || this.creatingDraft() || !this.capabilities().archive) {
      return;
    }
    this.runAndReload((context) => this.resourceActions.archive(context, this.title()));
  }

  reopen(): void {
    if (this.reopening() || !this.capabilities().reopen) {
      return;
    }
    this.runAndReload((context) => this.resourceActions.reopen(context, this.title()));
  }

  async openRejectDialog(): Promise<void> {
    if (this.rejecting() || !this.capabilities().reject) {
      return;
    }

    this.resourceActions.clearError();
    const note = await this.dialogService.prompt({
      eyebrow: 'Reject Resource',
      title: 'Reject this submission?',
      message: 'Add a reviewer note before rejecting.',
      inputLabel: 'Note',
      placeholder: 'Enter rejection note',
      confirmLabel: 'Reject',
      destructive: true,
      required: true
    });
    if (note) {
      this.reject(note);
    }
  }

  reject(note: string): void {
    if (this.rejecting() || !this.capabilities().reject) {
      return;
    }
    this.runAndReload((context) => this.resourceActions.reject(context, note, this.title()));
  }

  createDraft(): void {
    const status = this.status();
    if (
      this.creatingDraft() ||
      this.archiving() ||
      !this.capabilities().createDraft ||
      (status !== 'PUBLISHED' && status !== 'ARCHIVED')
    ) {
      return;
    }

    const versionContext = this.versionContext();
    const routeContext = this.context();
    if (!versionContext || !routeContext) {
      return;
    }

    this.resourceActions
      .createDraft(versionContext, status, routeContext.sectionPath)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((route) => {
        if (!route) {
          if (!this.actionError()) {
            this.resourceActions.setError('Draft created, but its detail page could not be opened.');
          }
          return;
        }
        void this.router.navigate(route, {
          queryParams: { groupId: versionContext.groupId, dataType: routeContext.resourceType }
        });
      });
  }

  async openDeleteDialog(): Promise<void> {
    if (this.deleting() || !this.canDelete()) {
      return;
    }

    this.resourceActions.clearError();
    const confirmed = await this.dialogService.confirm({
      eyebrow: 'Confirm Delete',
      title: 'Delete this draft?',
      message: 'You are about to permanently delete the current draft version. This action cannot be undone.',
      confirmLabel: 'Delete draft',
      destructive: true
    });
    if (confirmed) {
      this.delete();
    }
  }

  delete(): void {
    if (this.deleting() || !this.canDelete()) {
      return;
    }

    const versionContext = this.versionContext();
    const routeContext = this.context();
    if (!versionContext || !routeContext) {
      return;
    }

    this.resourceActions.delete(versionContext).pipe(takeUntilDestroyed(this.destroyRef)).subscribe((result) => {
      if (!result && this.actionError()) {
        return;
      }
      void this.router.navigate(['/', routeContext.sectionPath], {
        replaceUrl: true,
        queryParams: routeContext.groupId ? { groupId: routeContext.groupId } : undefined
      });
    });
  }

  private loadedState(
    context: ResourceDetailRouteContext,
    version: ResourceVersionResponse<DiscoveryResourceType>
  ): LoadedDatasetState | LoadedSoftwareState {
    if (context.resourceType === 'dataset') {
      const datasetVersion = version as ResourceVersionResponse<'dataset'>;
      return {
        kind: 'loaded-dataset',
        context,
        version: datasetVersion,
        form: toDatasetDetailForm(datasetVersion.metadata)
      };
    }

    const softwareVersion = version as SoftwareResourceVersion;
    if (context.resourceType === 'software-application') {
      return {
        kind: 'loaded-software',
        context,
        version: softwareVersion,
        applicationForm: structuredClone(version.metadata as SoftwareApplicationResourceMetadata),
        sourceCodeForm: null
      };
    }

    return {
      kind: 'loaded-software',
      context,
      version: softwareVersion,
      applicationForm: null,
      sourceCodeForm: structuredClone(version.metadata as SoftwareSourceCodeResourceMetadata)
    };
  }

  private updateDatasetForm(update: (form: DatasetDetailForm) => DatasetDetailForm): void {
    const state = this.state();
    if (state.kind === 'loaded-dataset') {
      this.state.set({ ...state, form: update(state.form) });
    }
  }

  private versionContext(): ResourceVersionContext | null {
    const routeContext = this.context();
    const version = this.datasetVersion() ?? this.resourceVersion();
    if (!routeContext || !version) {
      return null;
    }

    return {
      resourceType: routeContext.resourceType,
      resourceId: version.resource_id,
      versionId: version.version_id,
      groupId: version.group_id
    };
  }

  private runAndReload(
    action: (context: ResourceVersionContext) => ReturnType<ResourceActionsService['submit']>
  ): void {
    const versionContext = this.versionContext();
    const routeContext = this.context();
    if (!versionContext || !routeContext) {
      return;
    }

    action(versionContext).pipe(takeUntilDestroyed(this.destroyRef)).subscribe((result) => {
      if (result) {
        this.loadCommands.next({ kind: 'load', context: routeContext });
      }
    });
  }

  private applySavedVersion(resourceType: DiscoveryResourceType, savedVersion: ResourceVersion): void {
    const context = this.context();
    if (!context) {
      return;
    }

    this.state.set(this.loadedState(
      { ...context, resourceType },
      savedVersion as ResourceVersionResponse<DiscoveryResourceType>
    ));
  }
}
