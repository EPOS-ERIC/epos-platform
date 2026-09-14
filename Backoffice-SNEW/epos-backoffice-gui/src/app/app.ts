import { Component, DOCUMENT, effect, inject, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { DialogOutletComponent } from '../components/dialog-outlet/dialog-outlet.component';
import { MembershipRequestDialogComponent } from '../components/membership-request-dialog/membership-request-dialog.component';
import { ResourceCreationDialogComponent } from '../components/resource-creation-dialog/resource-creation-dialog.component';
import { ToastOutletComponent } from '../components/toast-outlet/toast-outlet.component';

@Component({
  selector: 'app-root',
  imports: [DialogOutletComponent, MembershipRequestDialogComponent, ResourceCreationDialogComponent, RouterOutlet, ToastOutletComponent],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private readonly document = inject(DOCUMENT);
  private readonly useScalarVariant = false;

  protected readonly title = signal('epos-backoffice-gui');

  constructor() {
    effect(() => {
      const variant = this.useScalarVariant ? 'scalar' : 'current';
      this.document.documentElement.setAttribute('data-ui-variant', variant);
    });
  }
}
