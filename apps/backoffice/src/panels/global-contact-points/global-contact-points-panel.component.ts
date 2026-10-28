import { Component, DestroyRef, HostListener, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EMPTY, catchError, finalize, of } from 'rxjs';

import type {
  ContactPoint,
  CreateContactPointRequest,
  ReplaceContactPointRequest
} from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { AppIconComponent } from '../../shared/ui/app-icon.component';
import { ListPaginationComponent } from '../../shared/ui/list-pagination.component';

type ContactPointForm = {
  email: string;
  role: string;
};

const emptyContactPointForm = (): ContactPointForm => ({ email: '', role: '' });

@Component({
  selector: 'app-global-contact-points-panel',
  imports: [AppIconComponent, FormsModule, ListPaginationComponent],
  templateUrl: './global-contact-points-panel.component.html',
  styleUrl: './global-contact-points-panel.component.css'
})
export class GlobalContactPointsPanelComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly contactPoints = signal<ContactPoint[]>([]);
  protected readonly loading = signal(true);
  protected readonly loadError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly currentPage = signal(1);
  protected readonly pageSize = signal(10);
  protected readonly pageSizeOptions = [10, 25, 50];
  protected readonly editorOpen = signal(false);
  protected readonly editingContactPoint = signal<ContactPoint | null>(null);
  protected readonly editorSaving = signal(false);
  protected readonly editorError = signal('');
  protected readonly pendingDelete = signal<ContactPoint | null>(null);
  protected readonly deletingContactPointId = signal('');
  protected readonly deleteError = signal('');
  protected contactPointForm = emptyContactPointForm();

  protected readonly filteredContactPoints = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const contactPoints = query
      ? this.contactPoints().filter(({ email, role }) => `${email} ${role}`.toLowerCase().includes(query))
      : this.contactPoints();
    return [...contactPoints].sort((left, right) => left.email.localeCompare(right.email));
  });
  protected readonly paginatedContactPoints = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredContactPoints().slice(start, start + this.pageSize());
  });
  private readonly totalPages = computed(() => Math.max(1, Math.ceil(this.filteredContactPoints().length / this.pageSize())));

  constructor() {
    this.loadContactPoints();
  }

  @HostListener('document:keydown.escape')
  protected closeOnEscape(): void {
    if (this.pendingDelete()) {
      this.closeDeleteDialog();
    } else {
      this.closeEditor();
    }
  }

  protected openCreateEditor(): void {
    this.editingContactPoint.set(null);
    this.contactPointForm = emptyContactPointForm();
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected openEditEditor(contactPoint: ContactPoint): void {
    this.editingContactPoint.set(contactPoint);
    this.contactPointForm = { email: contactPoint.email, role: contactPoint.role };
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected closeEditor(): void {
    if (this.editorSaving()) return;
    this.editorOpen.set(false);
    this.editingContactPoint.set(null);
    this.editorError.set('');
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

  protected saveContactPoint(): void {
    const email = this.contactPointForm.email.trim();
    const role = this.contactPointForm.role.trim();
    if (!email || !role || this.editorSaving()) return;

    const editing = this.editingContactPoint();
    const payload = { email, role };
    const request = editing
      ? this.apiService.replaceContactPoint(editing.id, payload satisfies ReplaceContactPointRequest)
      : this.apiService.createContactPoint(payload satisfies CreateContactPointRequest);

    this.editorSaving.set(true);
    this.editorError.set('');
    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.editorSaving.set(false)),
        catchError((error) => {
          console.error('Failed to save contact point', error);
          this.editorError.set('Failed to save the contact point. Check the entered values and your permissions.');
          return of(null);
        })
      )
      .subscribe((saved) => {
        if (!saved) return;
        // Update contact-points list
        this.contactPoints.update((contactPoints) =>
          editing
            ? contactPoints.map((contactPoint) => contactPoint.id === saved.id ? saved : contactPoint)
            : [...contactPoints, saved]
        );
        this.editorOpen.set(false);
        this.editingContactPoint.set(null);
      });
  }

  protected openDeleteDialog(contactPoint: ContactPoint): void {
    this.pendingDelete.set(contactPoint);
    this.deleteError.set('');
  }

  protected closeDeleteDialog(): void {
    if (this.deletingContactPointId()) return;
    this.pendingDelete.set(null);
    this.deleteError.set('');
  }

  protected deleteContactPoint(): void {
    const contactPoint = this.pendingDelete();
    if (!contactPoint || this.deletingContactPointId()) return;

    this.deletingContactPointId.set(contactPoint.id);
    this.deleteError.set('');
    this.apiService.deleteContactPoint(contactPoint.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.deletingContactPointId.set('')),
        catchError((error) => {
          console.error('Failed to delete contact point', error);
          this.deleteError.set('Failed to delete this contact point. It may still be referenced by other records.');
          return EMPTY;
        })
      )
      .subscribe(() => {
        // Update contact-points list
        this.contactPoints.update((contactPoints) => contactPoints.filter(({ id }) => id !== contactPoint.id));
        this.currentPage.update((page) => Math.min(page, this.totalPages()));
        this.pendingDelete.set(null);
      });
  }

  private loadContactPoints(): void {
    this.loading.set(true);
    this.loadError.set('');
    this.apiService.getContactPoints()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false)),
        catchError((error) => {
          console.error('Failed to load contact points', error);
          this.loadError.set('Failed to load contact points. This page requires contact point management access.');
          return of([] as ContactPoint[]);
        })
      )
      .subscribe((contactPoints) => this.contactPoints.set(contactPoints));
  }
}
