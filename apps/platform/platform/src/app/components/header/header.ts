import { Component, EventEmitter, Input, Output } from '@angular/core';

import { SearchBar } from '../search-bar/search-bar';

type MapViewMode = '2d' | 'globe';

@Component({
  selector: 'app-header',
  imports: [SearchBar],
  templateUrl: './header.html',
  styleUrl: './header.css',
})
export class Header {
  @Input() public isDarkMode = false;
  @Output() openSearch = new EventEmitter<void>();
  @Output() mapViewModeChange = new EventEmitter<MapViewMode>();
  @Output() toggleTheme = new EventEmitter<void>();

  public mapViewMode: MapViewMode = '2d';

  public onOpenSearch(): void {
    this.openSearch.emit();
  }

  public onMapViewModeChange(mode: MapViewMode): void {
    this.mapViewMode = mode;
    this.mapViewModeChange.emit(mode);
  }

  public onToggleTheme(): void {
    this.toggleTheme.emit();
  }
}
