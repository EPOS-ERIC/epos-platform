import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

export type IdentifierItem = {
  id?: string;
  type: string;
  value: string;
};

@Component({
  selector: 'app-identifiers',
  imports: [FormsModule],
  templateUrl: './identifiers.component.html',
  styleUrl: './identifiers.component.css'
})
export class IdentifiersComponent {
  @Input() model: IdentifierItem[] = [];
  @Input() disabled = false;
  @Input() anchor = 'persistent-identifiers';
  @Input() title = 'Identifiers';
  @Input() subtitle = 'Type and identifier value pairs.';
  @Output() modelChange = new EventEmitter<IdentifierItem[]>();

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected addItem(): void {
    if (this.disabled) {
      return;
    }

    this.model.push({ type: '', value: '' });
    this.emitModelChange();
  }

  protected removeItem(index: number): void {
    if (this.disabled) {
      return;
    }

    if (this.model.length === 1) {
      this.model[0] = { type: '', value: '' };
    } else {
      this.model.splice(index, 1);
    }

    this.emitModelChange();
  }
}
