import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import type { GlobalPermission, GroupPermission, User } from '../../generated/backofficeSchemas';
import { GLOBAL_MANAGEMENT_PERMISSIONS, GROUP_MANAGEMENT_PERMISSIONS } from '../models/access-control';
import { GROUP_PERMISSIONS } from '../models/resource-actions';
import { ApiService } from '../services/api.service';
import { AuthService } from '../services/auth.service';

@Injectable({ providedIn: 'root' })
export class CurrentUserStore {
  private readonly apiService = inject(ApiService);
  private readonly authService = inject(AuthService);
  private initialization?: Promise<void>;

  readonly user = signal<User | null>(null);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);
  readonly isAdmin = computed(() => this.user()?.global_roles.includes('ADMIN') ?? false);
  readonly globalPermissions = computed<GlobalPermission[]>(() => {
    const user = this.user();
    if (!user) return [];
    return [...new Set([...user.global_granted_permissions, ...user.global_role_permissions])];
  });

  hasGlobalPermission(permission: GlobalPermission): boolean {
    return this.globalPermissions().includes(permission);
  }

  canAccessGlobalManage(): boolean {
    return GLOBAL_MANAGEMENT_PERMISSIONS.some((permission) => this.hasGlobalPermission(permission));
  }

  canManageGroup(groupId: string): boolean {
    return GROUP_MANAGEMENT_PERMISSIONS.some((permission) => this.hasGroupPermission(groupId, permission));
  }

  belongsToGroup(groupId: string): { status: string } {
    const belongsToGroup = this.user()?.memberships.find((item) => item.group.id === groupId)
    if(belongsToGroup !== undefined && belongsToGroup.status.toUpperCase() === 'ACTIVE'){
      return { status: 'ACTIVE'};
    }
    else if(belongsToGroup !== undefined && belongsToGroup.status.toUpperCase() !== 'ACTIVE'){
      return { status: belongsToGroup.status };
    }
    else{
      return { status: 'null' };
    }
  }

  groupPermissions(groupId: string): GroupPermission[] {
    const user = this.user();
    if (!user || !groupId) return [];

    const globalPermissions = new Set([
      ...user.global_granted_permissions,
      ...user.global_role_permissions
    ]);
    const inheritedPermissions = GROUP_PERMISSIONS.filter((permission) =>
      globalPermissions.has(permission.replace(/^group\./, 'global.') as GlobalPermission)
    );
    const membership = user.memberships.find((item) => item.group.id === groupId && item.status === 'ACTIVE');
    return [...new Set([
      ...inheritedPermissions,
      ...(membership?.granted_permissions ?? []),
      ...(membership?.role_permissions ?? [])
    ])];
  }

  hasGroupPermission(groupId: string, permission: GroupPermission): boolean {
    return this.groupPermissions(groupId).includes(permission);
  }

  initialize(): Promise<void> {
    this.initialization ??= this.loadCurrentUser();
    return this.initialization;
  }

  refresh(): Promise<void> {
    return this.loadCurrentUser();
  }

  private async loadCurrentUser(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      await this.authService.initialize();
      if (!this.authService.authenticated()) return;

      this.user.set(await firstValueFrom(this.apiService.getUser('self')));
    } catch (error) {
      console.error('Failed to load current user', error);
      this.error.set('Failed to load current user.');
    } finally {
      this.loading.set(false);
    }
  }
}
