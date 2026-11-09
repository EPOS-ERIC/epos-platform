import { NgTemplateOutlet } from '@angular/common';
import { Component, DOCUMENT, DestroyRef, EventEmitter, Input, Output, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AppIconComponent, type IconName } from '../../shared/ui/app-icon.component';
import type { DiscoveryConcept, Group } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import type { ResourceActionItem, ResourceActionKey } from '../../models/resource-actions';

type NavigationItem = {
  label: string;
  route?: string;
  anchor?: string;
  icon?: IconName;
  concept?: DiscoveryConcept;
  children?: NavigationItem[];
};

@Component({
  selector: 'app-navigation-sidebar',
  imports: [AppIconComponent, NgTemplateOutlet, RouterLink],
  templateUrl: './navigation-sidebar.component.html',
  styleUrl: './navigation-sidebar.component.css'
})
export class NavigationSidebarComponent {
  private readonly document = inject(DOCUMENT);
  private readonly destroyRef = inject(DestroyRef);
  private readonly router = inject(Router);
  private detailNavigationItems: NavigationItem[] = [];
  private scrollFrame?: number;

  @Input() canAccessGlobalManage = false;
  @Input()
  set detailItems(value: NavigationItem[]) {
    this.detailNavigationItems = value;
    this.scheduleActiveDetailSync();
  }
  get detailItems(): NavigationItem[] {
    return this.detailNavigationItems;
  }
  @Input() detailSectionLabel = 'Detail';
  @Input() detailRootLabel = '';
  @Input() detailActions: ResourceActionItem[] = [];
  @Input() groups: Group[] = [];
  @Input() groupsLoading = false;
  @Input()
  set currentSectionPath(value: string) {
    this.currentSection = value;

    const route = `/${value}`;
    const item = this.items.find((candidate) => candidate.route === route);
    if (item) {
      this.expandedMainRoute.set(item.concept ? route : null);
    }
  }
  get currentSectionPath(): string {
    return this.currentSection;
  }
  @Input() selectedGroupId = '';
  @Input() globalManageReturnUrl = '/data';
  @Input() accountReturnUrl = '/data';
  @Output() detailAction = new EventEmitter<ResourceActionKey>();

  protected readonly collapsed = signal(false);
  protected readonly activeDetailAnchor = signal<string | null>(null);
  protected readonly detailRootExpanded = signal(true);
  protected readonly expandedNodes = signal<Set<string>>(new Set());
  protected readonly expandedMainRoute = signal<string | null>(null);

  protected readonly items: NavigationItem[] = [
    { label: 'Data', route: '/data', icon: 'database', concept: 'DATA' },
    { label: 'Software', route: '/software', icon: 'code', concept: 'SOFTWARE' },
    { label: 'Facilities', route: '/facilities', icon: 'building', concept: 'FACILITIES' },
    { label: 'ALL', route: '/all', icon: 'apps' }
  ];

  private readonly allAdminItems: NavigationItem[] = [
    { label: 'Manage', route: '/global-manage', icon: 'settings' }
  ];
  protected readonly groupIconPath = groupIconPath;
  private currentSection = '';

  constructor() {
    const onScroll = () => this.scheduleActiveDetailSync();
    this.document.addEventListener('scroll', onScroll, true);
    this.destroyRef.onDestroy(() => {
      this.document.removeEventListener('scroll', onScroll, true);
      if (this.scrollFrame !== undefined) {
        this.document.defaultView?.cancelAnimationFrame(this.scrollFrame);
      }
    });
  }

  protected get detailRootIcon(): IconName | undefined {
    if (this.detailRootLabel === 'Dataset') {
      return 'database';
    }

    if (this.detailRootLabel === 'Software Application') {
      return 'code';
    }

    if (this.detailRootLabel === 'Software Source Code') {
      return 'code';
    }

    return undefined;
  }

  protected get detailRootItems(): NavigationItem[] {
    return this.detailItems.filter((item) => !this.isDistributionItem(item));
  }

  protected get detailSiblingItems(): NavigationItem[] {
    return this.detailItems.filter((item) => this.isDistributionItem(item));
  }

  protected get adminItems(): NavigationItem[] {
    return this.canAccessGlobalManage ? this.allAdminItems : [];
  }

