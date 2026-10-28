import { Component, EventEmitter, Input, Output } from '@angular/core';

@Component({
  selector: 'app-list-pagination',
  templateUrl: './list-pagination.component.html',
  styleUrl: './list-pagination.component.css'
})
export class ListPaginationComponent {
  @Input() currentPage = 1;
  @Input() pageSize = 10;
  @Input() totalItems = 0;
  @Input() itemLabel = 'items';
  @Input() ariaLabel = 'List pagination';
  @Input() pageSizeLabel = 'Items per page';
  @Input() pageSizeOptions: readonly number[] = [];
  @Output() currentPageChange = new EventEmitter<number>();
  @Output() pageSizeChange = new EventEmitter<number>();

  protected get totalPages(): number {
    return Math.max(1, Math.ceil(this.totalItems / this.pageSize));
  }

  protected get firstItem(): number {
    return this.totalItems ? (this.currentPage - 1) * this.pageSize + 1 : 0;
  }

  protected get lastItem(): number {
    return Math.min(this.currentPage * this.pageSize, this.totalItems);
  }

  protected setPage(page: number): void {
    this.currentPageChange.emit(Math.min(Math.max(page, 1), this.totalPages));
  }

  protected changePageSize(event: Event): void {
    this.pageSizeChange.emit(Number((event.target as HTMLSelectElement).value));
  }
}
