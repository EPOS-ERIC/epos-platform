import { Component, DestroyRef, HostListener, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EMPTY, catchError, finalize, of } from 'rxjs';

import type {
  ConversionRuntime,
  ConversionRelation,
  CreateConversionPluginRequest,
  ManagedConversionPlugin,
  ReplaceConversionPluginRequest
} from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { ListPaginationComponent } from '../../shared/ui/list-pagination.component';

type PluginForm = {
  name: string;
  description: string;
  repository: string;
  branch: string;
  runtime: ConversionRuntime;
  arguments: string;
  enabled: boolean;
};

const emptyPluginForm = (): PluginForm => ({
  name: '',
  description: '',
  repository: '',
  branch: 'main',
  runtime: 'python',
  arguments: '',
  enabled: true
});

type DistributionRelationGroup = {
  distributionId: string;
  relations: ConversionRelation[];
};

@Component({
  selector: 'app-conversion-plugins-panel',
  imports: [FormsModule, ListPaginationComponent],
  templateUrl: './conversion-plugins-panel.component.html',
  styleUrl: './conversion-plugins-panel.component.css'
})
export class ConversionPluginsPanelComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly plugins = signal<ManagedConversionPlugin[]>([]);
  protected readonly relations = signal<ConversionRelation[]>([]);
  protected readonly loading = signal(true);
  protected readonly loadError = signal('');
  protected readonly relationsLoading = signal(true);
  protected readonly relationsError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly currentPage = signal(1);
  protected readonly pageSize = signal(10);
  protected readonly pageSizeOptions = [10, 25, 50];
  protected readonly runtimes: ConversionRuntime[] = ['python', 'java', 'binary'];
  protected readonly editorOpen = signal(false);
  protected readonly editingPlugin = signal<ManagedConversionPlugin | null>(null);
  protected readonly editorSaving = signal(false);
  protected readonly editorError = signal('');
  protected readonly syncingPluginId = signal('');
  protected readonly syncError = signal('');
  protected readonly pendingDelete = signal<ManagedConversionPlugin | null>(null);
  protected readonly deletingPluginId = signal('');
  protected readonly deleteError = signal('');
  protected readonly relationsPlugin = signal<ManagedConversionPlugin | null>(null);
  protected pluginForm = emptyPluginForm();

  protected readonly filteredPlugins = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const plugins = query
      ? this.plugins().filter((plugin) =>
          [plugin.name, plugin.description, plugin.repository, plugin.branch, plugin.runtime]
            .some((value) => value.toLowerCase().includes(query))
        )
      : this.plugins();
    return [...plugins].sort((left, right) => left.name.localeCompare(right.name));
  });
  protected readonly paginatedPlugins = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredPlugins().slice(start, start + this.pageSize());
  });
  protected readonly distributionRelationGroups = computed<DistributionRelationGroup[]>(() => {
    const plugin = this.relationsPlugin();
    if (!plugin) return [];

    const groups = new Map<string, ConversionRelation[]>();
    this.relations().filter(({ plugin_id }) => plugin_id === plugin.id).forEach((relation) => {
      groups.set(relation.distribution_id, [...(groups.get(relation.distribution_id) ?? []), relation]);
    });
    return Array.from(groups, ([distributionId, relations]) => ({
      distributionId,
      relations: [...relations].sort((left, right) =>
        left.input_format.localeCompare(right.input_format) || left.output_format.localeCompare(right.output_format)
      )
    })).sort((left, right) => left.distributionId.localeCompare(right.distributionId));
  });

  constructor() {
    this.loadPlugins();
    this.loadRelations();
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    if (this.pendingDelete()) {
      this.closeDeleteDialog();
    } else if (this.editorOpen()) {
      this.closeEditor();
    } else {
      this.closeRelationsDialog();
    }
  }

  protected openCreateEditor(): void {
    this.editingPlugin.set(null);
    this.pluginForm = emptyPluginForm();
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected openEditEditor(plugin: ManagedConversionPlugin): void {
    this.editingPlugin.set(plugin);
    this.pluginForm = {
      name: plugin.name,
      description: plugin.description,
      repository: plugin.repository,
      branch: plugin.branch,
      runtime: plugin.runtime,
      arguments: plugin.arguments.join('\n'),
      enabled: plugin.enabled
    };
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected closeEditor(): void {
    if (this.editorSaving()) return;
    this.editorOpen.set(false);
    this.editingPlugin.set(null);
    this.editorError.set('');
  }

  protected savePlugin(): void {
    const editing = this.editingPlugin();
    const commonPayload = {
      name: this.pluginForm.name.trim(),
      description: this.pluginForm.description.trim(),
      repository: this.pluginForm.repository.trim(),
      branch: this.pluginForm.branch.trim(),
      runtime: this.pluginForm.runtime,
      arguments: this.pluginForm.arguments.split('\n').map((argument) => argument.trim()).filter(Boolean)
    };
    if (!commonPayload.name || !commonPayload.repository || !commonPayload.branch || this.editorSaving()) return;

    const request = editing
      ? this.apiService.replaceConversionPlugin(editing.id, {
          ...commonPayload,
          enabled: this.pluginForm.enabled
        } satisfies ReplaceConversionPluginRequest)
      : this.apiService.createConversionPlugin(commonPayload satisfies CreateConversionPluginRequest);

    this.editorSaving.set(true);
    this.editorError.set('');
    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.editorSaving.set(false)),
        catchError((error) => {
          console.error('Failed to save conversion plugin', error);
          this.editorError.set('Failed to save the conversion plugin. Check the entered values and try again.');
          return of(null);
        })
      )
      .subscribe((saved) => {
        if (!saved) return;
        this.upsertPlugin(saved);
        this.editorOpen.set(false);
        this.editingPlugin.set(null);
      });
  }

  protected onSearchChange(value: string): void {
    this.searchQuery.set(value);
    this.currentPage.set(1);
  }

  protected setPage(page: number): void {
    this.currentPage.set(page);
  }

  protected changePageSize(pageSize: number): void {
    this.pageSize.set(pageSize);
    this.currentPage.set(1);
  }

  protected relationCount(pluginId: string): number {
    return this.relations().filter(({ plugin_id }) => plugin_id === pluginId).length;
  }

  protected distributionCount(pluginId: string): number {
    return new Set(
      this.relations().filter(({ plugin_id }) => plugin_id === pluginId).map(({ distribution_id }) => distribution_id)
    ).size;
  }

  protected openRelationsDialog(plugin: ManagedConversionPlugin): void {
    this.relationsPlugin.set(plugin);
  }

  protected closeRelationsDialog(): void {
    this.relationsPlugin.set(null);
  }

  protected syncPlugin(plugin: ManagedConversionPlugin): void {
    if (this.syncingPluginId()) return;
    this.syncingPluginId.set(plugin.id);
    this.syncError.set('');
    this.apiService.syncConversionPlugin(plugin.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.syncingPluginId.set('')),
        catchError((error) => {
          console.error('Failed to synchronize conversion plugin', error);
          this.syncError.set(`Failed to synchronize ${plugin.name}.`);
          return of(null);
        })
      )
      .subscribe((synced) => {
        if (synced) this.upsertPlugin(synced);
      });
  }

  protected openDeleteDialog(plugin: ManagedConversionPlugin): void {
    this.pendingDelete.set(plugin);
    this.deleteError.set('');
  }

  protected closeDeleteDialog(): void {
    if (this.deletingPluginId()) return;
    this.pendingDelete.set(null);
    this.deleteError.set('');
  }

  protected deletePlugin(): void {
    const plugin = this.pendingDelete();
    if (!plugin || this.deletingPluginId()) return;
    this.deletingPluginId.set(plugin.id);
    this.deleteError.set('');
    this.apiService.deleteConversionPlugin(plugin.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.deletingPluginId.set('')),
        catchError((error) => {
          console.error('Failed to delete conversion plugin', error);
          this.deleteError.set('Failed to remove this conversion plugin. Check whether it still has active relations.');
          return EMPTY;
        })
      )
      .subscribe(() => {
        this.plugins.update((plugins) => plugins.filter(({ id }) => id !== plugin.id));
        this.relations.update((relations) => relations.filter(({ plugin_id }) => plugin_id !== plugin.id));
        this.currentPage.update((page) => Math.min(page, Math.max(1, Math.ceil(this.filteredPlugins().length / this.pageSize()))));
        this.pendingDelete.set(null);
      });
  }

  private loadPlugins(): void {
    this.loading.set(true);
    this.loadError.set('');
    this.apiService.listConversionPlugins()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false)),
        catchError((error) => {
          console.error('Failed to load conversion plugins', error);
          this.loadError.set('Failed to load conversion plugins. Check your permissions and try again.');
          return of([] as ManagedConversionPlugin[]);
        })
      )
      .subscribe((plugins) => this.plugins.set(plugins));
  }

  private loadRelations(): void {
    this.relationsLoading.set(true);
    this.relationsError.set('');
    this.apiService.listConversionRelations()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.relationsLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load conversion relations', error);
          this.relationsError.set('Conversion relations could not be loaded.');
          return of([] as ConversionRelation[]);
        })
      )
      .subscribe((relations) => this.relations.set(relations));
  }

  private upsertPlugin(plugin: ManagedConversionPlugin): void {
    this.plugins.update((plugins) =>
      plugins.some(({ id }) => id === plugin.id)
        ? plugins.map((current) => current.id === plugin.id ? plugin : current)
        : [...plugins, plugin]
    );
  }
}
