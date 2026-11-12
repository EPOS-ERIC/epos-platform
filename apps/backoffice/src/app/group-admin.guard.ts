import { inject } from '@angular/core';
import { Router, type CanActivateFn } from '@angular/router';

import { CurrentUserStore } from '../stores/current-user.store';
import { GroupsStore } from '../stores/groups.store';

export const groupAdminGuard: CanActivateFn = async (route) => {
  const currentUserStore = inject(CurrentUserStore);
  const groupsStore = inject(GroupsStore);
  const router = inject(Router);
  const sectionPath = route.data['sectionPath'] as string;
  const groupId = route.queryParamMap.get('groupId');

  await Promise.all([currentUserStore.initialize(), groupsStore.initialize()]);

  const conceptBySection = {
    data: 'DATA',
    software: 'SOFTWARE',
    facilities: 'FACILITIES'
  } as const;
  const groupMatchesSection = groupsStore.groups().some((group) =>
    group.id === groupId && groupsStore.conceptName(group) === conceptBySection[sectionPath as keyof typeof conceptBySection]
  );

  const canManageGroup = !!groupId && groupMatchesSection && currentUserStore.canManageGroup(groupId);

  return canManageGroup
    ? true
    : router.createUrlTree(['/', sectionPath], { queryParams: groupId ? { groupId } : undefined });
};
