import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

import type { DatasetWebServiceDocumentationForm } from '../../main-container.component';

@Component({
  selector: 'app-web-service-documentation',
  imports: [FormsModule],
  templateUrl: './web-service-documentation.component.html',
  styleUrl: './web-service-documentation.component.css'
})
export class WebServiceDocumentationComponent {
  @Input() model: DatasetWebServiceDocumentationForm | null = null;
  @Input() disabled = false;
  @Output() modelChange = new EventEmitter<DatasetWebServiceDocumentationForm>();

  protected emitModelChange(): void {
    if (this.model) {
      this.modelChange.emit(this.model);
    }
  }
}
