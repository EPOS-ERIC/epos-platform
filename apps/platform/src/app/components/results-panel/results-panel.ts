import { ChangeDetectorRef, Component, OnInit, inject } from '@angular/core';
import { finalize } from 'rxjs';
import { PlatformApiService } from '../../api/services/platform-api.service';
import { SearchItem, SearchQueryParams } from '../../api/types/api-types';
import { SearchFilterStateService } from '../../services/search-filter-state.service';

type ResourceType = SearchItem['type'];
type ResourceName = SearchItem['name'];
type DatasetSearchItem = Extract<SearchItem, { type: 'dataset' }>;
type SearchItemRef = {
  id: string;
  name?: string;
  href: string;
};

type SearchCardVm = {
  id: string;
  name: ResourceName;
  type: ResourceType;
  href: string;
  source: SearchItem;
  providers: NonNullable<DatasetSearchItem['providers']>;
  concepts: NonNullable<DatasetSearchItem['concepts']>;
  distributions: NonNullable<DatasetSearchItem['distributions']>;
};

@Component({
  selector: 'app-results-panel',
  imports: [],
  templateUrl: './results-panel.html',
  styleUrl: './results-panel.css',
})
export class ResultsPanel implements OnInit {
  public readonly filterState = inject(SearchFilterStateService);

  constructor(platformApiService: PlatformApiService, cdr: ChangeDetectorRef) {
    this.platformApiService = platformApiService;
    this.cdr = cdr;
    console.warn('ResultsPanel component initialized');
  }

  public platformApiService: PlatformApiService;
  public cdr: ChangeDetectorRef;
  public isLoading = false;
  public errorMessage = '';
  public results: SearchCardVm[] = [];
  public selectedItemJson = '';
  public selectedDistributionByCard: Record<string, number> = {};
  public searchQuery = '';

  ngOnInit(): void {
    this.firstApiCall();
  }

  public firstApiCall(): void {
    this.isLoading = true;
    this.errorMessage = '';

    this.platformApiService
      .search(this.getSearchParams())
      .pipe(
        finalize(() => {
          this.isLoading = false;
          this.cdr.detectChanges();
        }),
      )
      .subscribe({
        next: (response) => {
          console.warn('Search Response', response);
          this.buildCards(response.items);
        },
        error: (error: unknown) => {
          this.errorMessage = 'Search request failed';
          console.error('API /search Error:', error);
        },
      });
  }

  public selectItem(item: SearchCardVm): void {
    this.selectedItemJson = JSON.stringify(item.source, null, 2);
  }

  public onSearchQuery(query: string): void {
    this.searchQuery = query;
    this.firstApiCall();
  }

  public onClearSearchQuery(): void {
    this.searchQuery = '';
    this.firstApiCall();
  }

  public setSelectedFilter(filter: 'dataset' | 'organization' | 'all'): void {
    this.filterState.setSelectedFilter(filter);
    this.firstApiCall();
  }

  public getSelectedDistributionIndex(item: SearchCardVm): number {
    const distributions = item.distributions ?? [];
    const savedIndex = this.selectedDistributionByCard[item.href] ?? 0;
    if (savedIndex < 0 || savedIndex >= distributions.length) {
      return 0;
    }

    return savedIndex;
  }

  public selectDistributionTab(item: SearchCardVm, index: number): void {
    this.selectedDistributionByCard[item.href] = index;
  }

  public refLabel(ref: SearchItemRef): string {
    return ref.name?.trim() || ref.id;
  }

  private getSearchParams(): SearchQueryParams {
    const selectedFilter = this.filterState.selectedFilter();

    if (selectedFilter === 'organization') {
      return { type: ['organization'] };
    }

    if (selectedFilter === 'all') {
      return this.searchQuery ? { q: this.searchQuery } : {};
    }

    return this.searchQuery ? { type: ['dataset'], q: this.searchQuery } : { type: ['dataset'] };
  }

  private buildCards(items: SearchItem[]): void {
    if (items.length === 0) {
      this.results = [];
      this.selectedItemJson = '';
      this.selectedDistributionByCard = {};
      return;
    }

    this.results = items.map((item) => this.toCard(item));
    this.selectedDistributionByCard = {};
    const firstCard = this.results[0];
    this.selectedItemJson = firstCard ? JSON.stringify(firstCard.source, null, 2) : '';
  }

  private toCard(item: SearchItem): SearchCardVm {
    const datasetItem = item.type === 'dataset' ? item : null;

    return {
      id: item.id,
      name: item.name,
      type: item.type,
      href: item.href,
      source: item,
      providers: datasetItem?.providers ?? [],
      concepts: datasetItem?.concepts ?? [],
      distributions: datasetItem?.distributions ?? [],
    };
  }
}
