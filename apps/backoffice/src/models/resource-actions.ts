import type { DiscoveryResourceType, GroupPermission, Permission, ReplaceResourceDraftRequest } from '../../generated/backofficeSchemas';

export type ResourceActionKey =
  | 'save'
  | 'submit'
  | 'publish'
  | 'create-new-draft'
  | 'archive'
  | 'reject'
  | 'revert-to-draft'
  | 'delete';

export const RESOURCE_ACTION_PERMISSIONS = {
  save: 'group.resource.edit',
  submit: 'group.resource.submit',
  publish: 'group.resource.publish',
  'create-new-draft': 'group.resource.create',
  archive: 'group.resource.unpublish',
  reject: 'group.resource.reject',
  'revert-to-draft': 'group.resource.reopen',
  delete: 'group.resource.discard'
} as const satisfies Record<ResourceActionKey, GroupPermission>;

export const GROUP_PERMISSIONS = [
  'group.resource.create',
  'group.resource.edit',
  'group.resource.submit',
  'group.resource.withdraw',
  'group.resource.reopen',
  'group.resource.discard',
  'group.resource.reject',
  'group.resource.publish',
  'group.resource.unpublish',
  'group.membership.manage',
  'group.role.manage',
  'group.permission.manage',
  'group.notification.manage',
  'group.category.manage'
] as const satisfies readonly GroupPermission[];

export type GroupPermissionCatalogItem = Omit<Permission, 'key'> & { key: GroupPermission };

export function isGroupPermissionCatalogItem(permission: Permission): permission is GroupPermissionCatalogItem {
  return permission.scope === 'GROUP';
}

export type ResourceActionVariant = 'primary' | 'ghost' | 'danger';

export type ResourceActionItem = {
  key: ResourceActionKey;
  label: string;
  loadingLabel?: string;
  disabled?: boolean;
  loading?: boolean;
  variant?: ResourceActionVariant;
};

export type ResourceVersionContext = {
  resourceType: DiscoveryResourceType;
  resourceId: string;
  versionId: string;
  groupId: string;
};

export type SaveResourceContext = {
  resourceType: DiscoveryResourceType;
  resourceId: string;
  versionId: string;
  targetStatus: 'draft' | 'submitted';
  payload: ReplaceResourceDraftRequest;
};
