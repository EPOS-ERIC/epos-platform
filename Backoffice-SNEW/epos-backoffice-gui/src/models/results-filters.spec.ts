import { normalizeResultsFilters } from './results-filters';

describe('normalizeResultsFilters', () => {
  it('sanitizes invalid persisted values and restores the default status', () => {
    expect(normalizeResultsFilters({
      searchQuery: 42,
      dataTypes: ['dataset', 'invalid'],
      versioningStatuses: ['invalid']
    })).toEqual({
      searchQuery: '',
      dataTypes: ['dataset'],
      versioningStatuses: ['PUBLISHED']
    });
  });
});
