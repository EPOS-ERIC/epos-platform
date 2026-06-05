import { Component, Input } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';

import { SearchResponse } from './api/types/api-types';
import { PlatformApiService } from './api/services/platform-api.service';
import { App } from './app';
import { MapView } from './components/map-view/map-view';

@Component({
  selector: 'app-map-view',
  template: '',
})
class StubMapView {
  @Input() public viewMode: '2d' | 'globe' = '2d';
}

const emptySearchResponse: SearchResponse = {
  page_number: 1,
  page_size: 0,
  total: 0,
  total_pages: 0,
  items: [],
};

describe('App', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [
        {
          provide: PlatformApiService,
          useValue: {
            search: () => of(emptySearchResponse),
          } satisfies Pick<PlatformApiService, 'search'>,
        },
      ],
    })
      .overrideComponent(App, {
        remove: { imports: [MapView] },
        add: { imports: [StubMapView] },
      })
      .compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });

  it('should render the main shell', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('app-header')).toBeTruthy();
  });
});
