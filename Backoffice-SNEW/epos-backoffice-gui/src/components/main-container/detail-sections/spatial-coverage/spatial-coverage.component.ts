import { AfterViewInit, Component, ElementRef, EventEmitter, Input, OnChanges, OnDestroy, Output, SimpleChanges, ViewChild, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { wktToGeoJSON, type WktGeoJsonGeometry } from '@terraformer/wkt';
import * as L from 'leaflet';

import type { DatasetSpatialForm } from '../../main-container.component';

@Component({
  selector: 'app-spatial-coverage',
  imports: [FormsModule],
  templateUrl: './spatial-coverage.component.html',
  styleUrl: './spatial-coverage.component.css'
})
export class SpatialCoverageComponent implements AfterViewInit, OnChanges, OnDestroy {
  @Input() model: DatasetSpatialForm = null;
  @Input() disabled = false;
  @Output() modelChange = new EventEmitter<DatasetSpatialForm>();

  @ViewChild('spatialMapContainer') private spatialMapContainerRef?: ElementRef<HTMLDivElement>;

  protected readonly validationError = signal<string | null>(null);

  private spatialMap?: L.Map;
  private spatialOverlayLayer?: L.FeatureGroup;
  private viewReady = false;
  private renderFrameId: number | null = null;
  private readonly pointIcon = L.divIcon({
    className: 'spatial-pin-marker',
    iconSize: [28, 40],
    iconAnchor: [14, 40],
    html: `
      <svg width="28" height="40" viewBox="0 0 28 40" aria-hidden="true" style="display:block;filter:drop-shadow(0 3px 3px rgba(15,23,42,.3))">
        <path d="M14 1C6.82 1 1 6.82 1 14c0 9.83 13 25 13 25s13-15.17 13-25C27 6.82 21.18 1 14 1Z" fill="#0f766e" stroke="#ffffff" stroke-width="2"/>
        <circle cx="14" cy="14" r="5" fill="#ffffff"/>
      </svg>
    `
  });

  ngAfterViewInit(): void {
    this.viewReady = true;
    this.scheduleRenderSpatialMap();
  }

  ngOnChanges(_changes: SimpleChanges): void {
    this.scheduleRenderSpatialMap();
  }

  ngOnDestroy(): void {
    if (this.renderFrameId !== null) {
      cancelAnimationFrame(this.renderFrameId);
    }

    this.spatialMap?.remove();
  }

  protected onWktChange(value: string): void {
    this.model = value.trim() ? value : null;
    this.modelChange.emit(this.model);
    this.scheduleRenderSpatialMap();
  }

  private scheduleRenderSpatialMap(): void {
    if (this.renderFrameId !== null) {
      cancelAnimationFrame(this.renderFrameId);
    }

    this.renderFrameId = requestAnimationFrame(() => {
      this.renderFrameId = null;
      this.renderSpatialMap();
    });
  }

  private renderSpatialMap(): void {
    const container = this.spatialMapContainerRef?.nativeElement;

    if (!this.viewReady || !container) {
      return;
    }

    if (!this.spatialMap) {
      this.spatialMap = L.map(container, { zoomControl: true, scrollWheelZoom: false }).setView([20, 0], 2);
      L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; OpenStreetMap contributors'
      }).addTo(this.spatialMap);
      this.spatialOverlayLayer = L.featureGroup().addTo(this.spatialMap);
    }

    this.spatialOverlayLayer?.clearLayers();

    if (!this.model?.trim()) {
      this.validationError.set(null);
      this.spatialMap.setView([20, 0], 2);
      return;
    }

    try {
      const geometry = wktToGeoJSON(this.withoutSrid(this.model));

      if (!this.geometryIsValid(geometry)) {
        throw new Error('Coordinate out of range');
      }

      this.validationError.set(null);
      const layer = L.geoJSON(geometry as Parameters<typeof L.geoJSON>[0], {
        style: { color: '#0f766e', weight: 2, fillOpacity: 0.22 },
        pointToLayer: (_feature, latLng) => L.marker(latLng, { icon: this.pointIcon })
      });
      this.spatialOverlayLayer?.addLayer(layer);
      const bounds = layer.getBounds();

      if (bounds.isValid()) {
        this.spatialMap.fitBounds(bounds, { padding: [20, 20], maxZoom: 8 });
      } else {
        this.spatialMap.setView([20, 0], 2);
      }
    } catch {
      this.validationError.set('Enter valid WKT geometry in EPSG:4326.');
    }

    requestAnimationFrame(() => this.spatialMap?.invalidateSize());
  }

  private withoutSrid(value: string): string {
    const normalized = value.trim();
    const sridMatch = normalized.match(/^SRID=(\d+);(.*)$/is);

    if (sridMatch) {
      if (sridMatch[1] !== '4326') {
        throw new Error('Unsupported SRID');
      }

      return sridMatch[2].trim();
    }

    return normalized;
  }

  private geometryIsValid(geometry: WktGeoJsonGeometry): boolean {
    if (geometry.type === 'GeometryCollection') {
      return Array.isArray(geometry.geometries) && geometry.geometries.every((item) => this.geometryIsValid(item));
    }

    return this.coordinatesAreValid(geometry.coordinates);
  }

  private coordinatesAreValid(coordinates: unknown): boolean {
    if (!Array.isArray(coordinates)) {
      return false;
    }

    if (coordinates.length === 0) {
      return true;
    }

    if (coordinates.every((value) => typeof value === 'number')) {
      const [longitude, latitude] = coordinates as number[];
      return Number.isFinite(longitude) && Number.isFinite(latitude) && longitude >= -180 && longitude <= 180 && latitude >= -90 && latitude <= 90;
    }

    return coordinates.every((value) => this.coordinatesAreValid(value));
  }
}
