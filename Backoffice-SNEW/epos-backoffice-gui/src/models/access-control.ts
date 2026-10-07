import type { GlobalPermission, GroupPermission } from '../../generated/backofficeSchemas';

export const GROUP_MANAGEMENT_PERMISSIONS = [
  'group.membership.manage',
  'group.role.manage',
  'group.permission.manage',
  'group.category.manage',
  'group.notification.manage'
] as const satisfies readonly GroupPermission[];

export const GLOBAL_MANAGEMENT_PERMISSIONS = [
  'global.category.manage',
  'global.category_scheme.manage',
  'global.contact_point.manage',
  'global.membership.manage',
  'global.role.manage',
  'global.permission.manage',
  'global.notification.manage',
  'global.organization.manage',
  'global.activity.read'
] as const satisfies readonly GlobalPermission[];
