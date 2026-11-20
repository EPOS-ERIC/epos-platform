import { Component, DestroyRef, EventEmitter, Input, Output, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { catchError, finalize, of, Subscription } from 'rxjs';

import type { Category, ResourceVersionReference } from '../../../../../generated/backofficeSchemas';
import { ApiService } from '../../../../services/api.service';

@Component({
  selector: 'app-categories',
  imports: [FormsModule],
  templateUrl: './categories.component.html',
  styleUrl: './categories.component.css'
})
export class CategoriesComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private categoryRequest?: Subscription;
  private _groupId = '';

  protected readonly categories = signal<Category[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly searchTerm = signal('');
  protected readonly availableCategories = computed(() => {
    const search = this.searchTerm().trim().toLowerCase();
    return search
      ? this.categories().filter((category) => category.name.toLowerCase().includes(search))
      : this.categories();
  });

  @Input() model: ResourceVersionReference[] = [];
  @Input() disabled = false;
  @Input() readonlyView = false;
  @Input() neutralAppearance = false;
  @Input() embedded = false;
  @Output() modelChange = new EventEmitter<ResourceVersionReference[]>();

  @Input()
  set groupId(value: string) {
    if (value === this._groupId) {
      return;
    }

    this._groupId = value;
    this.loadCategories();
  }

  get groupId(): string {
    return this._groupId;
  }

  protected selectedCategories(): ResourceVersionReference[] {
    const selected = new Map<string, ResourceVersionReference>();

    for (const category of this.model) {
      if (category.id && !selected.has(category.id)) {
        selected.set(category.id, category);
      }
    }

    return Array.from(selected.values());
  }

  protected selectedOnlyView(): boolean {
    return this.readonlyView || this.disabled;
  }

  protected isSelected(categoryId: string): boolean {
    return this.selectedCategories().some((category) => category.id === categoryId);
  }

  protected addCategory(category: Category): void {
    if (this.disabled || this.isSelected(category.id)) {
      return;
    }

    this.modelChange.emit([...this.selectedCategories(), { id: category.id, name: category.name }]);
  }

  protected removeCategory(categoryId: string): void {
    if (this.disabled) {
      return;
    }

    this.modelChange.emit(this.selectedCategories().filter((category) => category.id !== categoryId));
  }

  protected onSearchChange(value: string): void {
    this.searchTerm.set(value);
  }

  private loadCategories(): void {
    this.categoryRequest?.unsubscribe();
    this.categories.set([]);
    this.error.set(null);

    if (!this._groupId) {
      this.loading.set(false);
      return;
    }

    this.loading.set(true);
    this.categoryRequest = this.apiService
      .getGroupCategories(this._groupId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        catchError((error) => {
          console.error('Failed to load categories', error);
          this.error.set('Failed to load categories.');
          return of([] as Category[]);
        }),
        finalize(() => this.loading.set(false))
      )
      .subscribe((categories) => this.categories.set(categories));
  }
}
