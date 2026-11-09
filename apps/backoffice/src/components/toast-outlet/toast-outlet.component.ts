import { DOCUMENT } from '@angular/common';
import { Component, DestroyRef, inject, signal } from '@angular/core';

import { ToastService } from '../../services/toast.service';
import { AppIconComponent } from '../../shared/ui/app-icon.component';

const TOP_LEVEL_TARGET_ANCHORS: ReadonlyArray<readonly [string, string]> = [
  ['spatial_coverage', 'spatial-coverage'],
  ['temporal_coverage', 'temporal-coverage'],
  ['persistent_identifiers', 'persistent-identifiers'],
  ['identifiers', 'persistent-identifiers'],
  ['contact_points', 'contact-points'],
  ['providers', 'providers'],
  ['categories', 'categories'],
  ['distributions', 'distributions']
];

export function toastTargetAnchor(target: string): string | null {
  const normalized = target.toLowerCase();
  const distributionTarget = normalized.includes('distribution') || normalized.includes('web_service');

  if (distributionTarget) {
    const webServiceAnchors: ReadonlyArray<readonly [string, string]> = [
      ['documentation', 'webservice-documentation'],
      ['contact_points', 'webservice-contact-points'],
      ['parameters', 'webservice-parameters'],
      ['provider', 'webservice-provider'],
      ['body', 'webservice-body']
    ];
    const webServiceAnchor = webServiceAnchors.find(([field]) => normalized.includes(field))?.[1];

    return webServiceAnchor ?? (normalized.includes('web_service') ? 'web-service' : 'distributions');
  }

  const sectionAnchor = TOP_LEVEL_TARGET_ANCHORS.find(([field]) => normalized.includes(field))?.[1];
  return sectionAnchor ?? (normalized.includes('metadata') ? 'general-information' : null);
}

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

  protected issueAnchor(target: string | undefined): string | null {
    return target ? toastTargetAnchor(target) : null;
  }

  protected navigateToIssue(anchor: string): void {
    const target = this.document.getElementById(anchor);
    if (!target) {
      return;
    }

    if (target.tagName === 'DETAILS') {
      target.setAttribute('open', '');
    }
    target.closest('details')?.setAttribute('open', '');
    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
    this.document.defaultView?.history.replaceState(null, '', `#${anchor}`);
    this.toastService.dismiss();
  }
}
