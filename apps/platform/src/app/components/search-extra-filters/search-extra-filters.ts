import { Component } from '@angular/core';
import { DataproviderFilter } from '../dataprovider-filter/dataprovider-filter';
import { DatavisualizationFilter } from '../datavisualization-filter/datavisualization-filter';

@Component({
  selector: 'app-search-extra-filters',
  imports: [DataproviderFilter, DatavisualizationFilter],
  templateUrl: './search-extra-filters.html',
  styleUrl: './search-extra-filters.css',
})
export class SearchExtraFilters {}
