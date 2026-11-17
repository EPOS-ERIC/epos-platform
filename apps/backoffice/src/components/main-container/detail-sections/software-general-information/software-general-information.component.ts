import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type {
  ResourceVersionReference,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata
} from '../../../../../generated/backofficeSchemas';
import { KeywordChipsComponent } from '../../../../shared/ui/keyword-chips.component';
import { ProvidersComponent } from '../providers/providers.component';

type SoftwareResourceType = 'software-application' | 'software-source-code';
type SoftwareMetadata = SoftwareApplicationResourceMetadata | SoftwareSourceCodeResourceMetadata;

@Component({
  selector: 'app-software-general-information',
  imports: [FormsModule, KeywordChipsComponent, ProvidersComponent],
  templateUrl: './software-general-information.component.html',
  styleUrl: './software-general-information.component.css'
})
export class SoftwareGeneralInformationComponent {
  @Input() model: SoftwareMetadata | null = null;
  @Input() disabled = false;
  @Input() resourceType: SoftwareResourceType = 'software-application';
  @Output() modelChange = new EventEmitter<SoftwareMetadata>();

  protected applicationModel(): SoftwareApplicationResourceMetadata | null {
    return this.resourceType === 'software-application' ? (this.model as SoftwareApplicationResourceMetadata | null) : null;
  }

  protected sourceCodeModel(): SoftwareSourceCodeResourceMetadata | null {
    return this.resourceType === 'software-source-code' ? (this.model as SoftwareSourceCodeResourceMetadata | null) : null;
  }

  protected emitModelChange(): void {
    if (this.model) {
      this.modelChange.emit(this.model);
    }
  }

  protected updateKeywords(keywords: string[]): void {
    if (!this.model) return;
    this.model.keywords = keywords;
    this.emitModelChange();
  }

  protected updateCreators(creators: ResourceVersionReference[] | ResourceVersionReference): void {
    if (!this.model) return;
    this.model.creators = Array.isArray(creators) ? creators : [creators];
    this.emitModelChange();
  }
}
