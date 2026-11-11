import { Component, EventEmitter, Input, OnChanges, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { AppIconComponent } from '../../../../shared/ui/app-icon.component';
import type { DatasetEnumOptionForm, DatasetWebServiceParameterForm } from '../../../../models/dataset-detail-form';
import { enumLabel } from '../../../../shared/enum-label';
import { AllowedValuesEditorComponent } from './allowed-values-editor.component';
import { DefaultValuesEditorComponent } from './default-values-editor.component';
import { ScalarParametersEditorComponent } from './scalar-parameters-editor.component';

type ParameterFormKind = DatasetWebServiceParameterForm['kind'];
type ParameterComparisonStatus = 'unchanged' | 'changed' | 'added' | 'removed';

interface ParameterListItem {
  parameter: DatasetWebServiceParameterForm;
  globalIndex: number;
}

@Component({
  selector: 'app-web-service-parameters',
  imports: [AllowedValuesEditorComponent, AppIconComponent, DefaultValuesEditorComponent, FormsModule, ScalarParametersEditorComponent],
  templateUrl: './web-service-parameters.component.html',
  styleUrl: './web-service-parameters.component.css'
})
export class WebServiceParametersComponent implements OnChanges {
  protected readonly enumLabel = enumLabel;
  @Input() model: DatasetWebServiceParameterForm[] = [];
  @Input() comparisonModel: DatasetWebServiceParameterForm[] = [];
  @Input() sectionTitle = 'Parameters';
  @Input() disabled = false;
  @Input() comparisonMode = false;
  @Input() comparisonSide: 'left' | 'right' = 'left';
  @Input() comparisonSelectedParameterResourceId: string | null = null;
  @Input() reversed = false;
  @Input() dynamic = false;
  @Output() modelChange = new EventEmitter<DatasetWebServiceParameterForm[]>();
  @Output() comparisonSelectedParameterResourceIdChange = new EventEmitter<string>();

  protected parameterPage = 0;
  protected selectedParameterIndex = 0;
  protected parameterSearchQuery = '';
  protected readonly parameterPageSize = 10;
  protected readonly parameterKinds: readonly ParameterFormKind[] = ['STRING', 'BOOLEAN', 'INTEGER', 'FLOAT', 'DATETIME', 'TEMPLATE'];
  protected readonly parameterLocations = ['PATH', 'QUERY', 'HEADER'] as const;
  protected readonly standardParameterRoles = ['', 'FORMAT', 'START_DATE', 'END_DATE', 'MIN_LON', 'MAX_LON', 'MIN_LAT', 'MAX_LAT'] as const;
  protected addParameterDialogOpen = false;
  protected selectedParameterKind: ParameterFormKind | '' = '';
  protected parameterKindRequired = false;
  protected parameterListCollapsed = false;
  protected draggedParameterIndex: number | null = null;
  protected dragOverParameterIndex: number | null = null;

  ngOnChanges(): void {
    if (this.comparisonSelectedParameterResourceId) {
      const selectedIndex = this.model.findIndex(
        (parameter) => parameter.resourceId === this.comparisonSelectedParameterResourceId
      );
      if (selectedIndex >= 0) {
        this.parameterSearchQuery = '';
        this.selectedParameterIndex = selectedIndex;
        this.parameterPage = Math.floor(
          this.filteredParameterEntries().findIndex((entry) => entry.globalIndex === selectedIndex) /
          this.parameterPageSize
        );
      }
      return;
    }

    if (this.comparisonMode && this.comparisonSide === 'right') {
      this.selectedParameterIndex = this.filteredParameterEntries()[0]?.globalIndex ?? 0;
    }
  }

  protected toggleParameterList(): void {
    this.parameterListCollapsed = !this.parameterListCollapsed;
  }

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected parameterRoles(): readonly DatasetWebServiceParameterForm['role'][] {
    return this.dynamic
      ? [...this.standardParameterRoles, 'LONGITUDE', 'LATITUDE']
      : this.standardParameterRoles;
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
    if (parameter.resourceId?.trim()) {
      return `resource:${parameter.resourceId}`;
    }

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
    const resourceId = this.model[globalIndex]?.resourceId;
    if (this.comparisonMode && resourceId) {
      this.comparisonSelectedParameterResourceIdChange.emit(resourceId);
    }
  }

  protected moveParameterToPosition(
    parameter: DatasetWebServiceParameterForm,
    displayedPosition: number | null
  ): void {
    if (displayedPosition === null || !Number.isFinite(displayedPosition)) {
      return;
    }

    const sourceIndex = this.model.indexOf(parameter);
    if (sourceIndex < 0) {
      return;
    }

    const targetIndex = Math.min(
      this.model.length - 1,
      Math.max(0, Math.trunc(displayedPosition) - 1)
    );
    const reordered = [...this.model];
    const [movedParameter] = reordered.splice(sourceIndex, 1);
    reordered.splice(targetIndex, 0, movedParameter);

    this.selectedParameterIndex = targetIndex;
    this.parameterPage = Math.floor(targetIndex / this.parameterPageSize);
    this.modelChange.emit(reordered.map((item, position) => ({ ...item, position })));
  }

  protected startParameterDrag(event: DragEvent, globalIndex: number): void {
    if (this.disabled || this.comparisonMode) {
      event.preventDefault();
      return;
    }

    this.draggedParameterIndex = globalIndex;
    this.dragOverParameterIndex = globalIndex;
    event.dataTransfer?.setData('text/plain', String(globalIndex));
    if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
  }

  protected dragParameterOver(event: DragEvent, globalIndex: number): void {
    if (this.draggedParameterIndex === null || this.disabled || this.comparisonMode) return;

    event.preventDefault();
    this.dragOverParameterIndex = globalIndex;
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
  }

  protected dropParameter(event: DragEvent, targetIndex: number): void {
    event.preventDefault();
    const sourceIndex = this.draggedParameterIndex;
    if (sourceIndex === null || sourceIndex === targetIndex || this.disabled || this.comparisonMode) {
      this.endParameterDrag();
      return;
    }

    const selectedParameter = this.selectedParameter();
    const reordered = [...this.model];
    const [movedParameter] = reordered.splice(sourceIndex, 1);
    reordered.splice(targetIndex, 0, movedParameter);
    const nextModel = reordered.map((parameter, position) => ({ ...parameter, position }));

    if (selectedParameter) {
      this.selectedParameterIndex = reordered.indexOf(selectedParameter);
    }
    this.modelChange.emit(nextModel);
    this.endParameterDrag();
  }

  protected endParameterDrag(): void {
    this.draggedParameterIndex = null;
    this.dragOverParameterIndex = null;
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
    if (!parameter.multipleValues) {
      parameter.explode = false;
      parameter.defaultValue = parameter.defaultValue.slice(0, 1);
    }
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
    const entries = model.map((parameter, globalIndex) => ({ parameter, globalIndex }));
    if (this.comparisonMode && this.comparisonSide === 'right') {
      const comparisonOrder = new Map(
        this.comparisonModel.map((parameter, index) => [this.parameterComparisonKey(parameter), index])
      );
      entries.sort((left, right) => {
        const leftIndex = comparisonOrder.get(this.parameterComparisonKey(left.parameter)) ?? Number.MAX_SAFE_INTEGER;
        const rightIndex = comparisonOrder.get(this.parameterComparisonKey(right.parameter)) ?? Number.MAX_SAFE_INTEGER;
        return leftIndex - rightIndex || left.globalIndex - right.globalIndex;
      });
    }

    return entries
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
    const { id: _id, resourceId: _resourceId, position: _position, ...visibleValue } = parameter;
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
      role: '' as const,
      explode: false,
      readOnly: false
    };

    switch (kind) {
      case 'STRING':
        return { kind, ...common, multipleValues: false, defaultValue: [], allowedValues: [], pattern: '' };
      case 'BOOLEAN':
        return { kind, ...common, defaultValue: null };
      case 'INTEGER':
      case 'FLOAT':
        return { kind, ...common, multipleValues: false, minimum: '', maximum: '', defaultValue: [], allowedValues: [] };
      case 'DATETIME':
        return { kind, ...common, minimum: '', maximum: '', defaultValue: '', layout: '' };
      case 'TEMPLATE':
        return { kind, ...common, template: '', parameters: [] };
    }
  }

}