  protected get logoSrc(): string {
    return 'logo-white.png';
  }

  protected toggleSidebar(): void {
    this.collapsed.update((value) => !value);
  }

  protected toggleDetailRoot(): void {
    this.detailRootExpanded.update((value) => !value);
  }

  private isDistributionItem(item: NavigationItem): boolean {
    return item.anchor === 'distributions' || item.anchor === 'compare-distributions';
  }

  protected groupsForItem(item: NavigationItem): Group[] {
    return item.concept ? this.groups.filter((group) => group.concept === item.concept) : [];
  }

  protected isMainSectionActive(item: NavigationItem): boolean {
    return item.route === `/${this.currentSectionPath}`;
  }

  protected isMainItemExpanded(item: NavigationItem): boolean {
    return this.expandedMainRoute() === item.route;
  }

  protected expandMainItem(item: NavigationItem): void {
    this.expandedMainRoute.set(item.concept ? (item.route ?? null) : null);
  }

  protected toggleMainItem(event: MouseEvent, item: NavigationItem): void {
    event.preventDefault();
    event.stopPropagation();

    const expanding = this.expandedMainRoute() !== item.route;
    this.expandedMainRoute.set(expanding ? (item.route ?? null) : null);
    if (expanding && item.route && !this.isMainSectionActive(item)) {
      void this.router.navigateByUrl(item.route);
    }
  }

  protected isAdminSectionActive(item: NavigationItem): boolean {
    return item.route === `/${this.currentSectionPath}`;
  }

  protected adminQueryParams(item: NavigationItem): Record<string, string> | null {
    return item.route === '/global-manage'
      ? { returnUrl: this.globalManageReturnUrl }
      : null;
  }

  protected membershipsQueryParams(): Record<string, string> {
    return {
      ...(this.selectedGroupId ? { groupId: this.selectedGroupId } : {}),
      returnUrl: this.accountReturnUrl
    };
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
    this.setActiveDetailAnchor(anchor);
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    this.document.location.hash = anchor;
  }

  protected emitDetailAction(key: ResourceActionKey): void {
    this.detailAction.emit(key);
  }

  protected isDetailAnchorActive(anchor: string | undefined): boolean {
    return !!anchor && (this.activeDetailAnchor() ?? this.document.location.hash.slice(1)) === anchor;
  }

  private scheduleActiveDetailSync(): void {
    const window = this.document.defaultView;
    if (!window || this.scrollFrame !== undefined || !this.detailItems.length) {
      return;
    }

    this.scrollFrame = window.requestAnimationFrame(() => {
      this.scrollFrame = undefined;
      this.syncActiveDetailAnchor();
    });
  }

  private syncActiveDetailAnchor(): void {
    const anchors = this.flattenDetailAnchors(this.detailItems);
    const positions = anchors
      .map((anchor) => ({ anchor, top: this.document.getElementById(anchor)?.getBoundingClientRect().top }))
      .filter((entry): entry is { anchor: string; top: number } => entry.top !== undefined);
    if (!positions.length) {
      return;
    }

    const activationOffset = 32;
    const passed = positions.filter((entry) => entry.top <= activationOffset);
    const active = passed.length
      ? passed.reduce((closest, entry) => entry.top > closest.top ? entry : closest)
      : positions.reduce((closest, entry) => entry.top < closest.top ? entry : closest);
    this.setActiveDetailAnchor(active.anchor);
  }

  private setActiveDetailAnchor(anchor: string): void {
    if (this.activeDetailAnchor() === anchor) {
      return;
    }

    this.activeDetailAnchor.set(anchor);
    this.expandAncestors(anchor);
    if (this.detailRootItems.some((item) => this.containsAnchor(item, anchor))) {
      this.detailRootExpanded.set(true);
    }
  }

  private flattenDetailAnchors(items: NavigationItem[]): string[] {
    return items.flatMap((item) => [
      ...(item.anchor ? [item.anchor] : []),
      ...this.flattenDetailAnchors(item.children ?? [])
    ]);
  }

  private containsAnchor(item: NavigationItem, anchor: string): boolean {
    return item.anchor === anchor || !!item.children?.some((child) => this.containsAnchor(child, anchor));
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
