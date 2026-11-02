import { TestBed } from '@angular/core/testing';

import type { DatasetWebServiceScalarParameterForm } from '../../../../models/dataset-detail-form';
import { ScalarParametersEditorComponent } from './scalar-parameters-editor.component';

describe('ScalarParametersEditorComponent', () => {
  it('creates a valid empty scalar parameter at the next position', async () => {
    await TestBed.configureTestingModule({
      imports: [ScalarParametersEditorComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(ScalarParametersEditorComponent);
    const emissions: DatasetWebServiceScalarParameterForm[][] = [];

    fixture.componentRef.setInput('model', []);
    fixture.componentInstance.modelChange.subscribe((model) => emissions.push(model));
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.scalar-editor-add button') as HTMLButtonElement).click();

    expect(emissions.at(-1)).toEqual([
      {
        kind: 'STRING',
        position: 0,
        name: '',
        label: '',
        description: '',
        required: false,
        readOnly: false,
        role: '',
        defaultValue: null,
        allowedValues: [],
        pattern: ''
      }
    ]);
  });

  it('rejects duplicate scalar parameter names', async () => {
    await TestBed.configureTestingModule({
      imports: [ScalarParametersEditorComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(ScalarParametersEditorComponent);
    const parameters: DatasetWebServiceScalarParameterForm[] = [
      {
        kind: 'BOOLEAN', position: 0, name: 'first', label: 'First', description: '',
        required: false, readOnly: false, defaultValue: null
      },
      {
        kind: 'BOOLEAN', position: 1, name: 'second', label: 'Second', description: '',
        required: false, readOnly: false, defaultValue: null
      }
    ];
    fixture.componentRef.setInput('model', parameters);
    fixture.detectChanges();

    const secondNameInput = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.scalar-item'))[1]
      .querySelector('label input') as HTMLInputElement;
    secondNameInput.value = 'First';
    secondNameInput.dispatchEvent(new Event('input'));
    await fixture.whenStable();

    expect(parameters[1].name).toBe('second');
    expect(secondNameInput.value).toBe('second');
  });
});
