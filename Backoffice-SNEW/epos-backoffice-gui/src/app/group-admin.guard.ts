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
    group.id === groupId && group.concept === conceptBySection[sectionPath as keyof typeof conceptBySection]
  );

  const canManageGroup = groupMatchesSection && (currentUserStore.user()?.memberships.some((membership) =>
    membership.group.id === groupId
    && membership.status === 'ACTIVE'
    && membership.roles.includes('GROUP_ADMIN')
  ) ?? false);

  return canManageGroup
    ? true
    : router.createUrlTree(['/', sectionPath], { queryParams: groupId ? { groupId } : undefined });
};
