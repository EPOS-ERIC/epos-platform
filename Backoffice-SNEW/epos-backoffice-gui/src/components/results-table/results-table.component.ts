import { Component, Input, OnChanges, SimpleChanges } from '@angular/core';
import { RouterLink } from '@angular/router';

import type { DiscoveryItem } from '../../../generated/backofficeSchemas';

@Component({
  selector: 'app-results-table',
  imports: [RouterLink],
  templateUrl: './results-table.component.html',
  styleUrl: './results-table.component.css'
})
export class ResultsTableComponent implements OnChanges {
  @Input() sectionTitle = 'Data';
  @Input() sectionPath = 'data';
  @Input() selectedGroupId = '';
  @Input() selectedGroupName = '';
  @Input() canManageGroup = false;
  @Input() returnUrl = '/data';
  @Input() discoveryItems: DiscoveryItem[] | null = null;
  @Input() discoveryLoading = false;

  protected currentPage = 1;
  protected pageSize = 10;
  protected readonly pageSizeOptions = [10, 25, 50];

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['selectedGroupId'] || changes['discoveryItems']) {
      this.currentPage = 1;
    }
  }

  protected get paginatedItems(): DiscoveryItem[] {
    const start = (this.currentPage - 1) * this.pageSize;
    return this.discoveryItems?.slice(start, start + this.pageSize) ?? [];
  }

  protected get totalPages(): number {
    return Math.max(1, Math.ceil((this.discoveryItems?.length ?? 0) / this.pageSize));
  }

  protected get firstResult(): number {
    return (this.currentPage - 1) * this.pageSize + 1;
  }

  protected get lastResult(): number {
    return Math.min(this.currentPage * this.pageSize, this.discoveryItems?.length ?? 0);
  }

  protected setPage(page: number): void {
    this.currentPage = Math.min(Math.max(page, 1), this.totalPages);
  }

  protected changePageSize(event: Event): void {
    this.pageSize = Number((event.target as HTMLSelectElement).value);
    this.currentPage = 1;
  }

  protected discoveryStatusClass(status: DiscoveryItem['status']): string {
    return `discovery-status-${status.toLowerCase()}`;
  }

  protected discoveryAuthor(item: DiscoveryItem): string {
    return item.created_by?.name ?? 'Unknown';
  }

  protected discoveryLastChange(item: DiscoveryItem): string {
    return item.updated_at;
  }

  protected detailLink(item: DiscoveryItem): string[] {
    return ['/', this.sectionPath, item.resource_id, item.version_id];
  }

  protected detailQueryParams(item: DiscoveryItem): Record<string, string> {
    return {
      groupId: this.selectedGroupId,
      dataType: item.type,
      returnUrl: this.returnUrl
    };
  }

  protected manageGroupLink(): string[] {
    return ['/', this.sectionPath, 'manage-group'];
  }

  protected manageGroupQueryParams(): Record<string, string> {
    return {
      groupId: this.selectedGroupId,
      returnUrl: this.returnUrl
    };
  }
}
