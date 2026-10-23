import { Component, ElementRef, Input, Output, EventEmitter, HostListener, OnDestroy, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';

import type { DiscoveryResourceType, Reference, VersioningStatus } from '../../../generated/backofficeSchemas';
import { DISCOVERY_VERSIONING_STATUSES, type ResultsFilterSection, type ResultsFilters } from '../../models/results-filters';
import { AuthService } from '../../services/auth.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { AppIconComponent } from '../../shared/ui/app-icon.component';

export type ManagementHeaderContext = 'global' | 'group';

const SECTION_SUBTITLES: Record<string, string> = {
  Data: 'Manage resources and related metadata.',
  Software: 'Manage resources and related metadata.',
  Facilities: 'Manage resources and related metadata.',
  'All Resources': 'Browse resources across all your groups.',
  Dataset: 'Review dataset metadata and lifecycle.',
  'Software Application': 'Review application metadata and lifecycle.',
  'Software Source Code': 'Review source code metadata and lifecycle.',
  'My Account': 'Review your memberships and access.'
};

@Component({
  selector: 'app-backoffice-header',
  imports: [AppIconComponent, FormsModule, RouterLink],
  templateUrl: './backoffice-header.component.html',
  styleUrl: './backoffice-header.component.css'
})
export class BackofficeHeaderComponent implements OnDestroy {
  private readonly authService = inject(AuthService);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly router = inject(Router);
  private readonly elementRef = inject<ElementRef<HTMLElement>>(ElementRef);
  private highlightedFilterGroup?: HTMLElement;
  private filterOpenTimer?: ReturnType<typeof setTimeout>;
  private filterHighlightTimer?: ReturnType<typeof setTimeout>;

  protected readonly userMenuOpen = signal(false);
  protected readonly filterMenuOpen = signal(false);
  protected readonly authorSearchQuery = signal('');
  protected readonly currentUserName = computed(() => {
    const user = this.currentUserStore.user();
    if (!user) return 'Backoffice user';

    return `${user.given_name} ${user.family_name}`.trim() || user.email;
  });
  protected readonly isGlobalAdmin = computed(() => this.currentUserStore.user()?.global_roles.includes('ADMIN') ?? false);
  protected readonly dataTypeOptions: Array<{ label: string; value: DiscoveryResourceType }> = [
    { label: 'Dataset', value: 'dataset' },
    { label: 'Software Application', value: 'software-application' },
    { label: 'Software Source Code', value: 'software-source-code' }
  ];
  protected readonly versioningStatusOptions = DISCOVERY_VERSIONING_STATUSES;

  @Input() sectionTitle = 'Data';
  @Input() pageLabel = 'Results';
  @Input() darkMode = false;
  @Input() showSearchActions = true;
  @Input() canAddNewResource = false;
  @Input() searchQuery = '';
  @Input() filterByAuthor = '';
  @Input() availableAuthors: Reference[] = [];
  @Input() selectedDataTypes: DiscoveryResourceType[] = [];
  @Input() selectedVersioningStatuses: VersioningStatus[] = [];
  @Input() managementContext: ManagementHeaderContext | null = null;
  @Input() managementEyebrow = '';
  @Input() managementTitle = '';
  @Input() managementSubtitle = '';
  @Input() managementIconPath = '';
  @Input() managementBackRoute = '/data';
  @Input() headerBackRoute: string | null = null;
  @Output() toggleTheme = new EventEmitter<void>();
  @Output() addNewResource = new EventEmitter<void>();
  @Output() searchQueryChange = new EventEmitter<string>();
  @Output() filtersChange = new EventEmitter<ResultsFilters>();

  protected get headerBackRouteTree() {
    return this.headerBackRoute ? this.router.parseUrl(this.headerBackRoute) : null;
  }

  protected sectionSubtitle(): string {
    return SECTION_SUBTITLES[this.sectionTitle] ?? 'Manage resources and related metadata.';
  }

  @HostListener('document:click')
  protected closeUserMenu(): void {
    this.userMenuOpen.set(false);
    this.filterMenuOpen.set(false);
  }

  @HostListener('document:keydown.escape')
  protected closeUserMenuOnEscape(): void {
    this.userMenuOpen.set(false);
    this.filterMenuOpen.set(false);
  }

  protected toggleUserMenu(event: MouseEvent): void {
    event.stopPropagation();
    this.filterMenuOpen.set(false);
    this.userMenuOpen.update((open) => !open);
  }

  protected toggleFilterMenu(event: MouseEvent): void {
    event.stopPropagation();
    this.userMenuOpen.set(false);
    this.filterMenuOpen.update((open) => !open);
  }

  openFilterSection(section: ResultsFilterSection): void {
    this.userMenuOpen.set(false);
    this.filterMenuOpen.set(true);
    if (this.filterOpenTimer) {
      clearTimeout(this.filterOpenTimer);
    }
    this.filterOpenTimer = setTimeout(() => {
      this.highlightedFilterGroup?.classList.remove('filter-group-content--highlighted');
      if (this.filterHighlightTimer) {
        clearTimeout(this.filterHighlightTimer);
      }

      const filterGroup = this.elementRef.nativeElement
        .querySelector<HTMLElement>(`[data-filter-section="${section}"]`);
      filterGroup?.querySelector<HTMLInputElement>('input')?.focus({ preventScroll: true });
      const filterContent = filterGroup?.querySelector<HTMLElement>('.filter-group-content');
      if (filterContent) {
        filterContent.classList.add('filter-group-content--highlighted');
        this.highlightedFilterGroup = filterContent;
        this.filterHighlightTimer = setTimeout(() => {
          filterContent.classList.remove('filter-group-content--highlighted');
          this.highlightedFilterGroup = undefined;
        }, 1300);
      }
    }, 0);
  }

  ngOnDestroy(): void {
    if (this.filterOpenTimer) {
      clearTimeout(this.filterOpenTimer);
    }
    if (this.filterHighlightTimer) {
      clearTimeout(this.filterHighlightTimer);
    }
  }

  protected filteredAuthors(): Reference[] {
    const query = this.authorSearchQuery().trim().toLocaleLowerCase();
    return query
      ? this.availableAuthors.filter((author) => author.name.toLocaleLowerCase().includes(query))
      : this.availableAuthors;
  }

  protected authorHintsOpen(): boolean {
    return Boolean(this.authorSearchQuery().trim());
  }

  protected selectedAuthorName(): string {
    return this.availableAuthors.find((author) => author.id === this.filterByAuthor)?.name ?? 'Selected author';
  }

  protected selectAuthor(author: Reference): void {
    this.authorSearchQuery.set('');
    this.emitAuthorFilter(author.id);
  }

  protected clearAuthorFilter(): void {
    this.authorSearchQuery.set('');
    this.emitAuthorFilter('');
  }

  private emitAuthorFilter(authorId: string): void {
    this.filtersChange.emit({
      searchQuery: this.searchQuery,
      filterByAuthor: authorId,
      dataTypes: this.selectedDataTypes,
      versioningStatuses: this.selectedVersioningStatuses
    });
  }

  protected toggleDataType(dataType: DiscoveryResourceType): void {
    this.filtersChange.emit({
      searchQuery: this.searchQuery,
      filterByAuthor: this.filterByAuthor,
      dataTypes: this.toggleSelection(this.selectedDataTypes, dataType),
      versioningStatuses: this.selectedVersioningStatuses
    });
  }

  protected toggleVersioningStatus(status: VersioningStatus): void {
    const versioningStatuses = this.toggleSelection(this.selectedVersioningStatuses, status);

    this.filtersChange.emit({
      searchQuery: this.searchQuery,
      filterByAuthor: this.filterByAuthor,
      dataTypes: this.selectedDataTypes,
      versioningStatuses: versioningStatuses.length ? versioningStatuses : ['PUBLISHED']
    });
  }

  protected activeFilterCount(): number {
    return Number(Boolean(this.searchQuery.trim()))
      + Number(Boolean(this.filterByAuthor))
      + this.selectedDataTypes.length
      + this.selectedVersioningStatuses.length;
  }

  protected canClearFilters(): boolean {
    return Boolean(this.searchQuery.trim())
      || Boolean(this.filterByAuthor)
      || Boolean(this.selectedDataTypes.length)
      || this.selectedVersioningStatuses.length !== 1
      || this.selectedVersioningStatuses[0] !== 'PUBLISHED';
  }

  protected clearFilters(): void {
    this.authorSearchQuery.set('');
    this.filtersChange.emit({ searchQuery: '', filterByAuthor: '', dataTypes: [], versioningStatuses: ['PUBLISHED'] });
  }

  protected logout(): void {
    this.userMenuOpen.set(false);
    this.authService.logout();
  }

  private toggleSelection<T>(selection: T[], value: T): T[] {
    return selection.includes(value) ? selection.filter((item) => item !== value) : [...selection, value];
  }
}
