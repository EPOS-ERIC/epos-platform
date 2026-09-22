import { Component, DestroyRef, HostListener, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Observable, catchError, finalize, forkJoin, map, of, switchMap } from 'rxjs';

import type { Group, GroupMembership, GroupPermission, MembershipRole, Permission, RoleRequest, User, UserListItem, UserMembershipSummary } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { isGroupPermissionCatalogItem, type GroupPermissionCatalogItem } from '../../models/resource-actions';
import { ApiService } from '../../services/api.service';
import { GroupsStore } from '../../stores/groups.store';
import { GroupMembershipAccessEditorComponent } from '../group-membership-access-editor/group-membership-access-editor.component';

interface PendingMembershipRequest {
  group: Group;
  membership: GroupMembership;
  requestedRoles: MembershipRole[];
  selectedRoles: MembershipRole[];
}

interface PendingRoleRequest {
  group: Group;
  membership: GroupMembership;
  request: RoleRequest;
}

@Component({
  selector: 'app-admin-groups',
  imports: [DatePipe, FormsModule, GroupMembershipAccessEditorComponent],
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
  protected readonly roleRequests = signal<PendingRoleRequest[]>([]);
  protected readonly roleRequestsLoading = signal(true);
  protected readonly roleRequestsError = signal('');
  protected readonly roleRequestDecision = signal('');
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
  protected readonly groupPermissions = signal<GroupPermissionCatalogItem[]>([]);
  protected readonly filteredUsers = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    return query
      ? this.users().filter((user) => `${user.given_name} ${user.family_name} ${user.email}`.toLowerCase().includes(query))
      : this.users();
  });

  constructor() {
    this.loadPermissions();
    this.loadMembershipRequests();
    this.loadUsers();
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

    const enabled = !membership.granted_permissions.includes(permission);
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

  protected decideRoleRequest(item: PendingRoleRequest, decision: 'accept' | 'reject'): void {
    if (this.roleRequestDecision()) return;

    const key = this.roleRequestKey(item);
    this.roleRequestDecision.set(key);
    this.roleRequestsError.set('');
    this.apiService
      .decideGroupRoleRequest(item.group.id, item.request.id, decision)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.roleRequestDecision.set('')),
        catchError((error) => {
          console.error('Failed to decide role request', error);
          this.roleRequestsError.set('Failed to update this role request.');
          return of(null);
        })
      )
      .subscribe((membership) => {
        if (membership) {
          this.roleRequests.update((requests) => requests.filter((request) => request !== item));
        }
      });
  }

  protected roleRequestKey(item: PendingRoleRequest): string {
    return `${item.group.id}:${item.request.id}`;
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
    this.roleRequestsLoading.set(true);
    this.roleRequestsError.set('');

    const requests = this.groupsStore.groups().map((group) =>
      this.apiService.getGroupMemberships(group.id).pipe(
        map((memberships) => ({
          membershipRequests: memberships
            .filter((membership) => membership.status === 'PENDING')
            .map((membership): PendingMembershipRequest => {
            const requestedRoles = membership.role_requests
              .filter((request) => request.status === 'PENDING')
              .map((request) => request.role);
            return { group, membership, requestedRoles, selectedRoles: [...requestedRoles] };
            }),
          roleRequests: memberships
            .filter((membership) => membership.status === 'ACTIVE')
            .flatMap((membership) => membership.role_requests
              .filter((request) => request.status === 'PENDING')
              .map((request): PendingRoleRequest => ({ group, membership, request })))
        }))
      )
    );

    (requests.length ? forkJoin(requests) : of([]))
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => {
          this.membershipRequestsLoading.set(false);
          this.roleRequestsLoading.set(false);
        }),
        catchError((error) => {
          console.error('Failed to load membership and role requests', error);
          this.membershipRequestsError.set('Failed to load membership requests.');
          this.roleRequestsError.set('Failed to load role requests.');
          return of([]);
        })
      )
      .subscribe((groupRequests) => {
        this.membershipRequests.set(groupRequests.flatMap((group) => group.membershipRequests));
        this.roleRequests.set(groupRequests.flatMap((group) => group.roleRequests));
      });
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
