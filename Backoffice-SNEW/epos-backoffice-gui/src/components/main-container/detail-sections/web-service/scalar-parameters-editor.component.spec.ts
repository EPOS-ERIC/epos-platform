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
        defaultValue: null,
        allowedValues: [],
        pattern: ''
      }
    ]);
  });
});
