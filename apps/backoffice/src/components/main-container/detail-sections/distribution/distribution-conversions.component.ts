import { Component, DestroyRef, Input, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { EMPTY, Subscription, catchError, finalize } from 'rxjs';

import type {
  DistributionConversionPlugin,
  DistributionConversionRelation,
  WebServiceVisibleOn
} from '../../../../../generated/backofficeSchemas';
import { ApiService } from '../../../../services/api.service';

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
  private readonly destroyRef = inject(DestroyRef);
  private readonly cache = new Map<string, DistributionConversionPlugin[]>();
  private request?: Subscription;
  private currentDistributionId = '';

  protected readonly plugins = signal<DistributionConversionPlugin[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly rows = computed<ConversionRow[]>(() => this.plugins().flatMap((plugin) =>
    plugin.relations.map((relation) => ({ plugin, relation }))
  ));
  protected readonly visualizationOptions: readonly WebServiceVisibleOn[] = ['GRAPH', 'TABLE', 'MAP'];
  protected requestInputFormat = '';
  protected requestVisualizations: WebServiceVisibleOn[] = [];
  protected requestMessage = '';

  @Input() disabled = false;

  @Input()
  set distributionId(value: string) {
    if (value === this.currentDistributionId) {
      return;
    }

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

  protected toggleRequestVisualization(visualization: WebServiceVisibleOn, checked: boolean): void {
    this.requestVisualizations = checked
      ? Array.from(new Set([...this.requestVisualizations, visualization]))
      : this.requestVisualizations.filter((value) => value !== visualization);
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
    this.request = this.apiService.listDistributionConversionRelations(distributionId).pipe(
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
}
