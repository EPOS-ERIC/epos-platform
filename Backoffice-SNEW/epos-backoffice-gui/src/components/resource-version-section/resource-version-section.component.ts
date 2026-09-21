import { Component, Input } from '@angular/core';

import type {
  DatasetResourceMetadata,
  ResourceVersion,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata
} from '../../../generated/backofficeSchemas';
import { toDatasetDetailForm } from '../../mappers/dataset-detail-form.mapper';
import type { DatasetDetailForm } from '../../models/dataset-detail-form';
import { CategoriesComponent } from '../main-container/detail-sections/categories/categories.component';
import { ContactPointsComponent } from '../main-container/detail-sections/contact-points/contact-points.component';
import { DistributionComponent } from '../main-container/detail-sections/distribution/distribution.component';
import { GeneralInformationComponent } from '../main-container/detail-sections/general-information/general-information.component';
import { IdentifiersComponent } from '../main-container/detail-sections/identifiers/identifiers.component';
import { ProvidersComponent } from '../main-container/detail-sections/providers/providers.component';
import { SoftwareGeneralInformationComponent } from '../main-container/detail-sections/software-general-information/software-general-information.component';
import { SpatialCoverageComponent } from '../main-container/detail-sections/spatial-coverage/spatial-coverage.component';
import { TemporalCoverageComponent } from '../main-container/detail-sections/temporal-coverage/temporal-coverage.component';

type SoftwareMetadata = SoftwareApplicationResourceMetadata | SoftwareSourceCodeResourceMetadata;

@Component({
  selector: 'app-resource-version-section',
  imports: [
    CategoriesComponent,
    ContactPointsComponent,
    DistributionComponent,
    GeneralInformationComponent,
    IdentifiersComponent,
    ProvidersComponent,
    SoftwareGeneralInformationComponent,
    SpatialCoverageComponent,
    TemporalCoverageComponent
  ],
  templateUrl: './resource-version-section.component.html',
  styleUrl: './resource-version-section.component.css'
})
export class ResourceVersionSectionComponent {
  @Input({ required: true }) sectionTitle = '';
  @Input() comparisonSide: 'left' | 'right' = 'left';

  protected currentVersion: ResourceVersion | null = null;
  protected datasetForm: DatasetDetailForm | null = null;
  protected comparisonDatasetForm: DatasetDetailForm | null = null;
  protected softwareMetadata: SoftwareMetadata | null = null;

  @Input()
  set comparisonVersion(version: ResourceVersion | null) {
    this.comparisonDatasetForm = version?.metadata.resource_type === 'dataset'
      ? toDatasetDetailForm(version.metadata as DatasetResourceMetadata)
      : null;
  }

  @Input({ required: true })
  set version(version: ResourceVersion) {
    this.currentVersion = version;
    if (version.metadata.resource_type === 'dataset') {
      this.datasetForm = toDatasetDetailForm(version.metadata as DatasetResourceMetadata);
      this.softwareMetadata = null;
      return;
    }

    this.datasetForm = null;
    this.softwareMetadata = version.metadata as SoftwareMetadata;
  }

  protected softwareResourceType(): 'software-application' | 'software-source-code' {
    return this.softwareMetadata?.resource_type ?? 'software-application';
  }
}
