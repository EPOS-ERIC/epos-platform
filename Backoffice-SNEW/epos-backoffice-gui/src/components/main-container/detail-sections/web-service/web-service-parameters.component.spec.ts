import { TestBed } from '@angular/core/testing';

import type { DatasetWebServiceParameterForm } from '../../../../models/dataset-detail-form';
import { WebServiceParametersComponent } from './web-service-parameters.component';

describe('WebServiceParametersComponent', () => {
  it('keeps the parameter list navigable when the detail fields are read only', async () => {
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(WebServiceParametersComponent);
    const parameters: DatasetWebServiceParameterForm[] = [
      {
        kind: 'STRING',
        metadata: {
          id: 'first-id',
          label: 'First parameter',
          variable: 'first',
          property: 'firstProperty',
          required: false,
          multiple_values: false,
          read_only: false
        },
        defaultValue: null,
        allowedValues: [],
        pattern: ''
      },
      {
        kind: 'BOOLEAN',
        metadata: {
          id: 'second-id',
          label: 'Second parameter',
          variable: 'second',
          property: 'secondProperty',
          required: false,
          multiple_values: false,
          read_only: false
        },
        defaultValue: null
      }
    ];

    fixture.componentRef.setInput('model', parameters);
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    const listItems = fixture.nativeElement.querySelectorAll('.parameter-list-item') as NodeListOf<HTMLButtonElement>;
    expect(listItems.length).toBe(2);
    expect(listItems[0]?.disabled).toBe(false);
    expect(fixture.nativeElement.querySelector('.parameter-editor h3')?.textContent?.trim()).toBe('First parameter');

    listItems[1]?.click();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.parameter-editor h3')?.textContent?.trim()).toBe('Second parameter');
    expect(fixture.nativeElement.querySelector('.parameter-editor-fields')?.disabled).toBe(true);

    const searchInput = fixture.nativeElement.querySelector('.parameter-search-input') as HTMLInputElement;
    searchInput.value = 'firstProperty';
    searchInput.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('.parameter-list-item').length).toBe(1);
    expect(fixture.nativeElement.querySelector('.parameter-editor h3')?.textContent?.trim()).toBe('First parameter');
    expect(fixture.nativeElement.querySelector('.parameter-list-count')?.textContent?.trim()).toBe('1 / 2');
  });
});
