import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { DatasetDetailForm } from '../../../../models/dataset-detail-form';
import { KeywordChipsComponent } from '../../../../shared/ui/keyword-chips.component';

@Component({
  selector: 'app-general-information',
  imports: [FormsModule, KeywordChipsComponent],
  templateUrl: './general-information.component.html',
  styleUrl: './general-information.component.css'
})
export class GeneralInformationComponent {
  @Input() model: DatasetDetailForm | null = null;
  @Input() disabled = false;
  @Output() modelChange = new EventEmitter<DatasetDetailForm>();

  protected emitModelChange(): void {
    if (this.model) {
      this.modelChange.emit(this.model);
    }
  }

  protected updateKeywords(keywords: string[]): void {
    if (!this.model) return;
    this.model.keywords = keywords;
    this.emitModelChange();
  }
}
