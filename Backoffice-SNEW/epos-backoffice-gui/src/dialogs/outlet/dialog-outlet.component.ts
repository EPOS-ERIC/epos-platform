import { Component, HostListener, effect, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { DialogService } from '../../services/dialog.service';

@Component({
  selector: 'app-dialog-outlet',
  imports: [FormsModule],
  templateUrl: './dialog-outlet.component.html',
  styleUrl: './dialog-outlet.component.css'
})
export class DialogOutletComponent {
  protected readonly dialogService = inject(DialogService);
  protected promptValue = '';

  constructor() {
    effect(() => {
      const dialog = this.dialogService.state();
      this.promptValue = dialog?.kind === 'prompt' ? (dialog.initialValue ?? '') : '';
    });
  }

  @HostListener('document:keydown.escape')
  protected dismissOnEscape(): void {
    this.dialogService.dismiss();
  }
}
