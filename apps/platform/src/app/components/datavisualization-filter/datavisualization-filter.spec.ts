import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DatavisualizationFilter } from './datavisualization-filter';

describe('DatavisualizationFilter', () => {
  let component: DatavisualizationFilter;
  let fixture: ComponentFixture<DatavisualizationFilter>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DatavisualizationFilter],
    }).compileComponents();

    fixture = TestBed.createComponent(DatavisualizationFilter);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
