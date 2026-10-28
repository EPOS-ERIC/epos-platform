import { DOCUMENT } from '@angular/common';
import { Component, DestroyRef, inject, signal } from '@angular/core';

import { ToastService } from '../../services/toast.service';
import { AppIconComponent } from '../../shared/ui/app-icon.component';

@Component({
  selector: 'app-toast-outlet',
  imports: [AppIconComponent],
  templateUrl: './toast-outlet.component.html',
  styleUrl: './toast-outlet.component.css'
})
export class ToastOutletComponent {
  private readonly document = inject(DOCUMENT);
  private readonly destroyRef = inject(DestroyRef);
  private copyFeedbackTimeout?: ReturnType<typeof setTimeout>;

  protected readonly toastService = inject(ToastService);
  protected readonly copiedReference = signal<string | null>(null);
  protected readonly failedReference = signal<string | null>(null);

  constructor() {
    this.destroyRef.onDestroy(() => {
      if (this.copyFeedbackTimeout) {
        clearTimeout(this.copyFeedbackTimeout);
      }
    });
  }

  protected copyReference(reference: string): void {
    const clipboard = this.document.defaultView?.navigator.clipboard;
    if (!clipboard) {
      this.failedReference.set(reference);
      return;
    }

    void clipboard.writeText(reference).then(() => {
      this.failedReference.set(null);
      this.copiedReference.set(reference);
      if (this.copyFeedbackTimeout) {
        clearTimeout(this.copyFeedbackTimeout);
      }
      this.copyFeedbackTimeout = setTimeout(() => {
        this.copiedReference.set(null);
        this.copyFeedbackTimeout = undefined;
      }, 2000);
    }).catch(() => {
      this.copiedReference.set(null);
      this.failedReference.set(reference);
    });
  }
}
