import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { DatasetDetailForm } from '../../../../models/dataset-detail-form';
import { BackButtonComponent } from '../../../../shared/ui/back-button.component';

@Component({
  selector: 'app-general-information',
  imports: [BackButtonComponent, FormsModule],
  templateUrl: './general-information.component.html',
  styleUrl: './general-information.component.css'
})
export class GeneralInformationComponent {
  @Input() model: DatasetDetailForm | null = null;
  @Input() disabled = false;
  @Input() backLink = '/';
  @Input() showBackLink = true;
  @Output() modelChange = new EventEmitter<DatasetDetailForm>();

  protected emitModelChange(): void {
    if (this.model) {
      this.modelChange.emit(this.model);
    }
  }
}
