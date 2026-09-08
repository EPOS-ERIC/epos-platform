import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { ParameterDraftMetadata, ParameterKind } from '../../../../../generated/backofficeSchemas';
import { AppIconComponent } from '../../../app-icon/app-icon.component';
import type { DatasetWebServiceParameterForm } from '../../main-container.component';

@Component({
  selector: 'app-web-service-parameters',
  imports: [AppIconComponent, FormsModule],
  templateUrl: './web-service-parameters.component.html',
  styleUrl: './web-service-parameters.component.css'
})
export class WebServiceParametersComponent {
  @Input() model: DatasetWebServiceParameterForm[] = [];
  @Input() disabled = false;
  @Output() modelChange = new EventEmitter<DatasetWebServiceParameterForm[]>();

  protected parameterPage = 0;
  protected expandedParameterIndex: number | null = null;
  protected readonly parameterPageSize = 10;
  protected readonly parameterKinds: readonly ParameterKind[] = ['STRING', 'BOOLEAN', 'INTEGER', 'FLOAT', 'DATETIME'];
  protected addParameterDialogOpen = false;
  protected selectedParameterKind: ParameterKind | '' = '';
  protected parameterKindRequired = false;

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected parameterPagerLabel(): string {
    return `Page ${this.parameterPage + 1} of ${this.parameterPageCount()}`;
  }

  protected parameterPageCount(): number {
    return Math.max(1, Math.ceil(this.model.length / this.parameterPageSize));
  }

  protected pagedParameters(): DatasetWebServiceParameterForm[] {
    const start = this.parameterPage * this.parameterPageSize;
    return this.model.slice(start, start + this.parameterPageSize);
  }

  protected parameterGlobalIndex(localIndex: number): number {
    return this.parameterPage * this.parameterPageSize + localIndex;
  }

  protected isParameterExpanded(globalIndex: number): boolean {
    return this.expandedParameterIndex === globalIndex;
  }

  protected toggleParameterRow(globalIndex: number): void {
    this.expandedParameterIndex = this.expandedParameterIndex === globalIndex ? null : globalIndex;
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
    this.parameterPage = Math.floor(parameterIndex / this.parameterPageSize);
    this.expandedParameterIndex = parameterIndex;
    this.modelChange.emit([...this.model, parameter]);
    this.closeAddParameterDialog();
  }

  protected removeParameter(index: number): void {
    if (this.disabled || index < 0 || index >= this.model.length) {
      return;
    }

    const nextModel = this.model.filter((_, parameterIndex) => parameterIndex !== index);
    const lastPage = Math.max(0, Math.ceil(nextModel.length / this.parameterPageSize) - 1);
    this.parameterPage = Math.min(this.parameterPage, lastPage);

    if (this.expandedParameterIndex === index) {
      this.expandedParameterIndex = null;
    } else if (this.expandedParameterIndex !== null && this.expandedParameterIndex > index) {
      this.expandedParameterIndex -= 1;
    }

    this.modelChange.emit(nextModel);
  }

  protected addAllowedValue(parameter: DatasetWebServiceParameterForm): void {
    if (this.disabled || !('allowedValues' in parameter)) {
      return;
    }

    parameter.allowedValues.push('');
    this.emitModelChange();
  }

  protected removeAllowedValue(parameter: DatasetWebServiceParameterForm, index: number): void {
    if (this.disabled || !('allowedValues' in parameter)) {
      return;
    }

    parameter.allowedValues.splice(index, 1);
    this.emitModelChange();
  }

  protected isDefaultMissing(parameter: DatasetWebServiceParameterForm): boolean {
    if (!parameter.metadata.required) {
      return false;
    }

    return parameter.defaultValue === null
      || (typeof parameter.defaultValue === 'string' && !parameter.defaultValue.trim());
  }

  protected previousParameterPage(): void {
    this.parameterPage = Math.max(0, this.parameterPage - 1);
    this.expandedParameterIndex = null;
  }

  protected nextParameterPage(): void {
    this.parameterPage = Math.min(this.parameterPageCount() - 1, this.parameterPage + 1);
    this.expandedParameterIndex = null;
  }

  protected goToParameterPage(page: number): void {
    this.parameterPage = page;
    this.expandedParameterIndex = null;
  }

  protected parameterPageItems(): Array<number | null> {
    const totalPages = this.parameterPageCount();

    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, index) => index);
    }

    const currentPage = this.parameterPage;
    const pages = new Set<number>([0, totalPages - 1, Math.max(0, currentPage - 1), currentPage, Math.min(totalPages - 1, currentPage + 1)]);
    const sortedPages = Array.from(pages).sort((a, b) => a - b);
    const items: Array<number | null> = [];

    sortedPages.forEach((page, index) => {
      if (index > 0 && page - sortedPages[index - 1] > 1) {
        items.push(null);
      }

      items.push(page);
    });

    return items;
  }

  private isParameterKind(value: string): value is ParameterKind {
    return this.parameterKinds.some((kind) => kind === value);
  }

  private createParameter(kind: ParameterKind): DatasetWebServiceParameterForm {
    const draftMetadata: ParameterDraftMetadata = {
      kind,
      variable: '',
      property: '',
      label: '',
      required: false,
      multiple_values: false,
      read_only: false
    };
    const metadata = {
      id: '',
      variable: draftMetadata.variable,
      property: draftMetadata.property,
      label: draftMetadata.label,
      required: draftMetadata.required,
      multiple_values: draftMetadata.multiple_values,
      read_only: draftMetadata.read_only
    };

    switch (kind) {
      case 'STRING':
        return { kind, metadata, defaultValue: null, allowedValues: [], pattern: '' };
      case 'BOOLEAN':
        return { kind, metadata, defaultValue: null };
      case 'INTEGER':
      case 'FLOAT':
        return { kind, metadata, minimum: '', maximum: '', defaultValue: '', allowedValues: [] };
      case 'DATETIME':
        return { kind, metadata, minimum: '', maximum: '', defaultValue: '', pattern: '' };
    }
  }
}
