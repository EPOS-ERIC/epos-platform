import { Component, DestroyRef, EventEmitter, Input, OnInit, Output, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { catchError, finalize, of } from 'rxjs';

import type { DatasetProviderForm } from '../../../../models/dataset-detail-form';
import { ApiService } from '../../../../services/api.service';

@Component({
  selector: 'app-providers',
  imports: [FormsModule],
  templateUrl: './providers.component.html',
  styleUrl: './providers.component.css'
})
export class ProvidersComponent implements OnInit {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private _model: DatasetProviderForm[] | DatasetProviderForm | null = [];

  protected readonly organizations = signal<DatasetProviderForm[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly searchTerm = signal('');
  protected readonly availableOrganizations = computed(() => {
    const search = this.searchTerm().trim().toLowerCase();
    return search
      ? this.organizations().filter((organization) => organization.name.toLowerCase().includes(search))
      : this.organizations();
  });

  @Input()
  set model(value: DatasetProviderForm[] | DatasetProviderForm | null) {
    this._model = value;
  }

  get model(): DatasetProviderForm[] | DatasetProviderForm | null {
    return this._model;
  }

  @Input() single = false;
  @Input() disabled = false;
  @Input() readonlyView = false;
  @Input() neutralAppearance = false;
  @Output() modelChange = new EventEmitter<DatasetProviderForm[] | DatasetProviderForm>();

  ngOnInit(): void {
    if (this.readonlyView) {
      return;
    }
    this.loadOrganizations();
  }

  protected selectedProviders(): DatasetProviderForm[] {
    const values = Array.isArray(this._model) ? this._model : this._model ? [this._model] : [];
    const selected = new Map<string, DatasetProviderForm>();

    for (const provider of values) {
      if (provider.id && !selected.has(provider.id)) {
        selected.set(provider.id, provider);
      }
    }

    return Array.from(selected.values());
  }

  protected isSelected(organizationId: string): boolean {
    return this.selectedProviders().some((provider) => provider.id === organizationId);
  }

  protected selectionLimitReached(): boolean {
    return this.single && this.selectedProviders().length > 0;
  }

  protected addProvider(organization: DatasetProviderForm): void {
    if (this.disabled || this.isSelected(organization.id) || this.selectionLimitReached()) {
      return;
    }

    const provider = { id: organization.id, name: organization.name };
    this.modelChange.emit(this.single ? provider : [...this.selectedProviders(), provider]);
  }

  protected removeProvider(providerId: string): void {
    if (this.disabled) {
      return;
    }

    this.modelChange.emit(
      this.single
        ? { id: '', name: '' }
        : this.selectedProviders().filter((provider) => provider.id !== providerId)
    );
  }

  protected onSearchChange(value: string): void {
    this.searchTerm.set(value);
  }

  private loadOrganizations(): void {
    this.loading.set(true);
    this.error.set(null);

    this.apiService
      .getOrganizations()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        catchError((error) => {
          console.error('Failed to load organizations', error);
          this.error.set('Failed to load organizations.');
          return of([] as DatasetProviderForm[]);
        }),
        finalize(() => this.loading.set(false))
      )
      .subscribe((organizations) => this.organizations.set(organizations));
  }
}
