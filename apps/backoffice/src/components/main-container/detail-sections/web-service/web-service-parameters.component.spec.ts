import { TestBed } from '@angular/core/testing';

import type { DatasetWebServiceParameterForm } from '../../../../models/dataset-detail-form';
import { ToastService } from '../../../../services/toast.service';
import { WebServiceParametersComponent } from './web-service-parameters.component';

describe('WebServiceParametersComponent', () => {
  it('keeps the mirrored, collapsible parameter list navigable when detail fields are read only', async () => {
    const toast = { error: vi.fn() };
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent],
      providers: [{ provide: ToastService, useValue: toast }]
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
    fixture.componentRef.setInput('comparisonModel', [{ ...parameters[0], id: 'different-version-id' }]);
    fixture.componentRef.setInput('disabled', true);
    fixture.componentRef.setInput('comparisonMode', true);
    fixture.componentRef.setInput('comparisonSide', 'right');
    fixture.componentRef.setInput('reversed', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.parameter-workspace--reversed')).not.toBeNull();
    const listItems = fixture.nativeElement.querySelectorAll('.parameter-list-item') as NodeListOf<HTMLButtonElement>;
    expect(listItems.length).toBe(2);
    expect(listItems[0].classList.contains('parameter-list-item--positive')).toBe(false);
    expect(listItems[1].classList.contains('parameter-list-item--positive')).toBe(true);
    expect(listItems[1].textContent).toContain('Added');
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

    const toggle = fixture.nativeElement.querySelector('.parameter-list-toggle') as HTMLButtonElement;
    toggle.click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.parameter-workspace--collapsed')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.parameter-search-input')).toBeNull();
    expect(toggle.getAttribute('aria-expanded')).toBe('false');

    toggle.click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.parameter-search-input')).not.toBeNull();

    fixture.componentRef.setInput('disabled', false);
    fixture.detectChanges();
    const reopenedSearch = fixture.nativeElement.querySelector('.parameter-search-input') as HTMLInputElement;
    reopenedSearch.value = '';
    reopenedSearch.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelectorAll('.parameter-list-item')[1] as HTMLButtonElement).click();
    fixture.detectChanges();

    const nameField = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.detail-field'))
      .find((field) => field.querySelector('.detail-label')?.textContent?.trim() === 'Name *');
    const nameInput = nameField?.querySelector('input') as HTMLInputElement;
    nameInput.value = 'FIRST';
    nameInput.dispatchEvent(new Event('input'));
    await fixture.whenStable();

    expect(parameters[1].name).toBe('second');
    expect(nameInput.value).toBe('second');
    expect(toast.error).toHaveBeenCalledWith('A parameter named "FIRST" already exists.');
  });

  it('marks the workspace as comparison content', async () => {
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(WebServiceParametersComponent);

    fixture.componentRef.setInput('comparisonMode', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.webservice-parameters--comparison')).not.toBeNull();
  });

  it('offers longitude and latitude roles for DYNAMIC services', async () => {
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(WebServiceParametersComponent);
    fixture.componentRef.setInput('model', [createStringParameter('Coordinate', 'coordinate', 0)]);
    fixture.componentRef.setInput('dynamic', true);
    fixture.detectChanges();

    const roleField = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.detail-field'))
      .find((field) => field.querySelector('.detail-label')?.textContent?.trim() === 'Semantic role');
    const options = Array.from((roleField?.querySelector('select') as HTMLSelectElement).options)
      .map((option) => option.textContent?.trim());
    expect(options).toContain('LONGITUDE');
    expect(options).toContain('LATITUDE');
  });

  it('reorders parameters from the drag handle and updates every position', async () => {
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(WebServiceParametersComponent);
    const parameters: DatasetWebServiceParameterForm[] = [
      createStringParameter('First parameter', 'first', 7),
      createStringParameter('Second parameter', 'second', 3)
    ];
    const modelChange = vi.fn();
    fixture.componentInstance.modelChange.subscribe(modelChange);

    fixture.componentRef.setInput('model', parameters);
    fixture.detectChanges();

    const handles = fixture.nativeElement.querySelectorAll('.parameter-drag-handle') as NodeListOf<HTMLElement>;
    const items = fixture.nativeElement.querySelectorAll('.parameter-list-item') as NodeListOf<HTMLButtonElement>;
    expect(handles.length).toBe(2);

    handles[0].dispatchEvent(new Event('dragstart', { bubbles: true, cancelable: true }));
    items[1].dispatchEvent(new Event('dragover', { bubbles: true, cancelable: true }));
    items[1].dispatchEvent(new Event('drop', { bubbles: true, cancelable: true }));

    expect(modelChange).toHaveBeenCalledOnce();
    const reordered = modelChange.mock.calls[0][0] as DatasetWebServiceParameterForm[];
    expect(reordered.map((parameter) => parameter.name)).toEqual(['second', 'first']);
    expect(reordered.map((parameter) => parameter.position)).toEqual([0, 1]);
  });

  it('moves a parameter from a later page to the first position', async () => {
    await TestBed.configureTestingModule({
      imports: [WebServiceParametersComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(WebServiceParametersComponent);
    const parameters = Array.from({ length: 12 }, (_, index) =>
      createStringParameter(`Parameter ${index + 1}`, `parameter-${index + 1}`, index)
    );
    let reordered = parameters;
    fixture.componentInstance.modelChange.subscribe((value) => {
      reordered = value;
      fixture.componentRef.setInput('model', value);
    });
    fixture.componentRef.setInput('model', parameters);
    fixture.detectChanges();

    const pagerButtons = fixture.nativeElement.querySelectorAll('.parameter-page-button') as NodeListOf<HTMLButtonElement>;
    pagerButtons[1].click();
    fixture.detectChanges();
    const pageTwoItems = fixture.nativeElement.querySelectorAll('.parameter-list-item') as NodeListOf<HTMLButtonElement>;
    pageTwoItems[1].click();
    fixture.detectChanges();

    const positionField = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.detail-field'))
      .find((field) => field.querySelector('.detail-label')?.textContent?.trim() === 'Position');
    const positionInput = positionField?.querySelector('input') as HTMLInputElement;
    positionInput.value = '1';
    positionInput.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(reordered[0].name).toBe('parameter-12');
    expect(reordered.map((parameter) => parameter.position)).toEqual(parameters.map((_, index) => index));
    expect(fixture.nativeElement.querySelector('.parameter-editor h3')?.textContent?.trim()).toBe('Parameter 12');
    expect(fixture.nativeElement.querySelector('.parameter-list-footer')?.textContent).toContain('Page 1 of 2');
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

function createStringParameter(label: string, name: string, position: number): DatasetWebServiceParameterForm {
  return {
    kind: 'STRING',
    id: `${name}-id`,
    position,
    label,
    name,
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
}
