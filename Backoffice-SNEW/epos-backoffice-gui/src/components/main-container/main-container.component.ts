import { Component, DestroyRef, DOCUMENT, ElementRef, ViewChild, computed, effect, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router } from '@angular/router';
import { Subject, catchError, combineLatest, debounceTime, distinctUntilChanged, finalize, map, of } from 'rxjs';
import type { Subscription } from 'rxjs';

import type { DiscoveryConcept, DiscoveryItem, DiscoveryResourceType, Group } from '../../../generated/backofficeSchemas';
import { groupAccentColor, groupIconPath } from '../../group-icons';
import type { ResourceVersionComparisonContext } from '../../models/resource-version-comparison';
import { normalizeResultsFilters, type ResultsFilters } from '../../models/results-filters';
import { ApiService } from '../../services/api.service';
import { LocalStorageService } from '../../services/local-storage.service';
import { ResourceCreationDialogService } from '../../services/resource-creation-dialog.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GroupsStore } from '../../stores/groups.store';
import {
  ResourceDetailStore,
  resourceDetailNavigationItems,
  resourceDetailRootLabel,
  type ResourceDetailNavigationItem
} from '../../stores/resource-detail.store';
import { GlobalManagePageComponent } from '../../pages/global-manage/global-manage-page.component';
import { ManageGroupPageComponent } from '../../pages/manage-group/manage-group-page.component';
import { MembershipsPageComponent } from '../../pages/memberships/memberships-page.component';
import { ResourceDetailPageComponent } from '../../pages/resource-detail/resource-detail-page.component';
import { ResourceVersionComparisonPageComponent } from '../../pages/resource-version-comparison/resource-version-comparison-page.component';
import { BackofficeHeaderComponent, type ManagementHeaderContext } from '../backoffice-header/backoffice-header.component';
import { GroupSelectionComponent } from '../group-selection/group-selection.component';
import { NavigationSidebarComponent } from '../navigation-sidebar/navigation-sidebar.component';
import { ResultsTableComponent } from '../results-table/results-table.component';

type RouteData = {
  sectionTitle?: string;
  discoveryType?: DiscoveryResourceType[];
  accountPage?: 'memberships';
  manageGroup?: boolean;
  globalManage?: boolean;
  comparisonPage?: boolean;
  allGroups?: boolean;
};

type RouteState = {
  resourceId?: string | null;
  versionId?: string | null;
  groupId?: string | null;
  dataType?: string | null;
  leftVersionId?: string | null;
  rightVersionId?: string | null;
  sourceVersionId?: string | null;
  panel?: string | null;
  returnUrl?: string | null;
};

const RESULTS_FILTERS_STORAGE_KEY = 'epos-backoffice-results-filters';

@Component({
  selector: 'app-main-container',
  imports: [
    BackofficeHeaderComponent,
    GlobalManagePageComponent,
    GroupSelectionComponent,
    MembershipsPageComponent,
    ManageGroupPageComponent,
    NavigationSidebarComponent,
    ResourceDetailPageComponent,
    ResourceVersionComparisonPageComponent,
    ResultsTableComponent
  ],
  providers: [ResourceDetailStore],
  templateUrl: './main-container.component.html',
  styleUrl: './main-container.component.css'
})
export class MainContainerComponent {
  @ViewChild('detailScrollContainer') private detailScrollContainerRef?: ElementRef<HTMLElement>;

