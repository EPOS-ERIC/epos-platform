import { Component, DestroyRef, HostListener, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EMPTY, catchError, finalize, of } from 'rxjs';

import type { CreateGroupRequest, Group, ReplaceGroupRequest } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { ApiService } from '../../services/api.service';
import { GroupsStore } from '../../stores/groups.store';
import { ListPaginationComponent } from '../../shared/ui/list-pagination.component';

type GroupForm = {
  name: string;
  label: string;
  concept: Group['concept'];
  position: number | null;
  description: string;
  logo: string;
  page: string;
  color: string;
};

const emptyGroupForm = (): GroupForm => ({
  name: '',
  label: '',
  concept: 'DATA',
  position: null,
  description: '',
  logo: '',
  page: '',
  color: ''
});

@Component({
  selector: 'app-global-groups-panel',
  imports: [FormsModule, ListPaginationComponent],
  templateUrl: './global-groups-panel.component.html',
  styleUrl: './global-groups-panel.component.css'
})
export class GlobalGroupsPanelComponent {
  private readonly apiService = inject(ApiService);
  private readonly groupsStore = inject(GroupsStore);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly groups = this.groupsStore.groups;
  protected readonly loading = signal(true);
  protected readonly loadError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly currentPage = signal(1);
  protected readonly pageSize = signal(10);
  protected readonly pageSizeOptions = [10, 25, 50];
  protected readonly editorOpen = signal(false);
  protected readonly editingGroup = signal<Group | null>(null);
  protected readonly editorSaving = signal(false);
  protected readonly editorError = signal('');
  protected readonly pendingDelete = signal<Group | null>(null);
  protected readonly deletingGroupId = signal('');
  protected readonly deleteError = signal('');
  protected groupForm = emptyGroupForm();

  protected readonly filteredGroups = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const groups = query
      ? this.groups().filter((group) =>
          [group.name, group.label, group.description, group.concept]
            .filter(Boolean)
            .some((value) => value!.toLowerCase().includes(query))
        )
      : this.groups();
    return [...groups].sort((left, right) => left.position - right.position || left.name.localeCompare(right.name));
  });
  protected readonly paginatedGroups = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredGroups().slice(start, start + this.pageSize());
  });
  private readonly totalPages = computed(() => Math.max(1, Math.ceil(this.filteredGroups().length / this.pageSize())));

  constructor() {
    this.loadGroups();
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    if (this.pendingDelete()) {
      this.closeDeleteDialog();
    } else {
      this.closeEditor();
    }
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

  protected openCreateEditor(): void {
    this.editingGroup.set(null);
    this.groupForm = emptyGroupForm();
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected openEditEditor(group: Group): void {
    this.editingGroup.set(group);
    this.groupForm = {
      name: group.name,
      label: group.label ?? '',
      concept: group.concept,
      position: group.position,
      description: group.description ?? '',
      logo: group.logo ?? '',
      page: group.page ?? '',
      color: group.color ?? ''
    };
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected closeEditor(): void {
    if (this.editorSaving()) return;
    this.editorOpen.set(false);
    this.editingGroup.set(null);
    this.editorError.set('');
  }

  protected saveGroup(): void {
    const name = this.groupForm.name.trim();
    if (!name || this.editorSaving()) return;

    const editing = this.editingGroup();
    const payload = {
      name,
      label: this.toNullable(this.groupForm.label),
      concept: this.groupForm.concept,
      position: this.groupForm.position ?? undefined,
      description: this.toNullable(this.groupForm.description),
      logo: this.toNullable(this.groupForm.logo),
      page: this.toNullable(this.groupForm.page),
      color: this.toNullable(this.groupForm.color)
    };
    const request = editing
      ? this.apiService.replaceGroup(editing.id, payload satisfies ReplaceGroupRequest)
      : this.apiService.createGroup(payload satisfies CreateGroupRequest);

    this.editorSaving.set(true);
    this.editorError.set('');
    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.editorSaving.set(false)),
        catchError((error) => {
          console.error('Failed to save group', error);
          this.editorError.set('Failed to save the group. Check the entered values and your permissions.');
          return of(null);
        })
      )
      .subscribe((saved) => {
        if (!saved) return;
        this.groupsStore.upsert(saved);
        this.editorOpen.set(false);
        this.editingGroup.set(null);
      });
  }

  protected openDeleteDialog(group: Group): void {
    this.pendingDelete.set(group);
    this.deleteError.set('');
  }

  protected closeDeleteDialog(): void {
    if (this.deletingGroupId()) return;
    this.pendingDelete.set(null);
    this.deleteError.set('');
  }

  protected deleteGroup(): void {
    const group = this.pendingDelete();
    if (!group || this.deletingGroupId()) return;

    this.deletingGroupId.set(group.id);
    this.deleteError.set('');
    this.apiService.deleteGroup(group.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.deletingGroupId.set('')),
        catchError((error) => {
          console.error('Failed to delete group', error);
          this.deleteError.set('Failed to delete this group. It may still contain memberships, categories, or resources.');
          return EMPTY;
        })
      )
      .subscribe(() => {
        this.groupsStore.remove(group.id);
        this.currentPage.update((page) => Math.min(page, this.totalPages()));
        this.pendingDelete.set(null);
      });
  }

  protected groupLogo(group: Group): string | null {
    return groupIconPath(group.id) ?? group.logo ?? null;
  }

  private loadGroups(): void {
    this.loading.set(true);
    this.loadError.set('');
    this.apiService.getGroups()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false)),
        catchError((error) => {
          console.error('Failed to load groups', error);
          this.loadError.set('Failed to load groups. This page requires group management access.');
          return of([] as Group[]);
        })
      )
      .subscribe((groups) => this.groupsStore.replaceAll(groups));
  }

  private toNullable(value: string): string | null {
    return value.trim() || null;
  }
}
