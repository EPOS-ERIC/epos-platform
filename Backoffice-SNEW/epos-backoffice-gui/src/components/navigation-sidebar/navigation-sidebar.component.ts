import { Component, DOCUMENT, EventEmitter, Input, Output, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { AppIconComponent, type IconName } from '../app-icon/app-icon.component';
import type { DiscoveryConcept, Group } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';

type NavigationItem = {
  label: string;
  route?: string;
  anchor?: string;
  icon?: IconName;
  concept?: DiscoveryConcept;
  children?: NavigationItem[];
};

export type SidebarActionVariant = 'primary' | 'ghost' | 'danger';

export type SidebarActionItem = {
  key: string;
  label: string;
  loadingLabel?: string;
  disabled?: boolean;
  loading?: boolean;
  variant?: SidebarActionVariant;
};

@Component({
  selector: 'app-navigation-sidebar',
  imports: [AppIconComponent, RouterLink],
  templateUrl: './navigation-sidebar.component.html',
  styleUrl: './navigation-sidebar.component.css'
})
export class NavigationSidebarComponent {
  private readonly document = inject(DOCUMENT);

  @Input() isAdmin = true;
  @Input() detailItems: NavigationItem[] = [];
  @Input() detailRootLabel = '';
  @Input() detailBackLink: string | string[] = '/';
  @Input() detailBackQueryParams: Record<string, string> | null = null;
  @Input() detailActions: SidebarActionItem[] = [];
  @Input() groups: Group[] = [];
  @Input() groupsLoading = false;
  @Input() currentSectionPath = '';
  @Input() selectedGroupId = '';
  @Output() detailAction = new EventEmitter<string>();

  protected readonly collapsed = signal(false);
  protected readonly expandedNodes = signal<Set<string>>(new Set());

  protected readonly items: NavigationItem[] = [
    { label: 'Data', route: '/data', icon: 'database', concept: 'DATA' },
    { label: 'Software', route: '/software', icon: 'settings', concept: 'SOFTWARE' },
    { label: 'Facilities', route: '/facilities', icon: 'building', concept: 'FACILITIES' }
  ];

  protected readonly adminItems: NavigationItem[] = [
    { label: 'Categories', route: '/admin/categories', icon: 'tags' },
    { label: 'Groups', route: '/admin/groups', icon: 'users' },
    { label: 'Organizations', route: '/admin/organizations', icon: 'building' }
  ];
  protected readonly groupIconPath = groupIconPath;

  protected get detailRootIcon(): IconName | undefined {
    if (this.detailRootLabel === 'Dataset') {
      return 'database';
    }

    if (this.detailRootLabel === 'Software Application') {
      return 'settings';
    }

    if (this.detailRootLabel === 'Software Source Code') {
      return 'building';
    }

    return undefined;
  }

  protected get logoSrc(): string {
    return '/logo-white.png';
  }

  protected toggleSidebar(): void {
    this.collapsed.update((value) => !value);
  }

  protected groupsForItem(item: NavigationItem): Group[] {
    return item.concept ? this.groups.filter((group) => group.concept === item.concept) : [];
  }

  protected isMainSectionActive(item: NavigationItem): boolean {
    return item.route === `/${this.currentSectionPath}`;
  }

  protected isAdminSectionActive(item: NavigationItem): boolean {
    return item.route === `/${this.currentSectionPath}`;
  }

  protected isNodeExpanded(anchor: string | undefined): boolean {
    return !!anchor && this.expandedNodes().has(anchor);
  }

  protected toggleNode(event: MouseEvent, anchor: string | undefined): void {
    if (!anchor) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    this.expandedNodes.update((current) => {
      const next = new Set(current);

      if (next.has(anchor)) {
        next.delete(anchor);
      } else {
        next.add(anchor);
      }

      return next;
    });
  }

  protected scrollToSectionAndExpand(event: MouseEvent, anchor: string | undefined): void {
    this.expandAncestors(anchor);
    this.expandTargetSection(anchor);
    this.scrollToSection(event, anchor);
  }

  protected scrollToSection(event: MouseEvent, anchor: string | undefined): void {
    if (!anchor) {
      return;
    }

    event.preventDefault();

    const target = this.document.getElementById(anchor);
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    this.document.location.hash = anchor;
  }

  protected emitDetailAction(key: string): void {
    this.detailAction.emit(key);
  }

  protected isDetailAnchorActive(anchor: string | undefined): boolean {
    return !!anchor && this.document.location.hash.slice(1) === anchor;
  }

  private expandAncestors(anchor: string | undefined): void {
    if (!anchor) {
      return;
    }

    const ancestors = this.findAncestorAnchors(this.detailItems, anchor);

    if (!ancestors.length) {
      return;
    }

    this.expandedNodes.update((current) => {
      const next = new Set(current);

      ancestors.forEach((ancestor) => next.add(ancestor));

      return next;
    });
  }

  private findAncestorAnchors(items: NavigationItem[], targetAnchor: string, trail: string[] = []): string[] {
    for (const item of items) {
      if (item.anchor === targetAnchor) {
        return trail;
      }

      if (item.children?.length) {
        const found = this.findAncestorAnchors(item.children, targetAnchor, [...trail, item.anchor].filter(Boolean) as string[]);

        if (found.length) {
          return found;
        }
      }
    }

    return [];
  }

  private expandTargetSection(anchor: string | undefined): void {
    if (!anchor) {
      return;
    }

    const target = this.document.getElementById(anchor);

    if (target instanceof HTMLDetailsElement) {
      target.open = true;
      return;
    }

    target?.closest('details')?.setAttribute('open', '');
  }
}
