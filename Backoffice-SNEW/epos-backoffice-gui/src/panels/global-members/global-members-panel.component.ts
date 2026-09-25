import { DatePipe } from '@angular/common';
import { Component, DestroyRef, HostListener, OnInit, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { EMPTY, catchError, finalize, forkJoin, map, of, switchMap } from 'rxjs';

import type { GlobalNotificationTopicKey, GroupPermission, MembershipRole, NotificationSubscription, NotificationTopic, Permission, User, UserListItem, UserMembershipSummary } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { isGroupPermissionCatalogItem, type GroupPermissionCatalogItem } from '../../models/resource-actions';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GroupMembershipAccessEditorComponent } from '../../shared/access/group-membership-access-editor.component';
import { AppIconComponent } from '../../shared/ui/app-icon.component';

type GlobalNotificationTopic = NotificationTopic & {
  key: GlobalNotificationTopicKey;
  scope: 'GLOBAL';
};

@Component({
  selector: 'app-global-members-panel',
  imports: [AppIconComponent, DatePipe, FormsModule, GroupMembershipAccessEditorComponent],
  templateUrl: './global-members-panel.component.html',
  styleUrl: './global-members-panel.component.css'
})
export class GlobalMembersPanelComponent implements OnInit {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly dialog = inject(DialogService);
  private readonly toast = inject(ToastService);

