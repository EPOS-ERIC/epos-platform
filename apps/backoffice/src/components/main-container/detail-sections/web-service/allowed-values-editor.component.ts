import { Component, EventEmitter, Input, Output, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { AppIconComponent } from '../../../../shared/ui/app-icon.component';
import type { DatasetEnumOptionForm } from '../../../../models/dataset-detail-form';
import { ToastService } from '../../../../services/toast.service';

@Component({
  selector: 'app-allowed-values-editor',
  imports: [AppIconComponent, FormsModule],
  templateUrl: './allowed-values-editor.component.html',
  styleUrl: './allowed-values-editor.component.css'
})
export class AllowedValuesEditorComponent {
  private readonly toast = inject(ToastService);

  @Input() values: DatasetEnumOptionForm[] = [];
  @Input() kind: 'STRING' | 'INTEGER' | 'FLOAT' = 'STRING';
  @Input() defaultValue: string | null = null;
  @Input() disabled = false;
  @Output() valuesChange = new EventEmitter<DatasetEnumOptionForm[]>();

  protected searchQuery = '';
  protected addMenuOpen = false;
  protected bulkDialogOpen = false;
  protected bulkValues = '';
  protected bulkValuesError = '';

  protected filteredValues(): Array<{ option: DatasetEnumOptionForm; index: number }> {
    const query = this.searchQuery.trim().toLocaleLowerCase();

    return this.values
      .map((option, index) => ({ option, index }))
      .filter(({ option }) => !query || [option.value, option.label]
        .some((value) => value.toLocaleLowerCase().includes(query)));
  }

  protected defaultOutsideAllowedValues(): boolean {
    if (!this.values.length || this.defaultValue === null || !this.defaultValue.trim()) {
      return false;
    }

    if (this.kind === 'STRING') {
      return !this.values.some((option) => option.value === this.defaultValue);
    }

    const defaultValue = Number(this.defaultValue.replace(',', '.'));
    return Number.isFinite(defaultValue)
      && !this.values.some((option) => Number(option.value.replace(',', '.')) === defaultValue);
  }

  protected updateValue(index: number, value: string, input: HTMLInputElement): void {
    if (value.trim() && this.values.some(
      (option, currentIndex) => currentIndex !== index && this.optionKey(option.value) === this.optionKey(value)
    )) {
      this.notifyDuplicate(value);
      input.value = this.values[index]?.value ?? '';
      return;
    }

    this.valuesChange.emit(this.values.map((option, currentIndex) => currentIndex === index ? { ...option, value } : option));
  }

  protected updateLabel(index: number, label: string): void {
    this.valuesChange.emit(this.values.map((option, currentIndex) => currentIndex === index ? { ...option, label } : option));
  }

  protected removeValue(index: number): void {
    if (this.disabled) {
      return;
    }

    this.valuesChange.emit(this.values.filter((_, currentIndex) => currentIndex !== index));
  }

  protected addOption(): void {
    if (this.disabled) {
      return;
    }

    this.addMenuOpen = false;
    this.valuesChange.emit([...this.values, { value: '', label: '' }]);
  }

  protected toggleAddMenu(): void {
    if (!this.disabled) {
      this.addMenuOpen = !this.addMenuOpen;
    }
  }

  protected openBulkDialog(): void {
    if (this.disabled) {
      return;
    }

    this.addMenuOpen = false;
    this.bulkValues = '';
    this.bulkValuesError = '';
    this.bulkDialogOpen = true;
  }

  protected closeBulkDialog(): void {
    this.bulkDialogOpen = false;
    this.bulkValuesError = '';
  }

  protected addMultipleOptions(): void {
    const parsedValues = this.bulkValues
      .split(/[\r\n,]+/)
      .map((value) => value.trim())
      .filter(Boolean);

    if (!parsedValues.length) {
      this.bulkValuesError = 'Enter at least one option.';
      return;
    }

    if (this.kind === 'INTEGER' && parsedValues.some((value) => !Number.isInteger(Number(value)))) {
      this.bulkValuesError = 'Integer options must be whole numbers.';
      return;
    }

    if (this.kind === 'FLOAT' && parsedValues.some((value) => !Number.isFinite(Number(value)))) {
      this.bulkValuesError = 'Float options must be valid numbers. Use a dot for decimals.';
      return;
    }

    const knownValues = new Set(this.values
      .filter((option) => option.value.trim())
      .map((option) => this.optionKey(option.value)));
    const duplicate = parsedValues.find((value) => {
      const key = this.optionKey(value);
      if (knownValues.has(key)) {
        return true;
      }
      knownValues.add(key);
      return false;
    });
    if (duplicate) {
      this.bulkValuesError = `The option "${duplicate}" already exists.`;
      this.notifyDuplicate(duplicate);
      return;
    }

    this.valuesChange.emit([
      ...this.values,
      ...parsedValues.map((value) => ({ value, label: '' })),
    ]);
    this.closeBulkDialog();
  }

  private optionKey(value: string): string {
    const trimmedValue = value.trim();
    if (this.kind === 'STRING') {
      return trimmedValue;
    }

    return String(Number(trimmedValue.replace(',', '.')));
  }

  private notifyDuplicate(value: string): void {
    this.toast.error(`The option "${value.trim()}" already exists.`);
  }
}
