import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ContactPointsComponent } from '../contact-points/contact-points.component';
import { SpatialCoverageComponent } from '../spatial-coverage/spatial-coverage.component';
import { TemporalCoverageComponent } from '../temporal-coverage/temporal-coverage.component';
import { WebServiceDocumentationComponent } from './web-service-documentation.component';
import { ProvidersComponent } from '../providers/providers.component';
import { WebServiceParametersComponent } from './web-service-parameters.component';
import type {
  DatasetTemporalForm,
  DatasetWebServiceDocumentationForm,
  DatasetWebServiceForm,
  DatasetWebServiceParameterForm,
  DatasetProviderForm
} from '../../../../models/dataset-detail-form';

@Component({
  selector: 'app-web-service',
  imports: [FormsModule, ContactPointsComponent, SpatialCoverageComponent, TemporalCoverageComponent, WebServiceDocumentationComponent, WebServiceParametersComponent, ProvidersComponent],
  templateUrl: './web-service.component.html',
  styleUrl: './web-service.component.css'
})
export class WebServiceComponent {
  @Input() model: DatasetWebServiceForm | null = null;
  @Input() disabled = false;
  @Output() modelChange = new EventEmitter<DatasetWebServiceForm>();

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

  protected onParametersChange(parameters: DatasetWebServiceParameterForm[]): void {
    if (!this.model) {
      return;
    }

    this.model.parameters = parameters;
    this.emitModelChange();
  }
}
