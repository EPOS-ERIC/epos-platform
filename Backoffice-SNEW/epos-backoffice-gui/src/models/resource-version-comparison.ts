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
