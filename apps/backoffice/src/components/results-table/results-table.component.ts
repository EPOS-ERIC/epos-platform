import { DatePipe } from '@angular/common';
import { Component, EventEmitter, inject, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { RouterLink } from '@angular/router';

import type { DiscoveryItem, DiscoveryResourceType, VersioningStatus } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import type { ResultsFilterSection } from '../../models/results-filters';
import { AppIconComponent } from '../../shared/ui/app-icon.component';
import { ListPaginationComponent } from '../../shared/ui/list-pagination.component';
import { MembershipRequestDialogService } from '../../services/membership-request-dialog.service';

@Component({
  selector: 'app-results-table',
  imports: [AppIconComponent, DatePipe, ListPaginationComponent, RouterLink],
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
  @Input() belongsToGroup: {status: string} = {status: 'null'};
  @Input() returnUrl = '/data';
  @Input() discoveryItems: DiscoveryItem[] | null = null;
  @Input() discoveryLoading = false;
  @Input() canCopyResources = false;
  @Input() filterByAuthor = '';
  @Input() selectedDataTypes: DiscoveryResourceType[] = [];
  @Input() selectedVersioningStatuses: VersioningStatus[] = ['PUBLISHED'];
  @Output() copyResource = new EventEmitter<DiscoveryItem>();
  @Output() editFilter = new EventEmitter<ResultsFilterSection>();
  
  protected readonly requestDialog = inject(MembershipRequestDialogService);
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
  
  protected openJoinGroupDialog() {
    this.requestDialog.open(this.selectedGroupId);
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

  protected hasNonDefaultStatusFilter(): boolean {
    return this.selectedVersioningStatuses.length !== 1
      || this.selectedVersioningStatuses[0] !== 'PUBLISHED';
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

    const concept = item.concept.name.toUpperCase();
    return concept === 'DATA' ? 'data' : concept === 'SOFTWARE' ? 'software' : 'facilities';
  }
}
