import { Component, DestroyRef, DOCUMENT, ElementRef, Input, ViewChild, effect, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, catchError, combineLatest, debounceTime, distinctUntilChanged, finalize, map, of, switchMap } from 'rxjs';
import type { Subscription } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

import type {
  DiscoveryConcept,
  DiscoveryItem,
  DiscoveryResourceType,
  ResourceVersion,
  VersioningStatus,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata,
  Group
} from '../../../generated/backofficeSchemas';
import { ApiService, type ResourceVersionResponse } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { GroupsStore } from '../../services/groups.store';
import { LocalStorageService } from '../../services/local-storage.service';
import { ResourceActionsService } from '../../services/resource-actions.service';
import { ResourceCreationDialogService } from '../../services/resource-creation-dialog.service';
import type { ResourceActionItem, ResourceActionKey, ResourceVersionContext, SaveResourceContext } from '../../models/resource-actions';
import { resourceCapabilities, type ResourceCapabilities } from '../../models/resource-capabilities';
import type {
  DatasetCategoryForm,
  DatasetDetailForm,
  DatasetDistributionForm,
  DatasetProviderForm,
  DatasetSpatialForm,
  DatasetTemporalForm,
  DatasetWebServiceContactPointForm
} from '../../models/dataset-detail-form';
import { normalizeResultsFilters, type ResultsFilters } from '../../models/results-filters';
import {
  hasRequiredParameterWithoutDefault,
  temporalWarningKind,
  temporalWarningMessage,
  toDatasetDetailForm,
  toDatasetSaveRequest
} from '../../mappers/dataset-detail-form.mapper';
import {
  isSoftwareDescriptionValid,
  toSoftwareApplicationSaveRequest,
  toSoftwareSourceCodeSaveRequest
} from '../../mappers/software-detail-form.mapper';
import { BackofficeHeaderComponent } from '../backoffice-header/backoffice-header.component';
import { CategoriesComponent } from './detail-sections/categories/categories.component';
import { ContactPointsComponent } from './detail-sections/contact-points/contact-points.component';
import { DistributionComponent } from './detail-sections/distribution/distribution.component';
import { GeneralInformationComponent } from './detail-sections/general-information/general-information.component';
import { IdentifiersComponent } from './detail-sections/identifiers/identifiers.component';
import { NavigationSidebarComponent } from '../navigation-sidebar/navigation-sidebar.component';
import { ProvidersComponent } from './detail-sections/providers/providers.component';
import { SoftwareGeneralInformationComponent } from './detail-sections/software-general-information/software-general-information.component';
import { ResultsTableComponent } from '../results-table/results-table.component';
import { SpatialCoverageComponent } from './detail-sections/spatial-coverage/spatial-coverage.component';
import { TemporalCoverageComponent } from './detail-sections/temporal-coverage/temporal-coverage.component';
import { AdminGroupsComponent } from '../admin-groups/admin-groups.component';
import { AdminOrganizationsComponent } from '../admin-organizations/admin-organizations.component';
import { AdminCategoriesComponent } from '../admin-categories/admin-categories.component';

type RouteData = {
  sectionTitle?: string;
  discoveryType?: DiscoveryResourceType[];
  adminPage?: 'groups' | 'categories' | 'organizations';
};

type RouteState = {
  resourceId?: string | null;
  versionId?: string | null;
  groupId?: string | null;
  dataType?: string | null;
};

type SoftwareResourceMetadata = SoftwareApplicationResourceMetadata | SoftwareSourceCodeResourceMetadata;

const RESULTS_FILTERS_STORAGE_KEY = 'epos-backoffice-results-filters';

type DetailNavigationItem = {
  label: string;
  anchor: string;
  children?: DetailNavigationItem[];
};

