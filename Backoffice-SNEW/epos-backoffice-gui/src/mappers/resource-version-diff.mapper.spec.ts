import type { ResourceVersion } from '../../generated/backofficeSchemas';
import { mapResourceVersionDiff } from './resource-version-diff.mapper';

describe('mapResourceVersionDiff', () => {
  it('flattens useful version data with readable array paths in stable union order', () => {
    const left = version({
      metadata: metadata({
        resource_type: 'dataset',
        title: 'Original',
        nullable: null,
        persistent_identifiers: [{ id: 'identifier-id', type: 'DOI', value: '10.1234/example' }],
        providers: [{ id: 'provider-id', name: 'Provider' }],
        categories: [{ id: 'category-id', name: 'Category' }],
        contact_points: ['contact-ada'],
        left_only: 'left'
      })
    });
    const right = version({
      revision: 2,
      metadata: metadata({
        resource_type: 'dataset',
        title: 'Updated',
        nullable: 'null',
        persistent_identifiers: [{ id: 'identifier-id', type: 'DOI', value: '10.1234/example' }],
        providers: [{ id: 'provider-id', name: 'Provider' }],
        categories: [{ id: 'category-id', name: 'Category' }],
        contact_points: ['contact-ada', 'contact-lin'],
        right_only: 'right'
      })
    });

    const rows = mapResourceVersionDiff(left, right, {
      contactPoints: new Map([
        ['contact-ada', 'ada@example.test (Technical)'],
        ['contact-lin', 'lin@example.test (Legal)']
      ])
    });

    expect(rows.map((row) => row.path)).toEqual([
      'Title',
      'Nullable',
      'Persistent Identifier 1 (DOI) > Type',
      'Persistent Identifier 1 (DOI) > Value',
      'Provider 1 (Provider) > Name',
      'Category 1 (Category) > Name',
      'Contact Point 1',
      'Left Only',
      'Contact Point 2',
      'Right Only'
    ]);
    expect(row(rows, 'Title')).toMatchObject({
      left: { kind: 'value', value: 'Original' },
      right: { kind: 'value', value: 'Updated' },
      changed: true
    });
    expect(row(rows, 'Contact Point 1')).toMatchObject({
      section: 'Contact Points',
      left: { kind: 'value', value: 'ada@example.test (Technical)' },
      changed: false
    });
    expect(row(rows, 'Contact Point 2')).toMatchObject({
      left: { kind: 'missing' },
      right: { kind: 'value', value: 'lin@example.test (Legal)' }
    });
    expect(rows.some((item) => item.path.endsWith('> Id'))).toBe(false);
    expect(rows.some((item) => item.path.includes('Version Id'))).toBe(false);
  });

  it('excludes lifecycle fields and nested technical ids from visible changes', () => {
    const left = version({
      metadata: metadata({
        title: 'Same title',
        distributions: [{ id: 'left-distribution-id', title: 'Distribution' }]
      })
    });
    const right = version({
      revision: 2,
      status: 'PUBLISHED',
      created_at: '2026-02-01T00:00:00Z',
      updated_at: '2026-02-02T00:00:00Z',
      metadata: metadata({
        title: 'Same title',
        distributions: [{ id: 'right-distribution-id', title: 'Distribution' }]
      })
    });

    expect(mapResourceVersionDiff(left, right).every((item) => !item.changed)).toBe(true);
  });

  it('distinguishes strings, null, missing values, and empty containers', () => {
    const left = version({ metadata: metadata({ value: null, empty: [], object: {} }) });
    const right = version({ metadata: metadata({ value: 'null', absent: null, empty: {}, object: [] }) });

    const rows = mapResourceVersionDiff(left, right);

    expect(row(rows, 'Value')).toEqual({
      section: 'General Information',
      path: 'Value',
      left: { kind: 'null' },
      right: { kind: 'value', value: 'null' },
      changed: true
    });
    expect(row(rows, 'Absent')).toMatchObject({ left: { kind: 'missing' }, right: { kind: 'null' } });
    expect(row(rows, 'Empty')).toMatchObject({
      left: { kind: 'value', value: '[]' },
      right: { kind: 'value', value: '{}' },
      changed: true
    });
  });

  it('does not expose unresolved contact point ids but still compares their raw values', () => {
    const left = version({ metadata: metadata({ contact_points: ['left-contact-id'] }) });
    const right = version({ metadata: metadata({ contact_points: ['right-contact-id'] }) });

    expect(row(mapResourceVersionDiff(left, right), 'Contact Point 1')).toMatchObject({
      left: { kind: 'value', value: 'Contact point unavailable' },
      right: { kind: 'value', value: 'Contact point unavailable' },
      changed: true
    });
  });

  function row(rows: ReturnType<typeof mapResourceVersionDiff>, path: string) {
    const found = rows.find((item) => item.path === path);
    expect(found).toBeDefined();
    return found!;
  }

  function version(overrides: Partial<ResourceVersion> = {}): ResourceVersion {
    return {
      resource_id: 'resource-id',
      version_id: 'version-id',
      revision: 1,
      status: 'DRAFT',
      created_by: 'user',
      group_id: 'group-id',
      group_name: 'Group',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-02T00:00:00Z',
      metadata: { resource_type: 'dataset' } as ResourceVersion['metadata'],
      ...overrides
    };
  }

  function metadata(value: Record<string, unknown>): ResourceVersion['metadata'] {
    return value as ResourceVersion['metadata'];
  }
});
