import { DestroyRef, Injectable, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EMPTY, Observable, Subject, catchError, forkJoin, map, switchMap } from 'rxjs';

import type { ResourceVersion } from '../../generated/backofficeSchemas';
import { mapResourceVersionDiff } from '../mappers/resource-version-diff.mapper';
import type {
  ResourceVersionComparison,
  ResourceVersionComparisonContext,
  ResourceVersionDiffRow,
  ResourceVersionDiffSection
} from '../models/resource-version-comparison';
import { ApiService } from '../services/api.service';

type LoadCommand = { kind: 'load'; context: ResourceVersionComparisonContext } | { kind: 'clear' };

const DATASET_DIFF_SECTIONS = [
  'General Information',
  'Spatial Coverage',
  'Temporal Coverage',
  'Persistent Identifiers',
  'Contact Points',
  'Providers',
  'Categories',
  'Distributions'
];

const SOFTWARE_DIFF_SECTIONS = [
  'General Information',
  'Persistent Identifiers',
  'Contact Points',
  'Categories'
];

@Injectable()
export class ResourceVersionComparisonStore {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly loadCommands = new Subject<LoadCommand>();
  private readonly loadingState = signal(false);
  private readonly errorState = signal('');
  private readonly leftVersionState = signal<ResourceVersion | null>(null);
  private readonly rightVersionState = signal<ResourceVersion | null>(null);
  private readonly diffRowsState = signal<ResourceVersionDiffRow[]>([]);

  readonly loading = this.loadingState.asReadonly();
  readonly error = this.errorState.asReadonly();
  readonly leftVersion = this.leftVersionState.asReadonly();
  readonly rightVersion = this.rightVersionState.asReadonly();
  readonly diffRows = this.diffRowsState.asReadonly();
  readonly diffSections = computed<ResourceVersionDiffSection[]>(() => {
    const rowsBySection = new Map<string, ResourceVersionDiffRow[]>();
    this.diffRows().forEach((row) => {
      const rows = rowsBySection.get(row.section) ?? [];
      rows.push(row);
      rowsBySection.set(row.section, rows);
    });

    const resourceType = this.leftVersion()?.metadata.resource_type;
    const expectedSections = resourceType === 'dataset'
      ? DATASET_DIFF_SECTIONS
      : resourceType
        ? SOFTWARE_DIFF_SECTIONS
        : [];
    const sectionTitles = [
      ...expectedSections,
      ...[...rowsBySection.keys()].filter((title) => !expectedSections.includes(title))
    ];

    return sectionTitles
      .map((title) => {
        const rows = rowsBySection.get(title) ?? [];
        return {
          title,
          rows,
          changedCount: rows.filter((row) => row.changed).length
        };
      });
  });
  readonly changedCount = computed(() => this.diffRows().filter((row) => row.changed).length);

  constructor() {
    this.loadCommands
      .pipe(
        switchMap((command) => {
          this.reset();
          if (command.kind === 'clear') {
            return EMPTY;
          }

          this.loadingState.set(true);
          const context = command.context;
          return forkJoin({
            leftVersion: this.apiService.getResourceVersion(
              context.resourceType,
              context.resourceId,
              context.leftVersionId
            ) as Observable<ResourceVersion>,
            rightVersion: this.apiService.getResourceVersion(
              context.resourceType,
              context.resourceId,
              context.rightVersionId
            ) as Observable<ResourceVersion>
          }).pipe(
            map(({ leftVersion, rightVersion }) => ({
              leftVersion,
              rightVersion,
              diffRows: mapResourceVersionDiff(leftVersion, rightVersion)
            })),
            catchError((error) => {
              console.error('Failed to compare resource versions', error);
              this.loadingState.set(false);
              this.errorState.set('Failed to load resource versions for comparison.');
              return EMPTY;
            })
          );
        }),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe((comparison) => this.setComparison(comparison));
  }

  load(context: ResourceVersionComparisonContext): void {
    this.loadCommands.next({ kind: 'load', context });
  }

  clear(): void {
    this.loadCommands.next({ kind: 'clear' });
  }

  private reset(): void {
    this.loadingState.set(false);
    this.errorState.set('');
    this.leftVersionState.set(null);
    this.rightVersionState.set(null);
    this.diffRowsState.set([]);
  }

  private setComparison(comparison: ResourceVersionComparison): void {
    this.leftVersionState.set(comparison.leftVersion);
    this.rightVersionState.set(comparison.rightVersion);
    this.diffRowsState.set(comparison.diffRows);
    this.loadingState.set(false);
  }
}
