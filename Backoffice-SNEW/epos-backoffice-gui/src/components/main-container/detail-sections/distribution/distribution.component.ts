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

  protected readonly distributionFields: Array<keyof DatasetDistributionCommonForm> = ['title', 'description', 'issued', 'modified', 'license', 'type'];
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
      title: 'Title',
      description: 'Description',
      issued: 'Issued',
      modified: 'Modified',
      license: 'License',
      type: 'Type'
    };

    return labels[field];
  }

  private emptyDistribution(kind: DatasetDistributionForm['kind']): DatasetDistributionForm {
    const common = {
      title: '',
      description: '',
      issued: '',
      modified: '',
      license: ''
    };

    if (kind === 'downloadable-file') {
      return {
        ...common,
        kind,
        type: 'http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE',
        downloadableFile: { url: '', format: '' }
      };
    }

    return {
      ...common,
      kind,
      type: 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE',
      webService: {
        name: '',
        description: '',
        templateUrl: '',
        method: 'GET',
        type: 'DYNAMIC',
        visibleOn: [],
        format: '',
        bodyContentType: '',
        bodyTemplate: '',
        bodyParameters: [],
        documentation: { title: '', description: '', uri: '' },
        provider: { id: '', name: '' },
        contactPoints: [],
        spatialCoverage: null,
        temporalCoverage: { start: '', end: '' },
        parameters: []
      }
    };
  }
}
