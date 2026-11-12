import { Component, DestroyRef, computed, effect, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { catchError, EMPTY, finalize, forkJoin, from, map, of, switchMap } from 'rxjs';

import type { CreateRoleRequest, GroupNotificationTopicKey, GroupPermission, NotificationSubscription, NotificationTopic, Permission, UserMembershipSummary } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { isGroupPermissionCatalogItem, type GroupPermissionCatalogItem } from '../../models/resource-actions';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GroupsStore } from '../../stores/groups.store';

type RequestableRole = CreateRoleRequest['role'];
type GroupNotificationTopic = NotificationTopic & {
  key: GroupNotificationTopicKey;
  scope: 'GROUP';
};

@Component({
  selector: 'app-memberships-page',
  imports: [DatePipe, RouterLink],
  templateUrl: './memberships-page.component.html',
  styleUrl: './memberships-page.component.css'
})
export class MembershipsPageComponent {
  private readonly apiService = inject(ApiService);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly groupsStore = inject(GroupsStore);
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
  protected readonly groupPermissions = signal<GroupPermissionCatalogItem[]>([]);
  protected readonly notificationTopics = signal<GroupNotificationTopic[]>([]);
  protected readonly notificationSubscriptions = signal<NotificationSubscription[]>([]);
  protected readonly notificationsLoading = signal(false);
  protected readonly notificationMutation = signal('');
  protected readonly notificationError = signal('');
  protected readonly requestableRoles: readonly RequestableRole[] = ['EDITOR', 'REVIEWER'];
  private loadedNotificationUserId = '';
  protected readonly roleRequestMembership = computed(() => {
    const groupId = this.roleRequestGroupId();
    return this.user()?.memberships.find((membership) => membership.group.id === groupId && membership.status === 'ACTIVE') ?? null;
  });

  constructor() {
    this.apiService.listPermissions().pipe(
      takeUntilDestroyed(this.destroyRef),
      catchError((error) => {
        console.error('Failed to load permission catalog', error);
        return of([] as Permission[]);
      })
    ).subscribe((permissions) => this.groupPermissions.set(
      permissions.filter(isGroupPermissionCatalogItem)
    ));

    effect(() => {
      const userId = this.user()?.id ?? '';
      if (userId && userId !== this.loadedNotificationUserId) {
        this.loadNotificationSettings(userId);
      }
    });
  }

  protected permissionDetails(permission: GroupPermission): GroupPermissionCatalogItem | undefined {
    return this.groupPermissions().find((item) => item.key === permission);
  }

  protected pendingRoleNames(membership: UserMembershipSummary): string {
    const roles = membership.role_requests
      .filter((request) => request.status === 'PENDING')
      .map((request) => request.role);
    return roles.length ? roles.join(', ') : 'None';
  }

  protected groupRoute(membership: UserMembershipSummary): string | null {
    const concept = this.groupsStore.conceptName(
      this.groupsStore.groups().find((group) => group.id === membership.group.id)
    );
    switch (concept) {
      case 'DATA':
        return '/data';
      case 'SOFTWARE':
        return '/software';
      case 'FACILITIES':
        return '/facilities';
      default:
        return null;
    }
  }

  protected notificationSubscriptionCount(groupId: string): number {
    return this.notificationTopics().filter((topic) => this.isNotificationSubscribed(groupId, topic.key)).length;
  }

  protected isNotificationSubscribed(groupId: string, topic: GroupNotificationTopicKey): boolean {
    return this.notificationSubscriptions().some((subscription) =>
      subscription.group_id === groupId && subscription.topic_key === topic && subscription.channel === 'EMAIL'
    );
  }

  protected canReceiveNotification(membership: UserMembershipSummary, topic: GroupNotificationTopic): boolean {
    const permissions = this.currentUserStore.groupPermissions(membership.group.id);
    return !topic.required_permissions.length || topic.required_permissions.some((permission) =>
      permissions.includes(permission as GroupPermission)
    );
  }

  protected toggleNotification(membership: UserMembershipSummary, topic: GroupNotificationTopic): void {
    const userId = this.user()?.id;
    if (!userId || membership.status !== 'ACTIVE' || !this.canReceiveNotification(membership, topic) || this.notificationMutation()) return;

    const groupId = membership.group.id;
    const enabled = !this.isNotificationSubscribed(groupId, topic.key);
    this.notificationMutation.set(`${groupId}:${topic.key}`);
    this.notificationError.set('');
    this.apiService.setGroupUserNotificationSubscription(groupId, userId, topic.key, enabled).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.notificationMutation.set('')),
      catchError((error) => {
        console.error('Failed to update group notification subscription', error);
        this.notificationError.set('Failed to update this notification subscription.');
        return EMPTY;
      })
    ).subscribe(() => {
      this.notificationSubscriptions.update((subscriptions) => enabled
        ? [...subscriptions, {
            user_id: userId,
            group_id: groupId,
            topic_key: topic.key,
            channel: 'EMAIL'
          }]
        : subscriptions.filter((subscription) =>
            subscription.group_id !== groupId || subscription.topic_key !== topic.key || subscription.channel !== 'EMAIL'
          )
      );
    });
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

  private loadNotificationSettings(userId: string): void {
    this.loadedNotificationUserId = userId;
    this.notificationsLoading.set(true);
    this.notificationError.set('');
    forkJoin({
      topics: this.apiService.listNotificationTopics(),
      subscriptions: this.apiService.listUserNotificationSubscriptions(userId)
    }).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.notificationsLoading.set(false)),
      catchError((error) => {
        console.error('Failed to load notification settings', error);
        this.notificationError.set('Failed to load notification settings.');
        return of(null);
      })
    ).subscribe((result) => {
      if (!result || this.user()?.id !== userId) return;
      this.notificationTopics.set(result.topics.filter(isGroupNotificationTopic));
      this.notificationSubscriptions.set(result.subscriptions);
    });
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

function isGroupNotificationTopic(topic: NotificationTopic): topic is GroupNotificationTopic {
  return topic.scope === 'GROUP' && topic.key.startsWith('group.');
}
