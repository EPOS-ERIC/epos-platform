import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-default-values-editor',
  imports: [FormsModule],
  templateUrl: './default-values-editor.component.html',
  styleUrl: './default-values-editor.component.css'
})
export class DefaultValuesEditorComponent {
  @Input() values: string[] = [];
  @Input() kind: 'STRING' | 'INTEGER' | 'FLOAT' = 'STRING';
  @Input() allowMultiple = false;
  @Input() disabled = false;
  @Output() valuesChange = new EventEmitter<string[]>();

  protected displayedValues(): string[] {
    const values = this.values.length ? this.values : [''];
    return this.allowMultiple ? values : values.slice(0, 1);
  }

  protected updateValue(index: number, value: string): void {
    if (this.disabled) return;

    const values = [...this.displayedValues()];
    values[index] = value;
    this.valuesChange.emit(values);
  }

  protected addValue(): void {
    if (!this.disabled && this.allowMultiple) {
      this.valuesChange.emit([...this.displayedValues(), '']);
    }
  }

  protected removeValue(index: number): void {
    if (!this.disabled && this.allowMultiple && this.displayedValues().length > 1) {
      this.valuesChange.emit(this.displayedValues().filter((_, currentIndex) => currentIndex !== index));
    }
  }
}
