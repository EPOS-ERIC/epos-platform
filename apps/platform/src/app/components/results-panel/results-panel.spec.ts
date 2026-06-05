import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';

import { PlatformApiService } from '../../api/services/platform-api.service';
import { SearchResponse } from '../../api/types/api-types';
import { ResultsPanel } from './results-panel';

const emptySearchResponse: SearchResponse = {
  page_number: 1,
  page_size: 0,
  total: 0,
  total_pages: 0,
  items: [],
};

describe('ResultsPanel', () => {
  let component: ResultsPanel;
  let fixture: ComponentFixture<ResultsPanel>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ResultsPanel],
      providers: [
        {
          provide: PlatformApiService,
          useValue: {
            search: () => of(emptySearchResponse),
          } satisfies Pick<PlatformApiService, 'search'>,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ResultsPanel);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
