import { DatePipe } from '@angular/common';
import { Component, DestroyRef, HostListener, OnInit, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { EMPTY, catchError, finalize, forkJoin, map, of, switchMap } from 'rxjs';

import type { GlobalNotificationTopicKey, GlobalPermission, GroupNotificationTopicKey, GroupPermission, MembershipRole, NotificationSubscription, NotificationTopic, Permission, User, UserListItem, UserMembershipSummary } from '../../../generated/backofficeSchemas';
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

type GroupNotificationTopic = NotificationTopic & {
  key: GroupNotificationTopicKey;
  scope: 'GROUP';
};

type GlobalPermissionCatalogItem = Omit<Permission, 'key'> & {
  key: GlobalPermission;
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
  protected readonly canChangeUserStatus = computed(() => {
    const currentUser = this.currentUserStore.user();
    return currentUser?.status === 'ACTIVE' && currentUser.global_roles.includes('ADMIN');
  });
  protected readonly canManageMembers = computed(() =>
    this.canManageMemberships() || this.canManageRoles() || this.canManagePermissions() || this.canManageNotifications() || this.canChangeUserStatus()
  );
  protected readonly users = signal<UserListItem[]>([]);
  protected readonly usersLoading = signal(true);
  protected readonly usersError = signal('');
  protected readonly userStatusError = signal('');
  protected readonly userStatusMutation = signal('');
  protected readonly searchQuery = signal('');
  protected readonly selectedUserSummary = signal<UserListItem | null>(null);
  protected readonly selectedUser = signal<User | null>(null);
  protected readonly selectedUserLoading = signal(false);
  protected readonly membershipError = signal('');
  protected readonly globalAccessMutation = signal('');
  protected readonly globalAccessError = signal('');
  protected readonly membershipMutation = signal('');
  protected readonly removingMembershipGroupId = signal('');
  protected readonly notificationTopics = signal<GlobalNotificationTopic[]>([]);
  protected readonly groupNotificationTopics = signal<GroupNotificationTopic[]>([]);
  protected readonly notificationSubscriptions = signal<NotificationSubscription[]>([]);
  protected readonly notificationMutation = signal('');
  protected readonly notificationError = signal('');
  protected readonly groupNotificationErrors = signal<Record<string, string>>({});
  protected readonly groupIconPath = groupIconPath;
  protected readonly groupRoles: MembershipRole[] = ['EDITOR', 'REVIEWER', 'GROUP_ADMIN'];
  protected readonly globalPermissions = signal<GlobalPermissionCatalogItem[]>([]);
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
    ).subscribe((permissions) => {
      this.globalPermissions.set(permissions.filter(isGlobalPermissionCatalogItem));
      this.groupPermissions.set(permissions.filter(isGroupPermissionCatalogItem));
    });
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    this.closeUserMemberships();
  }

  protected openUserMemberships(user: UserListItem): void {
    this.selectedUserSummary.set(user);
    this.selectedUser.set(null);
    this.membershipError.set('');
    this.globalAccessError.set('');
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
        this.groupNotificationTopics.set(result.topics.filter(isGroupNotificationTopic));
        this.notificationSubscriptions.set(result.subscriptions);
      });
  }

  protected canChangeStatusFor(user: UserListItem): boolean {
    return this.canChangeUserStatus() && this.currentUserStore.user()?.id !== user.id;
  }

  protected async blockUser(user: UserListItem): Promise<void> {
    if (!this.canChangeStatusFor(user) || user.status !== 'ACTIVE' || this.userStatusMutation()) return;

    const reason = await this.dialog.prompt({
      eyebrow: 'User Access',
      title: `Block ${this.userName(user)}?`,
      message: 'This user will no longer be able to access the platform. Existing memberships and permissions will remain stored.',
      inputLabel: 'Reason',
      placeholder: 'Explain why this user is being blocked',
      confirmLabel: 'Block User',
      destructive: true,
      required: true
    });
    if (!reason?.trim()) return;

    this.runUserStatusMutation(user.id, 'block', this.apiService.blockUser(user.id, { reason: reason.trim() }));
  }

  protected async unblockUser(user: UserListItem): Promise<void> {
    if (!this.canChangeStatusFor(user) || user.status !== 'BLOCKED' || this.userStatusMutation()) return;

    const confirmed = await this.dialog.confirm({
      eyebrow: 'User Access',
      title: `Unblock ${this.userName(user)}?`,
      message: 'This user will regain access to the platform with their stored memberships and permissions.',
      confirmLabel: 'Unblock User'
    });
    if (!confirmed) return;

    this.runUserStatusMutation(user.id, 'unblock', this.apiService.unblockUser(user.id));
  }

  protected closeUserMemberships(): void {
    if (this.globalAccessMutation() || this.membershipMutation() || this.notificationMutation() || this.removingMembershipGroupId()) return;

    this.selectedUserSummary.set(null);
    this.selectedUser.set(null);
    this.membershipError.set('');
    this.globalAccessError.set('');
    this.notificationTopics.set([]);
    this.groupNotificationTopics.set([]);
    this.notificationSubscriptions.set([]);
    this.notificationError.set('');
    this.groupNotificationErrors.set({});
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
    this.notificationMutation.set(`global:${topic.key}`);
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

  protected groupNotificationSubscriptionCount(groupId: string): number {
    return this.groupNotificationTopics().filter((topic) => this.isGroupNotificationSubscribed(groupId, topic.key)).length;
  }

  protected isGroupNotificationSubscribed(groupId: string, topic: GroupNotificationTopicKey): boolean {
    return this.notificationSubscriptions().some((subscription) =>
      subscription.group_id === groupId && subscription.topic_key === topic && subscription.channel === 'EMAIL'
    );
  }

  protected canReceiveGroupNotification(
    user: User,
    membership: UserMembershipSummary,
    topic: GroupNotificationTopic
  ): boolean {
    const groupPermissions = new Set<string>([
      ...membership.granted_permissions,
      ...membership.role_permissions
    ]);
    const globalPermissions = new Set<string>([
      ...user.global_granted_permissions,
      ...user.global_role_permissions
    ]);
    return !topic.required_permissions.length || topic.required_permissions.some((permission) =>
      groupPermissions.has(permission) || globalPermissions.has(permission.replace(/^group\./, 'global.'))
    );
  }

  protected toggleGroupNotification(
    user: User,
    membership: UserMembershipSummary,
    topic: GroupNotificationTopic
  ): void {
    if (
      !this.canManageNotifications()
      || membership.status !== 'ACTIVE'
      || !this.canReceiveGroupNotification(user, membership, topic)
      || this.notificationMutation()
    ) return;

    const groupId = membership.group.id;
    const enabled = !this.isGroupNotificationSubscribed(groupId, topic.key);
    this.notificationMutation.set(`${groupId}:${topic.key}`);
    this.groupNotificationErrors.update((errors) => ({ ...errors, [groupId]: '' }));
    this.apiService.setGroupUserNotificationSubscription(groupId, user.id, topic.key, enabled).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.notificationMutation.set('')),
      catchError((error) => {
        console.error('Failed to update group notification subscription', error);
        this.groupNotificationErrors.update((errors) => ({
          ...errors,
          [groupId]: 'Failed to update this notification subscription.'
        }));
        return EMPTY;
      })
    ).subscribe(() => {
      this.notificationSubscriptions.update((subscriptions) => enabled
        ? [...subscriptions, {
            user_id: user.id,
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

  protected toggleGlobalRole(): void {
    const user = this.selectedUser();
    if (!this.canChangeUserStatus() || !user || this.globalAccessMutation()) return;

    const enabled = !user.global_roles.includes('ADMIN');
    this.runGlobalAccessMutation(
      'role:ADMIN',
      this.apiService.setGlobalRole(user.id, 'ADMIN', enabled),
      enabled ? 'Global administrator role assigned.' : 'Global administrator role removed.',
      true
    );
  }

  protected toggleGlobalPermission(permission: GlobalPermission): void {
    const user = this.selectedUser();
    if (!this.canManagePermissions() || !user || this.globalAccessMutation()) return;

    const enabled = !user.global_granted_permissions.includes(permission);
    this.runGlobalAccessMutation(
      `permission:${permission}`,
      this.apiService.setGlobalPermission(user.id, permission, enabled),
      enabled ? 'Global permission granted.' : 'Global permission revoked.'
    );
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

  private runUserStatusMutation(
    userId: string,
    action: 'block' | 'unblock',
    request: ReturnType<ApiService['blockUser']>
  ): void {
    this.userStatusMutation.set(`${userId}:${action}`);
    this.userStatusError.set('');
    request.pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.userStatusMutation.set('')),
      catchError((error) => {
        console.error(`Failed to ${action} user`, error);
        this.userStatusError.set(`Failed to ${action} this user.`);
        return EMPTY;
      })
    ).subscribe(() => {
      const status = action === 'block' ? 'BLOCKED' : 'ACTIVE';
      this.users.update((users) => users.map((user) => user.id === userId ? { ...user, status } : user));
      this.toast.success(action === 'block' ? 'User blocked.' : 'User unblocked.');
    });
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

  private runGlobalAccessMutation(
    key: string,
    request: ReturnType<ApiService['setGlobalRole']>,
    successMessage: string,
    refreshNotificationSubscriptions = false
  ): void {
    const userId = this.selectedUser()?.id;
    if (!userId) return;

    this.globalAccessMutation.set(key);
    this.globalAccessError.set('');
    request.pipe(
      switchMap(() => forkJoin({
        user: this.apiService.getUser(userId),
        subscriptions: refreshNotificationSubscriptions && this.canManageNotifications()
          ? this.apiService.listUserNotificationSubscriptions(userId)
          : of(null)
      })),
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.globalAccessMutation.set('')),
      catchError((error) => {
        console.error('Failed to update global access', error);
        this.globalAccessError.set('Failed to update global access.');
        return of(null);
      })
    ).subscribe((result) => {
      if (!result) return;
      const { user, subscriptions } = result;
      this.selectedUser.set(user);
      if (subscriptions) this.notificationSubscriptions.set(subscriptions);
      this.users.update((users) => users.map((item) => item.id === user.id ? {
        ...item,
        global_roles: user.global_roles,
        global_granted_permissions: user.global_granted_permissions,
        global_role_permissions: user.global_role_permissions,
        updated_at: user.updated_at
      } : item));
      this.toast.success(successMessage);
    });
  }
}

function isGlobalPermissionCatalogItem(permission: Permission): permission is GlobalPermissionCatalogItem {
  return permission.scope === 'GLOBAL' && permission.key.startsWith('global.');
}

function isGlobalNotificationTopic(topic: NotificationTopic): topic is GlobalNotificationTopic {
  return topic.scope === 'GLOBAL' && topic.key.startsWith('global.');
}

function isGroupNotificationTopic(topic: NotificationTopic): topic is GroupNotificationTopic {
  return topic.scope === 'GROUP' && topic.key.startsWith('group.');
}
