import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { DatasetWebServiceScalarParameterForm } from '../../../../models/dataset-detail-form';
import { AllowedValuesEditorComponent } from './allowed-values-editor.component';

type ScalarParameterKind = DatasetWebServiceScalarParameterForm['kind'];

// Component for scalarParameter type 
// (= parameters used inside of the Template parameters; do not have 'location', 'explode' and 'multiple_values' (but they have enum for options) properties)
@Component({
  selector: 'app-scalar-parameters-editor',
  imports: [AllowedValuesEditorComponent, FormsModule],
  templateUrl: './scalar-parameters-editor.component.html',
  styleUrl: './scalar-parameters-editor.component.css'
})
export class ScalarParametersEditorComponent {
  @Input() model: DatasetWebServiceScalarParameterForm[] = [];
  @Input() disabled = false;
  @Output() modelChange = new EventEmitter<DatasetWebServiceScalarParameterForm[]>();

  protected readonly kinds: readonly ScalarParameterKind[] = ['STRING', 'BOOLEAN', 'INTEGER', 'FLOAT', 'DATETIME'];
  protected selectedKind: ScalarParameterKind = 'STRING';

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected addParameter(): void {
    if (!this.disabled) {
      this.modelChange.emit([...this.model, this.createParameter(this.selectedKind)]);
    }
  }

  protected removeParameter(index: number): void {
    if (!this.disabled) {
      this.modelChange.emit(this.model.filter((_, currentIndex) => currentIndex !== index));
    }
  }

  protected updateAllowedValues(parameter: DatasetWebServiceScalarParameterForm, values: string[]): void {
    if ('allowedValues' in parameter) {
      parameter.allowedValues = values;
      this.emitModelChange();
    }
  }

  private createParameter(kind: ScalarParameterKind): DatasetWebServiceScalarParameterForm {
    const common = {
      position: this.model.length,
      name: '',
      label: '',
      description: '',
      required: false,
      readOnly: false
    };

    switch (kind) {
      case 'STRING':
        return { kind, ...common, defaultValue: null, allowedValues: [], pattern: '' };
      case 'BOOLEAN':
        return { kind, ...common, defaultValue: null };
      case 'INTEGER':
      case 'FLOAT':
        return { kind, ...common, minimum: '', maximum: '', defaultValue: '', allowedValues: [] };
      case 'DATETIME':
        return { kind, ...common, minimum: '', maximum: '', defaultValue: '', layout: '' };
    }
  }
}
