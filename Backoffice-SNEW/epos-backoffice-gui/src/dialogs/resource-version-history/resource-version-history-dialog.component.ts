import { Component, HostListener, inject } from '@angular/core';

import { ResourceDetailStore } from '../../stores/resource-detail.store';

@Component({
  selector: 'app-resource-version-history-dialog',
  templateUrl: './resource-version-history-dialog.component.html',
  styleUrl: './resource-version-history-dialog.component.css'
})
export class ResourceVersionHistoryDialogComponent {
  protected readonly store = inject(ResourceDetailStore);

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    if (this.store.historyOpen()) {
      this.store.closeVersionHistory();
    }
  }
}
