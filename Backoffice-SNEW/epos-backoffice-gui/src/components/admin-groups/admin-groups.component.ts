import { Component, DestroyRef, HostListener, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { catchError, finalize, of, switchMap } from 'rxjs';

import type { MembershipRole, User, UserListItem, UserMembershipSummary } from '../../../generated/backofficeSchemas';
import type { GroupPermission } from '../../../generated/backofficeParameters';
import { ApiService } from '../../services/api.service';

@Component({
  selector: 'app-admin-groups',
  imports: [DatePipe, FormsModule],
  templateUrl: './admin-groups.component.html',
  styleUrl: './admin-groups.component.css'
})
export class AdminGroupsComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly users = signal<UserListItem[]>([]);
  protected readonly usersLoading = signal(true);
  protected readonly usersError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly selectedUserSummary = signal<UserListItem | null>(null);
  protected readonly selectedUser = signal<User | null>(null);
  protected readonly selectedUserLoading = signal(false);
  protected readonly membershipError = signal('');
  protected readonly membershipMutation = signal('');
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
