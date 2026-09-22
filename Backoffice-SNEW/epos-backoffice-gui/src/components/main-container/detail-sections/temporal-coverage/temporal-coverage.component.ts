import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { AppIconComponent } from '../../../app-icon/app-icon.component';
import type { DatasetGeologicalTemporalForm, DatasetTemporalForm } from '../../../../models/dataset-detail-form';

@Component({
  selector: 'app-temporal-coverage',
  imports: [AppIconComponent, FormsModule],
  templateUrl: './temporal-coverage.component.html',
  styleUrl: './temporal-coverage.component.css'
})
export class TemporalCoverageComponent {
  @Input() model: DatasetTemporalForm = { start: '', end: '' };
  @Input() geologicalModel: DatasetGeologicalTemporalForm = { startMya: '', endMya: '' };
  @Input() mode: 'calendar' | 'geological' = 'calendar';
  @Input() allowGeological = false;
  @Input() disabled = false;
  @Input() warningKind: 'none' | 'warning' | 'error' = 'none';
  @Input() warningMessage = '';
  @Output() modelChange = new EventEmitter<DatasetTemporalForm>();
  @Output() geologicalModelChange = new EventEmitter<DatasetGeologicalTemporalForm>();
  @Output() modeChange = new EventEmitter<'calendar' | 'geological'>();

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected emitGeologicalModelChange(): void {
    this.geologicalModelChange.emit(this.geologicalModel);
  }
}
