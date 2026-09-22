import { Component, EventEmitter, Input, Output, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { AppIconComponent } from '../../../app-icon/app-icon.component';
import type { DatasetEnumOptionForm, DatasetWebServiceParameterForm } from '../../../../models/dataset-detail-form';
import { ToastService } from '../../../../services/toast.service';
import { AllowedValuesEditorComponent } from './allowed-values-editor.component';
import { ScalarParametersEditorComponent } from './scalar-parameters-editor.component';

type ParameterFormKind = DatasetWebServiceParameterForm['kind'];
type ParameterComparisonStatus = 'unchanged' | 'changed' | 'added' | 'removed';

interface ParameterListItem {
  parameter: DatasetWebServiceParameterForm;
  globalIndex: number;
}

@Component({
  selector: 'app-web-service-parameters',
  imports: [AllowedValuesEditorComponent, AppIconComponent, FormsModule, ScalarParametersEditorComponent],
  templateUrl: './web-service-parameters.component.html',
  styleUrl: './web-service-parameters.component.css'
})
export class WebServiceParametersComponent {
  private readonly toast = inject(ToastService);

  @Input() model: DatasetWebServiceParameterForm[] = [];
  @Input() comparisonModel: DatasetWebServiceParameterForm[] = [];
  @Input() disabled = false;
  @Input() comparisonMode = false;
  @Input() comparisonSide: 'left' | 'right' = 'left';
  @Input() reversed = false;
  @Output() modelChange = new EventEmitter<DatasetWebServiceParameterForm[]>();

  protected parameterPage = 0;
  protected selectedParameterIndex = 0;
  protected parameterSearchQuery = '';
  protected readonly parameterPageSize = 10;
  protected readonly parameterKinds: readonly ParameterFormKind[] = ['STRING', 'BOOLEAN', 'INTEGER', 'FLOAT', 'DATETIME', 'TEMPLATE'];
  protected readonly parameterLocations = ['PATH', 'QUERY', 'HEADER'] as const;
  protected addParameterDialogOpen = false;
  protected selectedParameterKind: ParameterFormKind | '' = '';
  protected parameterKindRequired = false;
  protected parameterListCollapsed = false;

  protected toggleParameterList(): void {
    this.parameterListCollapsed = !this.parameterListCollapsed;
  }

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected updateParameterName(
    parameter: DatasetWebServiceParameterForm,
    name: string,
    input: HTMLInputElement
  ): void {
    const normalizedName = name.trim().toLocaleLowerCase();
    const duplicate = normalizedName && this.model.some(
      (candidate) => candidate !== parameter && candidate.name.trim().toLocaleLowerCase() === normalizedName
    );
    if (duplicate) {
      this.toast.error(`A parameter named "${name.trim()}" already exists.`);
      input.value = parameter.name;
      return;
    }

    parameter.name = name;
    this.emitModelChange();
  }

  protected parameterPagerLabel(): string {
    return `Page ${this.parameterPage + 1} of ${this.parameterPageCount()}`;
  }

  protected parameterPageCount(): number {
    return Math.max(1, Math.ceil(this.filteredParameterCount() / this.parameterPageSize));
  }

  protected filteredParameterCount(): number {
    return this.filteredParameterEntries().length;
  }

  protected pagedParameters(): ParameterListItem[] {
    const start = this.parameterPage * this.parameterPageSize;
    return this.filteredParameterEntries().slice(start, start + this.parameterPageSize);
  }

  protected selectedParameter(): DatasetWebServiceParameterForm | null {
    return this.model[this.selectedParameterGlobalIndex()] ?? null;
  }

  protected selectedParameterGlobalIndex(): number {
    const entries = this.filteredParameterEntries();
    return entries.some(({ globalIndex }) => globalIndex === this.selectedParameterIndex)
      ? this.selectedParameterIndex
      : (entries[0]?.globalIndex ?? -1);
  }

  protected isParameterSelected(globalIndex: number): boolean {
    return this.selectedParameterGlobalIndex() === globalIndex;
  }

  protected parameterComparisonKey(parameter: DatasetWebServiceParameterForm): string {
    const name = parameter.name.trim().toLocaleLowerCase();
    if (name) {
      return `name:${name}`;
    }

    const property = parameter.property.trim().toLocaleLowerCase();
    if (property) {
      return `property:${property}`;
    }

    return parameter.id.trim() ? `id:${parameter.id}` : `position:${parameter.position}`;
  }

  protected parameterComparisonStatus(parameter: DatasetWebServiceParameterForm): ParameterComparisonStatus {
    if (!this.comparisonMode) {
      return 'unchanged';
    }

    const counterpart = this.comparisonModel.find(
      (candidate) => this.parameterComparisonKey(candidate) === this.parameterComparisonKey(parameter)
    );
    if (!counterpart) {
      return this.comparisonSide === 'right' ? 'added' : 'removed';
    }

    return this.parameterComparableValue(parameter) === this.parameterComparableValue(counterpart)
      ? 'unchanged'
      : 'changed';
  }

  protected parameterStatusLabel(parameter: DatasetWebServiceParameterForm): string {
    const status = this.parameterComparisonStatus(parameter);
    return status === 'unchanged' ? '' : status.charAt(0).toUpperCase() + status.slice(1);
  }

  protected parameterHasPositiveDiff(parameter: DatasetWebServiceParameterForm): boolean {
    const status = this.parameterComparisonStatus(parameter);
    return status === 'added' || (status === 'changed' && this.comparisonSide === 'right');
  }

  protected parameterHasNegativeDiff(parameter: DatasetWebServiceParameterForm): boolean {
    const status = this.parameterComparisonStatus(parameter);
    return status === 'removed' || (status === 'changed' && this.comparisonSide === 'left');
  }

  protected selectParameter(globalIndex: number): void {
    this.selectedParameterIndex = globalIndex;
  }

  protected onParameterSearch(query: string): void {
    this.parameterSearchQuery = query;
    this.parameterPage = 0;
    this.selectedParameterIndex = this.filteredParameterEntries()[0]?.globalIndex ?? 0;
  }

  protected openAddParameterDialog(): void {
    if (this.disabled) {
      return;
    }

    this.selectedParameterKind = '';
    this.parameterKindRequired = false;
    this.addParameterDialogOpen = true;
  }

  protected closeAddParameterDialog(): void {
    this.addParameterDialogOpen = false;
    this.parameterKindRequired = false;
  }

  protected confirmAddParameter(): void {
    if (this.disabled || !this.isParameterKind(this.selectedParameterKind)) {
      this.parameterKindRequired = true;
      return;
    }

    const parameter = this.createParameter(this.selectedParameterKind);
    const parameterIndex = this.model.length;
    this.parameterSearchQuery = '';
    this.parameterPage = Math.floor(parameterIndex / this.parameterPageSize);
    this.selectedParameterIndex = parameterIndex;
    this.modelChange.emit([...this.model, parameter]);
    this.closeAddParameterDialog();
  }

  protected removeParameter(index: number): void {
    if (this.disabled || index < 0 || index >= this.model.length) {
      return;
    }

    const nextModel = this.model.filter((_, parameterIndex) => parameterIndex !== index);
    const filteredEntries = this.filteredParameterEntries(nextModel);
    const lastPage = Math.max(0, Math.ceil(filteredEntries.length / this.parameterPageSize) - 1);
    this.parameterPage = Math.min(this.parameterPage, lastPage);

    if (this.selectedParameterIndex === index) {
      this.selectedParameterIndex = Math.min(index, nextModel.length - 1);
    } else if (this.selectedParameterIndex > index) {
      this.selectedParameterIndex -= 1;
    }

    this.selectedParameterIndex = Math.max(0, this.selectedParameterIndex);
    if (!filteredEntries.some(({ globalIndex }) => globalIndex === this.selectedParameterIndex)) {
      this.selectedParameterIndex = filteredEntries[this.parameterPage * this.parameterPageSize]?.globalIndex ?? 0;
    }

    this.modelChange.emit(nextModel);
  }

  protected updateAllowedValues(parameter: DatasetWebServiceParameterForm, values: DatasetEnumOptionForm[]): void {
    if (!('allowedValues' in parameter)) {
      return;
    }

    parameter.allowedValues = values;
    this.emitModelChange();
  }

  protected onMultipleValuesChange(
    parameter: Extract<DatasetWebServiceParameterForm, { kind: 'STRING' | 'INTEGER' | 'FLOAT' }>,
  ): void {
    if (!parameter.multipleValues) parameter.explode = false;
    this.emitModelChange();
  }

  protected onLocationChange(parameter: DatasetWebServiceParameterForm): void {
    if (parameter.location !== 'QUERY') parameter.explode = false;
    this.emitModelChange();
  }

  protected previousParameterPage(): void {
    this.parameterPage = Math.max(0, this.parameterPage - 1);
    this.selectFirstParameterOnPage();
  }

  protected nextParameterPage(): void {
    this.parameterPage = Math.min(this.parameterPageCount() - 1, this.parameterPage + 1);
    this.selectFirstParameterOnPage();
  }

  private selectFirstParameterOnPage(): void {
    this.selectedParameterIndex = this.pagedParameters()[0]?.globalIndex ?? 0;
  }

  private filteredParameterEntries(model = this.model): ParameterListItem[] {
    const query = this.parameterSearchQuery.trim().toLocaleLowerCase();

    return model
      .map((parameter, globalIndex) => ({ parameter, globalIndex }))
      .filter(({ parameter }) => !query || [
        parameter.kind,
        parameter.label,
        parameter.name,
        parameter.property
      ].some((value) => value.toLocaleLowerCase().includes(query)));
  }

  private isParameterKind(value: string): value is ParameterFormKind {
    return this.parameterKinds.some((kind) => kind === value);
  }

  private parameterComparableValue(parameter: DatasetWebServiceParameterForm): string {
    const { id: _id, position: _position, ...visibleValue } = parameter;
    return JSON.stringify(visibleValue);
  }

  private createParameter(kind: ParameterFormKind): DatasetWebServiceParameterForm {
    const common = {
      id: '',
      position: this.model.length,
      name: '',
      label: '',
      description: '',
      property: '',
      required: false,
      location: 'QUERY' as const,
      explode: false,
      readOnly: false
    };

    switch (kind) {
      case 'STRING':
        return { kind, ...common, multipleValues: false, defaultValue: null, allowedValues: [], pattern: '' };
      case 'BOOLEAN':
        return { kind, ...common, defaultValue: null };
      case 'INTEGER':
      case 'FLOAT':
        return { kind, ...common, multipleValues: false, minimum: '', maximum: '', defaultValue: '', allowedValues: [] };
      case 'DATETIME':
        return { kind, ...common, minimum: '', maximum: '', defaultValue: '', layout: '' };
      case 'TEMPLATE':
        return { kind, ...common, template: '', parameters: [] };
    }
  }
}
