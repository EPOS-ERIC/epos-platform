import {
  AfterViewInit,
  Component,
  ElementRef,
  Input,
  OnChanges,
  OnDestroy,
  SimpleChanges,
  ViewChild,
} from '@angular/core';
import { Map } from 'maplibre-gl';

type MapViewMode = '2d' | 'globe';

const MAP_STYLE_BY_MODE: Record<MapViewMode, string> = {
  '2d': 'https://demotiles.maplibre.org/style.json',
  globe: 'https://demotiles.maplibre.org/globe.json',
};

@Component({
  selector: 'app-map-view',
  imports: [],
  templateUrl: './map-view.html',
  styleUrl: './map-view.css',
})
export class MapView implements AfterViewInit, OnChanges, OnDestroy {
  @Input() public viewMode: MapViewMode = '2d';

  @ViewChild('mapContainer', { static: true })
  public mapContainer!: ElementRef<HTMLDivElement>;

  private map: Map | null = null;

  public ngAfterViewInit(): void {
    this.map = new Map({
      container: this.mapContainer.nativeElement,
      style: this.resolveStyle(this.viewMode),
      center: [12, 42],
      zoom: 2,
      /* renderWorldCopies: false, */ // for 'no-repeat' of the map
    });
  }

  public ngOnChanges(changes: SimpleChanges): void {
    if (!changes['viewMode'] || !this.map) {
      return;
    }

    this.map.setStyle(this.resolveStyle(this.viewMode));
  }

  public ngOnDestroy(): void {
    this.map?.remove();
    this.map = null;
  }

  private resolveStyle(mode: MapViewMode): string {
    return MAP_STYLE_BY_MODE[mode];
  }
}
