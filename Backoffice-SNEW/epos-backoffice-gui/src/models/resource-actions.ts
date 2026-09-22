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
  save: 'resource.edit',
  submit: 'resource.submit',
  publish: 'resource.publish',
  'create-new-draft': 'resource.create',
  archive: 'resource.unpublish',
  reject: 'resource.reject',
  'revert-to-draft': 'resource.reopen',
  delete: 'resource.discard'
} as const satisfies Record<ResourceActionKey, GroupPermission>;

export const GROUP_PERMISSIONS = [
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
