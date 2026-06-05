import { Injectable, computed, signal } from '@angular/core';

export type SearchFilter = 'dataset' | 'organization' | 'all';

@Injectable({
  providedIn: 'root',
})
export class SearchFilterStateService {
  private readonly selectedFilterSignal = signal<SearchFilter>('dataset');
  private readonly temporalFilterFromSignal = signal('');
  private readonly temporalFilterToSignal = signal('');
  private readonly spatialFilterMinLatSignal = signal('');
  private readonly spatialFilterMaxLatSignal = signal('');
  private readonly spatialFilterMinLonSignal = signal('');
  private readonly spatialFilterMaxLonSignal = signal('');

  public readonly selectedFilter = this.selectedFilterSignal.asReadonly();
  public readonly temporalFilterFrom = this.temporalFilterFromSignal.asReadonly();
  public readonly temporalFilterTo = this.temporalFilterToSignal.asReadonly();
  public readonly spatialFilterMinLat = this.spatialFilterMinLatSignal.asReadonly();
  public readonly spatialFilterMaxLat = this.spatialFilterMaxLatSignal.asReadonly();
  public readonly spatialFilterMinLon = this.spatialFilterMinLonSignal.asReadonly();
  public readonly spatialFilterMaxLon = this.spatialFilterMaxLonSignal.asReadonly();

  public readonly hasTemporalFilter = computed(
    () => this.temporalFilterFromSignal().trim().length > 0 || this.temporalFilterToSignal().trim().length > 0
  );

  public readonly hasSpatialFilter = computed(
    () =>
      this.spatialFilterMinLatSignal().trim().length > 0 ||
      this.spatialFilterMaxLatSignal().trim().length > 0 ||
      this.spatialFilterMinLonSignal().trim().length > 0 ||
      this.spatialFilterMaxLonSignal().trim().length > 0
  );

  public setSelectedFilter(filter: SearchFilter): void {
    this.selectedFilterSignal.set(filter);
  }

  public setTemporalFilterFrom(value: string): void {
    this.temporalFilterFromSignal.set(value);
  }

  public setTemporalFilterTo(value: string): void {
    this.temporalFilterToSignal.set(value);
  }

  public clearTemporalFilter(): void {
    this.temporalFilterFromSignal.set('');
    this.temporalFilterToSignal.set('');
  }

  public setSpatialFilterMinLat(value: string): void {
    this.spatialFilterMinLatSignal.set(value);
  }

  public setSpatialFilterMaxLat(value: string): void {
    this.spatialFilterMaxLatSignal.set(value);
  }

  public setSpatialFilterMinLon(value: string): void {
    this.spatialFilterMinLonSignal.set(value);
  }

  public setSpatialFilterMaxLon(value: string): void {
    this.spatialFilterMaxLonSignal.set(value);
  }

  public clearSpatialFilter(): void {
    this.spatialFilterMinLatSignal.set('');
    this.spatialFilterMaxLatSignal.set('');
    this.spatialFilterMinLonSignal.set('');
    this.spatialFilterMaxLonSignal.set('');
  }

  public clearAllFilters(): void {
    this.selectedFilterSignal.set('dataset');
    this.clearTemporalFilter();
    this.clearSpatialFilter();
  }
}
