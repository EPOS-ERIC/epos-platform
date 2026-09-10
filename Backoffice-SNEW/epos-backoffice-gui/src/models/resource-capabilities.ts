import type { VersioningStatus } from '../../generated/backofficeSchemas';

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

export function resourceCapabilities(status: VersioningStatus | null, isAdmin: boolean): ResourceCapabilities {
  const isDraft = status === 'DRAFT';
  const canReview = status === 'SUBMITTED' && isAdmin;

  return {
    edit: isDraft,
    delete: isDraft,
    save: isDraft,
    submit: isDraft,
    publish: canReview,
    reject: canReview,
    reopen: status === 'REJECTED',
    archive: status === 'PUBLISHED',
    createDraft: status === 'PUBLISHED' || status === 'ARCHIVED'
  };
}