  protected readonly canManageMemberships = computed(() =>
    this.currentUserStore.hasGlobalPermission('global.membership.manage')
  );
  protected readonly canManageRoles = computed(() =>
    this.currentUserStore.hasGlobalPermission('global.role.manage')
  );
  protected readonly canManagePermissions = computed(() =>
    this.currentUserStore.hasGlobalPermission('global.permission.manage')
  );
  protected readonly canManageNotifications = computed(() =>
    this.currentUserStore.hasGlobalPermission('global.notification.manage')
  );
  protected readonly canManageMembers = computed(() =>
    this.canManageMemberships() || this.canManageRoles() || this.canManagePermissions() || this.canManageNotifications()
  );
  protected readonly users = signal<UserListItem[]>([]);
  protected readonly usersLoading = signal(true);
  protected readonly usersError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly selectedUserSummary = signal<UserListItem | null>(null);
  protected readonly selectedUser = signal<User | null>(null);
  protected readonly selectedUserLoading = signal(false);
  protected readonly membershipError = signal('');
  protected readonly membershipMutation = signal('');
  protected readonly removingMembershipGroupId = signal('');
  protected readonly notificationTopics = signal<GlobalNotificationTopic[]>([]);
  protected readonly notificationSubscriptions = signal<NotificationSubscription[]>([]);
  protected readonly notificationMutation = signal('');
  protected readonly notificationError = signal('');
  protected readonly groupIconPath = groupIconPath;
  protected readonly groupRoles: MembershipRole[] = ['EDITOR', 'REVIEWER', 'GROUP_ADMIN'];
  protected readonly groupPermissions = signal<GroupPermissionCatalogItem[]>([]);
  protected readonly filteredUsers = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    return query
      ? this.users().filter((user) => `${user.given_name} ${user.family_name} ${user.email}`.toLowerCase().includes(query))
      : this.users();
  });

  ngOnInit(): void {
    if (this.canManagePermissions()) this.loadPermissions();
    if (this.canManageMembers()) this.loadUsers();
  }

  private loadPermissions(): void {
    this.apiService.listPermissions().pipe(
      takeUntilDestroyed(this.destroyRef),
      catchError((error) => {
        console.error('Failed to load permission catalog', error);
        return of([] as Permission[]);
      })
    ).subscribe((permissions) => this.groupPermissions.set(
      permissions.filter(isGroupPermissionCatalogItem)
    ));
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    this.closeUserMemberships();
  }

  protected openUserMemberships(user: UserListItem): void {
    this.selectedUserSummary.set(user);
    this.selectedUser.set(null);
    this.membershipError.set('');
    this.selectedUserLoading.set(true);

    this.notificationError.set('');
    const request = this.canManageNotifications()
      ? forkJoin({
          user: this.apiService.getUser(user.id),
          topics: this.apiService.listNotificationTopics().pipe(catchError((error) => {
            console.error('Failed to load global notification topics', error);
            this.notificationError.set('Failed to load notification settings for this user.');
            return of([] as NotificationTopic[]);
          })),
          subscriptions: this.apiService.listUserNotificationSubscriptions(user.id).pipe(catchError((error) => {
            console.error('Failed to load global notification subscriptions', error);
            this.notificationError.set('Failed to load notification settings for this user.');
            return of([] as NotificationSubscription[]);
          }))
        })
      : this.apiService.getUser(user.id).pipe(map((selectedUser) => ({
          user: selectedUser,
          topics: [] as NotificationTopic[],
          subscriptions: [] as NotificationSubscription[]
        })));

    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.selectedUserLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load user memberships', error);
          this.membershipError.set('Failed to load memberships for this user.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (!result) return;
        this.selectedUser.set(result.user);
        this.notificationTopics.set(result.topics.filter(isGlobalNotificationTopic));
        this.notificationSubscriptions.set(result.subscriptions);
      });
  }

  protected closeUserMemberships(): void {
    if (this.membershipMutation() || this.notificationMutation() || this.removingMembershipGroupId()) return;

    this.selectedUserSummary.set(null);
    this.selectedUser.set(null);
    this.membershipError.set('');
    this.notificationTopics.set([]);
    this.notificationSubscriptions.set([]);
    this.notificationError.set('');
  }

  protected canReceiveNotification(user: User, topic: GlobalNotificationTopic): boolean {
    const effectivePermissions = new Set<string>([
      ...user.global_granted_permissions,
      ...user.global_role_permissions
    ]);
    return !topic.required_permissions.length || topic.required_permissions.some((permission) => effectivePermissions.has(permission));
  }

  protected isNotificationSubscribed(topic: GlobalNotificationTopicKey): boolean {
    return this.notificationSubscriptions().some((subscription) =>
      !subscription.group_id && subscription.topic_key === topic && subscription.channel === 'EMAIL'
    );
  }

  protected toggleNotification(user: User, topic: GlobalNotificationTopic): void {
    if (!this.canManageNotifications() || !this.canReceiveNotification(user, topic) || this.notificationMutation()) return;
    const enabled = !this.isNotificationSubscribed(topic.key);
    this.notificationMutation.set(topic.key);
    this.notificationError.set('');
    this.apiService.setUserNotificationSubscription(user.id, topic.key, enabled).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.notificationMutation.set('')),
      catchError((error) => {
        console.error('Failed to update global notification subscription', error);
        this.notificationError.set('Failed to update this notification subscription.');
        return EMPTY;
      })
    ).subscribe(() => {
      this.notificationSubscriptions.update((subscriptions) => enabled
        ? [...subscriptions, { user_id: user.id, topic_key: topic.key, channel: 'EMAIL' }]
        : subscriptions.filter((subscription) => subscription.group_id || subscription.topic_key !== topic.key || subscription.channel !== 'EMAIL'));
    });
  }

  protected toggleRole(membership: UserMembershipSummary, role: MembershipRole): void {
    const user = this.selectedUser();
    if (!this.canManageRoles() || !user || membership.status !== 'ACTIVE' || this.membershipMutation()) return;

    const enabled = !membership.roles.includes(role);
    this.runMembershipMutation(
      `${membership.group.id}:role:${role}`,
      this.apiService.setGroupMembershipRole(membership.group.id, user.id, role, enabled)
    );
  }

  protected togglePermission(membership: UserMembershipSummary, permission: GroupPermission): void {
    const user = this.selectedUser();
    if (!this.canManagePermissions() || !user || membership.status !== 'ACTIVE' || this.membershipMutation()) return;

    const enabled = !membership.granted_permissions.includes(permission);
    this.runMembershipMutation(
      `${membership.group.id}:permission:${permission}`,
      this.apiService.setGroupMembershipPermission(membership.group.id, user.id, permission, enabled)
    );
  }

  protected async removeMembership(membership: UserMembershipSummary): Promise<void> {
    const user = this.selectedUser();
    if (!this.canManageMemberships() || !user || membership.status !== 'ACTIVE' || membership.roles.includes('GROUP_ADMIN') || this.removingMembershipGroupId()) return;

    const confirmed = await this.dialog.confirm({
      eyebrow: 'Group Membership',
      title: 'Remove this membership?',
      message: `${this.userName(user)} will lose access to ${membership.group.name}.`,
      confirmLabel: 'Remove',
      destructive: true
    });
    if (!confirmed) return;

    this.removingMembershipGroupId.set(membership.group.id);
    this.membershipError.set('');
    this.apiService.removeGroupMember(membership.group.id, user.id).pipe(
      switchMap(() => this.apiService.getUser(user.id)),
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.removingMembershipGroupId.set('')),
      catchError((error) => {
        console.error('Failed to remove group membership', error);
        this.membershipError.set('Failed to remove this group membership.');
        return of(null);
      })
    ).subscribe((updatedUser) => {
      if (!updatedUser) return;
      this.selectedUser.set(updatedUser);
      this.toast.success('Group membership removed.');
    });
  }

  protected userName(user: Pick<UserListItem, 'given_name' | 'family_name'>): string {
    return `${user.given_name} ${user.family_name}`.trim() || 'Unnamed user';
  }

  private loadUsers(): void {
    this.usersLoading.set(true);
    this.usersError.set('');

    this.apiService
      .getUsers()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.usersLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load users', error);
          this.usersError.set('Failed to load users. This page requires administrator access.');
          return of([] as UserListItem[]);
        })
      )
      .subscribe((users) => this.users.set(users));
  }

  private runMembershipMutation(key: string, request: ReturnType<ApiService['setGroupMembershipRole']>): void {
    const userId = this.selectedUser()?.id;
    if (!userId) return;

    this.membershipMutation.set(key);
    this.membershipError.set('');

    request
      .pipe(
        switchMap(() => this.apiService.getUser(userId)),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.membershipMutation.set('')),
        catchError((error) => {
          console.error('Failed to update group membership', error);
          this.membershipError.set('Failed to update this membership.');
          return of(null);
        })
      )
      .subscribe((user) => {
        if (user) this.selectedUser.set(user);
      });
  }
}

function isGlobalNotificationTopic(topic: NotificationTopic): topic is GlobalNotificationTopic {
  return topic.scope === 'GLOBAL' && topic.key.startsWith('global.');
}
