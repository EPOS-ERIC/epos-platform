import { Component, DestroyRef, HostListener, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Observable, catchError, finalize, forkJoin, map, of, switchMap } from 'rxjs';

import type { Group, GroupMembership, MembershipRole, User, UserListItem, UserMembershipSummary } from '../../../generated/backofficeSchemas';
import type { GroupPermission } from '../../../generated/backofficeParameters';
import { groupIconPath } from '../../group-icons';
import { ApiService } from '../../services/api.service';
import { GroupsStore } from '../../stores/groups.store';

interface PendingMembershipRequest {
  group: Group;
  membership: GroupMembership;
  requestedRoles: MembershipRole[];
  selectedRoles: MembershipRole[];
}

@Component({
  selector: 'app-admin-groups',
  imports: [DatePipe, FormsModule],
  templateUrl: './admin-groups.component.html',
  styleUrl: './admin-groups.component.css'
})
export class AdminGroupsComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly groupsStore = inject(GroupsStore);

  protected readonly membershipRequests = signal<PendingMembershipRequest[]>([]);
  protected readonly membershipRequestsLoading = signal(true);
  protected readonly membershipRequestsError = signal('');
  protected readonly membershipDecision = signal('');
  protected readonly users = signal<UserListItem[]>([]);
  protected readonly usersLoading = signal(true);
  protected readonly usersError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly selectedUserSummary = signal<UserListItem | null>(null);
  protected readonly selectedUser = signal<User | null>(null);
  protected readonly selectedUserLoading = signal(false);
  protected readonly membershipError = signal('');
  protected readonly membershipMutation = signal('');
  protected readonly groupIconPath = groupIconPath;
  protected readonly groupRoles: MembershipRole[] = ['EDITOR', 'REVIEWER', 'GROUP_ADMIN'];
  protected readonly groupPermissions: GroupPermission[] = [
    'resource.create',
    'resource.edit',
    'resource.submit',
    'resource.withdraw',
    'resource.reopen',
    'resource.discard',
    'resource.reject',
    'resource.publish',
    'resource.unpublish',
    'group.membership.manage',
    'group.role.manage',
    'group.permission.manage',
    'category.manage'
  ];
  protected readonly filteredUsers = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    return query
      ? this.users().filter((user) => `${user.given_name} ${user.family_name} ${user.email}`.toLowerCase().includes(query))
      : this.users();
  });

  constructor() {
    this.loadMembershipRequests();
    this.loadUsers();
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

    this.apiService
      .getUser(user.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.selectedUserLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load user memberships', error);
          this.membershipError.set('Failed to load memberships for this user.');
          return of(null);
        })
      )
      .subscribe((selectedUser) => this.selectedUser.set(selectedUser));
  }

  protected closeUserMemberships(): void {
    if (this.membershipMutation()) {
      return;
    }

    this.selectedUserSummary.set(null);
    this.selectedUser.set(null);
    this.membershipError.set('');
  }

  protected toggleRole(membership: UserMembershipSummary, role: MembershipRole): void {
    const user = this.selectedUser();

    if (!user || membership.status !== 'ACTIVE' || this.membershipMutation()) {
      return;
    }

    const enabled = !membership.roles.includes(role);
    this.runMembershipMutation(
      `${membership.group.id}:role:${role}`,
      this.apiService.setGroupMembershipRole(membership.group.id, user.id, role, enabled)
    );
  }

  protected togglePermission(membership: UserMembershipSummary, permission: GroupPermission): void {
    const user = this.selectedUser();

    if (!user || membership.status !== 'ACTIVE' || this.membershipMutation()) {
      return;
    }

    const enabled = !membership.permissions.includes(permission);
    this.runMembershipMutation(
      `${membership.group.id}:permission:${permission}`,
      this.apiService.setGroupMembershipPermission(membership.group.id, user.id, permission, enabled)
    );
  }

  protected toggleRequestedRole(request: PendingMembershipRequest, role: MembershipRole): void {
    if (this.membershipDecision()) return;

    this.membershipRequests.update((requests) => requests.map((item) => {
      if (item !== request) return item;

      const selectedRoles = item.selectedRoles.includes(role)
        ? item.selectedRoles.filter((selectedRole) => selectedRole !== role)
        : [...item.selectedRoles, role];
      return { ...item, selectedRoles };
    }));
  }

  protected decideMembership(request: PendingMembershipRequest, status: 'ACTIVE' | 'REJECTED'): void {
    if (this.membershipDecision()) return;

    const key = this.membershipRequestKey(request);
    this.membershipDecision.set(key);
    this.membershipRequestsError.set('');

    const decision = status === 'ACTIVE'
      ? this.acceptMembership(request)
      : this.apiService.decideGroupMembership(request.group.id, request.membership.user.id, 'reject');

    decision
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.membershipDecision.set('')),
        catchError((error) => {
          console.error('Failed to decide membership request', error);
          this.membershipRequestsError.set('Failed to update this membership request.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (result) {
          this.membershipRequests.update((requests) => requests.filter((item) => item !== request));
        }
      });
  }

  protected membershipRequestKey(request: PendingMembershipRequest): string {
    return `${request.group.id}:${request.membership.user.id}`;
  }

  protected isMutationLoading(groupId: string, kind: 'role' | 'permission', value: string): boolean {
    return this.membershipMutation() === `${groupId}:${kind}:${value}`;
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

  private loadMembershipRequests(): void {
    this.membershipRequestsLoading.set(true);
    this.membershipRequestsError.set('');

    const requests = this.groupsStore.groups().map((group) =>
      this.apiService.getGroupMemberships(group.id).pipe(
        map((memberships) => memberships
          .filter((membership) => membership.status === 'PENDING')
          .map((membership): PendingMembershipRequest => {
            const requestedRoles = membership.role_requests
              .filter((request) => request.status === 'PENDING')
              .map((request) => request.role);
            return { group, membership, requestedRoles, selectedRoles: [...requestedRoles] };
          }))
      )
    );

    (requests.length ? forkJoin(requests).pipe(map((groups) => groups.flat())) : of([] as PendingMembershipRequest[]))
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.membershipRequestsLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load membership requests', error);
          this.membershipRequestsError.set('Failed to load membership requests.');
          return of([] as PendingMembershipRequest[]);
        })
      )
      .subscribe((memberships) => this.membershipRequests.set(memberships));
  }

  private acceptMembership(request: PendingMembershipRequest): Observable<unknown> {
    return this.apiService
      .decideGroupMembership(request.group.id, request.membership.user.id, 'accept')
      .pipe(
        switchMap(() => {
          const selectedRoles = new Set(request.selectedRoles);
          const requestedRoles = new Set(request.requestedRoles);
          const decisions: Observable<unknown>[] = request.membership.role_requests
            .filter((roleRequest) => roleRequest.status === 'PENDING')
            .map((roleRequest) => this.apiService.decideGroupRoleRequest(
              request.group.id,
              roleRequest.id,
              selectedRoles.has(roleRequest.role) ? 'accept' : 'reject'
            ));

          for (const role of selectedRoles) {
            if (!requestedRoles.has(role)) {
              decisions.push(this.apiService.setGroupMembershipRole(request.group.id, request.membership.user.id, role, true));
            }
          }

          return decisions.length ? forkJoin(decisions) : of(request.membership);
        })
      );
  }

  private runMembershipMutation(key: string, request: ReturnType<ApiService['setGroupMembershipRole']>): void {
    const userId = this.selectedUser()?.id;

    if (!userId) {
      return;
    }

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
        if (user) {
          this.selectedUser.set(user);
        }
      });
  }
}
