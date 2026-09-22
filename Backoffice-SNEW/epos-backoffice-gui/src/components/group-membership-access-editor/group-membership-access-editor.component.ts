import { Component, EventEmitter, Input, Output } from '@angular/core';

import type { GroupPermission, MembershipRole } from '../../../generated/backofficeSchemas';
import type { GroupPermissionCatalogItem } from '../../models/resource-actions';

@Component({
  selector: 'app-group-membership-access-editor',
  templateUrl: './group-membership-access-editor.component.html',
  styleUrl: './group-membership-access-editor.component.css'
})
export class GroupMembershipAccessEditorComponent {
  @Input() groupId = '';
  @Input() roles: MembershipRole[] = [];
  @Input() grantedPermissions: GroupPermission[] = [];
  @Input() rolePermissions: GroupPermission[] = [];
  @Input() allowedRoles: MembershipRole[] = [];
  @Input() allowedPermissions: GroupPermissionCatalogItem[] = [];
  @Input() mutationKey = '';
  @Output() roleChange = new EventEmitter<MembershipRole>();
  @Output() permissionChange = new EventEmitter<GroupPermission>();

  protected isMutationLoading(kind: 'role' | 'permission', value: string): boolean {
    return this.mutationKey === `${this.groupId}:${kind}:${value}`;
  }

  protected permissionDetails(permission: GroupPermission): GroupPermissionCatalogItem | undefined {
    return this.allowedPermissions.find((item) => item.key === permission);
  }
}
