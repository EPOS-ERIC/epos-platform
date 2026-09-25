import { Component, Input, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import type { Group } from '../../../generated/backofficeSchemas';
import { CurrentUserStore } from '../../stores/current-user.store';
import { CategoriesPanelComponent } from '../../panels/categories/categories-panel.component';
import { GlobalMembersPanelComponent } from '../../panels/global-members/global-members-panel.component';
import { GlobalOrganizationsComponent } from '../../panels/global-organizations/global-organizations.component';
import { GlobalRequestsPanelComponent } from '../../panels/global-requests/global-requests-panel.component';
import { AppIconComponent, type IconName } from '../../shared/ui/app-icon.component';

type GlobalManagePanel = 'requests' | 'members' | 'contact-points' | 'categories' | 'organizations';

type GlobalManageDestination = {
  panel: GlobalManagePanel;
  title: string;
  description: string;
  icon: IconName;
};

const GLOBAL_MANAGE_DESTINATIONS: readonly GlobalManageDestination[] = [
  { panel: 'requests', title: 'Requests', description: 'Review membership and role requests across all groups.', icon: 'users' },
  { panel: 'members', title: 'Members', description: 'Manage users, group access, permissions, and global notifications.', icon: 'settings' },
  { panel: 'contact-points', title: 'Contact Points', description: 'Manage the contact points available across the platform.', icon: 'user' },
  { panel: 'categories', title: 'Categories', description: 'Manage category schemes and their category hierarchies.', icon: 'tags' },
  { panel: 'organizations', title: 'Organizations', description: 'Maintain the canonical organizations used by resources.', icon: 'building' }
];

@Component({
  selector: 'app-global-manage-page',
  imports: [AppIconComponent, CategoriesPanelComponent, GlobalMembersPanelComponent, GlobalOrganizationsComponent, GlobalRequestsPanelComponent, RouterLink],
  templateUrl: './global-manage-page.component.html',
  styleUrl: './global-manage-page.component.css'
})
export class GlobalManagePageComponent {
  private readonly currentUserStore = inject(CurrentUserStore);

  @Input() groups: Group[] = [];
  @Input() groupsLoading = false;
  @Input() set panel(value: string) {
    this.activePanel.set(this.isGlobalManagePanel(value) ? value : '');
  }

  protected readonly activePanel = signal<GlobalManagePanel | ''>('');
  protected readonly canManageMemberships = computed(() => this.hasPermission('global.membership.manage'));
  protected readonly canManageRoles = computed(() => this.hasPermission('global.role.manage'));
  protected readonly canManagePermissions = computed(() => this.hasPermission('global.permission.manage'));
  protected readonly canManageNotifications = computed(() => this.hasPermission('global.notification.manage'));
  protected readonly canManageContactPoints = computed(() => this.hasPermission('global.contact_point.manage'));
  protected readonly canManageCategories = computed(() => this.hasPermission('global.category.manage'));
  protected readonly canManageCategorySchemes = computed(() => this.hasPermission('global.category_scheme.manage'));
  protected readonly canManageOrganizations = computed(() => this.hasPermission('global.organization.manage'));
  protected readonly destinations = computed(() =>
    GLOBAL_MANAGE_DESTINATIONS.filter(({ panel }) => this.canAccessPanel(panel))
  );
  protected readonly showLanding = computed(() =>
    !this.activePanel() || !this.canAccessPanel(this.activePanel() as GlobalManagePanel)
  );

  protected isPanelActive(panel: GlobalManagePanel): boolean {
    return !this.showLanding() && this.activePanel() === panel;
  }

  private canAccessPanel(panel: GlobalManagePanel): boolean {
    switch (panel) {
      case 'requests': return this.canManageMemberships() || this.canManageRoles();
      case 'members': return this.canManageMemberships() || this.canManageRoles() || this.canManagePermissions() || this.canManageNotifications();
      case 'contact-points': return this.canManageContactPoints();
      case 'categories': return this.canManageCategories() || this.canManageCategorySchemes();
      case 'organizations': return this.canManageOrganizations();
    }
  }

  private hasPermission(permission: Parameters<CurrentUserStore['hasGlobalPermission']>[0]): boolean {
    return this.currentUserStore.hasGlobalPermission(permission);
  }

  private isGlobalManagePanel(value: string): value is GlobalManagePanel {
    return GLOBAL_MANAGE_DESTINATIONS.some(({ panel }) => panel === value);
  }
}
