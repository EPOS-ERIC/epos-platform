import { Component, DestroyRef, DOCUMENT, ElementRef, ViewChild, effect, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute } from '@angular/router';
import { Subject, catchError, combineLatest, debounceTime, distinctUntilChanged, finalize, map, of } from 'rxjs';
import type { Subscription } from 'rxjs';

import type { DiscoveryConcept, DiscoveryItem, DiscoveryResourceType, Group } from '../../../generated/backofficeSchemas';
import { normalizeResultsFilters, type ResultsFilters } from '../../models/results-filters';
import { ApiService } from '../../services/api.service';
import { LocalStorageService } from '../../services/local-storage.service';
import { ResourceCreationDialogService } from '../../services/resource-creation-dialog.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GroupsStore } from '../../stores/groups.store';
import { ResourceDetailStore } from '../../stores/resource-detail.store';
import { AdminCategoriesComponent } from '../admin-categories/admin-categories.component';
import { AdminGroupsComponent } from '../admin-groups/admin-groups.component';
import { AdminOrganizationsComponent } from '../admin-organizations/admin-organizations.component';
import { BackofficeHeaderComponent } from '../backoffice-header/backoffice-header.component';
import { MembershipsPageComponent } from '../memberships-page/memberships-page.component';
import { NavigationSidebarComponent } from '../navigation-sidebar/navigation-sidebar.component';
import { ResourceDetailPageComponent } from '../resource-detail-page/resource-detail-page.component';
import { ResultsTableComponent } from '../results-table/results-table.component';

type RouteData = {
  sectionTitle?: string;
  discoveryType?: DiscoveryResourceType[];
  adminPage?: 'groups' | 'categories' | 'organizations';
  accountPage?: 'memberships';
};

type RouteState = {
  resourceId?: string | null;
  versionId?: string | null;
  groupId?: string | null;
  dataType?: string | null;
};

const RESULTS_FILTERS_STORAGE_KEY = 'epos-backoffice-results-filters';

@Component({
  selector: 'app-main-container',
  imports: [
    AdminCategoriesComponent,
    AdminGroupsComponent,
    AdminOrganizationsComponent,
    BackofficeHeaderComponent,
    MembershipsPageComponent,
    NavigationSidebarComponent,
    ResourceDetailPageComponent,
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
  protected readonly sectionTitle = signal('Data');
  protected readonly pageLabel = signal('Results');
  protected readonly adminPage = signal<RouteData['adminPage'] | null>(null);
  protected readonly accountPage = signal<RouteData['accountPage'] | null>(null);
  protected readonly sectionPath = signal('data');
  protected readonly darkMode = signal(false);
  protected readonly discoveryItems = signal<DiscoveryItem[] | null>(null);
  protected readonly discoveryLoading = signal(false);
  protected readonly groups = this.groupsStore.groups;
  protected readonly groupsLoading = this.groupsStore.loading;
  protected readonly isAdmin = this.currentUserStore.isAdmin;
  protected readonly selectedGroupId = signal('');
  protected readonly selectedGroupName = signal('');
  protected readonly searchQuery = signal(this.initialResultsFilters.searchQuery);
  protected readonly selectedDataTypes = signal(this.initialResultsFilters.dataTypes);
  protected readonly selectedVersioningStatuses = signal(this.initialResultsFilters.versioningStatuses);

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
          dataType: queryParams.get('dataType')
        });
      });

    effect(() => {
      this.document.documentElement.setAttribute('data-theme', this.darkMode() ? 'dark' : 'light');
    });
  }

  private syncRouteState(routeData: RouteData, routeState: RouteState): void {
    const adminPage = routeData.adminPage ?? null;
    const accountPage = routeData.accountPage ?? null;
    const discoveryType = routeData.discoveryType;
    const detailDataType = this.toDiscoveryResourceType(routeState.dataType) ?? discoveryType?.[0];
    const isDetailRoute = Boolean(routeState.resourceId && routeState.versionId);
    this.adminPage.set(adminPage);
    this.accountPage.set(accountPage);
    this.sectionPath.set(accountPage ?? (adminPage ? `admin/${adminPage}` : this.currentSectionPath()));
    this.sectionTitle.set(
      isDetailRoute && detailDataType
        ? detailDataType === 'dataset'
          ? 'Dataset'
          : detailDataType === 'software-application'
            ? 'Software Application'
            : 'Software Source Code'
        : (routeData.sectionTitle ?? 'Data')
    );

    if (adminPage || accountPage) {
      this.detailStore.clear();
      this.discoveryRequest?.unsubscribe();
      const selectedGroup = accountPage
        ? this.groups().find((group) => group.id === routeState.groupId)
        : undefined;
      this.selectedGroupId.set(selectedGroup?.id ?? '');
      this.selectedGroupName.set(selectedGroup?.name ?? '');
      this.pageLabel.set(accountPage ? 'Memberships' : adminPage === 'groups' ? 'Users' : adminPage === 'categories' ? 'Categories' : 'Organizations');
      this.discoveryItems.set(null);
      this.discoveryLoading.set(false);
      return;
    }

    const selectedGroup = this.groupForCurrentSection(routeState.groupId);
    this.selectedGroupId.set(selectedGroup?.id ?? '');
    this.selectedGroupName.set(selectedGroup?.name ?? '');
    this.pageLabel.set(isDetailRoute ? 'Detail' : 'Results');

    if (isDetailRoute && detailDataType && routeState.resourceId && routeState.versionId) {
      this.discoveryLoading.set(false);
      this.discoveryItems.set(null);
      this.detailStore.load({
        resourceType: detailDataType,
        resourceId: routeState.resourceId,
        versionId: routeState.versionId,
        groupId: selectedGroup?.id ?? '',
        sectionPath: this.sectionPath(),
        isAdmin: this.isAdmin()
      });
      return;
    }

    this.detailStore.clear();
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
    this.resourceCreationDialog.open(this.selectedGroupId(), this.sectionPath());
  }

  protected scrollDetailToTop(): void {
    this.detailScrollContainerRef?.nativeElement.scrollTo({ top: 0, behavior: 'smooth' });
  }

  protected membershipBackLink(): string[] {
    const concept = this.groups().find((group) => group.id === this.selectedGroupId())?.concept;
    const sectionPath = concept === 'SOFTWARE' ? 'software' : concept === 'FACILITIES' ? 'facilities' : 'data';
    return ['/', sectionPath];
  }

  private currentSectionPath(): string {
    return this.activatedRoute.snapshot.routeConfig?.path?.split('/')[0] ?? 'data';
  }

  protected toggleTheme(): void {
    this.darkMode.update((value) => !value);
  }
}
