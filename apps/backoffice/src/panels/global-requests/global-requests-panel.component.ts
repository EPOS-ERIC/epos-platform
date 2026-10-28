import { Component, DestroyRef, OnInit, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Observable, catchError, finalize, forkJoin, map, of, switchMap } from 'rxjs';

import type { MembershipRole } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { ApiService } from '../../services/api.service';
import { CurrentUserStore } from '../../stores/current-user.store';
import { GroupsStore } from '../../stores/groups.store';
import {
  MembershipRequestsPanelComponent,
  type MembershipRequestDecision,
  type MembershipRequestItem,
  type MembershipRequestRoleToggle
} from '../access-requests/membership-requests-panel.component';
import {
  RoleRequestsPanelComponent,
  type RoleRequestDecision,
  type RoleRequestItem
} from '../access-requests/role-requests-panel.component';

@Component({
  selector: 'app-global-requests-panel',
  imports: [MembershipRequestsPanelComponent, RoleRequestsPanelComponent],
  templateUrl: './global-requests-panel.component.html',
  styleUrl: './global-requests-panel.component.css'
})
export class GlobalRequestsPanelComponent implements OnInit {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly groupsStore = inject(GroupsStore);
  private readonly currentUserStore = inject(CurrentUserStore);

  protected readonly canManageMemberships = computed(() =>
    this.currentUserStore.hasGlobalPermission('global.membership.manage')
  );
  protected readonly canManageRoles = computed(() =>
    this.currentUserStore.hasGlobalPermission('global.role.manage')
  );
  protected readonly membershipRequests = signal<MembershipRequestItem[]>([]);
  protected readonly membershipRequestsLoading = signal(true);
  protected readonly membershipRequestsError = signal('');
  protected readonly membershipDecision = signal('');
  protected readonly roleRequests = signal<RoleRequestItem[]>([]);
  protected readonly roleRequestsLoading = signal(true);
  protected readonly roleRequestsError = signal('');
  protected readonly roleRequestDecision = signal('');
  protected readonly groupRoles: MembershipRole[] = ['EDITOR', 'REVIEWER', 'GROUP_ADMIN'];

  ngOnInit(): void {
    if (this.canManageMemberships() || this.canManageRoles()) this.loadMembershipRequests();
  }

  protected toggleRequestedRole({ item: toggledItem, role }: MembershipRequestRoleToggle): void {
    if (this.membershipDecision()) return;

    this.membershipRequests.update((requests) => requests.map((item) => {
      if (item.key !== toggledItem.key) return item;

      const selectedRoles = item.selectedRoles.includes(role)
        ? item.selectedRoles.filter((selectedRole) => selectedRole !== role)
        : [...item.selectedRoles, role];
      return { ...item, selectedRoles };
    }));
  }

  protected decideMembership({ item, decision }: MembershipRequestDecision): void {
    if (!this.canManageMemberships() || this.membershipDecision()) return;

    this.membershipDecision.set(item.key);
    this.membershipRequestsError.set('');

    const request = decision === 'accept'
      ? this.acceptMembership(item)
      : this.apiService.decideGroupMembership(item.groupId, item.userId, 'reject');

    request
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
          this.membershipRequests.update((requests) => requests.filter((requestItem) => requestItem.key !== item.key));
        }
      });
  }

  protected decideRoleRequest({ item, decision }: RoleRequestDecision): void {
    if (!this.canManageRoles() || this.roleRequestDecision()) return;

    this.roleRequestDecision.set(item.key);
    this.roleRequestsError.set('');
    this.apiService
      .decideGroupRoleRequest(item.groupId, item.requestId, decision)
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
          this.roleRequests.update((requests) => requests.filter((request) => request.key !== item.key));
        }
      });
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
            .map((membership): MembershipRequestItem => {
              const requestedRoles = membership.role_requests
                .filter((request) => request.status === 'PENDING')
                .map((request) => request.role);
              return {
                key: `${group.id}:${membership.user.id}`,
                groupId: group.id,
                groupName: group.name,
                groupLogoPath: groupIconPath(group.id),
                userId: membership.user.id,
                userEmail: membership.user.email,
                userName: `${membership.user.given_name} ${membership.user.family_name}`.trim() || 'Unnamed user',
                requestedAt: membership.created_at,
                requestedRoles,
                selectedRoles: [...requestedRoles],
                pendingRoleRequests: membership.role_requests
                  .filter((request) => request.status === 'PENDING')
                  .map((request) => ({ id: request.id, role: request.role }))
              };
            }),
          roleRequests: memberships
            .filter((membership) => membership.status === 'ACTIVE')
            .flatMap((membership) => membership.role_requests
              .filter((request) => request.status === 'PENDING')
              .map((request): RoleRequestItem => ({
                key: `${group.id}:${request.id}`,
                groupId: group.id,
                groupName: group.name,
                groupLogoPath: groupIconPath(group.id),
                userId: membership.user.id,
                userEmail: membership.user.email,
                userName: `${membership.user.given_name} ${membership.user.family_name}`.trim() || 'Unnamed user',
                requestId: request.id,
                role: request.role,
                requestedAt: request.requested_at
              })))
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

  private acceptMembership(request: MembershipRequestItem): Observable<unknown> {
    if (!this.canManageRoles()) {
      return this.apiService.decideGroupMembership(request.groupId, request.userId, 'accept');
    }

    return this.apiService
      .decideGroupMembership(request.groupId, request.userId, 'accept')
      .pipe(
        switchMap(() => {
          const selectedRoles = new Set(request.selectedRoles);
          const requestedRoles = new Set(request.requestedRoles);
          const decisions: Observable<unknown>[] = request.pendingRoleRequests
            .map((roleRequest) => this.apiService.decideGroupRoleRequest(
              request.groupId,
              roleRequest.id,
              selectedRoles.has(roleRequest.role) ? 'accept' : 'reject'
            ));

          for (const role of selectedRoles) {
            if (!requestedRoles.has(role)) {
              decisions.push(this.apiService.setGroupMembershipRole(request.groupId, request.userId, role, true));
            }
          }

          return decisions.length ? forkJoin(decisions) : of(request);
        })
      );
  }
}
