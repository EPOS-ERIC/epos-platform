import { Component, inject } from '@angular/core';
import { DatePipe } from '@angular/common';

import type { UserMembershipSummary } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { CurrentUserStore } from '../../stores/current-user.store';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';

@Component({
  selector: 'app-memberships-page',
  imports: [DatePipe],
  templateUrl: './memberships-page.component.html',
  styleUrl: './memberships-page.component.css'
})
export class MembershipsPageComponent {
  private readonly currentUserStore = inject(CurrentUserStore);
  protected readonly requestDialog = inject(MembershipRequestDialogService);
  protected readonly user = this.currentUserStore.user;
  protected readonly loading = this.currentUserStore.loading;
  protected readonly error = this.currentUserStore.error;
  protected readonly groupIconPath = groupIconPath;

  protected pendingRoleNames(membership: UserMembershipSummary): string {
    const roles = membership.role_requests
      .filter((request) => request.status === 'PENDING')
      .map((request) => request.role);
    return roles.length ? roles.join(', ') : 'None';
  }
}
