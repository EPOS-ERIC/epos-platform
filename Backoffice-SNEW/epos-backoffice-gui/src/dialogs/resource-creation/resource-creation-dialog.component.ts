import { Component, HostListener, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ResourceCreationDialogService } from '../../services/resource-creation-dialog.service';

@Component({
  selector: 'app-resource-creation-dialog',
  imports: [FormsModule],
  templateUrl: './resource-creation-dialog.component.html',
  styleUrl: './resource-creation-dialog.component.css'
})
export class ResourceCreationDialogComponent {
  protected readonly dialog = inject(ResourceCreationDialogService);

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    this.dialog.close();
  }
}
