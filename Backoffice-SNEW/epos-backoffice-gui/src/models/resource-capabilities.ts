import type { GroupPermission, VersioningStatus } from '../../generated/backofficeSchemas';
import { RESOURCE_ACTION_PERMISSIONS } from './resource-actions';

export type ResourceCapabilities = {
  edit: boolean;
  delete: boolean;
  save: boolean;
  submit: boolean;
  publish: boolean;
  reject: boolean;
  reopen: boolean;
  archive: boolean;
  createDraft: boolean;
};

export function resourceCapabilities(
  status: VersioningStatus | null,
  permissions: readonly GroupPermission[]
): ResourceCapabilities {
  const isDraft = status === 'DRAFT';
  const hasPermission = (permission: GroupPermission): boolean => permissions.includes(permission);

  return {
    edit: isDraft && hasPermission(RESOURCE_ACTION_PERMISSIONS.save),
    delete: isDraft && hasPermission(RESOURCE_ACTION_PERMISSIONS.delete),
    save: isDraft && hasPermission(RESOURCE_ACTION_PERMISSIONS.save),
    submit: isDraft && hasPermission(RESOURCE_ACTION_PERMISSIONS.submit),
    publish: status === 'SUBMITTED' && hasPermission(RESOURCE_ACTION_PERMISSIONS.publish),
    reject: status === 'SUBMITTED' && hasPermission(RESOURCE_ACTION_PERMISSIONS.reject),
    reopen: status === 'REJECTED' && hasPermission(RESOURCE_ACTION_PERMISSIONS['revert-to-draft']),
    archive: status === 'PUBLISHED' && hasPermission(RESOURCE_ACTION_PERMISSIONS.archive),
    createDraft: (status === 'PUBLISHED' || status === 'ARCHIVED')
      && hasPermission(RESOURCE_ACTION_PERMISSIONS['create-new-draft'])
  };
}
