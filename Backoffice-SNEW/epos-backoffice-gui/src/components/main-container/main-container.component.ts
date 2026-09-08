import { Component, DestroyRef, DOCUMENT, ElementRef, Input, ViewChild, effect, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, catchError, combineLatest, debounceTime, distinctUntilChanged, finalize, map, of, switchMap } from 'rxjs';
import type { Observable, Subscription } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

import type {
  CreateResourceDraftRequest,
  Distribution,
  DatasetDraftMetadata,
  DatasetResourceMetadata,
  DiscoveryConcept,
  DiscoveryItem,
  VersioningStatus,
  DiscoveryResourceType,
  Documentation,
  DownloadableFile,
  DownloadableFileDistributionDraft,
  DistributionDraft,
  Parameter,
  ParameterMetadata,
  ParameterDraft,
  ParameterDraftMetadata,
  ReplaceDatasetDraftRequest,
  ReplaceSoftwareApplicationDraftRequest,
  ReplaceSoftwareSourceCodeDraftRequest,
  RejectResourceRequest,
  ResourceVersionReference,
  ResourceVersion,
  DownloadableFileDistribution,
  SoftwareApplicationDraftMetadata,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeDraftMetadata,
  SoftwareSourceCodeResourceMetadata,
  TemporalCoverage,
  Group,
  WebServiceDistributionDraft,
  WebServiceDistribution,
  WriteResult
} from '../../../generated/backofficeSchemas';
import { ApiService, type ResourceVersionResponse } from '../../services/api.service';
import { GroupsStore } from '../../services/groups.store';
import {
  BackofficeHeaderComponent,
  DISCOVERY_VERSIONING_STATUSES,
  type ResultsFilters
} from '../backoffice-header/backoffice-header.component';
import { CategoriesComponent } from './detail-sections/categories/categories.component';
import { ContactPointsComponent } from './detail-sections/contact-points/contact-points.component';
import { DistributionComponent } from './detail-sections/distribution/distribution.component';
import { GeneralInformationComponent } from './detail-sections/general-information/general-information.component';
import { IdentifiersComponent } from './detail-sections/identifiers/identifiers.component';
import { NavigationSidebarComponent, type SidebarActionItem } from '../navigation-sidebar/navigation-sidebar.component';
import { ProvidersComponent } from './detail-sections/providers/providers.component';
import { SoftwareGeneralInformationComponent } from './detail-sections/software-general-information/software-general-information.component';
import { ResultsTableComponent } from '../results-table/results-table.component';
import { SpatialCoverageComponent } from './detail-sections/spatial-coverage/spatial-coverage.component';
import { TemporalCoverageComponent } from './detail-sections/temporal-coverage/temporal-coverage.component';
import { AdminGroupsComponent } from '../admin-groups/admin-groups.component';

type RouteData = {
  sectionTitle?: string;
  discoveryType?: DiscoveryResourceType[];
  adminPage?: 'groups';
};

type RouteState = {
  resourceId?: string | null;
  versionId?: string | null;
  groupId?: string | null;
  dataType?: string | null;
};

type SoftwareResourceMetadata = SoftwareApplicationResourceMetadata | SoftwareSourceCodeResourceMetadata;

const RESULTS_FILTERS_STORAGE_KEY = 'epos-backoffice-results-filters';
const DISCOVERY_RESOURCE_TYPES: DiscoveryResourceType[] = ['dataset', 'software-application', 'software-source-code'];

function loadPersistedResultsFilters(): ResultsFilters {
  try {
    const storedValue: unknown = JSON.parse(localStorage.getItem(RESULTS_FILTERS_STORAGE_KEY) ?? 'null');

    if (!storedValue || typeof storedValue !== 'object') {
      return { searchQuery: '', dataTypes: [], versioningStatuses: ['PUBLISHED'] };
    }

    const storedFilters = storedValue as Partial<Record<keyof ResultsFilters, unknown>>;
    const dataTypes = Array.isArray(storedFilters.dataTypes)
      ? storedFilters.dataTypes.filter((value): value is DiscoveryResourceType => DISCOVERY_RESOURCE_TYPES.includes(value as DiscoveryResourceType))
      : [];
    const versioningStatuses = Array.isArray(storedFilters.versioningStatuses)
      ? storedFilters.versioningStatuses.filter((value): value is VersioningStatus => DISCOVERY_VERSIONING_STATUSES.includes(value as VersioningStatus))
      : [];

    return {
      searchQuery: typeof storedFilters.searchQuery === 'string' ? storedFilters.searchQuery : '',
      dataTypes,
      versioningStatuses: versioningStatuses.length ? versioningStatuses : ['PUBLISHED']
    };
  } catch {
    return { searchQuery: '', dataTypes: [], versioningStatuses: ['PUBLISHED'] };
  }
}

type DetailNavigationItem = {
  label: string;
  anchor: string;
  children?: DetailNavigationItem[];
};

type DatasetGeneralFieldKey = keyof Pick<
  DatasetResourceMetadata,
  'title' | 'description' | 'keywords' | 'accrual_periodicity' | 'quality_annotation' | 'created' | 'updated'
>;

type DatasetGeneralField = {
  label: string;
  key: DatasetGeneralFieldKey;
  inputType: 'text' | 'datetime-local';
};

export type DatasetSpatialForm = string | null;

export type DatasetTemporalForm = TemporalCoverage;

export type DatasetPersistentIdentifierForm = {
  id?: string;
  type: string;
  value: string;
};

export type DatasetProviderForm = ResourceVersionReference;

export type DatasetCategoryForm = ResourceVersionReference;

export type DatasetWebServiceDocumentationForm = Documentation;

export type DatasetWebServiceContactPointForm = string;

export type DatasetWebServiceProviderForm = ResourceVersionReference;

export type DatasetWebServiceParameterMetadataForm = ParameterMetadata;

export type DatasetWebServiceParameterForm =
  | {
      kind: 'STRING';
      metadata: DatasetWebServiceParameterMetadataForm;
      defaultValue: string | null;
      allowedValues: string[];
      pattern: string;
    }
  | {
      kind: 'BOOLEAN';
      metadata: DatasetWebServiceParameterMetadataForm;
      defaultValue: boolean | null;
    }
  | {
      kind: 'INTEGER';
      metadata: DatasetWebServiceParameterMetadataForm;
      minimum: string;
      maximum: string;
      defaultValue: string;
      allowedValues: string[];
    }
  | {
      kind: 'FLOAT';
      metadata: DatasetWebServiceParameterMetadataForm;
      minimum: string;
      maximum: string;
      defaultValue: string;
      allowedValues: string[];
    }
  | {
      kind: 'DATETIME';
      metadata: DatasetWebServiceParameterMetadataForm;
      minimum: string;
      maximum: string;
      defaultValue: string;
      pattern: string;
    };

