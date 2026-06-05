import { Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { SearchFilterStateService } from '../../services/search-filter-state.service';

@Component({
  selector: 'app-spatial-filter',
  imports: [FormsModule],
  templateUrl: './spatial-filter.html',
  styleUrl: './spatial-filter.css',
})
export class SpatialFilter {
  public readonly filterState = inject(SearchFilterStateService);
}
