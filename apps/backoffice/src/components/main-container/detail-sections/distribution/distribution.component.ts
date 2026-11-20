import { Component, EventEmitter, Input, OnChanges, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { DatasetDistributionCommonForm, DatasetDistributionForm, DatasetWebServiceForm } from '../../../../models/dataset-detail-form';
import { WebServiceComponent } from '../web-service/web-service.component';

type DistributionComparisonStatus = 'unchanged' | 'changed' | 'added' | 'removed';

@Component({
  selector: 'app-distribution',
  imports: [FormsModule, WebServiceComponent],
  templateUrl: './distribution.component.html',
  styleUrl: './distribution.component.css'
})
export class DistributionComponent implements OnChanges {
  @Input() model: DatasetDistributionForm[] = [];
  @Input() comparisonModel: DatasetDistributionForm[] = [];
  @Input() groupId = '';
  @Input() disabled = false;
  @Input() comparisonMode = false;
  @Input() comparisonSide: 'left' | 'right' = 'left';
  @Input() comparisonSelectedParameterResourceId: string | null = null;
  @Input() comparisonSelectedDistributionResourceId: string | null = null;
  @Output() modelChange = new EventEmitter<DatasetDistributionForm[]>();
  @Output() comparisonSelectedParameterResourceIdChange = new EventEmitter<string>();
  @Output() comparisonSelectedDistributionResourceIdChange = new EventEmitter<string>();

  protected readonly distributionFields: Array<keyof DatasetDistributionCommonForm> = ['title', 'description', 'license', 'byteSize', 'type'];
  protected distributionTabs = 0;
  protected newDistributionKind: DatasetDistributionForm['kind'] = 'web-service';

  ngOnChanges(): void {
    if (this.comparisonSelectedDistributionResourceId) {
      const selectedIndex = this.model.findIndex(
        (distribution) => distribution.resourceId === this.comparisonSelectedDistributionResourceId
      );
      if (selectedIndex >= 0) {
        this.distributionTabs = selectedIndex;
      }
      return;
    }

    if (this.comparisonMode && this.comparisonSide === 'right') {
      this.distributionTabs = this.displayedDistributions()[0]?.globalIndex ?? 0;
    }
  }

  protected displayedDistributions(): Array<{ distribution: DatasetDistributionForm; globalIndex: number }> {
    const entries = this.model.map((distribution, globalIndex) => ({ distribution, globalIndex }));
    if (!this.comparisonMode || this.comparisonSide !== 'right') {
      return entries;
    }

    const comparisonOrder = new Map(
      this.comparisonModel.map((distribution, index) => [distribution.resourceId, index])
    );
    return entries.sort((left, right) => {
      const leftIndex = left.distribution.resourceId
        ? (comparisonOrder.get(left.distribution.resourceId) ?? Number.MAX_SAFE_INTEGER)
        : left.globalIndex;
      const rightIndex = right.distribution.resourceId
        ? (comparisonOrder.get(right.distribution.resourceId) ?? Number.MAX_SAFE_INTEGER)
        : right.globalIndex;
      return leftIndex - rightIndex || left.globalIndex - right.globalIndex;
    });
  }

  protected selectDistributionTab(index: number): void {
    this.distributionTabs = index;
    const resourceId = this.model[index]?.resourceId;
    if (this.comparisonMode && resourceId) {
      this.comparisonSelectedDistributionResourceIdChange.emit(resourceId);
    }
  }

  protected distributionComparisonStatus(distribution: DatasetDistributionForm): DistributionComparisonStatus {
    if (!this.comparisonMode) {
      return 'unchanged';
    }
    const counterpart = distribution.resourceId
      ? this.comparisonModel.find((candidate) => candidate.resourceId === distribution.resourceId)
      : this.comparisonModel[this.model.indexOf(distribution)];
    if (!counterpart) {
      return this.comparisonSide === 'right' ? 'added' : 'removed';
    }
    return this.distributionComparableValue(distribution) === this.distributionComparableValue(counterpart)
      ? 'unchanged'
      : 'changed';
  }

  protected distributionStatusLabel(distribution: DatasetDistributionForm): string {
    const status = this.distributionComparisonStatus(distribution);
    return status === 'unchanged' ? '' : status.charAt(0).toUpperCase() + status.slice(1);
  }

  protected distributionComparisonKey(distribution: DatasetDistributionForm): string {
    return distribution.resourceId
      ? `resource:${distribution.resourceId}`
      : `index:${this.model.indexOf(distribution)}`;
  }

  protected distributionHasPositiveDiff(distribution: DatasetDistributionForm): boolean {
    const status = this.distributionComparisonStatus(distribution);
    return status === 'added' || (status === 'changed' && this.comparisonSide === 'right');
  }

  protected distributionHasNegativeDiff(distribution: DatasetDistributionForm): boolean {
    const status = this.distributionComparisonStatus(distribution);
    return status === 'removed' || (status === 'changed' && this.comparisonSide === 'left');
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
    const selected = this.selectedDistribution();
    if (selected?.resourceId) {
      return this.comparisonModel.find((distribution) => distribution.resourceId === selected.resourceId) ?? null;
    }
    return this.comparisonModel[this.distributionTabs] ?? this.comparisonModel[0] ?? null;
  }

  protected selectedComparisonWebService(): DatasetWebServiceForm | null {
    const distribution = this.selectedComparisonDistribution();
    return distribution?.kind === 'web-service' ? distribution.webService : null;
  }

  protected distributionFieldLabel(field: keyof DatasetDistributionCommonForm): string {
    const labels: Record<keyof DatasetDistributionCommonForm, string> = {
      id: 'ID',
      resourceId: 'Resource ID',
      nativeFormats: 'Native formats',
      title: 'Title',
      description: 'Description',
      license: 'License',
      byteSize: 'Byte size',
      type: 'Type'
    };

    return labels[field];
  }

  private distributionComparableValue(distribution: DatasetDistributionForm): string {
    return JSON.stringify(this.normalizeComparisonValue(distribution));
  }

  private normalizeComparisonValue(value: unknown): unknown {
    if (Array.isArray(value)) {
      const entries = value.map((item) => ({
        resourceId: item && typeof item === 'object' && 'resourceId' in item
          ? String(item.resourceId ?? '')
          : '',
        value: this.normalizeComparisonValue(item)
      }));
      if (entries.length > 0 && entries.every((entry) => entry.resourceId)) {
        entries.sort((left, right) => left.resourceId.localeCompare(right.resourceId));
      }
      return entries.map((entry) => entry.value);
    }
    if (value === null || typeof value !== 'object') {
      return value;
    }
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .filter(([key]) => key !== 'id' && key !== 'resourceId')
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, entry]) => [key, this.normalizeComparisonValue(entry)])
    );
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
        keywords: [],
        accessRights: '',
        license: '',
        categories: [],
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