export type DatasetWebServiceForm = {
  name: string;
  description: string;
  templateUrl: string;
  format: string;
  documentation: DatasetWebServiceDocumentationForm;
  provider: DatasetWebServiceProviderForm;
  contactPoints: DatasetWebServiceContactPointForm[];
  spatialCoverage: DatasetSpatialForm;
  temporalCoverage: DatasetTemporalForm;
  parameters: DatasetWebServiceParameterForm[];
};

export type DatasetDistributionCommonForm = {
  title: string;
  description: string;
  issued: string;
  modified: string;
  license: string;
  type: string;
};

export type DatasetDownloadableFileForm = DownloadableFile;

export type DatasetDistributionForm = DatasetDistributionCommonForm &
  (
    | {
        kind: 'web-service';
        webService: DatasetWebServiceForm;
      }
    | {
        kind: 'downloadable-file';
        downloadableFile: DatasetDownloadableFileForm;
      }
  );

export type DatasetDetailForm = {
  title: string;
  description: string;
  keywords: string;
  accrualPeriodicity: string;
  qualityAnnotation: string;
  created: string;
  updated: string;
  spatial: DatasetSpatialForm;
  temporal: DatasetTemporalForm;
  persistentIdentifiers: DatasetPersistentIdentifierForm[];
  contactPoints: DatasetWebServiceContactPointForm[];
  providers: DatasetProviderForm[];
  categories: DatasetCategoryForm[];
  distributions: DatasetDistributionForm[];
};

