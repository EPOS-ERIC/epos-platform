import { Component, DestroyRef, HostListener, Input, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EMPTY, Subscription, catchError, finalize, of } from 'rxjs';

import type { Category, CreateCategoryRequest, Group, ReplaceCategoryRequest } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { groupIconPath } from '../../group-icons';
import { ListPaginationComponent } from '../../shared/ui/list-pagination.component';
import { GroupsStore } from '../../stores/groups.store';

type CategoryRow = {
  category: Category;
  depth: number;
  parentName: string;
};

@Component({
  selector: 'app-categories-panel',
  imports: [FormsModule, ListPaginationComponent],
  templateUrl: './categories-panel.component.html',
  styleUrl: './categories-panel.component.css'
})
export class CategoriesPanelComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly groupsStore = inject(GroupsStore);
  private categoryRequest?: Subscription;
  private _groups: Group[] = [];

  @Input() groupsLoading = false;
  @Input() embedded = false;
  @Input() heading = '';

  @Input()
  set groups(value: Group[]) {
    this._groups = [...value].sort((left, right) => left.position - right.position || left.name.localeCompare(right.name));

    if (this._groups.length && !this._groups.some(({ id }) => id === this.selectedGroupId())) {
      this.selectGroup(this._groups[0].id);
    }
  }

  get groups(): Group[] {
    return this._groups;
  }

  protected readonly selectedGroupId = signal('');
  protected readonly categories = signal<Category[]>([]);
  protected readonly categoriesLoading = signal(false);
  protected readonly categoriesError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly currentPage = signal(1);
  protected readonly pageSize = signal(10);
  protected readonly pageSizeOptions = [10, 25, 50];
  protected readonly editorOpen = signal(false);
  protected readonly editingCategory = signal<Category | null>(null);
  protected readonly editorSaving = signal(false);
  protected readonly editorError = signal('');
  protected readonly pendingDelete = signal<Category | null>(null);
  protected readonly deletingCategoryId = signal('');
  protected readonly deleteError = signal('');
  protected categoryName = '';
  protected parentCategoryId = '';
  protected schemeLogoFailed = false;

  protected readonly selectedGroup = computed(() => this.groups.find(({ id }) => id === this.selectedGroupId()) ?? null);

  protected conceptName(group: Group): string {
    return this.groupsStore.conceptName(group) ?? 'Unknown';
  }
  protected readonly categoryRows = computed<CategoryRow[]>(() => {
    const categories = this.categories();
    const byParent = new Map<string | null, Category[]>();
    const byId = new Map(categories.map((category) => [category.id, category]));

    for (const category of categories) {
      const parentId = category.parent_category_id && byId.has(category.parent_category_id) ? category.parent_category_id : null;
      byParent.set(parentId, [...(byParent.get(parentId) ?? []), category]);
    }

    for (const children of byParent.values()) {
      children.sort((left, right) => left.name.localeCompare(right.name));
    }

    const rows: CategoryRow[] = [];
    const visited = new Set<string>();
    const append = (category: Category, depth: number): void => {
      if (visited.has(category.id)) {
        return;
      }

      visited.add(category.id);
      rows.push({ category, depth, parentName: category.parent_category_id ? (byId.get(category.parent_category_id)?.name ?? '') : '' });
      for (const child of byParent.get(category.id) ?? []) {
        append(child, depth + 1);
      }
    };

    for (const root of byParent.get(null) ?? []) {
      append(root, 0);
    }
    for (const category of categories) {
      append(category, 0);
    }

    return rows;
  });
  protected readonly filteredCategoryRows = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    return query
      ? this.categoryRows().filter(({ category, parentName }) => `${category.name} ${parentName}`.toLowerCase().includes(query))
      : this.categoryRows();
  });
  protected readonly paginatedCategoryRows = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredCategoryRows().slice(start, start + this.pageSize());
  });
  private readonly totalPages = computed(() => Math.max(1, Math.ceil(this.filteredCategoryRows().length / this.pageSize())));

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    if (this.pendingDelete()) {
      this.closeDeleteDialog();
    } else {
      this.closeEditor();
    }
  }

  protected selectGroup(groupId: string): void {
    if (!groupId || groupId === this.selectedGroupId()) {
      return;
    }

    this.selectedGroupId.set(groupId);
    this.schemeLogoFailed = false;
    this.searchQuery.set('');
    this.currentPage.set(1);
    this.closeEditor();
    this.closeDeleteDialog();
    this.loadCategories(groupId);
  }

  protected openCreateEditor(): void {
    if (!this.selectedGroupId()) {
      return;
    }

    this.editingCategory.set(null);
    this.categoryName = '';
    this.parentCategoryId = '';
    this.editorError.set('');
    this.editorOpen.set(true);
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

  protected schemeLogo(group: Group): string | null {
    return groupIconPath(group.id) ?? group.logo ?? null;
  }

  protected openEditEditor(category: Category): void {
    this.editingCategory.set(category);
    this.categoryName = category.name;
    this.parentCategoryId = category.parent_category_id ?? '';
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected closeEditor(): void {
    if (this.editorSaving()) {
      return;
    }

    this.editorOpen.set(false);
    this.editingCategory.set(null);
    this.editorError.set('');
  }

  protected parentOptions(): Category[] {
    const editingId = this.editingCategory()?.id;

    if (!editingId) {
      return [...this.categories()].sort((left, right) => left.name.localeCompare(right.name));
    }

    const excluded = new Set<string>([editingId]);
    let changed = true;
    while (changed) {
      changed = false;
      for (const category of this.categories()) {
        if (category.parent_category_id && excluded.has(category.parent_category_id) && !excluded.has(category.id)) {
          excluded.add(category.id);
          changed = true;
        }
      }
    }

    return this.categories()
      .filter(({ id }) => !excluded.has(id))
      .sort((left, right) => left.name.localeCompare(right.name));
  }

  protected saveCategory(): void {
    const groupId = this.selectedGroupId();
    const name = this.categoryName.trim();

    if (!groupId || !name || this.editorSaving()) {
      return;
    }

    const editing = this.editingCategory();
    const parentCategoryId = this.parentCategoryId || null;
    const request = editing
      ? this.apiService.replaceGroupCategory(groupId, editing.id, { name, parent_category_id: parentCategoryId } satisfies ReplaceCategoryRequest)
      : this.apiService.createGroupCategory(groupId, { name, parent_category_id: parentCategoryId } satisfies CreateCategoryRequest);

    this.editorSaving.set(true);
    this.editorError.set('');
    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.editorSaving.set(false)),
        catchError((error) => {
          console.error('Failed to save category', error);
          this.editorError.set('Failed to save the category. Check for duplicate names, invalid parents, or missing permissions.');
          return of(null);
        })
      )
      .subscribe((saved) => {
        if (!saved) {
          return;
        }

        this.editorOpen.set(false);
        this.editingCategory.set(null);
        this.loadCategories(groupId);
      });
  }

  protected openDeleteDialog(category: Category): void {
    this.pendingDelete.set(category);
    this.deleteError.set('');
  }

  protected closeDeleteDialog(): void {
    if (this.deletingCategoryId()) {
      return;
    }

    this.pendingDelete.set(null);
    this.deleteError.set('');
  }

  protected descendantCount(categoryId: string): number {
    const descendants = new Set<string>();
    let changed = true;
    while (changed) {
      changed = false;
      for (const category of this.categories()) {
        if (category.parent_category_id && (category.parent_category_id === categoryId || descendants.has(category.parent_category_id)) && !descendants.has(category.id)) {
          descendants.add(category.id);
          changed = true;
        }
      }
    }
    return descendants.size;
  }

  protected deleteCategory(): void {
    const groupId = this.selectedGroupId();
    const category = this.pendingDelete();

    if (!groupId || !category || this.deletingCategoryId()) {
      return;
    }

    this.deletingCategoryId.set(category.id);
    this.deleteError.set('');
    this.apiService
      .deleteGroupCategory(groupId, category.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.deletingCategoryId.set('')),
        catchError((error) => {
          console.error('Failed to delete category', error);
          this.deleteError.set('Failed to delete this category. Check your permissions and try again.');
          return EMPTY;
        })
      )
      .subscribe(() => {
        this.pendingDelete.set(null);
        this.loadCategories(groupId);
      });
  }

  private loadCategories(groupId: string): void {
    this.categoryRequest?.unsubscribe();
    this.categories.set([]);
    this.categoriesError.set('');
    this.categoriesLoading.set(true);
    this.categoryRequest = this.apiService
      .getGroupCategories(groupId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.categoriesLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load category scheme', error);
          this.categoriesError.set('Failed to load categories for this group.');
          return of([] as Category[]);
        })
      )
      .subscribe((categories) => {
        this.categories.set(categories);
        this.currentPage.update((page) => Math.min(page, this.totalPages()));
      });
  }
}
