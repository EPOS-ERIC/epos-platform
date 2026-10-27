import type { DiscoveryResourceType, ResourceVersion } from '../../generated/backofficeSchemas';

export type ResourceVersionComparisonContext = {
  resourceType: DiscoveryResourceType;
  resourceId: string;
  leftVersionId: string;
  rightVersionId: string;
  sourceVersionId: string;
  groupId: string;
  sectionPath: string;
};

export type ResourceVersionDiffValue =
  | { kind: 'value'; value: string }
  | { kind: 'null' }
  | { kind: 'missing' };

export type ResourceVersionDiffRow = {
  section: string;
  path: string;
  left: ResourceVersionDiffValue;
  right: ResourceVersionDiffValue;
  changed: boolean;
};

export type ResourceVersionDiffSection = {
  title: string;
  rows: ResourceVersionDiffRow[];
  changedCount: number;
};

export type ResourceVersionComparison = {
  leftVersion: ResourceVersion;
  rightVersion: ResourceVersion;
  diffRows: ResourceVersionDiffRow[];
};

export function orderVersionsByCreatedAt<T extends { created_at: string }>(first: T, second: T): [T, T] {
  const firstCreatedAt = Date.parse(first.created_at);
  const secondCreatedAt = Date.parse(second.created_at);

  return Number.isFinite(firstCreatedAt) && Number.isFinite(secondCreatedAt) && firstCreatedAt > secondCreatedAt
    ? [second, first]
    : [first, second];
}
