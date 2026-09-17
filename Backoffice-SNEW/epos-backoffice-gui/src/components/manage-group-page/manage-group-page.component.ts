import { DatePipe } from '@angular/common';
import { Component, DestroyRef, HostListener, Input, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { catchError, finalize, of, switchMap } from 'rxjs';

import type { GroupMembership, GroupPermission, MembershipRole } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { ApiService } from '../../services/api.service';
import { DialogService } from '../../services/dialog.service';
import { ToastService } from '../../services/toast.service';
import { GroupMembershipAccessEditorComponent } from '../group-membership-access-editor/group-membership-access-editor.component';

@Component({
  selector: 'app-manage-group-page',
  imports: [DatePipe, FormsModule, GroupMembershipAccessEditorComponent],
  templateUrl: './manage-group-page.component.html',
  styleUrl: './manage-group-page.component.css'
})
export class ManageGroupPageComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly dialog = inject(DialogService);
  private readonly toast = inject(ToastService);
  private currentGroupId = '';

  @Input()
  set groupId(value: string) {
    if (!value || value === this.currentGroupId) return;
    this.currentGroupId = value;
    this.selectedMembership.set(null);
    this.loadMemberships();
  }

  get groupId(): string {
    return this.currentGroupId;
  }

  @Input() groupName = '';

  protected readonly memberships = signal<GroupMembership[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly searchQuery = signal('');
  protected readonly selectedMembership = signal<GroupMembership | null>(null);
  protected readonly membershipMutation = signal('');
  protected readonly removingUserId = signal('');
  protected readonly groupIconPath = groupIconPath;
  protected readonly groupRoles: MembershipRole[] = ['EDITOR', 'REVIEWER'];
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
  protected readonly filteredMemberships = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const activeMemberships = this.memberships().filter((membership) => membership.status === 'ACTIVE');

    return query
      ? activeMemberships.filter((membership) =>
          `${this.userName(membership)} ${membership.user.email}`.toLowerCase().includes(query)
        )
      : activeMemberships;
  });

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    this.closeMembership();
  }

  protected openMembership(membership: GroupMembership): void {
    this.error.set('');
    this.selectedMembership.set(membership);
  }

  protected closeMembership(): void {
    if (!this.membershipMutation() && !this.removingUserId()) {
      this.selectedMembership.set(null);
    }
  }

  protected toggleRole(membership: GroupMembership, role: MembershipRole): void {
    if (!this.groupRoles.includes(role) || this.membershipMutation()) return;
    const enabled = !membership.roles.includes(role);
    this.runMembershipMutation(
      `${this.groupId}:role:${role}`,
      this.apiService.setGroupMembershipRole(this.groupId, membership.user.id, role, enabled)
    );
  }

  protected togglePermission(membership: GroupMembership, permission: GroupPermission): void {
    if (this.membershipMutation()) return;
    const enabled = !membership.granted_permissions.includes(permission);
    this.runMembershipMutation(
      `${this.groupId}:permission:${permission}`,
      this.apiService.setGroupMembershipPermission(this.groupId, membership.user.id, permission, enabled)
    );
  }

  protected async removeMembership(membership: GroupMembership): Promise<void> {
    if (membership.roles.includes('GROUP_ADMIN') || this.removingUserId() || this.membershipMutation()) return;

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

  protected userName(membership: GroupMembership): string {
    return `${membership.user.given_name} ${membership.user.family_name}`.trim() || 'Unnamed user';
  }

  private loadMemberships(): void {
    this.loading.set(true);
    this.error.set('');
    this.apiService
      .getGroupMemberships(this.groupId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false)),
        catchError((error) => {
          console.error('Failed to load group memberships', error);
          this.error.set('Failed to load group members.');
          return of([] as GroupMembership[]);
        })
      )
      .subscribe((memberships) => this.updateMemberships(memberships));
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
