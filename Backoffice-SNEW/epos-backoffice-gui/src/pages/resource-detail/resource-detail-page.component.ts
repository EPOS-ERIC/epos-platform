import { DatePipe } from '@angular/common';
import { Component, computed, inject } from '@angular/core';

import type { ResourceVersionReference } from '../../../generated/backofficeSchemas';
import { ResourceDetailStore } from '../../stores/resource-detail.store';
import { CategoriesComponent } from '../../components/main-container/detail-sections/categories/categories.component';
import { ContactPointsComponent } from '../../components/main-container/detail-sections/contact-points/contact-points.component';
import { DistributionComponent } from '../../components/main-container/detail-sections/distribution/distribution.component';
import { GeneralInformationComponent } from '../../components/main-container/detail-sections/general-information/general-information.component';
import { IdentifiersComponent } from '../../components/main-container/detail-sections/identifiers/identifiers.component';
import { ProvidersComponent } from '../../components/main-container/detail-sections/providers/providers.component';
import { SoftwareGeneralInformationComponent } from '../../components/main-container/detail-sections/software-general-information/software-general-information.component';
import { SpatialCoverageComponent } from '../../components/main-container/detail-sections/spatial-coverage/spatial-coverage.component';
import { TemporalCoverageComponent } from '../../components/main-container/detail-sections/temporal-coverage/temporal-coverage.component';
import { ResourceVersionHistoryDialogComponent } from '../../dialogs/resource-version-history/resource-version-history-dialog.component';
import { AppIconComponent } from '../../shared/ui/app-icon.component';
import { BackButtonComponent } from '../../shared/ui/back-button.component';

@Component({
  selector: 'app-resource-detail-page',
  imports: [
    AppIconComponent,
    CategoriesComponent,
    BackButtonComponent,
    ContactPointsComponent,
    DatePipe,
    DistributionComponent,
    GeneralInformationComponent,
    IdentifiersComponent,
    ProvidersComponent,
    ResourceVersionHistoryDialogComponent,
    SoftwareGeneralInformationComponent,
    SpatialCoverageComponent,
    TemporalCoverageComponent
  ],
  templateUrl: './resource-detail-page.component.html',
  styleUrl: './resource-detail-page.component.css'
})
export class ResourceDetailPageComponent {
  protected readonly store = inject(ResourceDetailStore);
  protected readonly backLink = computed(() => {
    const context = this.store.context();
    return context?.returnUrl ?? (context ? `/${context.sectionPath}?groupId=${encodeURIComponent(context.groupId)}` : '/');
  });

  protected updateSoftwareApplicationContactPoints(contactPoints: string[]): void {
    const form = this.store.softwareApplicationForm();
    if (form) {
      this.store.updateSoftwareApplicationForm({ ...form, contact_points: contactPoints });
    }
  }

  protected updateSoftwareApplicationCategories(categories: ResourceVersionReference[]): void {
    const form = this.store.softwareApplicationForm();
    if (form) {
      this.store.updateSoftwareApplicationForm({ ...form, categories });
    }
  }

  protected updateSoftwareSourceCodeContactPoints(contactPoints: string[]): void {
    const form = this.store.softwareSourceCodeForm();
    if (form) {
      this.store.updateSoftwareSourceCodeForm({ ...form, contact_points: contactPoints });
    }
  }

  protected updateSoftwareSourceCodeCategories(categories: ResourceVersionReference[]): void {
    const form = this.store.softwareSourceCodeForm();
    if (form) {
      this.store.updateSoftwareSourceCodeForm({ ...form, categories });
    }
  }
}
