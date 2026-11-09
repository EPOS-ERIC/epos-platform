import { Component, Input, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import type { ResourceVersion } from '../../../generated/backofficeSchemas';
import type { ResourceVersionComparisonContext } from '../../models/resource-version-comparison';
import { ResourceVersionComparisonStore } from '../../stores/resource-version-comparison.store';
import { ResourceVersionSectionComponent } from '../../components/resource-version-section/resource-version-section.component';
import { ComparisonFieldsDirective } from './comparison-fields.directive';

@Component({
  selector: 'app-resource-version-comparison-page',
  imports: [ComparisonFieldsDirective, ResourceVersionSectionComponent, RouterLink],
  providers: [ResourceVersionComparisonStore],
  templateUrl: './resource-version-comparison-page.component.html',
  styleUrl: './resource-version-comparison-page.component.css'
})
export class ResourceVersionComparisonPageComponent {
  protected readonly store = inject(ResourceVersionComparisonStore);
  protected readonly selectedParameterResourceId = signal<string | null>(null);
  protected readonly selectedDistributionResourceId = signal<string | null>(null);
  protected comparisonContext: ResourceVersionComparisonContext | null = null;

  @Input({ required: true })
  set context(context: ResourceVersionComparisonContext) {
    this.comparisonContext = context;
    this.selectedParameterResourceId.set(null);
    this.selectedDistributionResourceId.set(null);
    this.store.load(context);
  }

  protected selectDistribution(resourceId: string): void {
    this.selectedDistributionResourceId.set(resourceId);
    this.selectedParameterResourceId.set(null);
  }

  protected versionLabel(version: ResourceVersion): string {
    return `Revision ${version.revision}`;
  }

  protected sectionAnchor(title: string): string {
    return `compare-${title.toLowerCase().replace(/\s+/g, '-')}`;
  }
}