  private readonly apiService = inject(ApiService);
  private readonly activatedRoute = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly groupsStore = inject(GroupsStore);
  private readonly localStorage = inject(LocalStorageService);
  private readonly resourceCreationDialog = inject(ResourceCreationDialogService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly document = inject(DOCUMENT);
  private readonly initialResultsFilters = normalizeResultsFilters(this.localStorage.getJson(RESULTS_FILTERS_STORAGE_KEY));
  private discoveryRequest?: Subscription;
  private readonly searchQueryChanges = new Subject<string>();
  protected readonly detailStore = inject(ResourceDetailStore);
  protected readonly groupAccentColor = groupAccentColor;
  protected readonly groupIconPath = groupIconPath;
  protected readonly sectionTitle = signal('Data');
  protected readonly pageLabel = signal('Results');
  protected readonly accountPage = signal<RouteData['accountPage'] | null>(null);
  protected readonly manageGroup = signal(false);
  protected readonly manageGroupPanel = signal('');
  protected readonly globalManage = signal(false);
  protected readonly allGroups = signal(false);
  protected readonly globalManagePanel = signal('');
  protected readonly manageBackRoute = signal('/data');
  protected readonly sectionPath = signal('data');
  protected readonly darkMode = signal(false);
  protected readonly discoveryItems = signal<DiscoveryItem[] | null>(null);
  protected readonly discoveryLoading = signal(false);
  protected readonly groups = this.groupsStore.groups;
  protected readonly groupsLoading = this.groupsStore.loading;
  protected readonly canAccessGlobalManage = computed(() => this.currentUserStore.canAccessGlobalManage());
  protected readonly selectedGroupId = signal('');
  protected readonly selectedGroupName = signal('');
  protected readonly searchQuery = signal(this.initialResultsFilters.searchQuery);
  protected readonly selectedDataTypes = signal(this.initialResultsFilters.dataTypes);
  protected readonly selectedVersioningStatuses = signal(this.initialResultsFilters.versioningStatuses);
  protected readonly currentConcept = computed<DiscoveryConcept>(() => {
    const concepts: Record<string, DiscoveryConcept> = {
      data: 'DATA',
      software: 'SOFTWARE',
      facilities: 'FACILITIES'
    };
    return concepts[this.sectionPath()] ?? 'DATA';
  });
  protected readonly canCreateResource = computed(() =>
    this.currentUserStore.hasGroupPermission(this.selectedGroupId(), 'group.resource.create')
  );
  protected readonly canCopyResources = computed(() => this.resourceCreationDialog.groups().length > 0);
  protected readonly canManageSelectedGroup = computed(() => {
    const groupId = this.selectedGroupId();
    return !!groupId && this.currentUserStore.canManageGroup(groupId);
  });

  protected openCopyResource(item: DiscoveryItem): void {
    this.resourceCreationDialog.openCopy(
      item.type,
      item.resource_id,
      item.group.id,
      item.concept === 'DATA' ? 'data' : item.concept === 'SOFTWARE' ? 'software' : 'facilities'
    );
  }
  protected readonly comparisonContext = signal<ResourceVersionComparisonContext | null>(null);
  protected readonly managementHeaderContext = computed<ManagementHeaderContext | null>(() => {
    if (this.globalManage() && !this.globalManagePanel()) {
      return 'global';
    }

    return this.manageGroup() && !this.manageGroupPanel() ? 'group' : null;
  });
  protected readonly headerBackRoute = computed(() => {
    const comparison = this.comparisonContext();
    if (comparison) {
      return this.router.serializeUrl(this.router.createUrlTree(
        ['/', comparison.sectionPath, comparison.resourceId, comparison.sourceVersionId],
        { queryParams: { groupId: comparison.groupId, dataType: comparison.resourceType } }
      ));
    }

    return this.managementHeaderContext()
      || this.accountPage() === 'memberships'
      || this.pageLabel() === 'Detail'
      ? this.manageBackRoute()
      : null;
  });
  protected readonly sidebarDetailItems = computed<ResourceDetailNavigationItem[]>(() => {
    const comparison = this.comparisonContext();
    if (!comparison) {
      return this.detailStore.navigationItems();
    }

    return resourceDetailNavigationItems(comparison.resourceType).map(({ label, anchor }) => ({
      label,
      anchor: `compare-${anchor}`
    }));
  });
  protected readonly sidebarDetailRootLabel = computed(() => {
    const comparison = this.comparisonContext();
    return comparison ? resourceDetailRootLabel(comparison.resourceType) : this.detailStore.rootLabel();
  });

  constructor() {
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
          dataType: queryParams.get('dataType'),
          leftVersionId: queryParams.get('leftVersionId'),
          rightVersionId: queryParams.get('rightVersionId'),
          sourceVersionId: queryParams.get('sourceVersionId'),
          panel: queryParams.get('panel'),
          returnUrl: queryParams.get('returnUrl')
        });
      });

    effect(() => {
      this.document.documentElement.setAttribute('data-theme', this.darkMode() ? 'dark' : 'light');
    });
  }

  private syncRouteState(routeData: RouteData, routeState: RouteState): void {
    const accountPage = routeData.accountPage ?? null;
    const manageGroup = routeData.manageGroup ?? false;
    const globalManage = routeData.globalManage ?? false;
    const allGroups = routeData.allGroups ?? false;
    const discoveryType = routeData.discoveryType;
    const detailDataType = this.toDiscoveryResourceType(routeState.dataType) ?? discoveryType?.[0];
    const isComparisonRoute = routeData.comparisonPage === true;
    const isDetailRoute = !isComparisonRoute && Boolean(routeState.resourceId && routeState.versionId);
    this.accountPage.set(accountPage);
    this.manageGroup.set(manageGroup);
    this.manageGroupPanel.set(manageGroup ? (routeState.panel ?? '') : '');
    this.globalManage.set(globalManage);
    this.allGroups.set(allGroups);
    this.globalManagePanel.set(globalManage ? (routeState.panel ?? '') : '');
    this.sectionPath.set(accountPage ?? (globalManage ? 'global-manage' : this.currentSectionPath()));
    this.manageBackRoute.set(this.validReturnUrl(routeState.returnUrl) ?? this.defaultBackRoute(manageGroup || isDetailRoute || isComparisonRoute, routeState.groupId));
    this.sectionTitle.set(
      (isDetailRoute || isComparisonRoute) && detailDataType
        ? detailDataType === 'dataset'
          ? 'Dataset'
          : detailDataType === 'software-application'
            ? 'Software Application'
            : 'Software Source Code'
        : (routeData.sectionTitle ?? 'Data')
    );

    if (accountPage || manageGroup || globalManage) {
      this.detailStore.clear();
      this.comparisonContext.set(null);
      this.discoveryRequest?.unsubscribe();
      const selectedGroup = manageGroup
        ? this.groupForCurrentSection(routeState.groupId)
        : accountPage
          ? this.groups().find((group) => group.id === routeState.groupId)
          : undefined;
      this.selectedGroupId.set(selectedGroup?.id ?? '');
      this.selectedGroupName.set(selectedGroup?.name ?? '');
      this.pageLabel.set(
        manageGroup
          ? 'Manage Group'
          : globalManage
            ? 'Manage'
            : 'Memberships'
      );
      this.discoveryItems.set(null);
      this.discoveryLoading.set(false);
      return;
    }

    const selectedGroup = this.groupForCurrentSection(routeState.groupId);
    this.selectedGroupId.set(selectedGroup?.id ?? '');
    this.selectedGroupName.set(selectedGroup?.name ?? '');
    this.pageLabel.set(isComparisonRoute ? 'Compare' : isDetailRoute ? 'Detail' : 'Results');

    if (isComparisonRoute) {
      this.detailStore.clear();
      this.discoveryLoading.set(false);
      this.discoveryItems.set(null);
      this.comparisonContext.set(
        detailDataType && routeState.resourceId && routeState.leftVersionId && routeState.rightVersionId
          ? {
              resourceType: detailDataType,
              resourceId: routeState.resourceId,
              leftVersionId: routeState.leftVersionId,
              rightVersionId: routeState.rightVersionId,
              sourceVersionId: routeState.sourceVersionId ?? routeState.rightVersionId,
              groupId: selectedGroup?.id ?? routeState.groupId ?? '',
              sectionPath: this.sectionPath()
            }
          : null
      );
      return;
    }

    this.comparisonContext.set(null);

    if (isDetailRoute && detailDataType && routeState.resourceId && routeState.versionId) {
      this.discoveryLoading.set(false);
      this.discoveryItems.set(null);
      this.detailStore.load({
        resourceType: detailDataType,
        resourceId: routeState.resourceId,
        versionId: routeState.versionId,
        groupId: selectedGroup?.id ?? '',
        sectionPath: this.sectionPath(),
        returnUrl: this.manageBackRoute()
      });
      return;
    }

    this.detailStore.clear();
    this.loadRouteData(this.selectedGroupId());
  }

  private loadRouteData(groupId: string): void {
    this.discoveryRequest?.unsubscribe();

    if (!groupId && !this.allGroups()) {
      this.discoveryItems.set(null);
      this.discoveryLoading.set(false);
      return;
    }

    this.discoveryLoading.set(true);
    this.discoveryRequest = this.apiService
      .getDiscovery({
        groupId: groupId || undefined,
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
      .subscribe((items) => this.discoveryItems.set(items));
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
    if (this.pageLabel() === 'Results') {
      this.loadRouteData(this.selectedGroupId());
    }
  }

  private persistResultsFilters(): void {
    this.localStorage.setJson(RESULTS_FILTERS_STORAGE_KEY, {
      searchQuery: this.searchQuery(),
      dataTypes: this.selectedDataTypes(),
      versioningStatuses: this.selectedVersioningStatuses()
    } satisfies ResultsFilters);
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

  protected openCreateResourceDialog(): void {
    if (!this.canCreateResource()) return;
    this.resourceCreationDialog.open(this.selectedGroupId(), this.sectionPath());
  }

  protected scrollContentToTop(): void {
    this.detailScrollContainerRef?.nativeElement.scrollTo({ top: 0, behavior: 'smooth' });
  }

  protected currentUrl(): string {
    return this.router.url;
  }

  protected globalManageReturnUrl(): string {
    return this.globalManage() ? this.manageBackRoute() : this.currentUrl();
  }

  private defaultBackRoute(hasSectionBackRoute: boolean, groupId: string | null | undefined): string {
    if (!hasSectionBackRoute) {
      return '/data';
    }

    const groupQuery = groupId ? `?groupId=${encodeURIComponent(groupId)}` : '';
    return `/${this.currentSectionPath()}${groupQuery}`;
  }

  private validReturnUrl(returnUrl: string | null | undefined): string | null {
    return returnUrl?.startsWith('/') && !returnUrl.startsWith('//') ? returnUrl : null;
  }

  private currentSectionPath(): string {
    return this.activatedRoute.snapshot.routeConfig?.path?.split('/')[0] ?? 'data';
  }

  protected toggleTheme(): void {
    this.darkMode.update((value) => !value);
  }
}
