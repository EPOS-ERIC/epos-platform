import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { DatasetDistributionCommonForm, DatasetDistributionForm } from '../../../../models/dataset-detail-form';
import { WebServiceComponent } from '../web-service/web-service.component';

@Component({
  selector: 'app-distribution',
  imports: [FormsModule, WebServiceComponent],
  templateUrl: './distribution.component.html',
  styleUrl: './distribution.component.css'
})
export class DistributionComponent {
  @Input() model: DatasetDistributionForm[] = [];
  @Input() disabled = false;
  @Output() modelChange = new EventEmitter<DatasetDistributionForm[]>();

  protected readonly distributionFields: Array<keyof DatasetDistributionCommonForm> = ['title', 'description', 'issued', 'modified', 'license', 'type'];
  protected distributionTabs = 0;

  protected selectDistributionTab(index: number): void {
    this.distributionTabs = index;
  }

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected selectedDistribution(): DatasetDistributionForm | null {
    return this.model[this.distributionTabs] ?? this.model[0] ?? null;
  }

  protected distributionFieldLabel(field: keyof DatasetDistributionCommonForm): string {
    const labels: Record<keyof DatasetDistributionCommonForm, string> = {
      title: 'Title',
      description: 'Description',
      issued: 'Issued',
      modified: 'Modified',
      license: 'License',
      type: 'Type'
    };

    return labels[field];
  }
}