@Component({
  selector: 'app-main-container',
  imports: [
    AdminCategoriesComponent,
    AdminGroupsComponent,
    AdminOrganizationsComponent,
    BackofficeHeaderComponent,
    CategoriesComponent,
    ContactPointsComponent,
    DistributionComponent,
    GeneralInformationComponent,
    IdentifiersComponent,
    NavigationSidebarComponent,
    ProvidersComponent,
    ResultsTableComponent,
    RouterLink,
    SoftwareGeneralInformationComponent,
    SpatialCoverageComponent,
    TemporalCoverageComponent
  ],
  templateUrl: './main-container.component.html',
  styleUrl: './main-container.component.css'
})
export class MainContainerComponent {
  @Input() isAdmin = true;
  @ViewChild('detailScrollContainer') private detailScrollContainerRef?: ElementRef<HTMLElement>;

  private readonly apiService = inject(ApiService);
  private readonly dialogService = inject(DialogService);
  private readonly groupsStore = inject(GroupsStore);
  private readonly localStorage = inject(LocalStorageService);
  private readonly resourceActions = inject(ResourceActionsService);
  private readonly resourceCreationDialog = inject(ResourceCreationDialogService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly document = inject(DOCUMENT);
  private readonly router = inject(Router);
  private readonly initialResultsFilters = normalizeResultsFilters(this.localStorage.getJson(RESULTS_FILTERS_STORAGE_KEY));
  private discoveryRequest?: Subscription;
  private readonly searchQueryChanges = new Subject<string>();
  protected readonly sectionTitle = signal('Data');
  protected readonly pageLabel = signal('Results');
  protected readonly adminPage = signal<RouteData['adminPage'] | null>(null);
  protected readonly sectionPath = signal('data');
  protected readonly darkMode = signal(false);
  protected readonly discoveryItems = signal<DiscoveryItem[] | null>(null);
  protected readonly discoveryLoading = signal(false);
  protected readonly groups = this.groupsStore.groups;
  protected readonly groupsLoading = this.groupsStore.loading;
  protected readonly selectedGroupId = signal('');
  protected readonly selectedGroupName = signal('');
  protected readonly searchQuery = signal(this.initialResultsFilters.searchQuery);
  protected readonly selectedDataTypes = signal(this.initialResultsFilters.dataTypes);
  protected readonly selectedVersioningStatuses = signal(this.initialResultsFilters.versioningStatuses);
  private readonly activeDataType = signal<DiscoveryResourceType | null>(null);
  protected readonly resourceVersion = signal<ResourceVersionResponse<DiscoveryResourceType> | null>(null);
  protected readonly resourceVersionLoading = signal(false);
  protected readonly detailSaving = this.resourceActions.saving;
  protected readonly detailPublishing = this.resourceActions.publishing;
  protected readonly detailCreatingDraft = this.resourceActions.creatingDraft;
  protected readonly detailArchiving = this.resourceActions.archiving;
  protected readonly detailRejecting = this.resourceActions.rejecting;
  protected readonly detailReopening = this.resourceActions.reopening;
  protected readonly detailDeleting = this.resourceActions.deleting;
  protected readonly detailActionError = this.resourceActions.error;
  protected readonly datasetVersion = signal<ResourceVersionResponse<'dataset'> | null>(null);
  protected datasetForm: DatasetDetailForm | null = null;
  protected softwareApplicationForm: SoftwareApplicationResourceMetadata | null = null;
  protected softwareSourceCodeForm: SoftwareSourceCodeResourceMetadata | null = null;
  protected readonly detailNavigationItems = signal<DetailNavigationItem[]>([]);
  protected get detailRootLabel(): string {
    return this.activeDataType() ? this.resourceTypeLabel(this.activeDataType()!) : this.sectionTitle();
  }
  protected readonly datasetDetailSections: DetailNavigationItem[] = [
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
  protected readonly softwareDetailSections: DetailNavigationItem[] = [
    { label: 'General Information', anchor: 'general-information' },
    { label: 'Persistent Identifiers', anchor: 'persistent-identifiers' },
    { label: 'Contact Points', anchor: 'contact-points' },
    { label: 'Categories', anchor: 'categories' }
  ];

  constructor(private readonly activatedRoute: ActivatedRoute) {
    this.searchQueryChanges
      .pipe(
        map((query) => query.trim()),
        debounceTime(300),
        distinctUntilChanged(),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe(() => {
        if (this.pageLabel() === 'Results') {
          this.loadRouteData(this.selectedGroupId());
        }
      });

    combineLatest([this.activatedRoute.data, this.activatedRoute.paramMap, this.activatedRoute.queryParamMap])
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(([data, params, queryParams]) => {
        this.syncRouteState(data as RouteData, {
          resourceId: params.get('resourceId'),
          versionId: params.get('versionId'),
          groupId: queryParams.get('groupId'),
          dataType: queryParams.get('dataType')
        });
      });

    effect(() => {
      this.document.documentElement.setAttribute('data-theme', this.darkMode() ? 'dark' : 'light');
    });
  }

  private syncRouteState(routeData: RouteData, routeState: RouteState): void {
    const adminPage = routeData.adminPage ?? null;
    const discoveryType = routeData.discoveryType;
    const detailDataType = this.toDiscoveryResourceType(routeState.dataType) ?? discoveryType?.[0];
    const isDetailRoute = Boolean(routeState.resourceId && routeState.versionId);
    this.adminPage.set(adminPage);
    this.activeDataType.set(isDetailRoute ? (detailDataType ?? null) : null);

    this.resourceActions.clearError();
    this.dialogService.dismiss();

    this.sectionPath.set(adminPage ? `admin/${adminPage}` : this.currentSectionPath());
    this.sectionTitle.set(isDetailRoute && detailDataType ? this.resourceTypeLabel(detailDataType) : (routeData.sectionTitle ?? 'Data'));

    if (adminPage) {
      this.discoveryRequest?.unsubscribe();
      this.selectedGroupId.set('');
      this.selectedGroupName.set('');
      this.pageLabel.set(adminPage === 'groups' ? 'Users' : adminPage === 'categories' ? 'Categories' : 'Organizations');
      this.detailNavigationItems.set([]);
      this.discoveryItems.set(null);
      this.discoveryLoading.set(false);
      this.resourceVersion.set(null);
      this.datasetVersion.set(null);
      this.datasetForm = null;
      this.softwareApplicationForm = null;
      this.softwareSourceCodeForm = null;
      this.resourceVersionLoading.set(false);
      return;
    }

    const selectedGroup = this.groupForCurrentSection(routeState.groupId);
    this.selectedGroupId.set(selectedGroup?.id ?? '');
    this.selectedGroupName.set(selectedGroup?.name ?? '');
    this.pageLabel.set(isDetailRoute ? 'Detail' : 'Results');
    this.detailNavigationItems.set(isDetailRoute ? (detailDataType === 'dataset' ? this.datasetDetailSections : this.softwareDetailSections) : []);

    if (isDetailRoute && detailDataType && routeState.resourceId && routeState.versionId) {
      this.discoveryLoading.set(false);
      this.discoveryItems.set(null);
      if (detailDataType === 'dataset') {
        this.loadDatasetVersion(routeState.resourceId, routeState.versionId);
      } else {
        this.loadResourceVersion(detailDataType, routeState.resourceId, routeState.versionId);
      }
      return;
    }

    this.resourceVersion.set(null);
    this.datasetVersion.set(null);
    this.datasetForm = null;
    this.softwareApplicationForm = null;
    this.softwareSourceCodeForm = null;
    this.resourceVersionLoading.set(false);
    this.loadRouteData(this.selectedGroupId());
  }

  private loadRouteData(groupId: string): void {
    this.discoveryRequest?.unsubscribe();

    if (!groupId) {
      this.discoveryItems.set(null);
      this.discoveryLoading.set(false);
      return;
    }

    this.discoveryLoading.set(true);

    this.discoveryRequest = this.apiService
      .getDiscovery({
        groupId,
        q: this.searchQuery().trim() || undefined,
        type: this.selectedDataTypes().length ? this.selectedDataTypes() : undefined,
        versioningStatus: this.selectedVersioningStatuses().length ? this.selectedVersioningStatuses() : undefined
      })
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.discoveryLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load discovery results', error);
          this.discoveryItems.set([]);
          return of([] as DiscoveryItem[]);
        })
      )
      .subscribe((items) => {
        this.discoveryItems.set(items);
      });
  }