@Component({
  selector: 'app-main-container',
  imports: [
    AdminGroupsComponent,
    BackofficeHeaderComponent,
    CategoriesComponent,
    ContactPointsComponent,
    DistributionComponent,
    FormsModule,
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
  private readonly groupsStore = inject(GroupsStore);
  private readonly destroyRef = inject(DestroyRef);
  private readonly document = inject(DOCUMENT);
  private readonly router = inject(Router);
  private readonly initialResultsFilters = loadPersistedResultsFilters();
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
  protected readonly detailSaving = signal(false);
  protected readonly detailPublishing = signal(false);
  protected readonly detailCreatingDraft = signal(false);
  protected readonly detailArchiving = signal(false);
  protected readonly detailRejecting = signal(false);
  protected readonly detailReopening = signal(false);
  protected readonly detailDeleting = signal(false);
  protected readonly detailActionError = signal('');
  protected readonly detailDeleteDialogOpen = signal(false);
  protected readonly detailRejectDialogOpen = signal(false);
  protected readonly detailRejectNote = signal('');
  protected readonly detailToastMessage = signal('');
  protected readonly detailToastVisible = signal(false);
  protected readonly createResourceDialogOpen = signal(false);
  protected readonly createResourceTitle = signal('');
  protected readonly createResourceDescription = signal('');
  protected readonly createResourceType = signal<DiscoveryResourceType>('dataset');
  protected readonly createResourceGroupId = signal('');
  protected readonly createResourceSaving = signal(false);
  protected readonly createResourceError = signal('');
  protected readonly createResourceTypes: Array<{ label: string; value: DiscoveryResourceType }> = [
    { label: 'dataset', value: 'dataset' },
    { label: 'softwareapplication', value: 'software-application' },
    { label: 'softwaresourcecode', value: 'software-source-code' }
  ];
  protected readonly datasetVersion = signal<ResourceVersionResponse<'dataset'> | null>(null);
  protected datasetForm: DatasetDetailForm | null = null;
  protected softwareApplicationForm: SoftwareApplicationResourceMetadata | null = null;
  protected softwareSourceCodeForm: SoftwareSourceCodeResourceMetadata | null = null;
  protected readonly detailNavigationItems = signal<DetailNavigationItem[]>([]);
  protected get detailRootLabel(): string {
    return this.activeDataType() ? this.resourceTypeLabel(this.activeDataType()!) : this.sectionTitle();
  }
  protected readonly datasetGeneralFields: DatasetGeneralField[] = [
    { label: 'Title', key: 'title', inputType: 'text' },
    { label: 'Description', key: 'description', inputType: 'text' },
    { label: 'Keywords', key: 'keywords', inputType: 'text' },
    { label: 'Accrual periodicity', key: 'accrual_periodicity', inputType: 'text' },
    { label: 'Quality annotation', key: 'quality_annotation', inputType: 'text' },
    { label: 'Created', key: 'created', inputType: 'datetime-local' },
    { label: 'Updated', key: 'updated', inputType: 'datetime-local' }
  ];
  protected readonly datasetProviderFields: Array<keyof DatasetProviderForm> = ['id', 'name'];
  protected readonly datasetDistributionCommonFields: Array<keyof DatasetDistributionCommonForm> = [
    'title',
    'description',
    'issued',
    'modified',
    'license'
  ];
  protected readonly distributionTabs = signal(0);
  protected readonly parameterPage = signal(0);
  protected readonly expandedParameterIndex = signal<number | null>(0);
  protected readonly parameterPageSize = 10;
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

    this.detailActionError.set('');
    this.detailDeleteDialogOpen.set(false);
    this.detailRejectDialogOpen.set(false);
    this.detailRejectNote.set('');
    this.detailToastMessage.set('');
    this.detailToastVisible.set(false);

    this.sectionPath.set(adminPage ? `admin/${adminPage}` : this.currentSectionPath());
    this.sectionTitle.set(isDetailRoute && detailDataType ? this.resourceTypeLabel(detailDataType) : (routeData.sectionTitle ?? 'Data'));

    if (adminPage) {
      this.discoveryRequest?.unsubscribe();
      this.selectedGroupId.set('');
      this.selectedGroupName.set('');
      this.pageLabel.set('Users');
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
    try {
      localStorage.setItem(RESULTS_FILTERS_STORAGE_KEY, JSON.stringify({
        searchQuery: this.searchQuery(),
        dataTypes: this.selectedDataTypes(),
        versioningStatuses: this.selectedVersioningStatuses()
      } satisfies ResultsFilters));
    } catch {
      // Search remains usable when browser storage is unavailable.
    }
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
        this.datasetForm = this.buildDatasetForm(resourceVersion.metadata);
        this.softwareApplicationForm = null;
        this.softwareSourceCodeForm = null;
        this.distributionTabs.set(0);
        this.parameterPage.set(0);
        this.expandedParameterIndex.set(0);
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

  protected datasetMetadata(): DatasetResourceMetadata | null {
    const resourceVersion = this.datasetVersion();

    if (this.sectionPath() !== 'data' || !resourceVersion) {
      return null;
    }

    return resourceVersion.metadata;
  }

  protected datasetPersistentIdentifiers(): DatasetPersistentIdentifierForm[] {
    return this.datasetForm?.persistentIdentifiers ?? [];
  }

  protected canEditDetail(): boolean {
    const status = this.detailStatus()?.toUpperCase();

    if (status === 'DRAFT') {
      return true;
    }

    if (status === 'SUBMITTED') {
      return this.isAdmin;
    }

    return false;
  }

  protected detailStatus(): string | null {
    return this.datasetVersion()?.status ?? this.resourceVersion()?.status ?? null;
  }

  protected detailResourceType(): DiscoveryResourceType | null {
    return this.activeDataType();
  }

  protected detailDeleteContext(): { resourceType: DiscoveryResourceType; resourceId: string; versionId: string } | null {
    const resourceType = this.detailResourceType();

    if (!resourceType) {
      return null;
    }

    if (resourceType === 'dataset') {
      const version = this.datasetVersion();

      if (!version) {
        return null;
      }

      return {
        resourceType,
        resourceId: version.resource_id,
        versionId: version.version_id
      };
    }

    const version = this.resourceVersion();

    if (!version) {
      return null;
    }

    return {
      resourceType,
      resourceId: version.resource_id,
      versionId: version.version_id
    };
  }

  protected canDeleteDetail(): boolean {
    return this.detailStatus()?.toUpperCase() === 'DRAFT' && Boolean(this.detailDeleteContext());
  }

  protected detailSidebarActions(): SidebarActionItem[] {
    if (this.pageLabel() !== 'Detail') {
      return [];
    }

    const status = this.detailStatus()?.toUpperCase();

    if (status === 'PUBLISHED') {
      const actionPending = this.detailCreatingDraft() || this.detailArchiving();

      return [
        {
          key: 'create-new-draft',
          label: 'Create New Draft',
          loadingLabel: 'Creating...',
          variant: 'primary',
          disabled: actionPending,
          loading: this.detailCreatingDraft()
        },
        {
          key: 'archive',
          label: 'Archive',
          loadingLabel: 'Archiving...',
          variant: 'danger',
          disabled: actionPending,
          loading: this.detailArchiving()
        }
      ];
    }

    if (status === 'ARCHIVED') {
      return [
        {
          key: 'create-new-draft',
          label: 'Create New Draft',
          loadingLabel: 'Creating...',
          variant: 'primary',
          disabled: this.detailCreatingDraft(),
          loading: this.detailCreatingDraft()
        }
      ];
    }

    if (status === 'REJECTED') {
      return [
        {
          key: 'revert-to-draft',
          label: 'Revert to Draft',
          loadingLabel: 'Reverting...',
          variant: 'primary',
          disabled: this.detailReopening() || this.detailDeleting(),
          loading: this.detailReopening()
        }
      ];
    }

    if (status === 'SUBMITTED' && !this.isAdmin) {
      return [];
    }

    if (status === 'SUBMITTED' && this.isAdmin) {
      return [
        {
          key: 'publish',
          label: 'Publish',
          loadingLabel: 'Publishing...',
          variant: 'primary',
          disabled: this.detailPublishing() || this.detailRejecting() || this.detailDeleting(),
          loading: this.detailPublishing()
        },
        {
          key: 'reject',
          label: 'Reject',
          variant: 'danger',
          disabled: this.detailPublishing() || this.detailRejecting() || this.detailDeleting(),
          loading: this.detailRejecting()
        }
      ];
    }

    if (status !== 'DRAFT') {
      return [];
    }

    return [
      {
        key: 'save',
        label: 'Save',
        loadingLabel: 'Saving...',
        variant: 'primary',
        disabled: !this.canEditDetail() || this.detailSaving() || this.detailDeleting() || this.detailPublishing() || this.detailRejecting() || this.detailReopening(),
        loading: this.detailSaving()
      },
      {
        key: 'submit',
        label: 'Submit',
        variant: 'ghost' as const,
        disabled: this.detailSaving() || this.detailDeleting() || this.detailPublishing() || this.detailRejecting() || this.detailReopening()
      }
    ];
  }

  protected canSubmitDetail(): boolean {
    return this.detailStatus()?.toUpperCase() === 'DRAFT';
  }

  protected canPublishDetail(): boolean {
    return this.detailStatus()?.toUpperCase() === 'SUBMITTED' && this.isAdmin;
  }

  protected canRevertToDraftDetail(): boolean {
    return this.detailStatus()?.toUpperCase() === 'REJECTED';
  }

  private detailToastTimeout?: ReturnType<typeof setTimeout>;

  private detailVersionContext(): { resourceType: DiscoveryResourceType; resourceId: string; versionId: string; groupId: string } | null {
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

  private showDetailToast(message: string): void {
    this.detailToastMessage.set(message);
    this.detailToastVisible.set(true);

    if (this.detailToastTimeout) {
      clearTimeout(this.detailToastTimeout);
    }

    this.detailToastTimeout = setTimeout(() => {
      this.detailToastVisible.set(false);
      this.detailToastTimeout = setTimeout(() => {
        this.detailToastMessage.set('');
        this.detailToastTimeout = undefined;
      }, 180);
    }, 3500);
  }

  protected dismissDetailToast(): void {
    if (!this.detailToastMessage()) {
      return;
    }

    this.detailToastVisible.set(false);

    if (this.detailToastTimeout) {
      clearTimeout(this.detailToastTimeout);
    }

    this.detailToastTimeout = setTimeout(() => {
      this.detailToastMessage.set('');
      this.detailToastTimeout = undefined;
    }, 180);
  }

  private reloadCurrentDetail(resourceType: DiscoveryResourceType, resourceId: string, versionId: string): void {
    if (resourceType === 'dataset') {
      this.loadDatasetVersion(resourceId, versionId);
      return;
    }

    this.loadResourceVersion(resourceType, resourceId, versionId);
  }

  protected handleSidebarDetailAction(actionKey: string): void {
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
    const status = this.detailStatus()?.toUpperCase();

    if (this.detailCreatingDraft() || this.detailArchiving() || (status !== 'PUBLISHED' && status !== 'ARCHIVED')) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.detailCreatingDraft.set(true);
    this.detailActionError.set('');

    const request = status === 'PUBLISHED'
      ? this.apiService.createDraftFromPublication(context.resourceType, context.resourceId)
      : this.apiService.createDraftFromArchive(context.resourceType, context.resourceId);

    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        switchMap((result) => this.resolveCreatedResourceRoute(result, context.groupId)),
        finalize(() => this.detailCreatingDraft.set(false)),
        catchError((error) => {
          console.error(`Failed to create draft from ${status.toLowerCase()}`, error);
          this.detailActionError.set(`Failed to create a new draft from this ${status.toLowerCase()} resource.`);
          return of(null);
        })
      )
      .subscribe((route) => {
        if (!route) {
          if (!this.detailActionError()) {
            this.detailActionError.set('Draft created, but its detail page could not be opened.');
          }
          return;
        }

        void this.router.navigate(route, {
          queryParams: { groupId: context.groupId, dataType: context.resourceType }
        });
      });
  }

  protected archivePublishedDetail(): void {
    if (this.detailArchiving() || this.detailCreatingDraft() || this.detailStatus()?.toUpperCase() !== 'PUBLISHED') {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.detailArchiving.set(true);
    this.detailActionError.set('');

    this.apiService
      .unpublishResourceVersion(context.resourceType, context.resourceId, context.versionId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.detailArchiving.set(false)),
        catchError((error) => {
          console.error('Failed to archive published resource version', error);
          this.detailActionError.set('Failed to archive this resource version.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.showDetailToast(`Successfully Archived ${this.detailName()}`);
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

    this.detailSaving.set(true);

    this.apiService
      .submitResourceVersion(context.resourceType, context.resourceId, context.versionId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.detailSaving.set(false)),
        catchError((error) => {
          console.error('Failed to submit resource version', error);
          this.detailActionError.set('Failed to submit resource version.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.showDetailToast(`Successfully Submitted ${this.detailName()}`);
        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  protected openRejectDetailDialog(): void {
    if (this.detailRejecting() || !this.canPublishDetail()) {
      return;
    }

    this.detailActionError.set('');
    this.detailRejectNote.set('');
    this.detailRejectDialogOpen.set(true);
  }

  protected closeRejectDetailDialog(): void {
    this.detailRejectDialogOpen.set(false);
    this.detailRejectNote.set('');
  }

  protected publishDetail(): void {
    if (this.detailPublishing() || this.detailRejecting() || !this.canPublishDetail()) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.detailPublishing.set(true);
    this.detailActionError.set('');

    this.apiService
      .publishResourceVersion(context.resourceType, context.resourceId, context.versionId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.detailPublishing.set(false)),
        catchError((error) => {
          console.error('Failed to publish resource version', error);
          this.detailActionError.set('Failed to publish resource version.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.showDetailToast(`Successfully Published ${this.detailName()}`);
        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  protected confirmRejectDetail(): void {
    if (this.detailRejecting() || !this.canPublishDetail()) {
      return;
    }

    const note = this.detailRejectNote().trim();

    if (!note) {
      return;
    }

    const context = this.detailVersionContext();

    if (!context) {
      return;
    }

    this.detailRejectDialogOpen.set(false);
    this.detailRejecting.set(true);
    this.detailActionError.set('');

    const payload: RejectResourceRequest = { note };

    this.apiService
      .rejectResourceVersion(context.resourceType, context.resourceId, context.versionId, payload)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.detailRejecting.set(false)),
        catchError((error) => {
          console.error('Failed to reject resource version', error);
          this.detailActionError.set('Failed to reject resource version.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.showDetailToast(`Successfully Rejected ${this.detailName()}`);
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

    this.detailReopening.set(true);
    this.detailActionError.set('');

    this.apiService
      .reopenResourceVersion(context.resourceType, context.resourceId, context.versionId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.detailReopening.set(false)),
        catchError((error) => {
          console.error('Failed to reopen resource version', error);
          this.detailActionError.set('Failed to revert resource to draft.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (!result) {
          return;
        }

        this.showDetailToast(`Successfully Reverted ${this.detailName()} to Draft`);
        this.reloadCurrentDetail(context.resourceType, context.resourceId, context.versionId);
      });
  }

  protected saveDetail(): void {
    if (this.detailSaving() || !this.canEditDetail()) {
      return;
    }

    this.detailActionError.set('');
    const saveContext = this.buildSaveContext();

    if (!saveContext) {
      return;
    }

    this.detailSaving.set(true);

    this.apiService
      .saveResourceVersionDraft(saveContext.resourceId, saveContext.versionId, saveContext.targetStatus, saveContext.payload)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.detailSaving.set(false)),
        catchError((error) => {
          console.error('Failed to save resource version', error);
          return of(null);
        })
      )
      .subscribe((savedVersion) => {
        if (!savedVersion) {
          return;
        }

        this.apiService
          .getResourceVersion(saveContext.resourceType, saveContext.resourceId, saveContext.versionId)
          .pipe(takeUntilDestroyed(this.destroyRef))
          .subscribe((resourceVersion) => this.applySavedVersion(saveContext.resourceType, resourceVersion as ResourceVersion));
      });
  }

  protected deleteDetail(): void {
    this.openDeleteDetailDialog();
  }

  protected openDeleteDetailDialog(): void {
    if (this.detailDeleting() || !this.canDeleteDetail()) {
      return;
    }

    this.detailActionError.set('');
    this.detailDeleteDialogOpen.set(true);
  }

  protected closeDeleteDetailDialog(): void {
    this.detailDeleteDialogOpen.set(false);
  }

  protected confirmDeleteDetail(): void {
    if (this.detailDeleting() || !this.canDeleteDetail()) {
      return;
    }

    const deleteContext = this.detailDeleteContext();

    if (!deleteContext) {
      return;
    }

    this.detailDeleteDialogOpen.set(false);
    this.detailDeleting.set(true);
    this.detailActionError.set('');

    this.apiService
      .deleteResourceVersionDraft(deleteContext.resourceType, deleteContext.resourceId, deleteContext.versionId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.detailDeleting.set(false)),
        catchError((error) => {
          console.error('Failed to delete resource draft', error);
          this.detailActionError.set('Failed to delete resource draft.');
          return of(null);
        })
      )
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
    this.createResourceError.set('');
    this.createResourceTitle.set('');
    this.createResourceDescription.set('');
    this.createResourceDialogOpen.set(true);

    if (!this.createResourceGroupId() && this.groups().length) {
      this.createResourceGroupId.set(this.selectedGroupId() || this.groups()[0].id);
    }
  }

  protected closeCreateResourceDialog(): void {
    this.createResourceDialogOpen.set(false);
    this.createResourceError.set('');
    this.createResourceTitle.set('');
    this.createResourceDescription.set('');
    this.createResourceType.set('dataset');
  }

  protected submitCreateResource(): void {
    const title = this.createResourceTitle().trim();
    const description = this.createResourceDescription().trim();

    if (this.createResourceSaving() || !this.createResourceGroupId() || !title || (this.createResourceType() !== 'dataset' && !description)) {
      return;
    }

    const payload = this.buildCreateResourceDraftRequest(this.createResourceType(), title, description);

    if (!payload) {
      return;
    }

    this.createResourceSaving.set(true);
    this.createResourceError.set('');

    this.apiService
      .createResourceDraft(this.createResourceGroupId(), payload)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
          switchMap((result) => this.resolveCreatedResourceRoute(result, this.createResourceGroupId())),
        finalize(() => this.createResourceSaving.set(false)),
        catchError((error) => {
          console.error('Failed to create resource draft', error);
          this.createResourceError.set('Failed to create resource draft.');
          return of(null);
        })
      )
      .subscribe((route) => {
        if (!route) {
          this.createResourceError.set('Created resource, but could not open its detail page.');
          return;
        }

        const groupId = this.createResourceGroupId();
        const dataType = this.createResourceType();
        this.closeCreateResourceDialog();
        void this.router.navigate(route, { replaceUrl: true, queryParams: { groupId, dataType } });
      });
  }

  private resolveCreatedResourceRoute(result: WriteResult, groupId: string): Observable<readonly [string, string, string, string] | null> {
    return this.apiService
      .getDiscovery({
        groupId,
        type: [result.resourceType],
        versioningStatus: ['DRAFT'] satisfies VersioningStatus[]
      })
      .pipe(
        map((items) => items.find((item) => item.resource_id === result.id && item.type === result.resourceType)),
        map((item) => {
          if (!item) {
            return null;
          }

          return ['/', this.sectionPathForGroup(groupId), item.resource_id, item.version_id] as const;
        })
      );
  }

  private sectionPathForGroup(groupId: string): string {
    switch (this.groups().find((group) => group.id === groupId)?.concept) {
      case 'DATA':
        return 'data';
      case 'SOFTWARE':
        return 'software';
      case 'FACILITIES':
        return 'facilities';
      default:
        return this.sectionPath();
    }
  }

  protected scrollDetailToTop(): void {
    this.detailScrollContainerRef?.nativeElement.scrollTo({ top: 0, behavior: 'smooth' });
  }

  private buildSaveContext():
    | {
        resourceType: DiscoveryResourceType;
        resourceId: string;
        versionId: string;
        targetStatus: 'draft' | 'submitted';
        payload: ReplaceDatasetDraftRequest | ReplaceSoftwareApplicationDraftRequest | ReplaceSoftwareSourceCodeDraftRequest;
      }
    | null {
    const status = this.detailStatus()?.toLowerCase();
    const targetStatus: 'draft' | 'submitted' = status === 'submitted' ? 'submitted' : 'draft';

    if (this.datasetVersion() && this.datasetForm) {
      if (this.hasRequiredParameterWithoutDefault()) {
        this.detailActionError.set('Every required web service parameter must have a default value.');
        return null;
      }

      return {
        resourceType: 'dataset',
        resourceId: this.datasetVersion()!.resource_id,
        versionId: this.datasetVersion()!.version_id,
        targetStatus,
        payload: this.buildDatasetSaveRequest()
      };
    }

    if (this.resourceVersion() && this.softwareApplicationForm) {
      if (!this.softwareApplicationForm.description.trim()) {
        this.detailActionError.set('Software description is required.');
        return null;
      }

      return {
        resourceType: 'software-application',
        resourceId: this.resourceVersion()!.resource_id,
        versionId: this.resourceVersion()!.version_id,
        targetStatus,
        payload: this.buildSoftwareApplicationSaveRequest()
      };
    }

    if (this.resourceVersion() && this.softwareSourceCodeForm) {
      if (!this.softwareSourceCodeForm.description.trim()) {
        this.detailActionError.set('Software description is required.');
        return null;
      }

      return {
        resourceType: 'software-source-code',
        resourceId: this.resourceVersion()!.resource_id,
        versionId: this.resourceVersion()!.version_id,
        targetStatus,
        payload: this.buildSoftwareSourceCodeSaveRequest()
      };
    }

    return null;
  }

  private hasRequiredParameterWithoutDefault(): boolean {
    return this.datasetForm?.distributions.some((distribution) =>
      distribution.kind === 'web-service'
      && distribution.webService.parameters.some((parameter) =>
        parameter.metadata.required
        && (parameter.defaultValue === null
          || (typeof parameter.defaultValue === 'string' && !parameter.defaultValue.trim()))
      )
    ) ?? false;
  }

  private buildCreateResourceDraftRequest(resourceType: DiscoveryResourceType, title: string, description: string): CreateResourceDraftRequest | null {
    switch (resourceType) {
      case 'dataset':
        return this.buildCreateDatasetResourceRequest(title);
      case 'software-application':
        return this.buildCreateSoftwareApplicationResourceRequest(title, description);
      case 'software-source-code':
        return this.buildCreateSoftwareSourceCodeResourceRequest(title, description);
      default:
        return null;
    }
  }

  private buildCreateDatasetResourceRequest(title: string): CreateResourceDraftRequest {
    return {
      resource_type: 'dataset',
      metadata: {
        title
      }
    };
  }

  private buildCreateSoftwareApplicationResourceRequest(title: string, description: string): CreateResourceDraftRequest {
    return {
      resource_type: 'software-application',
      metadata: {
        name: title,
        description
      }
    };
  }

  private buildCreateSoftwareSourceCodeResourceRequest(title: string, description: string): CreateResourceDraftRequest {
    return {
      resource_type: 'software-source-code',
      metadata: {
        name: title,
        description
      }
    };
  }

  protected addPersistentIdentifier(): void {
    this.datasetForm?.persistentIdentifiers.push({ type: '', value: '' });
  }

  protected removePersistentIdentifier(index: number): void {
    if (!this.datasetForm) {
      return;
    }

    if (this.datasetForm.persistentIdentifiers.length === 1) {
      this.datasetForm.persistentIdentifiers[0] = { type: '', value: '' };
    } else {
      this.datasetForm.persistentIdentifiers.splice(index, 1);
    }
  }

  protected addProvider(): void {
    this.datasetForm?.providers.push({ id: '', name: '' });
  }

  protected removeProvider(index: number): void {
    if (!this.datasetForm) {
      return;
    }

    if (this.datasetForm.providers.length === 1) {
      this.datasetForm.providers[0] = { id: '', name: '' };
    } else {
      this.datasetForm.providers.splice(index, 1);
    }
  }

  protected selectDistributionTab(index: number): void {
    this.distributionTabs.set(index);
    this.parameterPage.set(0);
    this.expandedParameterIndex.set(0);
  }

  protected pagedParameters(): DatasetWebServiceParameterForm[] {
    const parameters = this.selectedWebServiceDistribution()?.webService.parameters ?? [];
    const start = this.parameterPage() * this.parameterPageSize;

    return parameters.slice(start, start + this.parameterPageSize);
  }

  protected parameterPageCount(): number {
    const parameters = this.selectedWebServiceDistribution()?.webService.parameters ?? [];
    return Math.max(1, Math.ceil(parameters.length / this.parameterPageSize));
  }

  protected parameterGlobalIndex(localIndex: number): number {
    return this.parameterPage() * this.parameterPageSize + localIndex;
  }

  protected isParameterExpanded(globalIndex: number): boolean {
    return this.expandedParameterIndex() === globalIndex;
  }

  protected toggleParameterRow(globalIndex: number): void {
    this.expandedParameterIndex.update((current) => (current === globalIndex ? null : globalIndex));
  }

  protected previousParameterPage(): void {
    this.parameterPage.update((page) => Math.max(0, page - 1));
    this.expandedParameterIndex.set(null);
  }

  protected nextParameterPage(): void {
    this.parameterPage.update((page) => Math.min(this.parameterPageCount() - 1, page + 1));
    this.expandedParameterIndex.set(null);
  }

  protected goToParameterPage(page: number): void {
    this.parameterPage.set(page);
    this.expandedParameterIndex.set(null);
  }

  protected hasDownloadableFile(distribution: DatasetDistributionForm): distribution is DatasetDistributionForm & { kind: 'downloadable-file' } {
    return distribution.kind === 'downloadable-file';
  }

  protected hasWebService(distribution: DatasetDistributionForm): distribution is DatasetDistributionForm & { kind: 'web-service' } {
    return distribution.kind === 'web-service';
  }

  protected selectedDistribution(): DatasetDistributionForm | null {
    const distributions = this.datasetForm?.distributions ?? [];
    return distributions[this.distributionTabs()] ?? distributions[0] ?? null;
  }

  protected selectedWebServiceDistribution(): Extract<DatasetDistributionForm, { kind: 'web-service' }> | null {
    const distribution = this.selectedDistribution();

    if (!distribution || distribution.kind !== 'web-service') {
      return null;
    }

    return distribution;
  }

  protected distributionFieldLabel(field: keyof DatasetDistributionCommonForm): string {
    const labels: Record<keyof DatasetDistributionCommonForm, string> = {
      title: 'Title',
      description: 'Description',
      issued: 'Issued',
      modified: 'Modified',
      license: 'License',
      type: 'Type'
    };

    return labels[field];
  }

  protected parameterPagerLabel(): string {
    return `Page ${this.parameterPage() + 1} of ${this.parameterPageCount()}`;
  }

  protected parameterPageItems(): Array<number | null> {
    const totalPages = this.parameterPageCount();

    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, index) => index);
    }

    const currentPage = this.parameterPage();
    const pages = new Set<number>([0, totalPages - 1, Math.max(0, currentPage - 1), currentPage, Math.min(totalPages - 1, currentPage + 1)]);
    const sortedPages = Array.from(pages).sort((a, b) => a - b);
    const items: Array<number | null> = [];

    sortedPages.forEach((page, index) => {
      if (index > 0 && page - sortedPages[index - 1] > 1) {
        items.push(null);
      }

      items.push(page);
    });

    return items;
  }

  private currentSectionPath(): string {
    return this.activatedRoute.snapshot.routeConfig?.path?.split('/')[0] ?? 'data';
  }

  private buildDatasetForm(metadata: DatasetResourceMetadata): DatasetDetailForm {
    const temporalCoverage = metadata.temporal_coverage;

    return {
      title: metadata.title ?? '',
      description: metadata.description ?? '',
        keywords: metadata.keywords.join(', '),
      accrualPeriodicity: metadata.accrual_periodicity ?? '',
      qualityAnnotation: metadata.quality_annotation ?? '',
      created: metadata.created ? this.toDateTimeLocalValue(metadata.created) : '',
      updated: metadata.updated ? this.toDateTimeLocalValue(metadata.updated) : '',
      spatial: metadata.spatial_coverage ?? null,
      temporal: {
        start: temporalCoverage?.start ? this.toDateTimeLocalValue(temporalCoverage.start) : '',
        end: temporalCoverage?.end ? this.toDateTimeLocalValue(temporalCoverage.end) : ''
      },
      persistentIdentifiers: metadata.persistent_identifiers?.length
        ? metadata.persistent_identifiers.map((identifier) => ({
            id: identifier.id,
            type: identifier.type,
            value: identifier.value
          }))
        : [{ type: '', value: '' }],
      contactPoints: metadata.contact_points?.length ? metadata.contact_points.map((contactPoint) => contactPoint) : [],
      providers: metadata.providers?.length
        ? metadata.providers.map((provider) => ({
            id: provider.id,
            name: provider.name
          }))
        : [{ id: '', name: '' }],
      categories: metadata.categories?.length
        ? metadata.categories.map((category) => ({
            id: category.id,
            name: category.name
          }))
        : [],
      distributions: metadata.distributions?.length
        ? metadata.distributions.map((distribution) => this.buildDistributionForm(distribution))
        : []
    };
  }

  private buildDistributionForm(distribution: Distribution): DatasetDistributionForm {
    const base: DatasetDistributionCommonForm = {
      title: distribution.title,
      description: distribution.description,
      issued: distribution.issued ? this.toDateTimeLocalValue(distribution.issued) : '',
      modified: distribution.modified ? this.toDateTimeLocalValue(distribution.modified) : '',
      license: distribution.license ?? '',
      type: distribution.type
    };

    if (distribution.type === 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE') {
      const webService = (distribution as WebServiceDistribution).web_service;

      return {
        ...base,
        kind: 'web-service',
        webService: {
          name: webService.name,
          description: webService.description,
          templateUrl: webService.template_url,
          format: webService.format,
          documentation: webService.documentation ? this.buildWebServiceDocumentationForm(webService.documentation) : { title: '', description: '', uri: '' },
          provider: webService.provider ? this.buildWebServiceProviderForm(webService.provider) : { id: '', name: '' },
          contactPoints: webService.contact_points?.length ? webService.contact_points.map((contactPoint) => contactPoint) : [],
          spatialCoverage: webService.spatial_coverage ?? null,
          temporalCoverage: this.buildTemporalForm(webService.temporal_coverage),
          parameters: webService.parameters?.length
            ? webService.parameters.map((parameter) => this.buildWebServiceParameterForm(parameter))
            : [this.buildEmptyWebServiceParameterForm()]
        }
      };
    }

    const downloadableFile = (distribution as DownloadableFileDistribution).downloadable_file;

    return {
      ...base,
      kind: 'downloadable-file',
      downloadableFile: {
        url: downloadableFile.url,
        format: downloadableFile.format
      }
    };
  }

  private buildWebServiceDocumentationForm(documentation: Documentation): DatasetWebServiceDocumentationForm {
    return {
      title: documentation.title ?? '',
      description: documentation.description ?? '',
      uri: documentation.uri
    };
  }

  private buildWebServiceProviderForm(provider: ResourceVersionReference): DatasetWebServiceProviderForm {
    return {
      id: provider.id,
      name: provider.name
    };
  }

  private buildTemporalForm(temporalCoverage: WebServiceDistribution['web_service']['temporal_coverage']): DatasetTemporalForm {
    return {
      start: temporalCoverage?.start ? this.toDateTimeLocalValue(temporalCoverage.start) : '',
      end: temporalCoverage?.end ? this.toDateTimeLocalValue(temporalCoverage.end) : ''
    };
  }

  private buildWebServiceParameterForm(parameter: Parameter): DatasetWebServiceParameterForm {
    const metadata: DatasetWebServiceParameterMetadataForm = {
      id: parameter.metadata.id,
      variable: parameter.metadata.variable,
      property: parameter.metadata.property,
      label: parameter.metadata.label,
      required: parameter.metadata.required,
      multiple_values: parameter.metadata.multiple_values,
      read_only: parameter.metadata.read_only
    };

    switch (parameter.kind) {
      case 'STRING':
        return {
          kind: 'STRING',
          metadata,
          defaultValue: parameter.default,
          allowedValues: parameter.allowed_values,
          pattern: parameter.pattern
        };
      case 'BOOLEAN':
        return {
          kind: 'BOOLEAN',
          metadata,
          defaultValue: parameter.default
        };
      case 'INTEGER':
        return {
          kind: 'INTEGER',
          metadata,
          minimum: parameter.minimum == null ? '' : String(parameter.minimum),
          maximum: parameter.maximum == null ? '' : String(parameter.maximum),
          defaultValue: parameter.default == null ? '' : String(parameter.default),
          allowedValues: parameter.allowed_values.map((value) => String(value))
        };
      case 'FLOAT':
        return {
          kind: 'FLOAT',
          metadata,
          minimum: parameter.minimum == null ? '' : String(parameter.minimum),
          maximum: parameter.maximum == null ? '' : String(parameter.maximum),
          defaultValue: parameter.default == null ? '' : String(parameter.default),
          allowedValues: parameter.allowed_values.map((value) => String(value))
        };
      case 'DATETIME':
        return {
          kind: 'DATETIME',
          metadata,
          minimum: parameter.minimum ?? '',
          maximum: parameter.maximum ?? '',
          defaultValue: parameter.default ?? '',
          pattern: parameter.pattern
        };
    }
  }

  private buildEmptyWebServiceParameterForm(): DatasetWebServiceParameterForm {
    return {
      kind: 'STRING',
      metadata: {
        id: '',
        variable: '',
        property: '',
        label: '',
        required: false,
        multiple_values: false,
        read_only: false
      },
      defaultValue: null,
      allowedValues: [],
      pattern: ''
    };
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
    const form = this.datasetForm;
    const temporal = form?.temporal;

    if (!temporal?.start) {
      return 'none';
    }

    if (!temporal.end) {
      return 'warning';
    }

    const start = new Date(temporal.start).getTime();
    const end = new Date(temporal.end).getTime();

    if (!Number.isFinite(start) || !Number.isFinite(end)) {
      return 'none';
    }

    return end <= start ? 'error' : 'none';
  }

  protected temporalWarningMessage(): string {
    const warningKind = this.temporalWarningKind();

    if (warningKind === 'warning') {
      return 'If no end date is provided, it is assumed that this dataset is acquired continuously.';
    }

    if (warningKind === 'error') {
      return 'End date must be after Start Date';
    }

    return '';
  }

  private toDateTimeLocalValue(value: string): string {
    const date = new Date(value);

    if (Number.isNaN(date.getTime())) {
      return value;
    }

    const pad = (input: number): string => String(input).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  private applySavedVersion(resourceType: DiscoveryResourceType, savedVersion: ResourceVersion): void {
    if (resourceType === 'dataset') {
      const typedVersion = savedVersion as ResourceVersionResponse<'dataset'>;
      this.datasetVersion.set(typedVersion);
      this.datasetForm = this.buildDatasetForm(typedVersion.metadata);
      return;
    }

    this.resourceVersion.set(savedVersion as ResourceVersionResponse<'software-application' | 'software-source-code'>);

    if (resourceType === 'software-application') {
      this.softwareApplicationForm = structuredClone(savedVersion.metadata as SoftwareApplicationResourceMetadata);
      return;
    }

    this.softwareSourceCodeForm = structuredClone(savedVersion.metadata as SoftwareSourceCodeResourceMetadata);
  }

  private buildDatasetSaveRequest(): ReplaceDatasetDraftRequest {
    const form = this.datasetForm!;
    const metadata = this.datasetVersion()?.metadata;

    return {
      resource_type: 'dataset',
      revision: this.datasetVersion()!.revision,
      metadata: {
        title: form.title || 'Untitled dataset',
        description: this.undefinedIfBlank(form.description),
        keywords: this.parseKeywords(form.keywords),
        accrual_periodicity: this.undefinedIfBlank(form.accrualPeriodicity),
        quality_annotation: this.undefinedIfBlank(form.qualityAnnotation),
        access_right: metadata?.access_right ?? undefined,
        distributions: form.distributions.map((distribution) => this.buildDistributionDraft(distribution)),
        spatial_coverage: this.buildSpatialCoverageDraft(form.spatial),
        temporal_coverage: this.buildTemporalCoverageDraft(form.temporal),
        persistent_identifiers: form.persistentIdentifiers
          .filter((identifier) => this.hasIdentifierValue(identifier.type) && this.hasIdentifierValue(identifier.value))
          .map((identifier) => ({ type: identifier.type, value: identifier.value })),
        contact_points: form.contactPoints.filter((contactPoint) => this.hasIdentifierValue(contactPoint)),
        providers: form.providers.filter((provider) => this.hasIdentifierValue(provider.id)).map((provider) => provider.id),
        categories: form.categories.filter((category) => this.hasIdentifierValue(category.id)).map((category) => category.id),
        created: this.toIsoDateTimeOrNull(form.created),
        updated: this.toIsoDateTimeOrNull(form.updated)
      }
    };
  }

  private buildSoftwareApplicationSaveRequest(): ReplaceSoftwareApplicationDraftRequest {
    const form = this.softwareApplicationForm!;

    return {
      resource_type: 'software-application',
      revision: this.resourceVersion()!.revision,
      metadata: {
        name: form.name || 'Untitled software application',
        description: form.description,
        keywords: form.keywords,
        license: this.undefinedIfBlank(form.license),
        version: this.undefinedIfBlank(form.version),
        download_url: this.undefinedIfBlank(form.download_url),
        requirements: this.undefinedIfBlank(form.requirements),
        operating_system: this.undefinedIfBlank(form.operating_system),
        main_entity_of_page: this.undefinedIfBlank(form.main_entity_of_page),
        creators: form.creators.filter((creator) => this.hasIdentifierValue(creator.id)).map((creator) => creator.id),
        identifiers: form.identifiers
          .filter((identifier) => this.hasIdentifierValue(identifier.type) && this.hasIdentifierValue(identifier.value))
          .map((identifier) => ({ type: identifier.type, value: identifier.value })),
        categories: form.categories.filter((category) => this.hasIdentifierValue(category.id)).map((category) => category.id),
        contact_points: (form.contact_points ?? []).filter((contactPoint) => this.hasIdentifierValue(contactPoint))
      }
    };
  }

  private buildSoftwareSourceCodeSaveRequest(): ReplaceSoftwareSourceCodeDraftRequest {
    const form = this.softwareSourceCodeForm!;

    return {
      resource_type: 'software-source-code',
      revision: this.resourceVersion()!.revision,
      metadata: {
        name: form.name || 'Untitled software source code',
        description: form.description,
        keywords: form.keywords,
        repository: this.undefinedIfBlank(form.repository),
        programming_language: this.undefinedIfBlank(form.programming_language),
        license: this.undefinedIfBlank(form.license),
        software_requirements: this.undefinedIfBlank(form.software_requirements),
        runtime_platform: this.undefinedIfBlank(form.runtime_platform),
        main_entity_of_page: this.undefinedIfBlank(form.main_entity_of_page),
        version: this.undefinedIfBlank(form.version),
        creators: form.creators.filter((creator) => this.hasIdentifierValue(creator.id)).map((creator) => creator.id),
        identifiers: form.identifiers
          .filter((identifier) => this.hasIdentifierValue(identifier.type) && this.hasIdentifierValue(identifier.value))
          .map((identifier) => ({ type: identifier.type, value: identifier.value })),
        categories: form.categories.filter((category) => this.hasIdentifierValue(category.id)).map((category) => category.id),
        contact_points: (form.contact_points ?? []).filter((contactPoint) => this.hasIdentifierValue(contactPoint))
      }
    };
  }

  private buildDistributionDraft(distribution: DatasetDistributionForm): DistributionDraft {
    const base = {
      title: distribution.title,
      description: distribution.description,
      license: this.undefinedIfBlank(distribution.license),
      issued: this.toIsoDateTimeOrNull(distribution.issued),
      modified: this.toIsoDateTimeOrNull(distribution.modified)
    };

    if (distribution.kind === 'web-service') {
      return {
        ...base,
        type: 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE',
        web_service: this.buildWebServiceDraft(distribution.webService)
      } satisfies WebServiceDistributionDraft;
    }

    return {
      ...base,
      type: 'http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE',
      downloadable_file: distribution.downloadableFile
    } satisfies DownloadableFileDistributionDraft;
  }

  private buildWebServiceDraft(webService: DatasetWebServiceForm): WebServiceDistributionDraft['web_service'] {
    return {
      name: webService.name,
      description: webService.description,
      template_url: webService.templateUrl,
      format: webService.format,
      provider: this.hasIdentifierValue(webService.provider.id) ? webService.provider.id : undefined,
      documentation: webService.documentation,
      spatial_coverage: this.buildSpatialCoverageDraft(webService.spatialCoverage),
      temporal_coverage: this.buildTemporalCoverageDraft(webService.temporalCoverage),
      contact_points: webService.contactPoints.filter((contactPoint) => this.hasIdentifierValue(contactPoint)),
      parameters: webService.parameters
        .filter((parameter) => !this.isEmptyWebServiceParameter(parameter))
        .map((parameter) => this.buildWebServiceParameterDraft(parameter))
    };
  }

  private buildWebServiceParameterDraft(parameter: DatasetWebServiceParameterForm): ParameterDraft {
    const metadata: ParameterDraftMetadata = {
      kind: parameter.kind,
      variable: parameter.metadata.variable,
      property: parameter.metadata.property,
      label: parameter.metadata.label,
      required: parameter.metadata.required,
      multiple_values: parameter.metadata.multiple_values,
      read_only: parameter.metadata.read_only
    };

    switch (parameter.kind) {
      case 'STRING':
        return {
          kind: 'STRING',
          metadata,
          default: this.blankToNull(parameter.defaultValue),
          allowed_values: parameter.allowedValues.filter((value) => this.hasIdentifierValue(value)),
          pattern: parameter.pattern
        };
      case 'BOOLEAN':
        return {
          kind: 'BOOLEAN',
          metadata,
          default: parameter.defaultValue
        };
      case 'INTEGER':
        return {
          kind: 'INTEGER',
          metadata,
          minimum: this.toNumberOrNull(parameter.minimum),
          maximum: this.toNumberOrNull(parameter.maximum),
          default: this.toNumberOrNull(parameter.defaultValue),
          allowed_values: parameter.allowedValues.map((value) => Number(value)).filter((value) => Number.isFinite(value))
        };
      case 'FLOAT':
        return {
          kind: 'FLOAT',
          metadata,
          minimum: this.toNumberOrNull(parameter.minimum),
          maximum: this.toNumberOrNull(parameter.maximum),
          default: this.toNumberOrNull(parameter.defaultValue),
          allowed_values: parameter.allowedValues.map((value) => Number(value)).filter((value) => Number.isFinite(value))
        };
      case 'DATETIME':
        return {
          kind: 'DATETIME',
          metadata,
          minimum: this.toIsoDateTimeOrNull(parameter.minimum),
          maximum: this.toIsoDateTimeOrNull(parameter.maximum),
          default: this.toIsoDateTimeOrNull(parameter.defaultValue),
          pattern: parameter.pattern
        };
    }
  }

  private buildSpatialCoverageDraft(spatial: DatasetSpatialForm): DatasetDraftMetadata['spatial_coverage'] {
    return this.blankToNull(spatial);
  }

  private buildTemporalCoverageDraft(temporal: DatasetTemporalForm): DatasetDraftMetadata['temporal_coverage'] {
    return {
      start: this.toIsoDateTimeOrNull(temporal.start),
      end: this.toIsoDateTimeOrNull(temporal.end)
    };
  }

  private isEmptyWebServiceParameter(parameter: DatasetWebServiceParameterForm): boolean {
    const isMetadataEmpty = [parameter.metadata.variable, parameter.metadata.property, parameter.metadata.label].every((value) => !this.hasIdentifierValue(value));

    switch (parameter.kind) {
      case 'STRING':
        return isMetadataEmpty && !this.hasIdentifierValue(parameter.defaultValue ?? '') && !parameter.allowedValues.length && !this.hasIdentifierValue(parameter.pattern);
      case 'BOOLEAN':
        return isMetadataEmpty && parameter.defaultValue === null;
      case 'INTEGER':
      case 'FLOAT':
        return isMetadataEmpty && !this.hasIdentifierValue(parameter.minimum) && !this.hasIdentifierValue(parameter.maximum) && !this.hasIdentifierValue(parameter.defaultValue) && !parameter.allowedValues.length;
      case 'DATETIME':
        return isMetadataEmpty && !this.hasIdentifierValue(parameter.minimum) && !this.hasIdentifierValue(parameter.maximum) && !this.hasIdentifierValue(parameter.defaultValue) && !this.hasIdentifierValue(parameter.pattern);
    }
  }

  private toIsoDateTimeOrNull(value: string | null | undefined): string | null {
    if (!this.hasIdentifierValue(value)) {
      return null;
    }

    const date = new Date(value);

    return Number.isNaN(date.getTime()) ? value : date.toISOString();
  }

  private toNumberOrNull(value: string | number | null | undefined): number | null {
    if (typeof value === 'number') {
      return Number.isFinite(value) ? value : null;
    }

    if (!this.hasIdentifierValue(value)) {
      return null;
    }

    const normalizedValue = value.trim().replace(',', '.');
    const numberValue = Number(normalizedValue);
    return Number.isFinite(numberValue) ? numberValue : null;
  }

  private blankToNull(value: string | null | undefined): string | null {
    if (value == null) {
      return null;
    }

    return this.hasIdentifierValue(value) ? value : null;
  }

  private undefinedIfBlank(value: string | null | undefined): string | undefined {
    if (value == null) {
      return undefined;
    }

    return this.hasIdentifierValue(value) ? value : undefined;
  }

  private parseKeywords(value: string): string[] {
    return value.split(',').map((keyword) => keyword.trim()).filter(Boolean);
  }

  private hasIdentifierValue(value: string | number | null | undefined): value is string {
    return typeof value === 'string' && value.trim().length > 0;
  }

  protected toggleTheme(): void {
    this.darkMode.update((value) => !value);
  }
}
