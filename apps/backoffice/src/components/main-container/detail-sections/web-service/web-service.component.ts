import { Component, EventEmitter, Input, Output } from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ContactPointsComponent } from '../contact-points/contact-points.component';
import { SpatialCoverageComponent } from '../spatial-coverage/spatial-coverage.component';
import { TemporalCoverageComponent } from '../temporal-coverage/temporal-coverage.component';
import { WebServiceDocumentationComponent } from './web-service-documentation.component';
import { ProvidersComponent } from '../providers/providers.component';
import { WebServiceParametersComponent } from './web-service-parameters.component';
import { ScalarParametersEditorComponent } from './scalar-parameters-editor.component';
import type {
  DatasetTemporalForm,
  DatasetWebServiceConfigurationForm,
  DatasetWebServiceDocumentationForm,
  DatasetWebServiceForm,
  DatasetWebServiceParameterForm,
  DatasetWebServiceScalarParameterForm,
  DatasetProviderForm
} from '../../../../models/dataset-detail-form';

@Component({
  selector: 'app-web-service',
  imports: [FormsModule, NgTemplateOutlet, ContactPointsComponent, ProvidersComponent, ScalarParametersEditorComponent, SpatialCoverageComponent, TemporalCoverageComponent, WebServiceDocumentationComponent, WebServiceParametersComponent],
  templateUrl: './web-service.component.html',
  styleUrl: './web-service.component.css'
})
export class WebServiceComponent {
  @Input() model: DatasetWebServiceForm | null = null;
  @Input() comparisonModel: DatasetWebServiceForm | null = null;
  @Input() disabled = false;
  @Input() comparisonMode = false;
  @Input() comparisonSide: 'left' | 'right' = 'left';
  @Output() modelChange = new EventEmitter<DatasetWebServiceForm>();

  protected readonly methods = ['GET', 'POST', 'PUT', 'DELETE'] as const;
  protected readonly types: readonly DatasetWebServiceConfigurationForm['type'][] = ['HTTP', 'DYNAMIC', 'WFS', 'WMS', 'WMTS'];
  protected readonly visibleOnOptions: readonly DatasetWebServiceForm['visibleOn'][number][] = ['GRAPH', 'TABLE', 'MAP'];

  protected emitModelChange(): void {
    this.modelChange.emit(this.model as DatasetWebServiceForm);
  }

  protected onProviderChange(provider: DatasetProviderForm): void {
    if (!this.model) {
      return;
    }

    this.model.provider = provider;
    this.emitModelChange();
  }

  protected onDocumentationChange(documentation: DatasetWebServiceDocumentationForm): void {
    if (!this.model) {
      return;
    }

    this.model.documentation = documentation;
    this.emitModelChange();
  }

  protected onContactPointsChange(contactPoints: string[]): void {
    if (!this.model) {
      return;
    }

    this.model.contactPoints = contactPoints;
    this.emitModelChange();
  }

  protected onSpatialChange(spatialCoverage: DatasetWebServiceForm['spatialCoverage']): void {
    if (!this.model) {
      return;
    }

    this.model.spatialCoverage = spatialCoverage;
    this.emitModelChange();
  }

  protected onTemporalChange(temporalCoverage: DatasetTemporalForm): void {
    if (!this.model) {
      return;
    }

    this.model.temporalCoverage = temporalCoverage;
    this.emitModelChange();
  }

  protected onConfigurationTypeChange(type: DatasetWebServiceConfigurationForm['type']): void {
    if (!this.model || this.model.configuration.type === type) return;
    this.model.configuration = this.emptyConfiguration(type);
    this.emitModelChange();
  }

  protected onBodyParametersChange(parameters: DatasetWebServiceScalarParameterForm[]): void {
    if (
      this.model?.configuration.type === 'HTTP' ||
      this.model?.configuration.type === 'DYNAMIC'
    ) {
      this.model.configuration.bodyParameters = parameters;
      this.emitModelChange();
    }
  }

  protected joinValues(values: string[]): string {
    return values.join(', ');
  }

  protected splitValues(value: string): string[] {
    return value.split(',').map((item) => item.trim()).filter(Boolean);
  }

  protected addLayer(): void {
    const configuration = this.model?.configuration;
    if (configuration?.type === 'WMS' || configuration?.type === 'WMTS') {
      configuration.layers.push({ name: '', defaultStyle: '', styles: [] });
      this.emitModelChange();
    }
  }

  protected removeLayer(index: number): void {
    const configuration = this.model?.configuration;
    if (configuration?.type === 'WMS' || configuration?.type === 'WMTS') {
      configuration.layers.splice(index, 1);
      this.emitModelChange();
    }
  }

  protected toggleVisibleOn(value: DatasetWebServiceForm['visibleOn'][number], checked: boolean): void {
    if (!this.model) {
      return;
    }

    this.model.visibleOn = checked
      ? Array.from(new Set([...this.model.visibleOn, value]))
      : this.model.visibleOn.filter((currentValue) => currentValue !== value);
    this.emitModelChange();
  }

  private emptyConfiguration(
    type: DatasetWebServiceConfigurationForm['type'],
  ): DatasetWebServiceConfigurationForm {
    switch (type) {
      case 'HTTP':
      case 'DYNAMIC':
        return { type, templateUrl: '', method: 'GET', format: '', bodyContentType: '', bodyTemplate: '', bodyRequired: true, bodyParameters: [], parameters: [] };
      case 'WFS':
        return { type, getFeatureTemplateUrl: '', version: '2.0.0', featureTypes: [], defaultFeatureType: '', outputFormats: [], defaultOutputFormat: '', resultLimit: '', resultLimitEditable: false, parameters: [] };
      case 'WMS':
        return { type, capabilitiesUrl: '', defaultLayer: '', layers: [{ name: '', defaultStyle: '', styles: [''] }], transparencyEditable: false, defaultTransparency: true, getMapParameters: [], getFeatureInfoParameters: [] };
      case 'WMTS':
        return { type, capabilitiesUrl: '', defaultLayer: '', layers: [{ name: '', defaultStyle: '', styles: [] }], getTileParameters: [], getFeatureInfoParameters: [] };
    }
  }
}
