import { Component, HostListener, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { groupIconPath } from '../../group-icons';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';

@Component({
  selector: 'app-membership-request-dialog',
  imports: [FormsModule],
  templateUrl: './membership-request-dialog.component.html',
  styleUrl: './membership-request-dialog.component.css'
})
export class MembershipRequestDialogComponent {
  protected readonly dialog = inject(MembershipRequestDialogService);

  protected selectedGroupIconPath(): string | undefined {
    return groupIconPath(this.dialog.groupId());
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    this.dialog.close();
  }
}
