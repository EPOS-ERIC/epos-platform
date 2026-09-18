import { Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { catchError, finalize, forkJoin, from, map, of, switchMap } from 'rxjs';

import type { CreateRoleRequest, UserMembershipSummary } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';

type RequestableRole = CreateRoleRequest['role'];

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
  protected readonly roleRequestGroupId = signal('');
  protected readonly selectedRoles = signal<RequestableRole[]>([]);
  protected readonly roleRequestSaving = signal(false);
  protected readonly roleRequestError = signal('');
  protected readonly requestableRoles: readonly RequestableRole[] = ['EDITOR', 'REVIEWER'];
  protected readonly roleRequestMembership = computed(() => {
    const groupId = this.roleRequestGroupId();
    return this.user()?.memberships.find((membership) => membership.group.id === groupId && membership.status === 'ACTIVE') ?? null;
  });

  protected pendingRoleNames(membership: UserMembershipSummary): string {
    const roles = membership.role_requests
      .filter((request) => request.status === 'PENDING')
      .map((request) => request.role);
    return roles.length ? roles.join(', ') : 'None';
  }

  protected async cancelMembershipRequest(membership: UserMembershipSummary): Promise<void> {
    if (membership.status !== 'PENDING') return;
    await this.confirmAndDeleteOwnMembership(membership, 'cancel');
  }

  protected async leaveOwnMembership(membership: UserMembershipSummary): Promise<void> {
    if (membership.status !== 'ACTIVE') return;
    await this.confirmAndDeleteOwnMembership(membership, 'leave');
  }

  protected openRoleRequestDialog(membership: UserMembershipSummary): void {
    if (membership.status !== 'ACTIVE') return;
    this.roleRequestGroupId.set(membership.group.id);
    this.selectedRoles.set([]);
    this.roleRequestError.set('');
  }

  protected closeRoleRequestDialog(): void {
    if (this.roleRequestSaving()) return;
    this.resetRoleRequestDialog();
  }

  protected toggleRequestedRole(membership: UserMembershipSummary, role: RequestableRole): void {
    if (this.roleRequestSaving() || this.hasRole(membership, role) || this.hasPendingRoleRequest(membership, role)) return;
    this.selectedRoles.update((roles) => roles.includes(role)
      ? roles.filter((selectedRole) => selectedRole !== role)
      : [...roles, role]
    );
  }

  protected hasRole(membership: UserMembershipSummary, role: RequestableRole): boolean {
    return membership.roles.includes(role);
  }

  protected hasPendingRoleRequest(membership: UserMembershipSummary, role: RequestableRole): boolean {
    return membership.role_requests.some((request) => request.role === role && request.status === 'PENDING');
  }

  protected submitRoleRequests(): void {
    const membership = this.roleRequestMembership();
    if (!membership || this.roleRequestSaving()) return;

    const roles = this.selectedRoles().filter((role) =>
      !this.hasRole(membership, role) && !this.hasPendingRoleRequest(membership, role)
    );
    if (!roles.length) return;

    this.roleRequestSaving.set(true);
    this.roleRequestError.set('');
    forkJoin(roles.map((role) => this.apiService.requestGroupRole(membership.group.id, { role }).pipe(
      map(() => ({ role, succeeded: true as const })),
      catchError((error) => {
        console.error(`Failed to request ${role} role`, error);
        return of({ role, succeeded: false as const });
      })
    )))
      .pipe(
        switchMap((results) => from(this.currentUserStore.refresh()).pipe(map(() => results))),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.roleRequestSaving.set(false))
      )
      .subscribe((results) => {
        const failedRoles = results.filter((result) => !result.succeeded).map((result) => result.role);
        const submittedCount = results.length - failedRoles.length;

        if (submittedCount) {
          this.toast.success(submittedCount === 1 ? 'Role request submitted.' : `${submittedCount} role requests submitted.`);
        }
        if (failedRoles.length) {
          this.selectedRoles.set(failedRoles);
          this.roleRequestError.set(`Failed to request: ${failedRoles.join(', ')}.`);
          return;
        }

        this.resetRoleRequestDialog();
      });
  }

  private resetRoleRequestDialog(): void {
    this.roleRequestGroupId.set('');
    this.selectedRoles.set([]);
    this.roleRequestError.set('');
  }

  private async confirmAndDeleteOwnMembership(
    membership: UserMembershipSummary,
    action: 'cancel' | 'leave'
  ): Promise<void> {
    if (this.membershipMutation()) return;

    const pending = action === 'cancel';

    const confirmed = await this.dialog.confirm({
      eyebrow: pending ? 'Membership Request' : 'Group Membership',
      title: pending ? 'Cancel membership request?' : 'Leave this group?',
      message: pending
        ? `Your pending request to join ${membership.group.name} will be cancelled.`
        : `Your membership and assigned roles in ${membership.group.name} will be removed.`,
      confirmLabel: pending ? 'Cancel Request' : 'Leave Group',
      destructive: true
    });
    if (!confirmed) return;

    this.membershipMutation.set(membership.group.id);
    this.membershipActionError.set('');
    const request = pending
      ? this.apiService.withdrawGroupMembershipRequest(membership.group.id)
      : this.apiService.leaveOwnGroupMembership(membership.group.id);

    request
      .pipe(
        switchMap(() => from(this.currentUserStore.refresh())),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.membershipMutation.set('')),
        catchError((error) => {
          console.error('Failed to remove membership', error);
          this.membershipActionError.set(pending
            ? 'Failed to cancel the membership request.'
            : 'Failed to leave the group.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (result !== null) {
          this.toast.success(pending ? 'Membership request cancelled.' : 'Group membership removed.');
        }
      });
  }
}
