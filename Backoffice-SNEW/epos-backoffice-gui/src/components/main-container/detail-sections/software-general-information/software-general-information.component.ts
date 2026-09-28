import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type {
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata,
  ResourceVersionReference
} from '../../../../../generated/backofficeSchemas';
import { BackButtonComponent } from '../../../../shared/ui/back-button.component';

type SoftwareResourceType = 'software-application' | 'software-source-code';
type SoftwareMetadata = SoftwareApplicationResourceMetadata | SoftwareSourceCodeResourceMetadata;

@Component({
  selector: 'app-software-general-information',
  imports: [BackButtonComponent, FormsModule],
  templateUrl: './software-general-information.component.html',
  styleUrl: './software-general-information.component.css'
})
export class SoftwareGeneralInformationComponent {
  @Input() model: SoftwareMetadata | null = null;
  @Input() disabled = false;
  @Input() resourceType: SoftwareResourceType = 'software-application';
  @Input() backLink = '/';
  @Input() showBackLink = true;
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

  protected updateKeywords(value: string): void {
    if (!this.model) {
      return;
    }

    this.model.keywords = value.split(',').map((keyword) => keyword.trim()).filter(Boolean);
    this.emitModelChange();
  }

  protected addCreator(): void {
    if (!this.model) {
      return;
    }

    this.model.creators.push({ id: '', name: '' } as ResourceVersionReference);
    this.emitModelChange();
  }

  protected removeCreator(index: number): void {
    if (!this.model) {
      return;
    }

    if (this.model.creators.length === 1) {
      this.model.creators[0] = { id: '', name: '' } as ResourceVersionReference;
    } else {
      this.model.creators.splice(index, 1);
    }

    this.emitModelChange();
  }
}
