import type { DiscoveryResourceType, ReplaceResourceDraftRequest } from '../../generated/backofficeSchemas';

export type ResourceActionKey =
  | 'save'
  | 'submit'
  | 'publish'
  | 'create-new-draft'
  | 'archive'
  | 'reject'
  | 'revert-to-draft'
  | 'delete';

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
