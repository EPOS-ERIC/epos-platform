import { Component, DestroyRef, HostListener, Input, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { EMPTY, Subscription, catchError, finalize, forkJoin, map } from 'rxjs';

import type {
  DistributionConversionPlugin,
  DistributionConversionRelation,
  ConversionRelationInput,
  ManagedConversionPlugin,
  WebServiceVisibleOn
} from '../../../../../generated/backofficeSchemas';
import { ApiService } from '../../../../services/api.service';
import { ToastService } from '../../../../services/toast.service';
import { CurrentUserStore } from '../../../../stores/current-user.store';

type ConversionRow = {
  plugin: DistributionConversionPlugin;
  relation: DistributionConversionRelation;
};

@Component({
  selector: 'app-distribution-conversions',
  imports: [FormsModule],
  templateUrl: './distribution-conversions.component.html',
  styleUrl: './distribution-conversions.component.css'
})
export class DistributionConversionsComponent {
  private readonly apiService = inject(ApiService);
  private readonly toast = inject(ToastService);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly destroyRef = inject(DestroyRef);
  private readonly cache = new Map<string, DistributionConversionPlugin[]>();
  private request?: Subscription;
  private submissionRequest?: Subscription;
  private associationRequest?: Subscription;
  private deletionRequest?: Subscription;
  private currentDistributionId = '';

  protected readonly plugins = signal<DistributionConversionPlugin[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly requestSubmitting = signal(false);
  protected readonly canManageConversions = computed(() =>
    this.currentUserStore.hasGlobalPermission('global.converter.manage')
  );
  protected readonly availablePlugins = computed(() =>
    [...this.plugins()].sort((left, right) => left.name.localeCompare(right.name))
  );
  protected readonly associationPlugin = signal<DistributionConversionPlugin | null>(null);
  protected readonly associationSubmitting = signal(false);
  protected readonly associationError = signal('');
  protected readonly pendingRemoval = signal<ConversionRow | null>(null);
  protected readonly deletionSubmitting = signal(false);
  protected readonly deletionError = signal('');
  protected readonly rows = computed<ConversionRow[]>(() => this.plugins().flatMap((plugin) =>
    plugin.relations.map((relation) => ({ plugin, relation }))
  ));
  protected readonly visualizationOptions: readonly WebServiceVisibleOn[] = ['GRAPH', 'TABLE', 'MAP'];
  protected requestInputFormat = '';
  protected requestVisualizations: WebServiceVisibleOn[] = [];
  protected requestMessage = '';
  protected associationInputFormat = '';
  protected associationOutputFormat = '';

  @Input() disabled = false;
  @Input() nativeFormats: string[] = [];

  @Input()
  set distributionId(value: string) {
    if (value === this.currentDistributionId) {
      return;
    }

    this.submissionRequest?.unsubscribe();
    this.associationRequest?.unsubscribe();
    this.deletionRequest?.unsubscribe();
    this.closeAssociationDialog();
    this.closeRemovalDialog();
    this.requestInputFormat = '';
    this.requestVisualizations = [];
    this.requestMessage = '';
    this.currentDistributionId = value;
    this.loadConversions();
  }

  get distributionId(): string {
    return this.currentDistributionId;
  }

  protected retry(): void {
    this.cache.delete(this.currentDistributionId);
    this.loadConversions();
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    if (this.pendingRemoval()) {
      this.closeRemovalDialog();
    } else {
      this.closeAssociationDialog();
    }
  }

  protected openAssociationDialog(plugin: DistributionConversionPlugin): void {
    this.associationPlugin.set(plugin);
    this.associationInputFormat = this.nativeFormats.length === 1 ? this.nativeFormats[0] : '';
    this.associationOutputFormat = '';
    this.associationError.set('');
  }

  protected closeAssociationDialog(): void {
    if (this.associationSubmitting()) return;
    this.associationPlugin.set(null);
    this.associationInputFormat = '';
    this.associationOutputFormat = '';
    this.associationError.set('');
  }

  protected canAssociate(): boolean {
    return this.canManageConversions()
      && !this.associationSubmitting()
      && Boolean(this.currentDistributionId)
      && Boolean(this.associationPlugin())
      && Boolean(this.associationInputFormat.trim())
      && Boolean(this.associationOutputFormat.trim());
  }

  protected associatePlugin(): void {
    const plugin = this.associationPlugin();
    if (!plugin || !this.canAssociate()) return;

    const payload = {
      distribution_id: this.currentDistributionId,
      plugin_id: plugin.id,
      input_format: this.associationInputFormat.trim(),
      output_format: this.associationOutputFormat.trim()
    } satisfies ConversionRelationInput;

    this.associationSubmitting.set(true);
    this.associationError.set('');
    this.associationRequest = this.apiService.createConversionRelation(payload).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.associationSubmitting.set(false))
    ).subscribe({
      next: () => {
        this.toast.success(`${plugin.name} associated with this distribution.`);
        this.associationPlugin.set(null);
        this.associationInputFormat = '';
        this.associationOutputFormat = '';
        this.cache.delete(this.currentDistributionId);
        this.loadConversions();
      },
      error: (error) => {
        console.error('Failed to create conversion relation', error);
        this.associationError.set(this.problemMessage(error, 'Failed to associate this plugin.'));
      }
    });
  }

  protected openRemovalDialog(row: ConversionRow): void {
    this.pendingRemoval.set(row);
    this.deletionError.set('');
  }

  protected closeRemovalDialog(): void {
    if (this.deletionSubmitting()) return;
    this.pendingRemoval.set(null);
    this.deletionError.set('');
  }

  protected removeRelation(): void {
    const row = this.pendingRemoval();
    if (!row || !this.canManageConversions() || this.deletionSubmitting()) return;

    this.deletionSubmitting.set(true);
    this.deletionError.set('');
    this.deletionRequest = this.apiService.deleteConversionRelation(row.relation.id).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.deletionSubmitting.set(false))
    ).subscribe({
      next: () => {
        this.toast.success('Conversion relation removed.');
        this.pendingRemoval.set(null);
        this.cache.delete(this.currentDistributionId);
        this.loadConversions();
      },
      error: (error) => {
        console.error('Failed to delete conversion relation', error);
        this.deletionError.set(this.problemMessage(error, 'Failed to remove this conversion relation.'));
      }
    });
  }

  protected toggleRequestVisualization(visualization: WebServiceVisibleOn, checked: boolean): void {
    this.requestVisualizations = checked
      ? Array.from(new Set([...this.requestVisualizations, visualization]))
      : this.requestVisualizations.filter((value) => value !== visualization);
  }

  protected canSubmitRequest(): boolean {
    return !this.disabled
      && !this.requestSubmitting()
      && Boolean(this.requestInputFormat.trim())
      && Boolean(this.requestVisualizations.length)
      && Boolean(this.requestMessage.trim());
  }

  protected submitRequest(): void {
    if (!this.canSubmitRequest()) {
      return;
    }

    this.requestSubmitting.set(true);
    this.submissionRequest = this.apiService.requestDistributionConversion(this.currentDistributionId, {
      input_format: this.requestInputFormat.trim(),
      visualizations: this.requestVisualizations,
      message: this.requestMessage.trim()
    }).pipe(
      takeUntilDestroyed(this.destroyRef),
      finalize(() => this.requestSubmitting.set(false))
    ).subscribe({
      next: () => {
        this.requestInputFormat = '';
        this.requestVisualizations = [];
        this.requestMessage = '';
        this.toast.success('Conversion request submitted.');
      },
      error: (error) => {
        console.error('Failed to request distribution conversion', error);
        this.toast.error(this.problemMessage(error, 'Failed to submit conversion request.'));
      }
    });
  }

  private problemMessage(error: unknown, fallback: string): string {
    const body: unknown = error instanceof HttpErrorResponse ? error.error : null;
    if (body && typeof body === 'object') {
      const problem = body as Record<string, unknown>;
      if (typeof problem['detail'] === 'string' && problem['detail'].trim()) {
        return problem['detail'].trim();
      }
      if (typeof problem['title'] === 'string' && problem['title'].trim()) {
        return problem['title'].trim();
      }
    }

    return fallback;
  }

  private loadConversions(): void {
    this.request?.unsubscribe();
    this.plugins.set([]);
    this.error.set('');

    if (!this.currentDistributionId) {
      this.loading.set(false);
      return;
    }

    const cached = this.cache.get(this.currentDistributionId);
    if (cached) {
      this.plugins.set(cached);
      this.loading.set(false);
      return;
    }

    const distributionId = this.currentDistributionId;
    this.loading.set(true);
    const conversionsRequest = this.canManageConversions()
      ? forkJoin({
          associatedPlugins: this.apiService.listDistributionConversionRelations(distributionId),
          availablePlugins: this.apiService.listConversionPlugins()
        }).pipe(map(({ associatedPlugins, availablePlugins }) => this.mergePlugins(availablePlugins, associatedPlugins)))
      : this.apiService.listDistributionConversionRelations(distributionId);
    this.request = conversionsRequest.pipe(
      takeUntilDestroyed(this.destroyRef),
      catchError((error) => {
        console.error('Failed to load distribution conversions', error);
        if (this.currentDistributionId === distributionId) {
          this.error.set('Failed to load conversions for this distribution.');
        }
        return EMPTY;
      }),
      finalize(() => {
        if (this.currentDistributionId === distributionId) {
          this.loading.set(false);
        }
      })
    ).subscribe((plugins) => {
      this.cache.set(distributionId, plugins);
      if (this.currentDistributionId === distributionId) {
        this.plugins.set(plugins);
      }
    });
  }

  private mergePlugins(
    availablePlugins: ManagedConversionPlugin[],
    associatedPlugins: DistributionConversionPlugin[]
  ): DistributionConversionPlugin[] {
    const associatedById = new Map(associatedPlugins.map((plugin) => [plugin.id, plugin]));
    const available = availablePlugins.map((plugin): DistributionConversionPlugin => ({
      id: plugin.id,
      name: plugin.name,
      description: plugin.description,
      repository: plugin.repository,
      enabled: plugin.enabled,
      installed: plugin.installed,
      branch: plugin.branch,
      runtime: plugin.runtime,
      arguments: plugin.arguments,
      relations: associatedById.get(plugin.id)?.relations ?? []
    }));
    const availableIds = new Set(available.map(({ id }) => id));
    return [...available, ...associatedPlugins.filter(({ id }) => !availableIds.has(id))];
  }
}