  protected updateSearchQuery(query: string): void {
    this.searchQuery.set(query);
    this.persistResultsFilters();
    this.searchQueryChanges.next(query);
  }

  protected updateResultsFilters(filters: ResultsFilters): void {
    this.searchQuery.set(filters.searchQuery);
    this.selectedDataTypes.set(filters.dataTypes);
    this.selectedVersioningStatuses.set(filters.versioningStatuses);
    this.persistResultsFilters();
    this.reloadResultsForFilterChange();
  }

  private persistResultsFilters(): void {
    this.localStorage.setJson(RESULTS_FILTERS_STORAGE_KEY, {
      searchQuery: this.searchQuery(),
      dataTypes: this.selectedDataTypes(),
      versioningStatuses: this.selectedVersioningStatuses()
    } satisfies ResultsFilters);
  }

  private reloadResultsForFilterChange(): void {
    if (this.pageLabel() === 'Results') {
      this.loadRouteData(this.selectedGroupId());
    }
  }

  private groupForCurrentSection(groupId: string | null | undefined): Group | undefined {
    if (!groupId) {
      return undefined;
    }

    const conceptBySection: Record<string, DiscoveryConcept> = {
      data: 'DATA',
      software: 'SOFTWARE',
      facilities: 'FACILITIES'
    };
    const concept = conceptBySection[this.sectionPath()];
    return this.groups().find((group) => group.id === groupId && group.concept === concept);
  }

