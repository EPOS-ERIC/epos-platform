import { Component, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { ToastOutletComponent } from '../components/toast-outlet/toast-outlet.component';
import { MembershipRequestDialogComponent } from '../dialogs/membership-request/membership-request-dialog.component';
import { DialogOutletComponent } from '../dialogs/outlet/dialog-outlet.component';
import { ResourceCreationDialogComponent } from '../dialogs/resource-creation/resource-creation-dialog.component';

@Component({
  selector: 'app-root',
  imports: [DialogOutletComponent, MembershipRequestDialogComponent, ResourceCreationDialogComponent, RouterOutlet, ToastOutletComponent],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  protected readonly title = signal('epos-backoffice-gui');
}
