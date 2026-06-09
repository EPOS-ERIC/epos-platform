import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DataproviderFilter } from './dataprovider-filter';

describe('DataproviderFilter', () => {
  let component: DataproviderFilter;
  let fixture: ComponentFixture<DataproviderFilter>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DataproviderFilter],
    }).compileComponents();

    fixture = TestBed.createComponent(DataproviderFilter);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
