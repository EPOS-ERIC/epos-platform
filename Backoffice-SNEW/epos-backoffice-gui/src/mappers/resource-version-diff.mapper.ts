import type { ResourceVersion } from '../../generated/backofficeSchemas';
import type { ResourceVersionDiffRow, ResourceVersionDiffValue } from '../models/resource-version-comparison';

type FlatValue = { value: unknown };
type PathSegment = { kind: 'field'; key: string } | { kind: 'item'; index: number };
type ResourceVersionDiffReferences = {
  contactPoints?: ReadonlyMap<string, string>;
};

const IGNORED_PATHS = new Set([
  'resource_id',
  'version_id',
  'revision',
  'status',
  'created_by',
  'group_id',
  'group_name',
  'created_at',
  'updated_at',
  'rejection_note',
  'metadata.resource_type',
  'metadata.id',
  'metadata.access_right'
]);

const METADATA_SECTIONS: Record<string, string> = {
  spatial_coverage: 'Spatial Coverage',
  temporal_coverage: 'Temporal Coverage',
  geological_temporal_coverage: 'Temporal Coverage',
  persistent_identifiers: 'Persistent Identifiers',
  identifiers: 'Persistent Identifiers',
  contact_points: 'Contact Points',
  providers: 'Providers',
  categories: 'Categories',
  distributions: 'Distributions'
};

export function mapResourceVersionDiff(
  leftVersion: ResourceVersion,
  rightVersion: ResourceVersion,
  references: ResourceVersionDiffReferences = {}
): ResourceVersionDiffRow[] {
  const left = flatten(leftVersion);
  const right = flatten(rightVersion);
  const paths = [...left.keys(), ...[...right.keys()].filter((path) => !left.has(path))];

  return paths.map((path) => {
    const segments = JSON.parse(path) as PathSegment[];
    const leftValue = toDiffValue(left.get(path), segments, references);
    const rightValue = toDiffValue(right.get(path), segments, references);
    const section = sectionFor(segments);

    return {
      section,
      path: readablePath(segments, leftVersion, rightVersion, section),
      left: leftValue,
      right: rightValue,
      changed: !sameValue(toRawDiffValue(left.get(path)), toRawDiffValue(right.get(path)))
    };
  });
}

function flatten(value: unknown): Map<string, FlatValue> {
  const result = new Map<string, FlatValue>();

  const visit = (current: unknown, segments: PathSegment[]): void => {
    if (Array.isArray(current)) {
      if (current.length === 0) {
        result.set(serializePath(segments), { value: [] });
        return;
      }
      current.forEach((item, index) => visit(item, [...segments, { kind: 'item', index }]));
      return;
    }

    if (isRecord(current)) {
      const entries = Object.entries(current);
      if (entries.length === 0) {
        result.set(serializePath(segments), { value: {} });
        return;
      }
      entries.forEach(([key, item]) => {
        const nextSegments: PathSegment[] = [...segments, { kind: 'field', key }];
        if (!shouldIgnorePath(nextSegments)) {
          visit(item, nextSegments);
        }
      });
      return;
    }

    result.set(serializePath(segments), { value: current });
  };

  visit(value, []);
  return result;
}

function serializePath(segments: PathSegment[]): string {
  return JSON.stringify(segments);
}

function fieldPath(segments: PathSegment[]): string {
  return segments
    .filter((segment): segment is Extract<PathSegment, { kind: 'field' }> => segment.kind === 'field')
    .map((segment) => segment.key)
    .join('.');
}

function shouldIgnorePath(segments: PathSegment[]): boolean {
  const path = fieldPath(segments);
  if (IGNORED_PATHS.has(path)) {
    return true;
  }

  return path.startsWith('metadata.') && path.endsWith('.id');
}

