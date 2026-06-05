import {
  Component,
  ElementRef,
  HostListener,
  OnInit,
  ViewChild,
  inject,
  signal,
} from '@angular/core';
import { DOCUMENT } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterOutlet } from '@angular/router';
import { Header } from './components/header/header';
import { MapView } from './components/map-view/map-view';
import { ResultsPanel } from './components/results-panel/results-panel';
import { SpatialFilter } from './components/spatial-filter/spatial-filter';
import { TemporalFilter } from './components/temporal-filter/temporal-filter';
import { SearchFilterStateService } from './services/search-filter-state.service';

type ThemeMode = 'light' | 'dark';

@Component({
  selector: 'app-root',
  imports: [
    RouterOutlet,
    ResultsPanel,
    Header,
    MapView,
    SpatialFilter,
    TemporalFilter,
    FormsModule,
  ],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App implements OnInit {
  private readonly document = inject(DOCUMENT);
  public readonly filterState = inject(SearchFilterStateService);
  private readonly themeStorageKey = 'platform-theme';
  protected readonly title = signal('platform');
  public mapViewMode: '2d' | 'globe' = '2d';
  public isSearchDialogOpen = false;
  public searchQuery = '';
  public isDarkMode = false;
  public isFiltersExpanded = false;
  @ViewChild('globalSearchInput') public globalSearchInput?: ElementRef<HTMLInputElement>;

  public ngOnInit(): void {
    const storedTheme = this.getStoredTheme();
    const prefersDark =
      typeof window !== 'undefined' &&
      typeof window.matchMedia === 'function' &&
      window.matchMedia('(prefers-color-scheme: dark)').matches;
    this.setTheme(storedTheme ?? (prefersDark ? 'dark' : 'light'));
  }

  public onMapViewModeChange(mode: '2d' | 'globe'): void {
    this.mapViewMode = mode;
  }

  public toggleTheme(): void {
    this.setTheme(this.isDarkMode ? 'light' : 'dark');
  }

  public toggleFilters(): void {
    this.isFiltersExpanded = !this.isFiltersExpanded;
  }

  @HostListener('window:keydown', ['$event'])
  public handleKeyDown(event: KeyboardEvent): void {
    const isShortcut = (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k';
    if (isShortcut) {
      if (this.isTypingTarget(event.target)) {
        return;
      }

      event.preventDefault();
      this.openSearchDialog();
      return;
    }

    if (event.key === 'Escape' && this.isSearchDialogOpen) {
      this.closeSearchDialog();
    }
  }

  public openSearchDialog(): void {
    this.isSearchDialogOpen = true;
    setTimeout(() => this.globalSearchInput?.nativeElement.focus(), 0);
  }

  public closeSearchDialog(): void {
    this.isSearchDialogOpen = false;
  }

  public submitSearch(resultsPanel: ResultsPanel): void {
    resultsPanel.onSearchQuery(this.searchQuery.trim());
    this.closeSearchDialog();
  }

  public clearSearch(resultsPanel: ResultsPanel): void {
    this.searchQuery = '';
    resultsPanel.onClearSearchQuery();
    this.closeSearchDialog();
  }

  private setTheme(theme: ThemeMode): void {
    this.isDarkMode = theme === 'dark';
    this.document.documentElement.classList.toggle('dark', this.isDarkMode);
    this.document.documentElement.style.colorScheme = theme;

    try {
      window.localStorage.setItem(this.themeStorageKey, theme);
    } catch {
      // Ignore storage failures.
    }
  }

  private getStoredTheme(): ThemeMode | null {
    try {
      const theme = window.localStorage.getItem(this.themeStorageKey);
      return theme === 'dark' || theme === 'light' ? theme : null;
    } catch {
      return null;
    }
  }

  private isTypingTarget(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) {
      return false;
    }

    const tagName = target.tagName.toLowerCase();
    return (
      target.isContentEditable ||
      tagName === 'input' ||
      tagName === 'textarea' ||
      tagName === 'select'
    );
  }
}
