import type { DiscoveryResourceType, VersioningStatus } from '../../generated/backofficeSchemas';

export type ResultsFilters = {
  searchQuery: string;
  dataTypes: DiscoveryResourceType[];
  versioningStatuses: VersioningStatus[];
};

export const DISCOVERY_RESOURCE_TYPES: DiscoveryResourceType[] = ['dataset', 'software-application', 'software-source-code'];

export const DISCOVERY_VERSIONING_STATUSES: VersioningStatus[] = [
  'DRAFT',
  'SUBMITTED',
  'PUBLISHED',
  'ARCHIVED',
  'REJECTED'
];

export const DEFAULT_RESULTS_FILTERS: ResultsFilters = {
  searchQuery: '',
  dataTypes: [],
  versioningStatuses: ['PUBLISHED']
};

export function normalizeResultsFilters(value: unknown): ResultsFilters {
  if (!value || typeof value !== 'object') {
    return defaultResultsFilters();
  }

  const storedFilters = value as Partial<Record<keyof ResultsFilters, unknown>>;
  const dataTypes = Array.isArray(storedFilters.dataTypes)
    ? storedFilters.dataTypes.filter((item): item is DiscoveryResourceType => DISCOVERY_RESOURCE_TYPES.includes(item as DiscoveryResourceType))
    : [];
  const versioningStatuses = Array.isArray(storedFilters.versioningStatuses)
    ? storedFilters.versioningStatuses.filter((item): item is VersioningStatus => DISCOVERY_VERSIONING_STATUSES.includes(item as VersioningStatus))
    : [];

  return {
    searchQuery: typeof storedFilters.searchQuery === 'string' ? storedFilters.searchQuery : '',
    dataTypes,
    versioningStatuses: versioningStatuses.length ? versioningStatuses : [...DEFAULT_RESULTS_FILTERS.versioningStatuses]
  };
}

function defaultResultsFilters(): ResultsFilters {
  return {
    searchQuery: DEFAULT_RESULTS_FILTERS.searchQuery,
    dataTypes: [...DEFAULT_RESULTS_FILTERS.dataTypes],
    versioningStatuses: [...DEFAULT_RESULTS_FILTERS.versioningStatuses]
  };
}
