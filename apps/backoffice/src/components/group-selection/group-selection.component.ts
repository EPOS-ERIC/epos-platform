import { Component, Input, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import type { DiscoveryConcept, Group } from '../../../generated/backofficeSchemas';
import { groupIconPath } from '../../group-icons';
import { GroupsStore } from '../../stores/groups.store';

@Component({
  selector: 'app-group-selection',
  imports: [RouterLink],
  templateUrl: './group-selection.component.html',
  styleUrl: './group-selection.component.css'
})
export class GroupSelectionComponent {
  private readonly groupsStore = inject(GroupsStore);
  @Input() groups: Group[] = [];
  @Input() groupsLoading = false;
  @Input() concept: DiscoveryConcept = 'DATA';
  @Input() sectionPath = 'data';
  @Input() sectionTitle = 'Data';

  protected get visibleGroups(): Group[] {
    return this.groups
      .filter((group) => this.groupsStore.conceptName(group) === this.concept)
      .sort((left, right) => left.position - right.position || left.name.localeCompare(right.name));
  }

  protected groupLogo(group: Group): string | null {
    return groupIconPath(group.id) ?? group.logo ?? null;
  }
}
