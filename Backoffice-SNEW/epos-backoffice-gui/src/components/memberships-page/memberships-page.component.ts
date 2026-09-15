import { Component, DestroyRef, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { catchError, finalize, from, of, switchMap } from 'rxjs';

import type { UserMembershipSummary } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';

@Component({
  selector: 'app-memberships-page',
  imports: [DatePipe],
  templateUrl: './memberships-page.component.html',
  styleUrl: './memberships-page.component.css'
})
export class MembershipsPageComponent {
  private readonly apiService = inject(ApiService);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly destroyRef = inject(DestroyRef);
  private readonly dialog = inject(DialogService);
  private readonly toast = inject(ToastService);
  protected readonly requestDialog = inject(MembershipRequestDialogService);
  protected readonly user = this.currentUserStore.user;
  protected readonly loading = this.currentUserStore.loading;
  protected readonly error = this.currentUserStore.error;
  protected readonly groupIconPath = groupIconPath;
  protected readonly membershipMutation = signal('');
  protected readonly membershipActionError = signal('');

  protected pendingRoleNames(membership: UserMembershipSummary): string {
    const roles = membership.role_requests
      .filter((request) => request.status === 'PENDING')
      .map((request) => request.role);
    return roles.length ? roles.join(', ') : 'None';
  }

  protected async cancelMembershipRequest(membership: UserMembershipSummary): Promise<void> {
    if (this.membershipMutation() || membership.status !== 'PENDING') return;

    const confirmed = await this.dialog.confirm({
      eyebrow: 'Membership Request',
      title: 'Cancel membership request?',
      message: `Your pending request to join ${membership.group.name} will be cancelled.`,
      confirmLabel: 'Cancel Request',
      destructive: true
    });
    if (!confirmed) return;

    this.membershipMutation.set(membership.group.id);
    this.membershipActionError.set('');
    this.apiService
      .withdrawGroupMembershipRequest(membership.group.id)
      .pipe(
        switchMap(() => from(this.currentUserStore.refresh())),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.membershipMutation.set('')),
        catchError((error) => {
          console.error('Failed to remove membership', error);
          this.membershipActionError.set('Failed to cancel the membership request.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (result !== null) {
          this.toast.success('Membership request cancelled.');
        }
      });
  }
}
