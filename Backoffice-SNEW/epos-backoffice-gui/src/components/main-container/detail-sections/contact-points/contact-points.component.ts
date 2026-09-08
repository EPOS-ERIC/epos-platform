import { Component, DestroyRef, EventEmitter, Input, OnInit, Output, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { catchError, finalize, of } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

import type { ContactPoint } from '../../../../../generated/backofficeSchemas';
import { ApiService } from '../../../../services/api.service';

@Component({
  selector: 'app-contact-points',
  imports: [FormsModule],
  templateUrl: './contact-points.component.html',
  styleUrl: './contact-points.component.css'
})
export class ContactPointsComponent implements OnInit {
  private readonly apiService = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  private _model: string[] | null = null;

  protected readonly contactPoints = signal<ContactPoint[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly searchTerm = signal('');
  protected readonly roleFilter = signal('all');
  protected readonly availableContactPoints = computed(() => {
    const search = this.searchTerm().trim().toLowerCase();
    const roleFilter = this.roleFilter();

    return this.contactPoints().filter((contactPoint) => {
      if (roleFilter !== 'all' && contactPoint.role !== roleFilter) {
        return false;
      }

      if (!search) {
        return true;
      }

      return [contactPoint.email, contactPoint.role].some((value) => value.toLowerCase().includes(search));
    });
  });
  protected readonly availableRoles = computed(() => {
    const roles = new Set(this.contactPoints().map((contactPoint) => contactPoint.role).filter((role) => role.trim().length > 0));
    return Array.from(roles).sort((left, right) => left.localeCompare(right));
  });

  @Input() disabled = false;
  @Input() editable = false;
  @Output() modelChange = new EventEmitter<string[]>();

  @Input()
  set model(value: string[] | null) {
    this._model = value;
  }

  get model(): string[] | null {
    return this._model;
  }

  ngOnInit(): void {
    this.loadContactPoints();
  }

  protected selectedContactPoints(): ContactPoint[] {
    const selectedIds = new Set(this._model ?? []);
    const selectedOrder = this._model ?? [];

    return selectedOrder
      .map((id) => this.contactPoints().find((contactPoint) => contactPoint.id === id))
      .filter((contactPoint): contactPoint is ContactPoint => !!contactPoint && selectedIds.has(contactPoint.id));
  }

  protected hasSelection(): boolean {
    return this.selectedContactPoints().length > 0;
  }

  protected hasAvailableContactPoints(): boolean {
    return this.availableContactPoints().length > 0;
  }

  protected isSelected(contactPointId: string): boolean {
    return (this._model ?? []).includes(contactPointId);
  }

  protected addContactPoint(contactPointId: string): void {
    if (this.disabled || !this.editable) {
      return;
    }

    const current = this._model ?? [];

    if (current.includes(contactPointId)) {
      return;
    }

    this.modelChange.emit([...current, contactPointId]);
  }

  protected removeContactPoint(contactPointId: string): void {
    if (this.disabled || !this.editable) {
      return;
    }

    this.modelChange.emit((this._model ?? []).filter((id) => id !== contactPointId));
  }

  protected contactPointDescription(contactPoint: ContactPoint): string {
    return contactPoint.role;
  }

  protected onSearchChange(value: string): void {
    this.searchTerm.set(value);
  }

  protected onRoleFilterChange(value: string): void {
    this.roleFilter.set(value);
  }

  private loadContactPoints(): void {
    this.loading.set(true);
    this.error.set(null);

    this.apiService
      .getContactPoints()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        catchError((error) => {
          console.error('Failed to load contact points', error);
          this.error.set('Failed to load contact points.');
          return of([] as ContactPoint[]);
        }),
        finalize(() => this.loading.set(false))
      )
      .subscribe((items) => {
        this.contactPoints.set(items);
      });
  }
}
