import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';

import type { DatasetDetailForm } from '../../../../models/dataset-detail-form';

@Component({
  selector: 'app-general-information',
  imports: [FormsModule, RouterLink],
  templateUrl: './general-information.component.html',
  styleUrl: './general-information.component.css'
})
export class GeneralInformationComponent {
  @Input() model: DatasetDetailForm | null = null;
  @Input() disabled = false;
  @Input() backLink: string | string[] = '/';
  @Input() showBackLink = true;
  @Output() modelChange = new EventEmitter<DatasetDetailForm>();

  protected emitModelChange(): void {
    if (this.model) {
      this.modelChange.emit(this.model);
    }
  }
}
