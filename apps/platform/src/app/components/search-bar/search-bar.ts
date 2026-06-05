import { Component, EventEmitter, Output } from '@angular/core';

@Component({
  selector: 'app-search-bar',
  imports: [],
  templateUrl: './search-bar.html',
  styleUrl: './search-bar.css',
})
export class SearchBar {
  @Output() openSearch = new EventEmitter<void>();

  public onOpenSearch(): void {
    this.openSearch.emit();
  }
}
