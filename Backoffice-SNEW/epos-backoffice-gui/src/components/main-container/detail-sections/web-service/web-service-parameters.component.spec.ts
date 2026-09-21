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
        id: 'first-id',
        position: 0,
        label: 'First parameter',
        name: 'first',
        description: '',
        property: 'firstProperty',
        location: 'QUERY',
        explode: false,
        required: false,
        multipleValues: false,
        readOnly: false,
        defaultValue: null,
        allowedValues: [],
        pattern: ''
      },
      {
        kind: 'BOOLEAN',
        id: 'second-id',
        position: 1,
        label: 'Second parameter',
        name: 'second',
        description: '',
        property: 'secondProperty',
        location: 'QUERY',
        explode: false,
        required: false,
        readOnly: false,
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

  it('marks the workspace for the vertical comparison layout', async () => {
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(WebServiceParametersComponent);

    fixture.componentRef.setInput('comparisonMode', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.webservice-parameters--comparison')).not.toBeNull();
  });

  it('only enables explode for supported multiple-value query parameters', async () => {
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(WebServiceParametersComponent);
    const parameter: DatasetWebServiceParameterForm = {
      kind: 'STRING',
      id: '',
      position: 0,
      label: 'Value',
      name: 'value',
      description: '',
      property: '',
      location: 'QUERY',
      explode: false,
      required: false,
      multipleValues: false,
      readOnly: false,
      defaultValue: null,
      allowedValues: [],
      pattern: ''
    };

    fixture.componentRef.setInput('model', [parameter]);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const explodeField = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.detail-field'))
      .find((field) => field.querySelector('.detail-label')?.textContent?.trim() === 'Explode');
    expect((explodeField?.querySelector('input') as HTMLInputElement).disabled).toBe(true);

    const multipleValuesField = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.detail-field'))
      .find((field) => field.querySelector('.detail-label')?.textContent?.trim() === 'Multiple values');
    (multipleValuesField?.querySelector('input') as HTMLInputElement).click();
    await fixture.whenStable();
    fixture.detectChanges();
    const enabledExplodeField = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.detail-field'))
      .find((field) => field.querySelector('.detail-label')?.textContent?.trim() === 'Explode');
    expect((enabledExplodeField?.querySelector('input') as HTMLInputElement).disabled).toBe(false);

    const templateParameter: DatasetWebServiceParameterForm = {
      kind: 'TEMPLATE',
      id: '',
      position: 0,
      label: 'Template',
      name: 'template',
      description: '',
      property: '',
      location: 'QUERY',
      explode: false,
      required: false,
      readOnly: false,
      template: '{}',
      parameters: []
    };
    fixture.componentRef.setInput('model', [templateParameter]);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Template parameters do not support Multiple values or Explode.');
    expect(Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.detail-label'))
      .some((label) => label.textContent?.trim() === 'Explode')).toBe(false);
  });
});
