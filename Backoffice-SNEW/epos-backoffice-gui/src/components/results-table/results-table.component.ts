import { Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { RouterLink } from '@angular/router';

import type { DiscoveryItem } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { ListPaginationComponent } from '../../shared/ui/list-pagination.component';

@Component({
  selector: 'app-results-table',
  imports: [ListPaginationComponent, RouterLink],
  templateUrl: './results-table.component.html',
  styleUrl: './results-table.component.css'
})
export class ResultsTableComponent implements OnChanges {
  @Input() sectionTitle = 'Data';
  @Input() sectionPath = 'data';
  @Input() allGroups = false;
  @Input() selectedGroupId = '';
  @Input() selectedGroupName = '';
  @Input() canManageGroup = false;
  @Input() returnUrl = '/data';
  @Input() discoveryItems: DiscoveryItem[] | null = null;
  @Input() discoveryLoading = false;
  @Input() canCopyResources = false;
  @Output() copyResource = new EventEmitter<DiscoveryItem>();

  protected currentPage = 1;
  protected pageSize = 10;
  protected readonly pageSizeOptions = [10, 25, 50];
  protected readonly skeletonRows = Array.from({ length: 10 }, (_, index) => index);
  protected readonly groupIconPath = groupIconPath;

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['selectedGroupId'] || changes['discoveryItems']) {
      this.currentPage = 1;
    }
  }

  protected get paginatedItems(): DiscoveryItem[] {
    const start = (this.currentPage - 1) * this.pageSize;
    return this.discoveryItems?.slice(start, start + this.pageSize) ?? [];
  }

  protected setPage(page: number): void {
    this.currentPage = page;
  }

  protected changePageSize(pageSize: number): void {
    this.pageSize = pageSize;
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

  protected canCopy(item: DiscoveryItem): boolean {
    return this.canCopyResources && item.status === 'PUBLISHED';
  }

  protected detailLink(item: DiscoveryItem): string[] {
    return ['/', this.detailSectionPath(item), item.resource_id, item.version_id];
  }

  protected detailQueryParams(item: DiscoveryItem): Record<string, string> {
    return {
      groupId: item.group.id,
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

  private detailSectionPath(item: DiscoveryItem): string {
    if (!this.allGroups) {
      return this.sectionPath;
    }

    return item.concept === 'DATA' ? 'data' : item.concept === 'SOFTWARE' ? 'software' : 'facilities';
  }
}
