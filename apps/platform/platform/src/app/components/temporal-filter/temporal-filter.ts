import { Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { SearchFilterStateService } from '../../services/search-filter-state.service';

@Component({
  selector: 'app-temporal-filter',
  imports: [FormsModule],
  templateUrl: './temporal-filter.html',
  styleUrl: './temporal-filter.css',
})
export class TemporalFilter {
  public readonly filterState = inject(SearchFilterStateService);
}
