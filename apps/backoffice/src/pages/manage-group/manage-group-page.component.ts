import { DatePipe } from '@angular/common';
import { Component, DestroyRef, HostListener, Input, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { catchError, EMPTY, finalize, forkJoin, Observable, of, switchMap } from 'rxjs';

import type { Group, GroupMembership, GroupNotificationTopicKey, GroupPermission, MembershipRole, NotificationSubscription, NotificationTopic, Permission } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { isGroupPermissionCatalogItem, type GroupPermissionCatalogItem } from '../../models/resource-actions';
import { ApiService } from '../../services/api.service';
import { CapabilitiesService } from '../../services/capabilities.service';
import { DialogService } from '../../services/dialog.service';
import { ToastService } from '../../services/toast.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import {
  MembershipRequestsPanelComponent,
  type MembershipRequestDecision,
  type MembershipRequestItem,
  type MembershipRequestRoleToggle
} from '../../panels/access-requests/membership-requests-panel.component';
import {
  RoleRequestsPanelComponent,
  type RoleRequestDecision,
  type RoleRequestItem
} from '../../panels/access-requests/role-requests-panel.component';
import { CategoriesPanelComponent } from '../../panels/categories/categories-panel.component';
import { GroupMembershipAccessEditorComponent } from '../../shared/access/group-membership-access-editor.component';
import { AppIconComponent, type IconName } from '../../shared/ui/app-icon.component';

type ManageGroupPanel = 'membership-requests' | 'member-access' | 'categories';

type GroupNotificationTopic = NotificationTopic & {
  key: GroupNotificationTopicKey;
  scope: 'GROUP';
};

type ManageGroupDestination = {
  panel: ManageGroupPanel;
  title: string;
  description: string;
  icon: IconName;
};

const MANAGE_GROUP_DESTINATIONS: readonly ManageGroupDestination[] = [
  { panel: 'membership-requests', title: 'Requests', description: 'Review requests to join this group or receive a new role.', icon: 'inbox' },
  { panel: 'member-access', title: 'Members', description: 'Manage active members, roles, direct permissions, and notification subscriptions.', icon: 'users' },
  { panel: 'categories', title: 'Group Categories', description: 'Maintain this group\'s category hierarchy.', icon: 'tags' }
];

