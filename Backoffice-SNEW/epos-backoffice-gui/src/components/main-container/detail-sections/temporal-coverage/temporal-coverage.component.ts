import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { AppIconComponent } from '../../../app-icon/app-icon.component';
import type { DatasetTemporalForm } from '../../../../models/dataset-detail-form';

@Component({
  selector: 'app-temporal-coverage',
  imports: [AppIconComponent, FormsModule],
  templateUrl: './temporal-coverage.component.html',
  styleUrl: './temporal-coverage.component.css'
})
export class TemporalCoverageComponent {
  @Input() model: DatasetTemporalForm = { start: '', end: '' };
  @Input() disabled = false;
  @Input() warningKind: 'none' | 'warning' | 'error' = 'none';
  @Input() warningMessage = '';
  @Output() modelChange = new EventEmitter<DatasetTemporalForm>();

  protected emitModelChange(): void {
    if (this.model) {
      this.modelChange.emit(this.model);
    }
  }
}
