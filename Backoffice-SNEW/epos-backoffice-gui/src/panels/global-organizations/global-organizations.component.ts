import { Component, DestroyRef, HostListener, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EMPTY, catchError, finalize, of } from 'rxjs';

import type { CanonicalOrganization, CreateOrganizationRequest, ReplaceOrganizationRequest } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { ListPaginationComponent } from '../../shared/ui/list-pagination.component';

type OrganizationForm = {
  name: string;
  email: string;
  pageUrl: string;
  logoUrl: string;
  leiCode: string;
  street: string;
  locality: string;
  postalCode: string;
  country: string;
};

const emptyOrganizationForm = (): OrganizationForm => ({
  name: '',
  email: '',
  pageUrl: '',
  logoUrl: '',
  leiCode: '',
  street: '',
  locality: '',
  postalCode: '',
  country: ''
});

@Component({
  selector: 'app-global-organizations',
  imports: [FormsModule, ListPaginationComponent],
  templateUrl: './global-organizations.component.html',
  styleUrl: './global-organizations.component.css'
})
export class GlobalOrganizationsComponent {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly organizations = signal<CanonicalOrganization[]>([]);
  protected readonly organizationsLoading = signal(true);
  protected readonly organizationsError = signal('');
  protected readonly searchQuery = signal('');
  protected readonly currentPage = signal(1);
  protected readonly pageSize = signal(10);
  protected readonly pageSizeOptions = [10, 25, 50];
  protected readonly editingOrganization = signal<CanonicalOrganization | null>(null);
  protected readonly editorOpen = signal(false);
  protected readonly editorSaving = signal(false);
  protected readonly editorError = signal('');
  protected readonly pendingDelete = signal<CanonicalOrganization | null>(null);
  protected readonly deletingOrganizationId = signal('');
  protected readonly deleteError = signal('');
  protected organizationForm = emptyOrganizationForm();
  protected logoPreviewFailed = false;

  protected readonly filteredOrganizations = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const organizations = query
      ? this.organizations().filter((organization) =>
          [organization.name, organization.email, organization.lei_code, organization.address.locality, organization.address.country]
            .filter(Boolean)
            .some((value) => value!.toLowerCase().includes(query))
        )
      : this.organizations();

    return [...organizations].sort((left, right) => left.name.localeCompare(right.name));
  });
  protected readonly paginatedOrganizations = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredOrganizations().slice(start, start + this.pageSize());
  });
  private readonly totalPages = computed(() => Math.max(1, Math.ceil(this.filteredOrganizations().length / this.pageSize())));

  constructor() {
    this.loadOrganizations();
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
    this.editingOrganization.set(null);
    this.organizationForm = emptyOrganizationForm();
    this.logoPreviewFailed = false;
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

  protected openEditEditor(organization: CanonicalOrganization): void {
    this.editingOrganization.set(organization);
    this.organizationForm = {
      name: organization.name,
      email: organization.email ?? '',
      pageUrl: organization.page_url ?? '',
      logoUrl: organization.logo_url ?? '',
      leiCode: organization.lei_code ?? '',
      street: organization.address.street ?? '',
      locality: organization.address.locality ?? '',
      postalCode: organization.address.postal_code ?? '',
      country: organization.address.country ?? ''
    };
    this.logoPreviewFailed = false;
    this.editorError.set('');
    this.editorOpen.set(true);
  }

  protected closeEditor(): void {
    if (this.editorSaving()) {
      return;
    }

    this.editorOpen.set(false);
    this.editingOrganization.set(null);
    this.editorError.set('');
  }

  protected saveOrganization(): void {
    const name = this.organizationForm.name.trim();

    if (!name || this.editorSaving()) {
      return;
    }

    const editing = this.editingOrganization();
    const address = {
      street: this.toNullable(this.organizationForm.street),
      locality: this.toNullable(this.organizationForm.locality),
      postal_code: this.toNullable(this.organizationForm.postalCode),
      country: this.toNullable(this.organizationForm.country)
    };
    const common = {
      name,
      address,
      logo_url: this.toNullable(this.organizationForm.logoUrl),
      page_url: this.toNullable(this.organizationForm.pageUrl),
      email: this.toNullable(this.organizationForm.email),
      lei_code: this.toNullable(this.organizationForm.leiCode)
    };
    const request = editing
      ? this.apiService.replaceOrganization(editing.id, {
          ...common,
          identifiers: editing.identifiers.map(({ type, value }) => ({ type, value })),
          parents: editing.parents,
          contact_point_ids: editing.contact_point_ids
        } satisfies ReplaceOrganizationRequest)
      : this.apiService.createOrganization({
          ...common,
          identifiers: [],
          parents: [],
          contact_point_ids: []
        } satisfies CreateOrganizationRequest);

    this.editorSaving.set(true);
    this.editorError.set('');
    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.editorSaving.set(false)),
        catchError((error) => {
          console.error('Failed to save organization', error);
          this.editorError.set('Failed to save the organization. Check the entered values and your permissions.');
          return of(null);
        })
      )
      .subscribe((saved) => {
        if (!saved) {
          return;
        }

        this.organizations.update((organizations) =>
          editing ? organizations.map((organization) => (organization.id === saved.id ? saved : organization)) : [...organizations, saved]
        );
        this.editorOpen.set(false);
        this.editingOrganization.set(null);
        this.editorError.set('');
      });
  }

  protected updateLogoUrl(value: string): void {
    this.organizationForm.logoUrl = value;
    this.logoPreviewFailed = false;
  }

  protected openDeleteDialog(organization: CanonicalOrganization): void {
    this.pendingDelete.set(organization);
    this.deleteError.set('');
  }

  protected closeDeleteDialog(): void {
    if (this.deletingOrganizationId()) {
      return;
    }

    this.pendingDelete.set(null);
    this.deleteError.set('');
  }

  protected deleteOrganization(): void {
    const organization = this.pendingDelete();

    if (!organization || this.deletingOrganizationId()) {
      return;
    }

    this.deletingOrganizationId.set(organization.id);
    this.deleteError.set('');
    this.apiService
      .deleteOrganization(organization.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.deletingOrganizationId.set('')),
        catchError((error) => {
          console.error('Failed to delete organization', error);
          this.deleteError.set('Failed to delete this organization. It may still be referenced by other records.');
          return EMPTY;
        })
      )
      .subscribe(() => {
        this.organizations.update((organizations) => organizations.filter(({ id }) => id !== organization.id));
        this.currentPage.update((page) => Math.min(page, this.totalPages()));
        this.pendingDelete.set(null);
        this.deleteError.set('');
      });
  }

  protected organizationLocation(organization: CanonicalOrganization): string {
    return [organization.address.locality, organization.address.country].filter(Boolean).join(', ') || 'No location provided';
  }

  private loadOrganizations(): void {
    this.organizationsLoading.set(true);
    this.organizationsError.set('');
    this.apiService
      .getCanonicalOrganizations()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.organizationsLoading.set(false)),
        catchError((error) => {
          console.error('Failed to load organizations', error);
          this.organizationsError.set('Failed to load organizations. This page requires organization management access.');
          return of([] as CanonicalOrganization[]);
        })
      )
      .subscribe((organizations) => this.organizations.set(organizations));
  }

  private toNullable(value: string): string | null {
    return value.trim() || null;
  }
}