  private toDiscoveryResourceType(value: string | null | undefined): DiscoveryResourceType | undefined {
    return value === 'dataset' || value === 'software-application' || value === 'software-source-code'
      ? value
      : undefined;
  }

  private resourceTypeLabel(resourceType: DiscoveryResourceType): string {
    switch (resourceType) {
      case 'dataset':
        return 'Dataset';
      case 'software-application':
        return 'Software Application';
      case 'software-source-code':
        return 'Software Source Code';
    }
  }

  private loadDatasetVersion(resourceId: string, versionId: string): void {
    this.resourceVersionLoading.set(true);

    this.apiService
      .getResourceVersion('dataset', resourceId, versionId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.resourceVersionLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load dataset resource version', error);
          this.datasetVersion.set(null);
          return of(null);
        })
      )
      .subscribe((resourceVersion) => {
        if (!resourceVersion) {
          this.datasetVersion.set(null);
          this.datasetForm = null;
          this.softwareApplicationForm = null;
          this.softwareSourceCodeForm = null;
          return;
        }

        this.datasetVersion.set(resourceVersion);
        this.datasetForm = toDatasetDetailForm(resourceVersion.metadata);
        this.softwareApplicationForm = null;
        this.softwareSourceCodeForm = null;
      });
  }

  private loadResourceVersion(resourceType: DiscoveryResourceType, resourceId: string, versionId: string): void {
    this.resourceVersionLoading.set(true);

    this.apiService
      .getResourceVersion(resourceType, resourceId, versionId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.resourceVersionLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load resource version', error);
          this.resourceVersion.set(null);
          return of(null);
        })
      )
      .subscribe((resourceVersion) => {
        this.resourceVersion.set(resourceVersion);

        if (!resourceVersion) {
          this.softwareApplicationForm = null;
          this.softwareSourceCodeForm = null;
          return;
        }

        this.datasetVersion.set(null);
        this.datasetForm = null;

        if (resourceType === 'software-application') {
          this.softwareApplicationForm = structuredClone(resourceVersion.metadata as SoftwareApplicationResourceMetadata);
          this.softwareSourceCodeForm = null;
        } else if (resourceType === 'software-source-code') {
          this.softwareSourceCodeForm = structuredClone(resourceVersion.metadata as SoftwareSourceCodeResourceMetadata);
          this.softwareApplicationForm = null;
        }
      });
  }

  protected detailName(): string {
    const metadata = this.resourceVersion()?.metadata as { title?: string; name?: string } | undefined;
    return metadata?.title ?? metadata?.name ?? 'Resource detail';
  }

  protected detailTitleValue(): string {
    if (this.datasetForm) {
      return this.datasetForm.title || this.detailName();
    }

    if (this.softwareApplicationForm) {
      return this.softwareApplicationForm.name || this.detailName();
    }

    if (this.softwareSourceCodeForm) {
      return this.softwareSourceCodeForm.name || this.detailName();
    }

    return this.detailName();
  }

  protected updateSoftwareApplicationForm(metadata: SoftwareResourceMetadata): void {
    if (metadata.resource_type === 'software-application') {
      this.softwareApplicationForm = metadata;
    }
  }

  protected updateSoftwareSourceCodeForm(metadata: SoftwareResourceMetadata): void {
    if (metadata.resource_type === 'software-source-code') {
      this.softwareSourceCodeForm = metadata;
    }
  }

  protected detailDescription(): string {
    const metadata = this.resourceVersion()?.metadata as { description?: string } | undefined;
    return metadata?.description ?? 'No description available.';
  }

  protected detailKeywords(): string {
    const keywords = this.resourceVersion()?.metadata.keywords;
    return keywords?.length ? keywords.join(', ') : 'None';
  }

  protected detailDistributionCount(): number {
    const metadata = this.resourceVersion()?.metadata as { distributions?: unknown[] } | undefined;
    return metadata?.distributions?.length ?? 0;
  }

  protected canEditDetail(): boolean {
    return this.detailCapabilities().edit;
  }

  protected detailStatus(): VersioningStatus | null {
    return this.datasetVersion()?.status ?? this.resourceVersion()?.status ?? null;
  }

  protected detailResourceType(): DiscoveryResourceType | null {
    return this.activeDataType();
  }

  protected canDeleteDetail(): boolean {
    return this.detailCapabilities().delete && Boolean(this.detailVersionContext());
  }

  protected detailSidebarActions(): ResourceActionItem[] {
    if (this.pageLabel() !== 'Detail') {
      return [];
    }

    return this.resourceActions.sidebarActions(this.detailStatus(), this.isAdmin);
  }

  protected canSubmitDetail(): boolean {
    return this.detailCapabilities().submit;
  }

  protected canPublishDetail(): boolean {
    return this.detailCapabilities().publish;
  }

  protected canRejectDetail(): boolean {
    return this.detailCapabilities().reject;
  }

  protected canRevertToDraftDetail(): boolean {
    return this.detailCapabilities().reopen;
  }

  private detailCapabilities(): ResourceCapabilities {
    return resourceCapabilities(this.detailStatus(), this.isAdmin);
  }

  private detailVersionContext(): ResourceVersionContext | null {
    const resourceType = this.detailResourceType();

    if (!resourceType) {
      return null;
    }

    const version = resourceType === 'dataset' ? this.datasetVersion() : this.resourceVersion();

    if (!version) {
      return null;
    }

    return {
      resourceType,
      resourceId: version.resource_id,
      versionId: version.version_id,
      groupId: version.group_id
    };
  }

  private reloadCurrentDetail(resourceType: DiscoveryResourceType, resourceId: string, versionId: string): void {
    if (resourceType === 'dataset') {
      this.loadDatasetVersion(resourceId, versionId);
      return;
    }

    this.loadResourceVersion(resourceType, resourceId, versionId);
  }

  protected handleSidebarDetailAction(actionKey: ResourceActionKey): void {
    if (actionKey === 'save') {
      this.saveDetail();
      return;
    }

    if (actionKey === 'submit') {
      this.submitDetail();
      return;
    }

    if (actionKey === 'publish') {
      this.publishDetail();
      return;
    }

    if (actionKey === 'create-new-draft') {
      this.createNewDraftFromCurrentVersion();
      return;
    }

    if (actionKey === 'archive') {
      this.archivePublishedDetail();
      return;
    }

    if (actionKey === 'reject') {
      this.openRejectDetailDialog();
      return;
    }

    if (actionKey === 'revert-to-draft') {
      this.revertToDraftDetail();
    }
  }

  protected createNewDraftFromCurrentVersion(): void {
    const status = this.detailStatus();

    if (this.detailCreatingDraft() || this.detailArchiving() || !this.detailCapabilities().createDraft || (status !== 'PUBLISHED' && status !== 'ARCHIVED')) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.resourceActions
      .createDraft(context, status, this.sectionPath())
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((route) => {
        if (!route) {
          if (!this.detailActionError()) {
            this.resourceActions.setError('Draft created, but its detail page could not be opened.');
          }
          return;
        }

        void this.router.navigate(route, {
          queryParams: { groupId: context.groupId, dataType: context.resourceType }
        });
      });
  }

  protected archivePublishedDetail(): void {
    if (this.detailArchiving() || this.detailCreatingDraft() || !this.detailCapabilities().archive) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.resourceActions
      .archive(context, this.detailTitleValue())
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  protected submitDetail(): void {
    if (this.detailSaving() || this.detailDeleting() || !this.canSubmitDetail()) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.resourceActions
      .submit(context, this.detailTitleValue())
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  protected async openRejectDetailDialog(): Promise<void> {
    if (this.detailRejecting() || !this.canRejectDetail()) {
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
      this.confirmRejectDetail(note);
    }
  }

  protected publishDetail(): void {
    if (this.detailPublishing() || this.detailRejecting() || !this.canPublishDetail()) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.resourceActions
      .publish(context, this.detailTitleValue())
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  private confirmRejectDetail(note: string): void {
    if (this.detailRejecting() || !this.canRejectDetail()) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.resourceActions
      .reject(context, note, this.detailTitleValue())
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  protected revertToDraftDetail(): void {
    if (this.detailReopening() || !this.canRevertToDraftDetail()) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.resourceActions
      .reopen(context, this.detailTitleValue())
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  protected saveDetail(): void {
    if (this.detailSaving() || !this.canEditDetail()) {
      return;
    }

    this.resourceActions.clearError();
    const saveContext = this.buildSaveContext();

    if (!saveContext) {
      return;
    }

    this.resourceActions
      .save(saveContext)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((savedVersion) => {
        if (!savedVersion) {
          return;
        }

        this.applySavedVersion(saveContext.resourceType, savedVersion as ResourceVersion);
      });
  }

  protected deleteDetail(): void {
    this.openDeleteDetailDialog();
  }

  protected async openDeleteDetailDialog(): Promise<void> {
    if (this.detailDeleting() || !this.canDeleteDetail()) {
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
      this.confirmDeleteDetail();
    }
  }

  private confirmDeleteDetail(): void {
    if (this.detailDeleting() || !this.canDeleteDetail()) {
      return;
    }

    const deleteContext = this.detailVersionContext();

    if (!deleteContext) {
      return;
    }

    this.resourceActions
      .delete(deleteContext)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((result) => {
        if (!result && this.detailActionError()) {
          return;
        }

        void this.router.navigate(['/', this.sectionPath()], {
          replaceUrl: true,
          queryParams: this.selectedGroupId() ? { groupId: this.selectedGroupId() } : undefined
        });
      });
  }

  protected openCreateResourceDialog(): void {
    this.resourceCreationDialog.open(this.selectedGroupId(), this.sectionPath());
  }

  protected scrollDetailToTop(): void {
    this.detailScrollContainerRef?.nativeElement.scrollTo({ top: 0, behavior: 'smooth' });
  }

  private buildSaveContext(): SaveResourceContext | null {
    const status = this.detailStatus()?.toLowerCase();
    const targetStatus: 'draft' | 'submitted' = status === 'submitted' ? 'submitted' : 'draft';

    if (this.datasetVersion() && this.datasetForm) {
      if (hasRequiredParameterWithoutDefault(this.datasetForm)) {
        this.resourceActions.setError('Every required web service parameter must have a default value.');
        return null;
      }

      return {
        resourceType: 'dataset',
        resourceId: this.datasetVersion()!.resource_id,
        versionId: this.datasetVersion()!.version_id,
        targetStatus,
        payload: toDatasetSaveRequest(this.datasetForm, this.datasetVersion()!.revision, this.datasetVersion()!.metadata)
      };
    }

    if (this.resourceVersion() && this.softwareApplicationForm) {
      if (!isSoftwareDescriptionValid(this.softwareApplicationForm)) {
        this.resourceActions.setError('Software description is required.');
        return null;
      }

      return {
        resourceType: 'software-application',
        resourceId: this.resourceVersion()!.resource_id,
        versionId: this.resourceVersion()!.version_id,
        targetStatus,
        payload: toSoftwareApplicationSaveRequest(this.softwareApplicationForm, this.resourceVersion()!.revision)
      };
    }

    if (this.resourceVersion() && this.softwareSourceCodeForm) {
      if (!isSoftwareDescriptionValid(this.softwareSourceCodeForm)) {
        this.resourceActions.setError('Software description is required.');
        return null;
      }

      return {
        resourceType: 'software-source-code',
        resourceId: this.resourceVersion()!.resource_id,
        versionId: this.resourceVersion()!.version_id,
        targetStatus,
        payload: toSoftwareSourceCodeSaveRequest(this.softwareSourceCodeForm, this.resourceVersion()!.revision)
      };
    }

    return null;
  }

  private currentSectionPath(): string {
    return this.activatedRoute.snapshot.routeConfig?.path?.split('/')[0] ?? 'data';
  }

  protected updateDatasetSpatial(spatial: DatasetSpatialForm | null): void {
    if (!this.datasetForm) {
      return;
    }

    this.datasetForm.spatial = spatial;
  }

  protected updateDatasetTemporal(temporal: DatasetTemporalForm): void {
    if (!this.datasetForm) {
      return;
    }

    this.datasetForm.temporal = temporal;
  }

  protected updateDatasetContactPoints(contactPoints: DatasetWebServiceContactPointForm[]): void {
    if (!this.datasetForm) {
      return;
    }

    this.datasetForm.contactPoints = contactPoints;
  }

  protected updateDatasetProviders(providers: DatasetProviderForm[]): void {
    if (!this.datasetForm) {
      return;
    }

    this.datasetForm.providers = providers;
  }

  protected updateDatasetCategories(categories: DatasetCategoryForm[]): void {
    if (!this.datasetForm) {
      return;
    }

    this.datasetForm.categories = categories;
  }

  protected updateDatasetDistributions(distributions: DatasetDistributionForm[]): void {
    if (!this.datasetForm) {
      return;
    }

    this.datasetForm.distributions = distributions;
  }

  protected temporalWarningKind(): 'none' | 'warning' | 'error' {
    return this.datasetForm ? temporalWarningKind(this.datasetForm) : 'none';
  }

  protected temporalWarningMessage(): string {
    return this.datasetForm ? temporalWarningMessage(this.datasetForm) : '';
  }

  private applySavedVersion(resourceType: DiscoveryResourceType, savedVersion: ResourceVersion): void {
    if (resourceType === 'dataset') {
      const typedVersion = savedVersion as ResourceVersionResponse<'dataset'>;
      this.datasetVersion.set(typedVersion);
      this.datasetForm = toDatasetDetailForm(typedVersion.metadata);
      return;
    }

    this.resourceVersion.set(savedVersion as ResourceVersionResponse<'software-application' | 'software-source-code'>);

    if (resourceType === 'software-application') {
      this.softwareApplicationForm = structuredClone(savedVersion.metadata as SoftwareApplicationResourceMetadata);
      return;
    }

    this.softwareSourceCodeForm = structuredClone(savedVersion.metadata as SoftwareSourceCodeResourceMetadata);
  }

  protected toggleTheme(): void {
    this.darkMode.update((value) => !value);
  }
}