function readablePath(
  segments: PathSegment[],
  left: ResourceVersion,
  right: ResourceVersion,
  section: string
): string {
  const labels: string[] = [];
  let leftValue: unknown = left;
  let rightValue: unknown = right;

  segments.forEach((segment) => {
    if (segment.kind === 'field') {
      labels.push(label(segment.key));
      leftValue = propertyValue(leftValue, segment.key);
      rightValue = propertyValue(rightValue, segment.key);
      return;
    }

    const itemLabel = singularLabel(labels.at(-1) ?? 'Item');
    const leftItem = arrayValue(leftValue, segment.index);
    const rightItem = arrayValue(rightValue, segment.index);
    const description = itemDescription(leftItem, rightItem);
    labels.push(`${itemLabel} ${segment.index + 1}${description ? ` (${description})` : ''}`);
    leftValue = leftItem;
    rightValue = rightItem;
  });

  if (segments[0]?.kind === 'field' && segments[0].key === 'metadata') {
    labels.shift();
    if (section !== 'General Information' && labels.length > 1) {
      labels.shift();
    }
  }

  return labels.join(' > ');
}

function sectionFor(segments: PathSegment[]): string {
  if (segments[0]?.kind !== 'field' || segments[0].key !== 'metadata') {
    return 'General Information';
  }

  const metadataField = segments.find((segment, index) => index > 0 && segment.kind === 'field');
  return metadataField?.kind === 'field'
    ? (METADATA_SECTIONS[metadataField.key] ?? 'General Information')
    : 'General Information';
}

function propertyValue(value: unknown, key: string): unknown {
  return isRecord(value) ? value[key] : undefined;
}

function arrayValue(value: unknown, index: number): unknown {
  return Array.isArray(value) ? value[index] : undefined;
}

function itemDescription(left: unknown, right: unknown): string {
  const leftDescription = describeItem(left);
  const rightDescription = describeItem(right);
  if (!leftDescription) {
    return rightDescription;
  }
  if (!rightDescription || leftDescription === rightDescription) {
    return leftDescription;
  }
  return `${leftDescription} / ${rightDescription}`;
}

function describeItem(value: unknown): string {
  if (!isRecord(value)) {
    return '';
  }

  for (const key of ['title', 'name', 'label', 'identifier', 'email', 'type', 'format', 'url']) {
    const candidate = value[key];
    if (typeof candidate === 'string' && candidate.trim()) {
      return truncate(candidate.trim());
    }
  }
  return '';
}

function truncate(value: string): string {
  return value.length > 64 ? `${value.slice(0, 61)}...` : value;
}

function singularLabel(value: string): string {
  if (value.endsWith('ies')) {
    return `${value.slice(0, -3)}y`;
  }
  if (value.endsWith('sses')) {
    return value.slice(0, -2);
  }
  return value.endsWith('s') ? value.slice(0, -1) : value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function label(key: string): string {
  return key
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function toDiffValue(
  entry: FlatValue | undefined,
  segments: PathSegment[],
  references: ResourceVersionDiffReferences
): ResourceVersionDiffValue {
  if (entry && fieldPath(segments) === 'metadata.contact_points' && typeof entry.value === 'string') {
    return {
      kind: 'value',
      value: references.contactPoints?.get(entry.value) ?? 'Contact point unavailable'
    };
  }
  return toRawDiffValue(entry);
}

function toRawDiffValue(entry: FlatValue | undefined): ResourceVersionDiffValue {
  if (!entry) {
    return { kind: 'missing' };
  }
  if (entry.value === null) {
    return { kind: 'null' };
  }
  if (Array.isArray(entry.value)) {
    return { kind: 'value', value: '[]' };
  }
  if (isRecord(entry.value)) {
    return { kind: 'value', value: '{}' };
  }
  return { kind: 'value', value: String(entry.value) };
}

function sameValue(left: ResourceVersionDiffValue, right: ResourceVersionDiffValue): boolean {
  return left.kind === right.kind && (left.kind !== 'value' || (right.kind === 'value' && left.value === right.value));
}
