import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { DatasetDistributionCommonForm, DatasetDistributionForm, DatasetWebServiceForm } from '../../../../models/dataset-detail-form';
import { WebServiceComponent } from '../web-service/web-service.component';

@Component({
  selector: 'app-distribution',
  imports: [FormsModule, WebServiceComponent],
  templateUrl: './distribution.component.html',
  styleUrl: './distribution.component.css'
})
export class DistributionComponent {
  @Input() model: DatasetDistributionForm[] = [];
  @Input() comparisonModel: DatasetDistributionForm[] = [];
  @Input() disabled = false;
  @Input() comparisonMode = false;
  @Input() comparisonSide: 'left' | 'right' = 'left';
  @Output() modelChange = new EventEmitter<DatasetDistributionForm[]>();

  protected readonly distributionFields: Array<keyof DatasetDistributionCommonForm> = ['title', 'description', 'license', 'byteSize', 'type'];
  protected distributionTabs = 0;
  protected newDistributionKind: DatasetDistributionForm['kind'] = 'web-service';

  protected selectDistributionTab(index: number): void {
    this.distributionTabs = index;
  }

  protected emitModelChange(): void {
    this.modelChange.emit(this.model);
  }

  protected addDistribution(): void {
    if (this.disabled) {
      return;
    }

    const model = [...this.model, this.emptyDistribution(this.newDistributionKind)];
    this.distributionTabs = model.length - 1;
    this.modelChange.emit(model);
  }

  protected deleteSelectedDistribution(): void {
    if (this.disabled || this.comparisonMode || !this.model.length) {
      return;
    }

    const selectedIndex = Math.min(this.distributionTabs, this.model.length - 1);
    const model = this.model.filter((_, index) => index !== selectedIndex);
    this.distributionTabs = Math.max(0, Math.min(selectedIndex, model.length - 1));
    this.modelChange.emit(model);
  }

  protected selectedDistribution(): DatasetDistributionForm | null {
    return this.model[this.distributionTabs] ?? this.model[0] ?? null;
  }

  protected selectedComparisonDistribution(): DatasetDistributionForm | null {
    return this.comparisonModel[this.distributionTabs] ?? this.comparisonModel[0] ?? null;
  }

  protected selectedComparisonWebService(): DatasetWebServiceForm | null {
    const distribution = this.selectedComparisonDistribution();
    return distribution?.kind === 'web-service' ? distribution.webService : null;
  }

  protected distributionFieldLabel(field: keyof DatasetDistributionCommonForm): string {
    const labels: Record<keyof DatasetDistributionCommonForm, string> = {
      id: 'ID',
      nativeFormats: 'Native formats',
      title: 'Title',
      description: 'Description',
      license: 'License',
      byteSize: 'Byte size',
      type: 'Type'
    };

    return labels[field];
  }

  private emptyDistribution(kind: DatasetDistributionForm['kind']): DatasetDistributionForm {
    const common = {
      title: '',
      description: '',
      license: '',
      byteSize: ''
    };

    if (kind === 'downloadable-file') {
      return {
        ...common,
        kind,
        type: 'DOWNLOADABLE_FILE',
        downloadableFile: { url: '', format: '' }
      };
    }

    return {
      ...common,
      kind,
      type: 'WEB_SERVICE',
      webService: {
        visibleOn: [],
        documentationUri: '',
        provider: { id: '', name: '' },
        contactPoints: [],
        configuration: {
          type: 'HTTP',
          templateUrl: '',
          method: 'GET',
          format: '',
          bodyContentType: '',
          bodyTemplate: '',
          bodyRequired: true,
          bodyParameters: [],
          parameters: []
        }
      }
    };
  }
}