@Component({
  selector: 'app-manage-group-page',
  imports: [AppIconComponent, CategoriesPanelComponent, DatePipe, FormsModule, GroupMembershipAccessEditorComponent, MembershipRequestsPanelComponent, RoleRequestsPanelComponent, RouterLink],
  templateUrl: './manage-group-page.component.html',
  styleUrl: './manage-group-page.component.css'
})
export class ManageGroupPageComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly dialog = inject(DialogService);
  private readonly toast = inject(ToastService);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly capabilitiesService = inject(CapabilitiesService);
  private currentGroupId = '';
  private membershipsLoadedGroupId = '';
  private readonly groupIdValue = signal('');
  private readonly groupList = signal<Group[]>([]);

  @Input()
  set groupId(value: string) {
    if (!value || value === this.currentGroupId) return;
    this.currentGroupId = value;
    this.groupIdValue.set(value);
    this.memberships.set([]);
    this.membershipRequestRoles.set({});
    this.membershipsLoadedGroupId = '';
    this.selectedMembership.set(null);
    this.loadMembershipsForActivePanel();
  }

  get groupId(): string {
    return this.currentGroupId;
  }

  @Input() groupName = '';
  @Input() set panel(value: string) {
    this.activePanel.set(this.isManageGroupPanel(value) ? value : '');
    this.loadMembershipsForActivePanel();
  }
  @Input() set groups(value: Group[]) {
    this.groupList.set(value);
  }

  protected readonly memberships = signal<GroupMembership[]>([]);
  protected readonly activePanel = signal<ManageGroupPanel | ''>('');
  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly searchQuery = signal('');
  protected readonly selectedMembership = signal<GroupMembership | null>(null);
  protected readonly membershipMutation = signal('');
  protected readonly membershipRequestDecision = signal('');
  protected readonly membershipRequestRoles = signal<Record<string, MembershipRole[]>>({});
  protected readonly roleRequestDecision = signal('');
  protected readonly requestError = signal('');
  protected readonly removingUserId = signal('');
  protected readonly notificationTopics = signal<GroupNotificationTopic[]>([]);
  protected readonly notificationSubscriptions = signal<NotificationSubscription[]>([]);
  protected readonly notificationsLoading = signal(false);
  protected readonly notificationMutation = signal('');
  protected readonly notificationError = signal('');
  protected readonly groupIconPath = groupIconPath;
  protected readonly requestableGroupRoles: MembershipRole[] = ['EDITOR', 'REVIEWER'];
  protected readonly groupRoles: MembershipRole[] = ['EDITOR', 'REVIEWER', 'GROUP_ADMIN'];
  protected readonly groupPermissions = signal<GroupPermissionCatalogItem[]>([]);
  protected readonly canManageMemberships = computed(() =>
    this.currentUserStore.hasGroupPermission(this.groupIdValue(), 'group.membership.manage')
  );
  protected readonly canManageRoles = computed(() =>
    this.currentUserStore.hasGroupPermission(this.groupIdValue(), 'group.role.manage')
  );
  protected readonly canManagePermissions = computed(() =>
    this.currentUserStore.hasGroupPermission(this.groupIdValue(), 'group.permission.manage')
  );
  protected readonly canManageCategories = computed(() =>
    this.currentUserStore.hasGroupPermission(this.groupIdValue(), 'group.category.manage')
  );
  protected readonly canManageNotifications = computed(() =>
    this.capabilitiesService.emailNotificationsEnabled()
      && this.currentUserStore.hasGroupPermission(this.groupIdValue(), 'group.notification.manage')
  );
  protected readonly canManageMembers = computed(() =>
    this.canManageMemberships() || this.canManageRoles() || this.canManagePermissions() || this.canManageNotifications()
  );
  protected readonly canManageRequests = computed(() =>
    this.canManageMemberships() || this.canManageRoles()
  );
  protected readonly selectedGroup = computed(() =>
    this.groupList().filter(({ id }) => id === this.groupIdValue())
  );
  protected readonly destinations = computed(() =>
    MANAGE_GROUP_DESTINATIONS.filter(({ panel }) => this.canAccessPanel(panel))
  );
  protected readonly showLanding = computed(() =>
    !this.activePanel() || !this.canAccessPanel(this.activePanel() as ManageGroupPanel)
  );
  protected readonly filteredMemberships = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const activeMemberships = this.memberships().filter((membership) => membership.status === 'ACTIVE');

    return query
      ? activeMemberships.filter((membership) =>
          `${this.userName(membership)} ${membership.user.email}`.toLowerCase().includes(query)
        )
      : activeMemberships;
  });
  protected readonly pendingMembershipRequests = computed<MembershipRequestItem[]>(() =>
    this.memberships()
      .filter((membership) => membership.status === 'PENDING')
      .map((membership) => {
        const requestedRoles = membership.role_requests
          .filter((request) => request.status === 'PENDING')
          .map((request) => request.role);
        const key = `${this.groupIdValue()}:${membership.user.id}`;
        return {
          key,
          groupId: this.groupIdValue(),
          groupName: this.groupName || 'This group',
          groupLogoPath: groupIconPath(this.groupIdValue()),
          userId: membership.user.id,
          userEmail: membership.user.email,
          userName: this.userName(membership),
          requestedAt: membership.created_at,
          requestedRoles,
          selectedRoles: this.membershipRequestRoles()[key] ?? requestedRoles,
          pendingRoleRequests: membership.role_requests
            .filter((request) => request.status === 'PENDING')
            .map((request) => ({ id: request.id, role: request.role }))
        };
      })
  );
  protected readonly pendingRoleRequests = computed<RoleRequestItem[]>(() =>
    this.memberships()
      .filter((membership) => membership.status === 'ACTIVE')
      .flatMap((membership) => membership.role_requests
        .filter((request) => request.status === 'PENDING')
        .map((request) => ({
          key: `${this.groupIdValue()}:${request.id}`,
          groupId: this.groupIdValue(),
          groupName: this.groupName || 'This group',
          groupLogoPath: groupIconPath(this.groupIdValue()),
          userId: membership.user.id,
          userEmail: membership.user.email,
          userName: this.userName(membership),
          requestId: request.id,
          role: request.role,
          requestedAt: request.requested_at
        })))
  );

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
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    this.closeMembership();
  }

  protected openMembership(membership: GroupMembership): void {
    if (!this.canManageRoles() && !this.canManagePermissions() && !this.canManageNotifications()) return;
    this.error.set('');
    this.selectedMembership.set(membership);
    if (this.canManageNotifications()) {
      this.loadNotificationSettings(membership.user.id);
    }
  }

  protected closeMembership(): void {
    if (!this.membershipMutation() && !this.notificationMutation() && !this.removingUserId()) {
      this.selectedMembership.set(null);
      this.notificationTopics.set([]);
      this.notificationSubscriptions.set([]);
      this.notificationError.set('');
    }
  }

  protected toggleRole(membership: GroupMembership, role: MembershipRole): void {
    if (!this.canManageRoles() || !this.groupRoles.includes(role) || this.membershipMutation()) return;
    const enabled = !membership.roles.includes(role);
    this.runMembershipMutation(
      `${this.groupId}:role:${role}`,
      this.apiService.setGroupMembershipRole(this.groupId, membership.user.id, role, enabled)
    );
  }

  protected togglePermission(membership: GroupMembership, permission: GroupPermission): void {
    if (!this.canManagePermissions() || this.membershipMutation()) return;
    const enabled = !membership.granted_permissions.includes(permission);
    this.runMembershipMutation(
      `${this.groupId}:permission:${permission}`,
      this.apiService.setGroupMembershipPermission(this.groupId, membership.user.id, permission, enabled)
    );
  }

  protected isNotificationSubscribed(topic: GroupNotificationTopicKey): boolean {
    return this.notificationSubscriptions().some((subscription) =>
      subscription.group_id === this.groupId && subscription.topic_key === topic && subscription.channel === 'EMAIL'
    );
  }

  protected canReceiveNotification(membership: GroupMembership, topic: GroupNotificationTopic): boolean {
    return !topic.required_permissions.length || topic.required_permissions.some((permission) =>
      membership.granted_permissions.includes(permission as GroupPermission)
      || membership.role_permissions.includes(permission as GroupPermission)
    );
  }

  protected toggleNotification(membership: GroupMembership, topic: GroupNotificationTopic): void {
    if (!this.canManageNotifications() || !this.canReceiveNotification(membership, topic) || this.notificationMutation()) return;
    const enabled = !this.isNotificationSubscribed(topic.key);
    this.notificationMutation.set(topic.key);
    this.notificationError.set('');
    this.apiService.setGroupUserNotificationSubscription(this.groupId, membership.user.id, topic.key, enabled).pipe(
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
            user_id: membership.user.id,
            group_id: this.groupId,
            topic_key: topic.key,
            channel: 'EMAIL'
          }]
        : subscriptions.filter((subscription) =>
            subscription.group_id !== this.groupId || subscription.topic_key !== topic.key || subscription.channel !== 'EMAIL'
          ));
    });
  }

  protected async removeMembership(membership: GroupMembership): Promise<void> {
    if (!this.canManageMemberships() || membership.roles.includes('GROUP_ADMIN') || this.removingUserId() || this.membershipMutation()) return;

    const confirmed = await this.dialog.confirm({
      eyebrow: 'Group Membership',
      title: 'Remove this member?',
      message: `${this.userName(membership)} will lose access to ${this.groupName || 'this group'}.`,
      confirmLabel: 'Remove',
      destructive: true
    });
    if (!confirmed) return;

    this.removingUserId.set(membership.user.id);
    this.error.set('');
    this.apiService
      .removeGroupMember(this.groupId, membership.user.id)
      .pipe(
        switchMap(() => this.apiService.getGroupMemberships(this.groupId)),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.removingUserId.set('')),
        catchError((error) => {
          console.error('Failed to remove group member', error);
          this.error.set('Failed to remove this group member.');
          return of(null);
        })
      )
      .subscribe((memberships) => {
        if (!memberships) return;
        this.memberships.set(memberships);
        this.selectedMembership.set(null);
        this.toast.success('Group member removed.');
      });
  }

  protected decideMembershipRequest({ item, decision }: MembershipRequestDecision): void {
    if (!this.canManageMemberships() || this.membershipRequestDecision()) return;

    this.membershipRequestDecision.set(item.key);
    this.requestError.set('');
    const request = decision === 'accept'
      ? this.acceptMembershipRequest(item)
      : this.apiService.decideGroupMembership(this.groupId, item.userId, 'reject');

    request.pipe(
      switchMap(() => this.apiService.getGroupMemberships(this.groupId)),
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.membershipRequestDecision.set('')),
      catchError((error) => {
        console.error('Failed to decide group membership request', error);
        this.requestError.set('Failed to update this membership request.');
        return of(null);
      })
    ).subscribe((memberships) => {
      if (!memberships) return;
      this.updateMemberships(memberships);
      this.membershipRequestRoles.update((roles) => {
        const remaining = { ...roles };
        delete remaining[item.key];
        return remaining;
      });
      this.toast.success(decision === 'accept' ? 'Membership request accepted.' : 'Membership request rejected.');
    });
  }

  protected toggleRequestedRole({ item, role }: MembershipRequestRoleToggle): void {
    if (!this.canManageRoles() || this.membershipRequestDecision()) return;

    const selectedRoles = item.selectedRoles.includes(role)
      ? item.selectedRoles.filter((selectedRole) => selectedRole !== role)
      : [...item.selectedRoles, role];
    this.membershipRequestRoles.update((roles) => ({ ...roles, [item.key]: selectedRoles }));
  }

  protected decideRoleRequest({ item, decision }: RoleRequestDecision): void {
    if (!this.canManageRoles() || this.roleRequestDecision()) return;

    this.roleRequestDecision.set(item.key);
    this.requestError.set('');
    this.apiService.decideGroupRoleRequest(this.groupId, item.requestId, decision).pipe(
      switchMap(() => this.apiService.getGroupMemberships(this.groupId)),
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.roleRequestDecision.set('')),
      catchError((error) => {
        console.error('Failed to decide group role request', error);
        this.requestError.set('Failed to update this role request.');
        return of(null);
      })
    ).subscribe((memberships) => {
      if (!memberships) return;
      this.updateMemberships(memberships);
      this.toast.success(decision === 'accept' ? 'Role request accepted.' : 'Role request rejected.');
    });
  }

  protected isPanelActive(panel: ManageGroupPanel): boolean {
    return !this.showLanding() && this.activePanel() === panel;
  }

  private canAccessPanel(panel: ManageGroupPanel): boolean {
    switch (panel) {
      case 'membership-requests': return this.canManageRequests();
      case 'member-access': return this.canManageMembers();
      case 'categories': return this.canManageCategories();
    }
  }

  private acceptMembershipRequest(item: MembershipRequestItem): Observable<unknown> {
    if (!this.canManageRoles()) {
      return this.apiService.decideGroupMembership(this.groupId, item.userId, 'accept');
    }

    return this.apiService.decideGroupMembership(this.groupId, item.userId, 'accept').pipe(
      switchMap(() => {
        const selectedRoles = new Set(item.selectedRoles);
        const requestedRoles = new Set(item.requestedRoles);
        const decisions: Observable<unknown>[] = item.pendingRoleRequests.map((roleRequest) =>
          this.apiService.decideGroupRoleRequest(
            this.groupId,
            roleRequest.id,
            selectedRoles.has(roleRequest.role) ? 'accept' : 'reject'
          )
        );

        for (const role of selectedRoles) {
          if (!requestedRoles.has(role)) {
            decisions.push(this.apiService.setGroupMembershipRole(this.groupId, item.userId, role, true));
          }
        }

        return decisions.length ? forkJoin(decisions) : of(item);
      })
    );
  }

  private isManageGroupPanel(value: string): value is ManageGroupPanel {
    return MANAGE_GROUP_DESTINATIONS.some(({ panel }) => panel === value);
  }

  private loadMembershipsForActivePanel(): void {
    const panel = this.activePanel();
    if (!this.groupId || this.membershipsLoadedGroupId === this.groupId) return;
    if (panel !== 'membership-requests' && panel !== 'member-access') return;
    if (!this.canAccessPanel(panel)) return;
    this.loadMemberships();
  }

  protected userName(membership: GroupMembership): string {
    return `${membership.user.given_name} ${membership.user.family_name}`.trim() || 'Unnamed user';
  }

  private loadMemberships(): void {
    this.membershipsLoadedGroupId = this.groupId;
    this.loading.set(true);
    this.error.set('');
    this.apiService
      .getGroupMemberships(this.groupId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false)),
        catchError((error) => {
          console.error('Failed to load group memberships', error);
          this.membershipsLoadedGroupId = '';
          this.error.set('Failed to load group members.');
          return of([] as GroupMembership[]);
        })
      )
      .subscribe((memberships) => this.updateMemberships(memberships));
  }

  private loadNotificationSettings(userId: string): void {
    this.notificationsLoading.set(true);
    this.notificationError.set('');
    forkJoin({
      topics: this.apiService.listNotificationTopics(),
      subscriptions: this.apiService.listUserNotificationSubscriptions(userId)
    }).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.notificationsLoading.set(false)),
      catchError((error) => {
        console.error('Failed to load group notification settings', error);
        this.notificationError.set('Failed to load notification settings for this member.');
        return of(null);
      })
    ).subscribe((result) => {
      if (!result || this.selectedMembership()?.user.id !== userId) return;
      this.notificationTopics.set(result.topics.filter(isGroupNotificationTopic));
      this.notificationSubscriptions.set(result.subscriptions);
    });
  }

  private runMembershipMutation(
    key: string,
    request: ReturnType<ApiService['setGroupMembershipRole']>
  ): void {
    const selectedUserId = this.selectedMembership()?.user.id;
    if (!selectedUserId) return;

    this.membershipMutation.set(key);
    this.error.set('');
    request
      .pipe(
        switchMap(() => this.apiService.getGroupMemberships(this.groupId)),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.membershipMutation.set('')),
        catchError((error) => {
          console.error('Failed to update group membership', error);
          this.error.set('Failed to update this membership.');
          return of(null);
        })
      )
      .subscribe((memberships) => {
        if (!memberships) return;
        this.updateMemberships(memberships, selectedUserId);
      });
  }

  private updateMemberships(memberships: GroupMembership[], selectedUserId?: string): void {
    this.memberships.set(memberships);
    if (selectedUserId) {
      this.selectedMembership.set(memberships.find((membership) => membership.user.id === selectedUserId) ?? null);
    }
  }
}

function isGroupNotificationTopic(topic: NotificationTopic): topic is GroupNotificationTopic {
  return topic.scope === 'GROUP' && topic.key.startsWith('group.');
}
